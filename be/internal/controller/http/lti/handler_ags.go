package lti

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quipper/poc/lti/be/pkg/common/logger"
	scoresRepo "github.com/quipper/poc/lti/be/pkg/repositories/scores"
)

// apiLineItem is the public shape per AGS spec (camelCase) with id as a URL.
type apiLineItem struct {
	ID             string     `json:"id"`
	Label          string     `json:"label"`
	ResourceID     string     `json:"resourceId,omitempty"`
	ResourceLinkID string     `json:"resourceLinkId,omitempty"`
	Tag            string     `json:"tag,omitempty"`
	ScoreMaximum   float64    `json:"scoreMaximum"`
	StartAt        *time.Time `json:"startDateTime,omitempty"`
	EndAt          *time.Time `json:"endDateTime,omitempty"`
}

func buildBaseURL(r *http.Request) string {
	if pub := os.Getenv("PUBLIC_BASE_URL"); pub != "" {
		return pub
	}
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func itemURL(r *http.Request, contextID string, id int64) string {
	base := buildBaseURL(r)
	// Ensure contextID is URL-escaped
	return base + "/api/ags/contexts/" + url.PathEscape(contextID) + "/lineitems/" + strconv.FormatInt(id, 10)
}

func toAPI(r *http.Request, li *scoresRepo.LineItem) apiLineItem {
	return apiLineItem{
		ID:             itemURL(r, li.ContextID, li.ID),
		Label:          li.Label,
		ResourceID:     li.ResourceID,
		ResourceLinkID: li.ResourceLinkID,
		Tag:            li.Tag,
		ScoreMaximum:   li.ScoreMaximum,
		StartAt:        li.StartAt,
		EndAt:          li.EndAt,
	}
}

// agsListLineItems GET /api/ags/contexts/{contextId}/lineitems
func (h *Handler) agsListLineItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	// Log raw request details
	logger.Debug("AGS list lineitems raw: method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)
	if r.Body != nil {
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			logger.Debug("AGS list lineitems body: %s", string(b))
		}
		r.Body = io.NopCloser(bytes.NewReader(nil))
	}
	logger.Debug("AGS list lineitems: context_id=%s", contextID)
	items, err := h.scores.ListLineItems(ctx, contextID)
	if err != nil {
		logger.Debug("AGS list lineitems error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Optional filtering by resource_link_id as per AGS spec
	q := r.URL.Query()
	if rl := q.Get("resource_link_id"); rl != "" {
		filtered := make([]*scoresRepo.LineItem, 0, len(items))
		for i := range items {
			if items[i].ResourceLinkID == rl {
				filtered = append(filtered, items[i])
			}
		}
		items = filtered
	}
	// Map to API shape with URL ids
	resp := make([]apiLineItem, 0, len(items))
	for i := range items {
		resp = append(resp, toAPI(r, items[i]))
	}
	logger.Debug("AGS list lineitems ok: count=%d", len(resp))
	if b, err := json.Marshal(resp); err == nil {
		logger.Debug("AGS list lineitems response: %s", string(b))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// agsCreateLineItem POST /api/ags/contexts/{contextId}/lineitems
func (h *Handler) agsCreateLineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	// Log raw request body before decoding
	var raw []byte
	if r.Body != nil {
		raw, _ = io.ReadAll(r.Body)
		logger.Debug("AGS create lineitem raw body: %s", string(raw))
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	logger.Debug("AGS create lineitem: context_id=%s", contextID)
	var li scoresRepo.LineItem
	if err := json.NewDecoder(r.Body).Decode(&li); err != nil {
		logger.Debug("AGS create lineitem decode error: %v", err)
		http.Error(w, "invalidJson", http.StatusBadRequest)
		return
	}
	li.ContextID = contextID
	if li.ScoreMaximum == 0 {
		logger.Debug("AGS create lineitem validation failed: missingScoreMaximum")
		http.Error(w, "scoreMaximumIsRequired", http.StatusBadRequest)
		return
	}
	id, err := h.scores.CreateLineItem(ctx, &li)
	if err != nil {
		logger.Debug("AGS create lineitem repo error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	li.ID = id
	logger.Debug("AGS create lineitem ok: id=%d", id)
	// Content-Location to the created resource
	w.Header().Set("Location", itemURL(r, contextID, id))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp := toAPI(r, &li)
	if b, err := json.Marshal(resp); err == nil {
		logger.Debug("AGS create lineitem response: %s", string(b))
	}
	_ = json.NewEncoder(w).Encode(&resp)
}

// agsGetLineItem GET /api/ags/contexts/{contextId}/lineitems/{lineItemId}
func (h *Handler) agsGetLineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	idStr := chi.URLParam(r, "lineItemId")
	// Log raw request details
	logger.Debug("AGS get lineitem raw: method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)
	if r.Body != nil {
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			logger.Debug("AGS get lineitem body: %s", string(b))
		}
		r.Body = io.NopCloser(bytes.NewReader(nil))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Debug("AGS get lineitem parse id error: %v", err)
		http.Error(w, "invalidLineItemId", http.StatusBadRequest)
		return
	}
	logger.Debug("AGS get lineitem: context_id=%s id=%d", contextID, id)
	li, err := h.scores.GetLineItem(ctx, id, contextID)
	if err != nil {
		logger.Debug("AGS get lineitem repo error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if li == nil {
		logger.Debug("AGS get lineitem not found: id=%d", id)
		http.NotFound(w, r)
		return
	}
	logger.Debug("AGS get lineitem ok: id=%d", id)
	w.Header().Set("Content-Type", "application/json")
	resp := toAPI(r, li)
	if b, err := json.Marshal(resp); err == nil {
		logger.Debug("AGS get lineitem response: %s", string(b))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// agsUpdateLineItem PUT /api/ags/contexts/{contextId}/lineitems/{lineItemId}
func (h *Handler) agsUpdateLineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	idStr := chi.URLParam(r, "lineItemId")
	// Log raw request body before decoding
	var raw []byte
	if r.Body != nil {
		raw, _ = io.ReadAll(r.Body)
		logger.Debug("AGS update lineitem raw body: %s", string(raw))
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Debug("AGS update lineitem parse id error: %v", err)
		http.Error(w, "invalidLineItemId", http.StatusBadRequest)
		return
	}
	logger.Debug("AGS update lineitem: context_id=%s id=%d", contextID, id)
	var li scoresRepo.LineItem
	if err := json.NewDecoder(r.Body).Decode(&li); err != nil {
		logger.Debug("AGS update lineitem decode error: %v", err)
		http.Error(w, "invalidJson", http.StatusBadRequest)
		return
	}
	li.ID = id
	li.ContextID = contextID

	if err := h.scores.UpdateLineItem(ctx, &li); err != nil {
		logger.Debug("AGS update lineitem repo error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Debug("AGS update lineitem ok: id=%d", id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&li)
}

// agsDeleteLineItem DELETE /api/ags/contexts/{contextId}/lineitems/{lineItemId}
func (h *Handler) agsDeleteLineItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	idStr := chi.URLParam(r, "lineItemId")
	// Log raw request details
	logger.Debug("AGS delete lineitem raw: method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)
	if r.Body != nil {
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			logger.Debug("AGS delete lineitem body: %s", string(b))
		}
		r.Body = io.NopCloser(bytes.NewReader(nil))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Debug("AGS delete lineitem parse id error: %v", err)
		http.Error(w, "invalidLineItemId", http.StatusBadRequest)
		return
	}
	logger.Debug("AGS delete lineitem: context_id=%s id=%d", contextID, id)
	if err := h.scores.DeleteLineItem(ctx, id, contextID); err != nil {
		logger.Debug("AGS delete lineitem repo error: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Debug("AGS delete lineitem ok: id=%d", id)
	w.WriteHeader(http.StatusNoContent)
}

// agsPostScore POST /api/ags/contexts/{contextId}/lineitems/{lineItemId}/scores
func (h *Handler) agsPostScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	idStr := chi.URLParam(r, "lineItemId")
	// Log raw request body before decoding
	var raw []byte
	if r.Body != nil {
		raw, _ = io.ReadAll(r.Body)
		logger.Debug("AGS post score raw body: %s", string(raw))
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Debug("AGS post score parse id error: %v", err)
		http.Error(w, "invalidLineItemId", http.StatusBadRequest)
		return
	}
	logger.Debug("AGS post score: context_id=%s id=%d", contextID, id)
	var s scoresRepo.Score
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		logger.Debug("AGS post score decode error: %v", err)
		http.Error(w, "invalidJson", http.StatusBadRequest)
		return
	}
	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now().UTC()
	}
	if err := h.scores.UpsertResultFromScore(ctx, id, contextID, &s); err != nil {
		logger.Debug("AGS post score repo error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var sg any
	if s.ScoreGiven != nil {
		sg = *s.ScoreGiven
	} else {
		sg = nil
	}
	// Log additional status fields
	logger.Debug("AGS post score ok: user=%s scoreGiven=%v activity=%s grading=%s", s.UserID, sg, s.ActivityProgress, s.GradingProgress)
	w.WriteHeader(http.StatusNoContent)
}

// agsListResults GET /api/ags/contexts/{contextId}/lineitems/{lineItemId}/results
func (h *Handler) agsListResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	idStr := chi.URLParam(r, "lineItemId")
	// Log raw request details
	logger.Debug("AGS list results raw: method=%s path=%s query=%s", r.Method, r.URL.Path, r.URL.RawQuery)
	if r.Body != nil {
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			logger.Debug("AGS list results body: %s", string(b))
		}
		r.Body = io.NopCloser(bytes.NewReader(nil))
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logger.Debug("AGS list results parse id error: %v", err)
		http.Error(w, "invalid lineItemId", http.StatusBadRequest)
		return
	}
	logger.Debug("AGS list results: context_id=%s id=%d", contextID, id)
	results, err := h.scores.ListResultsByLineItem(ctx, id, contextID)
	if err != nil {
		logger.Debug("AGS list results repo error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Debug("AGS list results ok: count=%d", len(results))
	if b, err := json.Marshal(results); err == nil {
		logger.Debug("AGS list results response: %s", string(b))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
