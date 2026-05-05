package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"flag-gen/delivery/http/response"

	"github.com/gin-gonic/gin"
)

const CSRFHeaderName = "X-CSRF-Token"

type CSRFProtection struct {
	secret     []byte
	cookieName string
}

type CSRFContract struct {
	HeaderName string    `json:"headerName"`
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func NewCSRFProtection(secret string, cookieName string) *CSRFProtection {
	if cookieName == "" {
		cookieName = "flag_gen_csrf_session"
	}
	key := []byte(secret)
	if len(key) == 0 {
		key = randomBytes(32)
	}
	return &CSRFProtection{secret: key, cookieName: cookieName}
}

func (p *CSRFProtection) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if p == nil || isSafeMethod(c.Request.Method) || strings.HasPrefix(c.Request.URL.Path, "/api/security/") {
			c.Next()
			return
		}
		if err := p.Validate(c); err != nil {
			response.AbortError(c, http.StatusForbidden, "csrf_invalid", err.Error(), nil, time.Now().UTC().Format(time.RFC3339Nano))
			return
		}
		c.Next()
	}
}

func (p *CSRFProtection) Issue(c *gin.Context) CSRFContract {
	session := p.session(c)
	expires := time.Now().UTC().Add(30 * time.Minute)
	return CSRFContract{
		HeaderName: CSRFHeaderName,
		Token:      p.sign(session, expires),
		ExpiresAt:  expires,
	}
}

func (p *CSRFProtection) IssueHTTP(r *http.Request, headers http.Header) CSRFContract {
	session := ""
	if cookie, err := r.Cookie(p.cookieName); err == nil {
		session = cookie.Value
	}
	if session == "" {
		session = hex.EncodeToString(randomBytes(32))
		secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
		cookie := &http.Cookie{
			Name:     p.cookieName,
			Value:    session,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
		}
		headers.Add("Set-Cookie", cookie.String())
	}
	expires := time.Now().UTC().Add(30 * time.Minute)
	return CSRFContract{
		HeaderName: CSRFHeaderName,
		Token:      p.sign(session, expires),
		ExpiresAt:  expires,
	}
}

func (p *CSRFProtection) Validate(c *gin.Context) error {
	session, err := c.Cookie(p.cookieName)
	if err != nil || session == "" {
		return errCSRF("missing csrf session")
	}
	token := c.GetHeader(CSRFHeaderName)
	if token == "" {
		return errCSRF("missing csrf token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errCSRF("malformed csrf token")
	}
	unix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return errCSRF("malformed csrf expiry")
	}
	expires := time.Unix(unix, 0).UTC()
	if time.Now().UTC().After(expires) {
		return errCSRF("expired csrf token")
	}
	want := p.sign(session, expires)
	if !hmac.Equal([]byte(token), []byte(want)) {
		return errCSRF("invalid csrf token")
	}
	return nil
}

func (p *CSRFProtection) session(c *gin.Context) string {
	if existing, err := c.Cookie(p.cookieName); err == nil && existing != "" {
		return existing
	}
	session := hex.EncodeToString(randomBytes(32))
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     p.cookieName,
		Value:    session,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	return session
}

func (p *CSRFProtection) sign(session string, expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(session))
	mac.Write([]byte("|"))
	mac.Write([]byte(exp))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return exp + "." + sig
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

type csrfError string

func (e csrfError) Error() string { return string(e) }

func errCSRF(message string) error { return csrfError(message) }

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
