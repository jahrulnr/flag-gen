package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"flag-gen/infrastructure/config"

	"github.com/gin-gonic/gin"
)

func RegisterDashboardSSRProxy(r *gin.Engine, cfg config.Config) error {
	target, err := url.Parse(cfg.Dashboard.SSRURL)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalHost := req.Host
		originalDirector(req)
		req.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		req.Header.Set("X-Forwarded-Host", originalHost)
		req.Header.Set("X-Forwarded-Proto", forwardedProto(req))
	}

	r.GET("/app", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/app/")
	})
	r.Any("/app/*path", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}

func forwardedProto(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	return "http"
}
