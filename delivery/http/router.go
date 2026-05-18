package http

import (
	"strings"

	"flag-gen/application/usecase"
	"flag-gen/delivery/http/handler"
	"flag-gen/delivery/http/middleware"
	"flag-gen/infrastructure/di"

	"github.com/gin-gonic/gin"
)

func RegisterV1Routes(r *gin.Engine, app *di.ApplicationState) {
	RegisterRoutes(r.Group("/v1"), app, nil)
}

func RegisterAPIRoutes(r *gin.Engine, app *di.ApplicationState, csrf *middleware.CSRFProtection) {
	api := r.Group("/api")
	api.Use(csrf.Middleware())
	authHandler := handler.NewAuthHandler(app.MySQL.SQL())
	api.POST("/auth/login", authHandler.Login)
	api.Use(middleware.DashboardSession(app.MySQL.SQL(), handler.DashboardSessionCookie))
	api.GET("/auth/me", authHandler.Me)
	api.POST("/auth/logout", authHandler.Logout)
	api.Use(middleware.DashboardPermission())
	RegisterRoutes(api, app, csrf)
}

func RegisterRoutes(group *gin.RouterGroup, app *di.ApplicationState, csrf *middleware.CSRFProtection) {
	systemHandler := handler.NewSystemHandler(app.System)
	group.GET("/system/status", systemHandler.Status)
	group.GET("/system/metrics", systemHandler.Metrics)
	group.POST("/system/install", systemHandler.Install)
	group.POST("/system/migrate", systemHandler.Migrate)

	if csrf != nil {
		securityHandler := handler.NewSecurityHandler(csrf)
		group.GET("/security/csrf", securityHandler.CSRF)
	}

	listFlags := usecase.NewListFlagsUsecase(app.Flags)

	projectsHandler := handler.NewProjectsHandler(
		usecase.NewListProjectsUsecase(app.Projects),
		usecase.NewCreateProjectUsecase(app.Projects),
		usecase.NewDeleteProjectUsecase(app.Projects, app.Environments, app.Segments),
	)
	envHandler := handler.NewEnvironmentsHandler(
		usecase.NewListEnvironmentsUsecase(app.Environments),
		usecase.NewCreateEnvironmentUsecase(app.Environments),
		usecase.NewDeleteEnvironmentUsecase(app.Environments, app.Flags),
	)
	segmentsHandler := handler.NewSegmentsHandler(
		usecase.NewListSegmentsUsecase(app.Segments),
		usecase.NewCreateSegmentUsecase(app.Segments),
		usecase.NewUpdateSegmentUsecase(app.Segments),
		usecase.NewDeleteSegmentUsecase(app.Segments),
	)
	flagsHandler := handler.NewFlagsHandler(
		listFlags,
		usecase.NewGetFlagUsecase(app.Flags),
		usecase.NewCreateFlagUsecase(app.Flags, app.SnapshotPointer),
		usecase.NewUpdateFlagUsecase(app.Flags, app.SnapshotPointer),
		usecase.NewDeleteFlagUsecase(app.Flags, app.SnapshotPointer),
		app.Idempotency,
	)
	evalHandler := handler.NewEvaluateHandler(
		usecase.NewEvaluateFlagsUsecase(app.ScopedFlags),
	)

	dashboardHandler := handler.NewDashboardHandler(
		app.AdminRead,
		usecase.NewRollbackFlagUsecase(app.Flags, app.AdminRead, app.SnapshotPointer),
	)

	public := group.Group("/public")
	public.Use(middleware.RateLimit(20, 60))
	public.Use(middleware.StaticAPIKey("X-Public-Api-Key", app.Config.PublicReadAPIKey))
	publicHandler := handler.NewPublicFlagsHandler(listFlags, usecase.NewEvaluateFlagsUsecase(app.ScopedFlags))
	public.GET("/projects/:projectId/environments/:environmentKey/flags", publicHandler.List)
	public.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/evaluate", publicHandler.Evaluate)
	client := group.Group("/client")
	client.Use(middleware.RateLimit(20, 60))
	client.Use(middleware.StaticAPIKey("X-Public-Api-Key", app.Config.PublicReadAPIKey))
	client.GET("/projects/:projectId/environments/:environmentKey/flags", publicHandler.List)
	client.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/evaluate", publicHandler.Evaluate)

	apiKeysHandler := handler.NewAPIKeysHandler(app.MySQL.SQL())
	group.GET("/projects/:projectId/environments/:environmentKey/api-keys", apiKeysHandler.List)
	group.POST("/projects/:projectId/environments/:environmentKey/api-keys", apiKeysHandler.Create)
	group.DELETE("/projects/:projectId/environments/:environmentKey/api-keys/:keyId", apiKeysHandler.Revoke)
	group.POST("/projects/:projectId/environments/:environmentKey/api-keys/:keyId/rotate", apiKeysHandler.Rotate)

	usersHandler := handler.NewUsersHandler(app.MySQL.SQL())
	rolesHandler := handler.NewRolesHandler(app.MySQL.SQL())
	group.GET("/users", usersHandler.List)
	group.POST("/users", usersHandler.Create)
	group.GET("/users/:userId", usersHandler.Get)
	group.PATCH("/users/:userId", usersHandler.Update)
	group.DELETE("/users/:userId", usersHandler.Disable)
	group.GET("/roles", rolesHandler.List)
	group.POST("/roles", rolesHandler.Create)
	group.GET("/roles/:roleId", rolesHandler.Get)
	group.PATCH("/roles/:roleId", rolesHandler.Update)
	group.DELETE("/roles/:roleId", rolesHandler.Delete)

	group.GET("/projects", projectsHandler.List)
	group.POST("/projects", projectsHandler.Create)
	group.DELETE("/projects/:projectId", projectsHandler.Delete)

	group.GET("/projects/:projectId/environments", envHandler.List)
	group.POST("/projects/:projectId/environments", envHandler.Create)
	group.DELETE("/projects/:projectId/environments/:environmentKey", envHandler.Delete)

	group.GET("/projects/:projectId/segments", segmentsHandler.List)
	group.POST("/projects/:projectId/segments", segmentsHandler.Create)
	group.PATCH("/projects/:projectId/segments/:segmentId", segmentsHandler.Update)
	group.DELETE("/projects/:projectId/segments/:segmentId", segmentsHandler.Delete)

	group.GET("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.List)
	group.POST("/projects/:projectId/environments/:environmentKey/flags", flagsHandler.Create)
	// Static /flags/evaluate before /flags/:flagKey so "evaluate" is never captured as a key.
	requireRuntimeKey := strings.EqualFold(app.Config.App.Env, "production")
	group.POST("/projects/:projectId/environments/:environmentKey/flags/evaluate",
		middleware.APIKeyRateLimit(20, 60),
		middleware.RequireAPIKeyScope(app.MySQL.SQL(), "flags:evaluate", requireRuntimeKey),
		evalHandler.Evaluate)
	group.GET("/projects/:projectId/environments/:environmentKey/flags/evaluate",
		middleware.APIKeyRateLimit(20, 60),
		middleware.RequireAPIKeyScope(app.MySQL.SQL(), "flags:evaluate", requireRuntimeKey),
		evalHandler.EvaluateGET)
	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/evaluate",
		middleware.APIKeyRateLimit(20, 60),
		middleware.RequireAPIKeyScope(app.MySQL.SQL(), "flags:evaluate", requireRuntimeKey),
		evalHandler.EvaluateFlagGET)

	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.GetByKey)
	group.PATCH("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.Update)
	group.DELETE("/projects/:projectId/environments/:environmentKey/flags/:flagKey", flagsHandler.Delete)

	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/versions", dashboardHandler.ListFlagVersions)
	group.GET("/projects/:projectId/environments/:environmentKey/flags/:flagKey/versions/diff", dashboardHandler.DiffFlagVersions)
	group.POST("/projects/:projectId/environments/:environmentKey/flags/:flagKey/rollback", dashboardHandler.Rollback)

	group.GET("/projects/:projectId/audit", dashboardHandler.ListAudit)

	wsHandler := handler.NewBroadcastWSHandler(app.Config, app.BroadcastHub, app.ScopedFlags, app.Environments)
	group.GET("/broadcast/flags", middleware.RequireAPIKeyScope(app.MySQL.SQL(), "broadcast:subscribe", requireRuntimeKey), wsHandler.Connect)
}
