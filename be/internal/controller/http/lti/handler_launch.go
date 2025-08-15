package lti

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
)

// launchStart starts the LTI 1.3 third-party initiated login by redirecting
// the user-agent to the Tool's Login Initiation URL with the required query parameters.
func (h *Handler) launchStart(w http.ResponseWriter, r *http.Request) {
	logger.Debug("launchStart: method=%s content_type=%s", r.Method, r.Header.Get("Content-Type"))
	type reqBody struct {
		Issuer             string `json:"issuer"`
		ClientID           string `json:"client_id"`
		LoginInitiationURL string `json:"login_initiation_url"`
		TargetLinkURI      string `json:"target_link_uri"`
		ContextID          string `json:"context_id"`
		LoginHint          string `json:"login_hint"`
		LTIMessageHint     string `json:"lti_message_hint"`
		ResourceLinkID     string `json:"resource_link_id"`
	}

	var body reqBody
	ct := r.Header.Get("Content-Type")
	if ct != "" && (ct == "application/json" || ct == "application/json; charset=utf-8") {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			logger.Debug("launchStart: invalid JSON body: %v", err)
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		// Accept form submissions as well for browser form-post redirects
		_ = r.ParseForm()
		body.Issuer = r.FormValue("issuer")
		body.ClientID = r.FormValue("client_id")
		body.LoginInitiationURL = r.FormValue("login_initiation_url")
		body.TargetLinkURI = r.FormValue("target_link_uri")
		body.ContextID = r.FormValue("context_id")
		body.LoginHint = r.FormValue("login_hint")
		body.LTIMessageHint = r.FormValue("lti_message_hint")
		body.ResourceLinkID = r.FormValue("resource_link_id")

		logger.Debug("issuer: %s", body.Issuer)
		logger.Debug("client_id: %s", body.ClientID)
		logger.Debug("login_initiation_url: %s", body.LoginInitiationURL)
		logger.Debug("target_link_uri: %s", body.TargetLinkURI)
		logger.Debug("context_id: %s", body.ContextID)
		logger.Debug("login_hint: %s", body.LoginHint)
		logger.Debug("lti_message_hint: %s", body.LTIMessageHint)
		logger.Debug("resource_link_id: %s", body.ResourceLinkID)
	}
	if body.Issuer == "" || body.ClientID == "" || body.LoginInitiationURL == "" || body.TargetLinkURI == "" {
		logger.Debug("launchStart: missing required fields issuer/client_id/login_initiation_url/target_link_uri")
		http.Error(w, "missing required fields: issuer, client_id, login_initiation_url, target_link_uri", http.StatusBadRequest)
		return
	}

	// Build redirect URL with required params.
	u, err := url.Parse(body.LoginInitiationURL)
	if err != nil {
		logger.Debug("launchStart: invalid login_initiation_url: %v", err)
		http.Error(w, "invalid login_initiation_url", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("iss", body.Issuer)
	q.Set("client_id", body.ClientID)

	if body.LTIMessageHint != "" {
		q.Set("lti_message_hint", body.LTIMessageHint)
	}

	q.Set("target_link_uri", body.TargetLinkURI)

	// For POC it retrieves email from body
	// TODO: Get email from user session/ DB
	if body.LoginHint != "" {
		q.Set("login_hint", body.LoginHint)
	}

	if body.ResourceLinkID != "" {
		q.Set("resource_link_id", body.ResourceLinkID)
	}

	// Generate and persist state/nonce (15 min TTL)
	state := uuid.NewString()
	nonce := uuid.NewString()
	exp := time.Now().Add(15 * time.Minute)
	if h.validationRepo == nil {
		logger.Debug("launchStart: validation repository not configured")
		http.Error(w, "validation repository not configured", http.StatusInternalServerError)
		return
	}
	if err := h.validationRepo.CreateOIDCState(r.Context(), state, body.ClientID,
		body.TargetLinkURI, body.ContextID, body.ResourceLinkID, exp); err != nil {
		logger.Debug("launchStart: failed to create state: %v", err)
		http.Error(w, "failed to create state", http.StatusInternalServerError)
		return
	}

	// Set a first-party correlation cookie to survive tool rewrites of state/nonce
	http.SetCookie(w, &http.Cookie{
		Name:     "lti_corr",
		Value:    state,
		Path:     "/",
		MaxAge:   120, // seconds
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Use plain state value; correlation relies on first-party cookie
	q.Set("state", state)
	q.Set("nonce", nonce)
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
	logger.Debug("launchStart: redirected to %s", u.String())
}
