package lti

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
    "github.com/quipper/poc/lti/be/pkg/common/logger"
)

func (h *Handler) listSelections(w http.ResponseWriter, r *http.Request) {
    logger.Debug("listSelections: start")
    sels, err := h.repo.ListDeepLinkSelections(r.Context())
    if err != nil {
        logger.Debug("listSelections: repo error: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    if sels == nil {
        // Ensure [] instead of null
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("[]"))
        logger.Debug("listSelections: returned empty list")
        return
    }
    logger.Debug("listSelections: returned %d items", len(sels))
    _ = json.NewEncoder(w).Encode(sels)
}

func (h *Handler) getSelectionByID(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        logger.Debug("getSelectionByID: invalid id=%q", idStr)
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    logger.Debug("getSelectionByID: id=%d", id)
    sel, err := h.repo.GetDeepLinkSelection(r.Context(), id)
    if err != nil {
        logger.Debug("getSelectionByID: repo error: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if sel == nil {
        logger.Debug("getSelectionByID: not found id=%d", id)
        http.NotFound(w, r)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(sel)
}

func (h *Handler) deleteSelectionByID(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        logger.Debug("deleteSelectionByID: invalid id=%q", idStr)
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
     logger.Debug("deleteSelectionByID: id=%d", id)
    if err := h.repo.DeleteDeepLinkSelection(r.Context(), id); err != nil {
        logger.Debug("deleteSelectionByID: repo error: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
    logger.Debug("deleteSelectionByID: deleted id=%d", id)
}
