package lti

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
	ltiRepo "github.com/quipper/poc/lti/be/pkg/repositories/lti"
)

// LTI Tools use this endpoint to obtain an access token.
// The access token is then used to make API calls to the platform (AGS/ NRPS)
func (h *Handler) oauth2Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid form: "+err.Error())
		return
	}
	grantType := r.Form.Get("grant_type")
	scope := r.Form.Get("scope")
	clientID := r.Form.Get("client_id")
	clientAssertionType := r.Form.Get("client_assertion_type")
	clientAssertion := r.Form.Get("client_assertion")

	// Debug: log received fields (not logging full assertions)
	logger.Debug("/api/oauth2/token: grant_type=%s scope=%q client_id=%q client_assertion_type=%q has_assertion=%t",
		grantType, scope, clientID, clientAssertionType, clientAssertion != "")
	if clientAssertion != "" {
		// Try to log JWT header kid and payload aud without verifying
		parts := strings.Split(clientAssertion, ".")
		if len(parts) == 3 {
			if hb, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
				var hmap map[string]any
				if json.Unmarshal(hb, &hmap) == nil {
					if kid, _ := hmap["kid"].(string); kid != "" {
						logger.Debug("client_assertion header.kid=%s", kid)
					}
				}
			}
			if pb, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
				var pmap map[string]any
				if json.Unmarshal(pb, &pmap) == nil {
					if aud, ok := pmap["aud"]; ok {
						logger.Debug("client_assertion payload.aud=%v", aud)
					}
					if iss, _ := pmap["iss"].(string); clientID == "" && iss != "" {
						logger.Debug("inferring client_id from assertion iss=%s", iss)
						clientID = iss
					}
					if sub, _ := pmap["sub"].(string); clientID == "" && sub != "" {
						logger.Debug("inferring client_id from assertion sub=%s", sub)
						clientID = sub
					}
				}
			}
		}
	}

	// Basic validation according to IMS LTI services: client_credentials + private_key_jwt
	if grantType != "client_credentials" {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be client_credentials")
		return
	}
	if scope == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing scope")
		return
	}

	// Validate client assertion type
	if clientAssertionType != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid client_assertion_type")
		return
	}
	if clientAssertion == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing client_assertion")
		return
	}

	// Lookup tool by client_id. Some tools omit client_id when using private_key_jwt.
	// We'll attempt discovery by verifying the assertion against registered tools if needed.
	var tool *ltiRepo.Tool
	var parsed jwt.Token
	var err error
	tokenEndpoint := h.issuer + "/api/oauth2/token"

	// Helper to parse with a given key set
	parseWithSet := func(set jwk.Set) (jwt.Token, error) {
		return jwt.ParseString(clientAssertion,
			jwt.WithKeySet(set),
			jwt.WithValidate(true),
			jwt.WithAudience(tokenEndpoint),
		)
	}

	// Primary path: have a client_id
	if clientID != "" {
		t, err1 := h.repo.GetToolByClientID(r.Context(), clientID)
		if err1 != nil {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "repository error")
			return
		}
		if t != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			set, err2 := h.jwksCache.Get(ctx, t.KeySetURL)
			if err2 == nil {
				if tok, err3 := parseWithSet(set); err3 == nil {
					tool = t
					parsed = tok
				}
			}
		}
	}

	// Fallback discovery: iterate tools and try to validate
	if parsed == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		all, err1 := h.repo.ListTools(r.Context())
		if err1 == nil {
			for _, t := range all {
				if t.KeySetURL == "" {
					continue
				}
				set, err2 := h.jwksCache.Get(ctx, t.KeySetURL)
				if err2 != nil {
					continue
				}
				tok, err3 := parseWithSet(set)
				if err3 != nil {
					continue
				}
				tool = t
				parsed = tok
				logger.Debug("resolved client_assertion to tool=%s via JWKS", t.Name)
				break
			}
		}
	}
	if parsed == nil || tool == nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "unable to validate client_assertion against any registered tool")
		return
	}

	// Fetch tool JWKS
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	set, err := h.jwksCache.Get(ctx, tool.KeySetURL)
	if err != nil {
		writeOAuthError(w, http.StatusBadGateway, "invalid_client", "failed to fetch client JWKS")
		return
	}

	// Validate client_assertion JWT signature and claims
	parsed, err = jwt.ParseString(clientAssertion,
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithAudience(tokenEndpoint),
	)
	if err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "invalid client_assertion: "+err.Error())
		return
	}

	// Additional consistency checks (relaxed for interop): accept if either iss or sub matches registered client_id
	iss := parsed.Issuer()
	sub := parsed.Subject()
	logger.Debug("client_assertion iss=%q sub=%q registered_client_id=%q", iss, sub, tool.ClientID)
	if tool.ClientID != "" {
		if iss != tool.ClientID && sub != tool.ClientID {
			writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client_assertion iss/sub do not match registered client_id")
			return
		}
	} else if iss == "" && sub == "" {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client_assertion missing iss/sub")
		return
	}
	// Use the registered client_id when available, otherwise fallback to sub then iss
	effectiveClientID := tool.ClientID
	if effectiveClientID == "" {
		if sub != "" {
			effectiveClientID = sub
		} else {
			effectiveClientID = iss
		}
	}
	// iat/exp already validated by WithValidate(true)
	// Enforce jti uniqueness to prevent replay
	jti := parsed.JwtID()
	if jti == "" {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client_assertion missing jti")
		return
	}
	exp := parsed.Expiration()
	if h.validationRepo == nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "validation repository not configured")
		return
	}
	ok, err := h.validationRepo.TryUseClientAssertionJTI(r.Context(), jti, effectiveClientID, exp)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "repository error")
		return
	}
	if !ok {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client_assertion replay detected")
		return
	}

	// Issue a JWT access token signed by platform keys
	if err := keys.Init(); err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to initialize keys")
		return
	}
	now := time.Now()
	//TODO: extend token expiry to 1 hour
	//this short time is for easier logging since Tool will cache the token and will not call /oauth2/token again
	exp2 := now.Add(1 * time.Minute)
	aud := h.issuer + "/api" // audience for your APIs; adjust per service if needed
	accessJWT, err := jwt.NewBuilder().
		Issuer(h.issuer).
		Subject(effectiveClientID).
		Audience([]string{aud}).
		IssuedAt(now).
		Expiration(exp2).
		JwtID(anonSub()).
		Claim("scope", scope).
		Build()
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to build access token")
		return
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jwk.KeyIDKey, keys.Kid())
	rawToken, err := jwt.Sign(accessJWT, jwt.WithKey(jwa.RS256, keys.PrivateKey(), jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to sign access token")
		return
	}

	resp := map[string]any{
		"access_token": string(rawToken),
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(exp2).Seconds()),
		"scope":        scope,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	logger.Debug("/api/oauth2/token: issued token client_id=%s scope=%q exp=%s", effectiveClientID, scope, exp2.Format(time.RFC3339))
}

// writeOAuthError writes an RFC 6749 style error response
func writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":             code,
		"error_description": desc,
	})
	logger.Debug("/api/oauth2/token error: status=%d error=%s desc=%s", status, code, desc)
}
