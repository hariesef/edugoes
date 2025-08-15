package lti

import (
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
)

// agsRequireScopes validates the Bearer token and enforces one of the required scopes.
// It verifies signature, exp, aud and checks the `scope` claim contains at least one required scope.
func (h *Handler) agsRequireScopes(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			logger.Debug("AGS auth: path=%s required_scopes=%v", r.URL.Path, required)
			// Debug log all incoming headers for troubleshooting
			// for name, values := range r.Header {
			// 	for _, v := range values {
			// 		logger.Debug("AGS auth header: %s=%s", name, v)
			// 	}
			// }
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				logger.Debug("AGS auth: missing bearer token")
				w.Header().Set("WWW-Authenticate", `Bearer realm="lti-ags", error="invalid_request", error_description="missing bearer token"`)
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			tokStr := strings.TrimSpace(auth[len("Bearer "):])

			if err := keys.Init(); err != nil {
				logger.Debug("AGS auth: keys init error: %v", err)
				http.Error(w, "server keys not initialized", http.StatusInternalServerError)
				return
			}
			// Validate token
			pub := &keys.PrivateKey().PublicKey
			aud := h.issuer + "/api"
			tok, err := jwt.ParseString(tokStr,
				jwt.WithKey(jwa.RS256, pub),
				jwt.WithValidate(true),
				jwt.WithAudience(aud),
			)
			if err != nil {
				logger.Debug("AGS auth: token parse/validate error: %v", err)
				w.Header().Set("WWW-Authenticate", `Bearer realm="lti-ags", error="invalid_token", error_description="expired or invalid token"`)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}
			// Scope check (space-delimited per RFC 6749)
			scopes, _ := tok.Get("scope")
			scopeStr, _ := scopes.(string)
			ok := false
			if scopeStr != "" && len(required) == 0 {
				ok = true
			} else if scopeStr != "" {
				parts := strings.Fields(scopeStr)
				for _, need := range required {
					for _, have := range parts {
						if have == need {
							ok = true
							break
						}
					}
					if ok {
						break
					}
				}
			}
			if !ok {
				logger.Debug("AGS auth: insufficient scope. have=%q need=%v", scopeStr, required)
				w.Header().Set("WWW-Authenticate", `Bearer realm="lti-ags", error="insufficient_scope", scope="`+strings.Join(required, " ")+`"`)
				http.Error(w, "insufficient_scope", http.StatusForbidden)
				return
			}
			logger.Debug("AGS auth: ok for path=%s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
