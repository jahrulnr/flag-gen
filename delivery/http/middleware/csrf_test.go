// nolint

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCSRFProtection_IssueAndValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	p := NewCSRFProtection("test-secret", "csrf_session")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/security/csrf", nil)
	contract := p.Issue(c)
	require.NotEmpty(t, contract.Token)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", nil)
	for _, cookie := range w.Result().Cookies() {
		req.AddCookie(cookie)
	}
	req.Header.Set(CSRFHeaderName, contract.Token)
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = req

	require.NoError(t, p.Validate(c2))
}

func TestCSRFProtection_RejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	p := NewCSRFProtection("test-secret", "csrf_session")

	req := httptest.NewRequest(http.MethodPost, "/api/projects", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_session", Value: "session"})
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	require.Error(t, p.Validate(c))
}
