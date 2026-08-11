package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a gin middleware for handling CORS
func CORS() gin.HandlerFunc {
	allowedOrigins := parseAllowedOrigins(os.Getenv("LINKTOR_CORS_ALLOWED_ORIGINS"))

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if !IsSameOrigin(origin, c.Request) && !isAllowedOrigin(origin, allowedOrigins) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    "FORBIDDEN",
					"message": "origin not allowed",
				})
				return
			}

			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// IsSameOrigin reports whether the Origin header points at the very host the
// request was addressed to.
//
// Browsers send Origin on same-origin non-GET requests too, so without this a
// single-origin deployment — admin and API behind one reverse proxy, as in the
// on-prem install — would 403 its own front-end unless every hostname or IP a
// user might type were listed in LINKTOR_CORS_ALLOWED_ORIGINS.
//
// Only hosts are compared, never schemes: behind a TLS-terminating proxy the
// request reaching us is plain HTTP while the page origin is https. That grants
// nothing to a third-party page — an attacker cannot make the browser send our
// Host alongside their Origin. Subdomain deployments (app.x calling api.x) are
// unaffected: the hosts differ, so the allowlist still decides.
func IsSameOrigin(origin string, r *http.Request) bool {
	if origin == "" || r == nil || r.Host == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return u.Host == r.Host
}

func parseAllowedOrigins(value string) map[string]struct{} {
	origins := make(map[string]struct{})
	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return origins
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	if _, ok := allowed[origin]; ok {
		return true
	}

	if len(allowed) > 0 {
		return false
	}

	// No origins configured: deny-by-default in release mode. The localhost
	// fallback below is a development convenience only.
	if gin.Mode() == gin.ReleaseMode {
		return false
	}

	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:")
}
