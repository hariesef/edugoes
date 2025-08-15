package lti

import (
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
)

// nrpsRequireScopes enforces OAuth2 Bearer token and required scopes for NRPS endpoints.
func (h *Handler) nrpsRequireScopes(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := requireBearerScopes(w, r, scopes); !ok {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireBearerScopes validates Authorization: Bearer access token and required scopes.
// Returns false and writes an error response when invalid.
func requireBearerScopes(w http.ResponseWriter, r *http.Request, required []string) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		w.Header().Set("WWW-Authenticate", "Bearer realm=\"NRPS\"")
		http.Error(w, "missingAuthorization", http.StatusUnauthorized)
		return false
	}
	tokenStr := strings.TrimSpace(auth[len("Bearer "):])
	if err := keys.Init(); err != nil {
		http.Error(w, "serverKeyInitFailed", http.StatusInternalServerError)
		return false
	}
	// Verify JWT using platform key (in this PoC we issue tokens ourselves)
	priv := keys.PrivateKey()
	if priv == nil {
		http.Error(w, "signingKeyUnavailable", http.StatusInternalServerError)
		return false
	}
	pub := priv.Public()
	tok, err := jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.RS256, pub))
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
		http.Error(w, "invalidToken", http.StatusUnauthorized)
		return false
	}
	// Validate scope claim contains all required scopes
	scopesOK := false
	if v, ok := tok.Get("scope"); ok {
		scopesOK = hasAllScopes(v, required)
	}
	if !scopesOK {
		w.Header().Set("WWW-Authenticate", "Bearer error=\"insufficient_scope\"")
		http.Error(w, "insufficientScope", http.StatusForbidden)
		return false
	}
	return true
}

func hasAllScopes(scopeClaim any, required []string) bool {
	// OAuth often encodes scopes as space-delimited string; support []string as well
	have := map[string]struct{}{}
	switch s := scopeClaim.(type) {
	case string:
		for _, p := range strings.Fields(s) {
			have[p] = struct{}{}
		}
	case []string:
		for _, p := range s {
			have[p] = struct{}{}
		}
	case []any:
		for _, x := range s {
			if str, ok := x.(string); ok {
				have[str] = struct{}{}
			}
		}
	}
	for _, req := range required {
		if _, ok := have[req]; !ok {
			return false
		}
	}
	return true
}
