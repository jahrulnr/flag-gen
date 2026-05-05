package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"flag-gen/delivery/http/middleware"
	"flag-gen/infrastructure/config"

	"github.com/gin-gonic/gin"
)

func RegisterDashboardSSRProxy(r *gin.Engine, cfg config.Config, csrf *middleware.CSRFProtection) error {
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
	proxy.ModifyResponse = func(resp *http.Response) error {
		if csrf == nil || !isHTMLResponse(resp) {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		contract := csrf.IssueHTTP(resp.Request, resp.Header)
		body = injectDashboardBootstrap(body, contract)
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		resp.Header.Del("Content-Encoding")
		return nil
	}

	r.GET("/app", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/app/")
	})
	r.Any("/app/*path", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}

func isHTMLResponse(resp *http.Response) bool {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	return resp.StatusCode == http.StatusOK && strings.Contains(contentType, "text/html")
}

func injectDashboardBootstrap(body []byte, csrf middleware.CSRFContract) []byte {
	if bytes.Contains(body, []byte(`id="flag-gen-bootstrap"`)) {
		return body
	}
	payload, err := json.Marshal(struct {
		CSRF middleware.CSRFContract `json:"csrf"`
	}{CSRF: csrf})
	if err != nil {
		return body
	}
	script := `<script id="flag-gen-bootstrap" type="application/json">` + strings.ReplaceAll(string(payload), "</", "<\\/") + `</script>`
	needle := []byte("</head>")
	if idx := bytes.Index(bytes.ToLower(body), needle); idx >= 0 {
		out := make([]byte, 0, len(body)+len(script))
		out = append(out, body[:idx]...)
		out = append(out, script...)
		out = append(out, body[idx:]...)
		return out
	}
	return append(body, []byte(script)...)
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
