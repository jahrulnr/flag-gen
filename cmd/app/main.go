package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flag-gen/application/jobs"
	deliveryhttp "flag-gen/delivery/http"
	"flag-gen/delivery/http/middleware"
	"flag-gen/infrastructure/broker"
	"flag-gen/infrastructure/di"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager/integrations/lmgin"
	helmet "github.com/danielkov/gin-helmet/ginhelmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {
	app, err := di.InitializeApplication()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	// Setup standard log
	appLog := logmanager.NewApplication(
		logmanager.WithDebug(),
		logmanager.WithAppName("flag-gen"),
		logmanager.WithExposeHeaders("Referer", "X-Forwarded-For", "User-Agent"),
		logmanager.WithMaskingConfig([]logmanager.MaskingConfig{
			{Field: "Authorization", Type: logmanager.PartialMask, ShowFirst: 4, ShowLast: 4},
			{Field: "Api_key", Type: logmanager.PartialMask, ShowFirst: 4, ShowLast: 4},
			{Field: "X-Signature", Type: logmanager.PartialMask, ShowFirst: 4, ShowLast: 4},
			{Field: "X-Msisdn", Type: logmanager.PartialMask, ShowFirst: 4, ShowLast: 4},
			{Field: "x-token", Type: logmanager.PartialMask, ShowFirst: 4, ShowLast: 4},
			{Field: "token", Type: logmanager.HideMask},
			{Field: "base64", Type: logmanager.HideMask},
			{Field: "authID", Type: logmanager.HideMask},
			{Field: "auth_id", Type: logmanager.HideMask},
		}),
	)

	router := gin.New()
	router.MaxMultipartMemory = 10 << 20 // 10 MiB
	router.RemoveExtraSlash = true

	if app.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Credentialed CORS cannot use AllowOrigins: *; use AllowOriginFunc (see CORS_ALLOWED_ORIGINS, config.IsCORSOriginAllowed).
	cfg := app.Config
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return cfg.IsCORSOriginAllowed(origin)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"User-Agent", "Content-Type", "X-Forwarded-For", "X-Correlation-ID", "X-CSRF-Token", "Idempotency-Key", "If-Match"},
		ExposeHeaders:    []string{"Content-Length", "X-Correlation-ID", "X-Schema-Migration-State"},
		AllowCredentials: true,
	}))
	router.Use(helmet.Default())
	// Do not gzip /app/* — dashboard SSR is reverse-proxied from Node/Express; wrapping that
	// stream breaks Content-Encoding / Content-Length (browser "Content Encoding Error").
	router.Use(gzip.Gzip(gzip.BestCompression, gzip.WithExcludedPaths([]string{"/app"})))

	// Limit the request body size to 2MB (2 * 1024 * 1024 bytes)
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2*1024*1024)
		c.Next()
	})

	router.Use(lmgin.Middleware(appLog))
	router.Use(middleware.CorrelationIDMiddleware())
	router.Use(middleware.RecoverMiddleware())
	router.Use(middleware.MigrationGate(app.System))

	// Register routes
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": true, "message": "OK Service Running"})
	})
	csrf := middleware.NewCSRFProtection(app.Config.Security.CSRFSecret, app.Config.Security.CSRFSessionKey)
	deliveryhttp.RegisterV1Routes(router, app)
	deliveryhttp.RegisterAPIRoutes(router, app, csrf)
	if err := deliveryhttp.RegisterDashboardSSRProxy(router, app.Config, csrf); err != nil {
		log.Fatalf("failed to register dashboard SSR proxy: %v", err)
	}

	// not found route
	router.NoRoute(func(ctx *gin.Context) {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"status":  false,
			"message": "The page you are looking for doesn't exist or may have been moved.",
		})
	})

	srv := &http.Server{
		Addr:              ":" + app.Config.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: app.Config.Server.ReadHeaderTimeout,
	}

	// Graceful shutdown setup
	go gracefulShutdown(srv, 5*time.Second)
	if app.Config.Outbox.Enabled && app.OutboxDispatcher != nil {
		go func() {
			_ = app.OutboxDispatcher.Run(context.Background())
		}()
	}

	if app.Config.BroadcastConsumerEnabled && app.Config.RabbitMQ.AMQPURL != "" && app.BroadcastHub != nil {
		go broker.DialBackoff(context.Background(), app.Config.RabbitMQ.AMQPURL, app.Config.RabbitMQ.ExchangeName, app.BroadcastHub, app.ScopedFlags)
	}

	if app.MySQL != nil && app.Config.CleanupIntervalSeconds > 0 && app.Config.SoftDeleteRetentionDays > 0 {
		ret := time.Duration(app.Config.SoftDeleteRetentionDays) * 24 * time.Hour
		iv := time.Duration(app.Config.CleanupIntervalSeconds) * time.Second
		go jobs.StartSoftDeleteCleanup(context.Background(), app.MySQL.SQL(), app.Redis, iv, ret, app.Config.CleanupLockTTL, app.Config.CleanupPruneMaxPerScope)
	}

	if app.Config.Snapshot.SchedulerEnabled && app.SnapshotPointer != nil && app.Snapshots != nil && app.Singleflight != nil {
		go jobs.StartSnapshotConsistencyScheduler(context.Background(), app.Config, app.Redis, app.Snapshots, app.SnapshotPointer, app.Singleflight)
	}

	if app.Config.TLS.Enabled {
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP521, tls.CurveP384, tls.CurveP256,
			},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		}

		log.Println("Server is running with SSL on port " + app.Config.Server.Port)
		if err := srv.ListenAndServeTLS(app.Config.TLSCertPath(), app.Config.TLSKeyPath()); err != nil && err != http.ErrServerClosed {
			log.Fatalf("SSL Server failed: %v", err)
		}
	} else {
		log.Println("Server is running without SSL on port " + app.Config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Non-SSL Server failed: %v", err)
		}
	}
}

func gracefulShutdown(srv *http.Server, timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	} else {
		log.Println("Server exited properly")
	}
}
