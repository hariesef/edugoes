package lti

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
	repoPkg "github.com/quipper/poc/lti/be/pkg/repositories/lti"
	scoresRepo "github.com/quipper/poc/lti/be/pkg/repositories/scores"
)

// deeplinkReturn receives the Tool's Deep Linking Response (JWT via form_post).
// Verifies the JWT against any registered tool JWKS (PoC approach).
func (h *Handler) deeplinkReturn(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	// Per 1EdTech Deep Linking spec, the Tool posts the response as form field "JWT".
	// Accept "JWT" first, with "id_token" as a fallback for vendor quirks.
	idToken := firstNonEmpty(
		r.Form.Get("JWT"),
		r.Form.Get("id_token"),
		r.URL.Query().Get("JWT"),
		r.URL.Query().Get("id_token"),
	)
	logger.Debug("DeepLink return: id_token_len=%d", len(idToken))
	if idToken == "" {
		logger.Debug("DeepLink return: missing JWT in form_post")
		http.Error(w, "missing deep linking JWT (expected form field 'JWT')", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Try to verify against each registered tool's JWKS
	var matchedTool string
	tools, _ := h.repo.ListTools(ctx)
	for _, t := range tools {
		if t.KeySetURL == "" {
			continue
		}
		set, err := jwk.Fetch(ctx, t.KeySetURL)
		if err != nil {
			continue
		}
		if _, err := jwt.ParseString(idToken, jwt.WithKeySet(set), jwt.WithValidate(true)); err == nil {
			matchedTool = t.Name
			break
		}
	}
	if matchedTool == "" {
		logger.Debug("DeepLink return: no tool matched JWKS verification")
	} else {
		logger.Debug("DeepLink return: verified with tool=%s", matchedTool)
	}

	// Pretty-print JWT payload by decoding the middle segment
	var prettyPayload string
	parts := strings.Split(idToken, ".")
	if len(parts) == 3 {
		if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var obj any
			if err := json.Unmarshal(b, &obj); err == nil {
				if pb, err := json.MarshalIndent(obj, "", "  "); err == nil {
					prettyPayload = string(pb)
				}
			}
		}
	}
	if prettyPayload == "" {
		prettyPayload = "{}"
	}

	// Attempt to persist deep link selections if present
	// Decode payload to map for extraction
	var payload map[string]any
	if prettyPayload != "{}" {
		_ = json.Unmarshal([]byte(prettyPayload), &payload)
	} else {
		// try decode raw
		parts := strings.Split(idToken, ".")
		if len(parts) == 3 {
			if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
				_ = json.Unmarshal(b, &payload)
			}
		}
	}
	if payload == nil {
		payload = map[string]any{}
	}
	// Extract client_id/aud best-effort
	clientID := ""
	if v, ok := payload["aud"]; ok {
		switch t := v.(type) {
		case string:
			clientID = t
		case []any:
			if len(t) > 0 {
				if s, ok := t[0].(string); ok {
					clientID = s
				}
			}
		}
	}
	// Extract content_items (IMS claim)

	contextId := ""
	logger.Debug("DeepLink return: payload=%v", payload)
	if v, ok := payload["https://purl.imsglobal.org/spec/lti-dl/claim/data"].(string); ok {
		contextId = v
		logger.Debug("DeepLink return: contextId=%s", contextId)
	}

	const dlItemsClaim = "https://purl.imsglobal.org/spec/lti-dl/claim/content_items"
	if raw, ok := payload[dlItemsClaim]; ok {
		if arr, ok := raw.([]any); ok {
			logger.Debug("DeepLink return: content_items found count=%d client_id=%s", len(arr), clientID)
			for _, it := range arr {
				m, ok := it.(map[string]any)
				if !ok {
					continue
				}
				// logger.Debug("DeepLink return: content_items item: %v", m)
				// url
				url := ""
				if v, ok := m["url"].(string); ok {
					url = v
				}
				// full item json
				fullJSON := "{}"
				if b, err := json.Marshal(m); err == nil {
					fullJSON = string(b)
				}
				// persist minimal fields + full JSON
				resourceLinkID, _ := h.repo.CreateDeepLinkSelection(r.Context(), &repoPkg.DeepLinkSelection{
					ClientID:        clientID,
					ToolName:        matchedTool,
					URL:             url,
					ContentItemJSON: fullJSON,
				})
				logger.Debug("DeepLink return: persisted selection url=%s", url)

				// If claim carries a lineItem, create a new AGS line item and mapping
				if liRaw, ok := m["lineItem"].(map[string]any); ok {
					label := ""
					if v, ok := liRaw["label"].(string); ok {
						label = v
					}
					var scoreMax float64 = 0
					if v, ok := liRaw["scoreMaximum"].(float64); ok {
						scoreMax = v
					} else if v, ok := liRaw["scoreMaximum"].(json.Number); ok {
						if f, err := v.Float64(); err == nil {
							scoreMax = f
						}
					}
					// resourceLinkId and contextId are expected in custom fields if provided
					logger.Debug("DeepLink return: trying to create lineitem label=%s scoreMax=%f resourceLinkID=%d contextID=%s", label, scoreMax, resourceLinkID, contextId)
					if label != "" && scoreMax > 0 && resourceLinkID != 0 && contextId != "" {
						li := scoresRepo.LineItem{
							ContextID:      contextId,
							Label:          label,
							ResourceLinkID: strconv.FormatInt(resourceLinkID, 10),
							ScoreMaximum:   scoreMax,
						}
						newID, err := h.scores.CreateLineItem(r.Context(), &li)
						if err != nil {
							logger.Debug("DeepLink return: create lineitem error: %v", err)
						} else {
							if err := h.scores.CreateLineItemMapping(r.Context(), newID, strconv.FormatInt(resourceLinkID, 10)); err != nil {
								logger.Debug("DeepLink return: create mapping error: %v", err)
							} else {
								logger.Debug("DeepLink return: created lineitem id=%d mapped to resourceLinkId=%s", newID, strconv.FormatInt(resourceLinkID, 10))
							}
						}
					} else {
						logger.Debug("DeepLink return: lineItem present but missing required fields label/scoreMaximum/resource_link_id/context_id; skipping create")
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := `<!DOCTYPE html>
<html>
  <head><meta charset="utf-8"/><title>Deep Linking Result</title></head>
  <body>
    <h1>Deep Linking Result</h1>
    <p>verified_with_tool: ` + template.HTMLEscapeString(matchedTool) + `</p>
    <h2>id_token claims</h2>
    <pre>` + template.HTMLEscapeString(prettyPayload) + `</pre>
    <p><a href="/">Back to App</a></p>
  </body>
  </html>`
	_, _ = w.Write([]byte(page))
	logger.Debug("DeepLink return: response rendered.")
}
