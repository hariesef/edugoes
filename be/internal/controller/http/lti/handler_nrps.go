package lti

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	roster "github.com/quipper/poc/lti/be/pkg/repositories/roster"
)

// nrpsListMembers GET /api/nrps/contexts/{contextId}/members
// TODO: contextId should have intermediate mapping to students class Id.
// Class has students, has teachers
// Class has courses (contextIds)
// This way when a class needs to assign new course, they are not populated again as new rows.
func (h *Handler) nrpsListMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	// Pagination params
	q := r.URL.Query()
	limit := 50
	if ls := q.Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 {
			limit = v
		}
	}
	offset := 0
	if os := q.Get("offset"); os != "" {
		if v, err := strconv.Atoi(os); err == nil && v >= 0 {
			offset = v
		}
	}
	// DB-level pagination
	page, total, err := h.roster.ListMembersPage(ctx, contextID, offset, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	containerID := absoluteURL(r)
	// Set Link: rel="next" if more pages
	if offset+limit < total {
		nextURL := buildPageURL(r, offset+limit, limit)
		w.Header().Add("Link", "<"+nextURL+">; rel=\"next\"")
	}
    // Ensure members serializes as [] instead of null when empty
    if page == nil {
        page = []*roster.Member{}
    }
	resp := map[string]any{
		"id":      containerID,
		"context": map[string]any{"id": contextID},
		"members": page,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// This is NOT LTI Spec. Provided only for POC convenience.
// nrpsUpsertMember POST /api/nrps/contexts/{contextId}/members
func (h *Handler) nrpsUpsertMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	var m roster.Member
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalidJson", http.StatusBadRequest)
		return
	}
	if m.UserID == "" {
		http.Error(w, "userIdRequired", http.StatusBadRequest)
		return
	}
	if err := h.roster.UpsertMember(ctx, contextID, &m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// This is NOT LTI Spec. Provided only for POC convenience.
// nrpsDeleteMember DELETE /api/nrps/contexts/{contextId}/members/{userId}
func (h *Handler) nrpsDeleteMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	contextID := chi.URLParam(r, "contextId")
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		http.Error(w, "userIdRequired", http.StatusBadRequest)
		return
	}
	if err := h.roster.DeleteMember(ctx, contextID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// absoluteURL builds an absolute URL for the current request path using
// X-Forwarded-* headers when present, otherwise falls back to r.Host and TLS.
func absoluteURL(r *http.Request) string {
	scheme, host := schemeHost(r)
	return scheme + "://" + host + r.URL.Path
}

func schemeHost(r *http.Request) (string, string) {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme, host
}

func buildPageURL(r *http.Request, offset, limit int) string {
	scheme, host := schemeHost(r)
	u := url.URL{Scheme: scheme, Host: host, Path: r.URL.Path}
	q := r.URL.Query()
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()
	return u.String()
}
