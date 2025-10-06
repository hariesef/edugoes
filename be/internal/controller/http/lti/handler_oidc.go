package lti

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
)

// oidcAuth handles the tool's redirect to the platform authorization endpoint and issues an id_token.
// For PoC, we perform minimal validation and sign with the platform RSA key.
func (h *Handler) oidcAuth(w http.ResponseWriter, r *http.Request) {
	// Accept both GET and POST; read params from query + form.
	_ = r.ParseForm()
	clientID := firstNonEmpty(r.Form.Get("client_id"), r.URL.Query().Get("client_id"))
	redirectURI := firstNonEmpty(r.Form.Get("redirect_uri"), r.URL.Query().Get("redirect_uri"))
	// Some tools (e.g., IMS RI) send a JWT in `state` and embed the platform-provided state inside it.
	// Also, some UIs may pass `tool_state` separately. Capture both.
	rawState := firstNonEmpty(r.Form.Get("state"), r.URL.Query().Get("state"))
	_ = firstNonEmpty(r.Form.Get("tool_state"), r.URL.Query().Get("tool_state"))
	nonce := firstNonEmpty(r.Form.Get("nonce"), r.URL.Query().Get("nonce"))
	ltiMessageHint := firstNonEmpty(r.Form.Get("lti_message_hint"), r.URL.Query().Get("lti_message_hint"))

	if clientID == "" || redirectURI == "" {
		http.Error(w, "missing client_id or redirect_uri", http.StatusBadRequest)
		return
	}

	// Validate and consume state
	if h.validationRepo == nil {
		http.Error(w, "validation repository not configured", http.StatusInternalServerError)
		return
	}

	// Consume our stored state using first-party correlation cookie only
	corrCookie, _ := r.Cookie("lti_corr")
	if corrCookie == nil || corrCookie.Value == "" {
		http.Error(w, "missing correlation cookie", http.StatusUnauthorized)
		return
	}
	logger.Info("Using correlation cookie lti_corr=%s", corrCookie.Value)
	storedClientID, targetLinkUri, resourceLinkID, contextID, ok, err := h.validationRepo.ConsumeOIDCState(r.Context(), corrCookie.Value)
	if !ok || err != nil {
		http.Error(w, "invalid or expired correlation state", http.StatusUnauthorized)
		return
	}
	// No nonce comparison; we no longer persist nonce in validation repository
	if targetLinkUri == "" && resourceLinkID != "" {
		logger.Debug("oidcAuth: target_link_uri empty, have resource_link_id=%s (tool may resolve target via this)", resourceLinkID)
	}
	// Clear the correlation cookie to avoid reuse
	http.SetCookie(w, &http.Cookie{
		Name:     "lti_corr",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	// Strict cookie-based correlation: we've already handled err above.
	if storedClientID != "" && storedClientID != clientID {
		http.Error(w, "client_id mismatch", http.StatusUnauthorized)
		return
	}

	// Validate client and redirectUri against repo
	tool, err := h.repo.GetToolByClientID(r.Context(), clientID)
	if err != nil {
		http.Error(w, "repository error", http.StatusInternalServerError)
		return
	}
	if tool == nil {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	// Compare host+path of redirectURI with the allowed redirect for this client.
	// For deep linking, tools typically require redirect_uri == target_link_url (deep_link_launches).
	// Fallback to auth_url if target_link_url is not set.
	reqURI, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	allowed := tool.TargetLaunchURL
	if ltiMessageHint == "deep_linking" {
		allowed = tool.TargetLinkURL
	}

	if allowed == "" {
		allowed = tool.AuthURL
	}
	allowedURI, err := url.Parse(allowed)
	if err != nil {
		http.Error(w, "server misconfig: tool redirect url invalid", http.StatusInternalServerError)
		return
	}
	if reqURI.Scheme != allowedURI.Scheme || reqURI.Host != allowedURI.Host || reqURI.Path != allowedURI.Path {
		logger.Debug("requri.scheme=%s alloweduri.scheme=%s requri.host=%s alloweduri.host=%s requri.path=%s alloweduri.path=%s", reqURI.Scheme, allowedURI.Scheme, reqURI.Host, allowedURI.Host, reqURI.Path, allowedURI.Path)
		http.Error(w, "redirect_uri not allowed for this client", http.StatusBadRequest)
		return
	}

	// Build LTI id_token claims (minimal set for PoC)
	now := time.Now()
	exp := now.Add(5 * time.Minute)
	iss := h.issuer // platform issuer; define on Handler init in server.go

	// Decide message type: ResourceLink vs DeepLinking
	msgType := "LtiResourceLinkRequest"
	var extraClaims map[string]any
	if ltiMessageHint == "deep_linking" {
		msgType = "LtiDeepLinkingRequest"
		extraClaims = map[string]any{
			"https://purl.imsglobal.org/spec/lti-dl/claim/deep_linking_settings": map[string]any{
				"deep_link_return_url":                 h.issuer + "/api/deeplink/return",
				"data":                                 contextID,
				"accept_types":                         []string{"ltiResourceLink"},
				"accept_presentation_document_targets": []string{"iframe", "window"},
				"accept_multiple":                      false,
			},
		}
	}

	// Create JWT
	// Determine subject and optional profile claims
	subject := anonSub()
	email := ""
	fullName := ""
	// given := ""
	// family := ""
	// Use login_hint as a stable subject if available
	if lh := firstNonEmpty(r.Form.Get("login_hint"), r.URL.Query().Get("login_hint")); lh != "" {
		subject = lh
		if email == "" && strings.Contains(lh, "@") {
			email = lh
		}
		// TODO: In real case this should be populated from LMS session/auth, resolving to user data
		// For PoC, hardcode the LMS user; later, populate from your session/auth
		fullName = "Haries Efrika"
		// given = "Haries"
		// family = "Efrika"
	}

	builder := jwt.NewBuilder().
		Issuer(iss).
		Subject(subject).
		Audience([]string{clientID}).
		IssuedAt(now).
		Expiration(exp).
		Claim("nonce", nonce).
		Claim("https://purl.imsglobal.org/spec/lti/claim/version", "1.3.0").
		Claim("https://purl.imsglobal.org/spec/lti/claim/message_type", msgType).
		Claim("https://purl.imsglobal.org/spec/lti/claim/deployment_id", "dev-deployment")

	// Roles are required by many tools (including ltijs). For PoC, default to Instructor.
	roles := []string{
		"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Instructor",
	}

	// POC Workaround
	// TODO: Set proper roles based on userid/ email
	if resourceLinkID != "" {
		logger.Debug("resource_link_id is set: %s", resourceLinkID)
		// Workaround for POC
		// To differentiate between teacher (haries@efrika.net) and student (student@efrika.net)
		// when we have target_link_uri cookie, means this is student access.
		roles = []string{
			"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Student",
		}
		fullName = "Student Efrika"
		// given = "Student"
		// family = "Efrika"
	}
	// this claim is important to tell Tool what Content Item to launch
	builder = builder.Claim("https://purl.imsglobal.org/spec/lti/claim/target_link_uri", targetLinkUri)
	builder = builder.Claim("https://purl.imsglobal.org/spec/lti/claim/roles", roles)

	// Optional user profile claims
	if fullName != "" {
		builder = builder.Claim("name", fullName)
	}
	// if given != "" {
	// 	builder = builder.Claim("given_name", given)
	// }
	// if family != "" {
	// 	builder = builder.Claim("family_name", family)
	// }
	if email != "" {
		builder = builder.Claim("email", email)
	}

	// For a resource launch, include a resource_link claim
	if msgType == "LtiResourceLinkRequest" {
		builder = builder.Claim("https://purl.imsglobal.org/spec/lti/claim/resource_link", map[string]any{
			"id": resourceLinkID,
		})

		// Add AGS endpoint claim with context-scoped lineitems URL and allowed scopes
		// contextID was stored during launchStart and retrieved from validation state.
		if contextID == "" {
			logger.Debug("oidcAuth: missing context_id in state; falling back to dev-context")
			contextID = "dev-context"
		} else {
			logger.Debug("oidcAuth: using context_id=%s", contextID)
		}
		base := h.issuer
		if pub := os.Getenv("PUBLIC_BASE_URL"); pub != "" {
			base = pub
		}

		// Resolve line item id from resourceLinkID using repository reverse lookup
		lineItemId := ""
		if resourceLinkID != "" {
			if id, err := h.scores.GetLineItemIDByResourceLinkID(r.Context(), resourceLinkID); err != nil {
				logger.Debug("oidcAuth: failed to get line item id for resourceLinkID %s: %v", resourceLinkID, err)
			} else if id > 0 {
				lineItemId = strconv.FormatInt(id, 10)
				logger.Debug("oidcAuth: resolved lineItemId %s for resourceLinkID %s", lineItemId, resourceLinkID)
			}
		}

		agsClaim := map[string]any{
			// Tool can only access the lineItemId we specified/ linked to resourceLinkId.
			"lineitem":  base + "/api/ags/contexts/" + contextID + "/lineitems/" + lineItemId,
			"lineitems": base + "/api/ags/contexts/" + contextID + "/lineitems",
			"scope": []string{
				"https://purl.imsglobal.org/spec/lti-ags/scope/lineitem.readonly",
				// TODO: In production, scope lineitem (write/del) below must be removed,
				// so that Tool can't messly delete the lineitem that we already tightly coupled with resourceLinkID.
				// LMS should manage the final state of lineitem/gradebook, not the Tool.
				"https://purl.imsglobal.org/spec/lti-ags/scope/lineitem", // <-- to be removed
				"https://purl.imsglobal.org/spec/lti-ags/scope/result.readonly",
				"https://purl.imsglobal.org/spec/lti-ags/scope/score",
			},
		}
		logger.Debug("OIDC id_token AGS claim: %+v", agsClaim)
		builder = builder.Claim("https://purl.imsglobal.org/spec/lti-ags/claim/endpoint", agsClaim)

		// NRPS claim: advertise context memberships endpoint and version
		nrpsClaim := map[string]any{
			"context_memberships_url": base + "/api/nrps/contexts/" + contextID + "/members",
			"service_versions":        []string{"2.0"},
		}
		logger.Debug("OIDC id_token NRPS claim: %+v", nrpsClaim)
		builder = builder.Claim("https://purl.imsglobal.org/spec/lti-nrps/claim/namesroleservice", nrpsClaim)
		// Also advertise token endpoint via LTI Services claim so tools know where to obtain an access token
		services := []map[string]any{
			{
				"endpoint": base + "/oauth2/token",
				"scope": []string{
					"https://purl.imsglobal.org/spec/lti-ags/scope/lineitem.readonly",
					"https://purl.imsglobal.org/spec/lti-ags/scope/lineitem",
					"https://purl.imsglobal.org/spec/lti-ags/scope/result.readonly",
					"https://purl.imsglobal.org/spec/lti-ags/scope/score",
				},
			},
			{
				"endpoint": base + "/api/nrps/contexts/" + contextID + "/members",
				"scope": []string{
					"https://purl.imsglobal.org/spec/lti-nrps/scope/contextmembership.readonly",
				},
			},
		}
		logger.Debug("OIDC id_token Services claim: %+v", services)
		builder = builder.Claim("https://purl.imsglobal.org/spec/lti/claim/service", services)
	}

	for k, v := range extraClaims {
		builder = builder.Claim(k, v)
	}
	tok, _ := builder.Build()
	// Debug: log all claims we are about to issue in id_token
	if m, err := tok.AsMap(r.Context()); err == nil {
		if b, err := json.MarshalIndent(m, "", "  "); err == nil {
			logger.Info("id_token full claims:\n%s", string(b))
		}
	}

	// Ensure keys are initialized
	if err := keys.Init(); err != nil {
		http.Error(w, "failed to initialize keys", http.StatusInternalServerError)
		return
	}
	// Prepare signing key (after keys.Init())
	priv := keys.PrivateKey()
	if priv == nil {
		http.Error(w, "signing key not initialized", http.StatusInternalServerError)
		return
	}
	key, err := jwk.FromRaw(priv)
	if err != nil {
		http.Error(w, "failed to load signing key", http.StatusInternalServerError)
		return
	}
	_ = key.Set(jwk.KeyIDKey, keys.Kid())
	_ = key.Set(jwk.AlgorithmKey, jwa.RS256)

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		http.Error(w, "failed to sign id_token", http.StatusInternalServerError)
		return
	}

	// Return form_post
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := `<!DOCTYPE html>
<html><body onload="document.forms[0].submit()">
<form method="post" action="` + template.HTMLEscapeString(redirectURI) + `">
<input type="hidden" name="state" value="` + template.HTMLEscapeString(rawState) + `"/>
<input type="hidden" name="id_token" value="` + template.HTMLEscapeString(string(signed)) + `"/>
<noscript><button type="submit">Continue</button></noscript>
</form></body></html>`
	_, _ = w.Write([]byte(page))
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func anonSub() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
