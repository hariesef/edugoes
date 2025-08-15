package lti

import (
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    repoIface "github.com/quipper/poc/lti/be/pkg/repositories/lti"
    "github.com/quipper/poc/lti/be/pkg/common/logger"
    "github.com/go-chi/chi/v5"
)

func (h *Handler) createTool(w http.ResponseWriter, r *http.Request) {
    logger.Debug("createTool: start")
    w.Header().Set("Content-Type", "application/json")
    var req repoIface.Tool
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        logger.Debug("createTool: invalid JSON: %v", err)
        http.Error(w, "invalid JSON body", http.StatusBadRequest)
        return
    }
    // Minimal validation
    if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.ClientID) == "" {
        logger.Debug("createTool: missing name/client_id")
        http.Error(w, "name and client_id are required", http.StatusBadRequest)
        return
    }
    id, err := h.repo.RegisterTool(r.Context(), &req)
    if err != nil {
        logger.Error("register tool: %v", err)
        http.Error(w, "failed to register tool", http.StatusInternalServerError)
        return
    }
    logger.Debug("createTool: created id=%d name=%s client_id=%s", id, req.Name, req.ClientID)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "id":        id,
        "created_at": req.CreatedAt,
    })
}

func (h *Handler) deleteToolByID(w http.ResponseWriter, r *http.Request, id int64) {
    logger.Debug("deleteToolByID: id=%d", id)
    if err := h.repo.DeleteToolByID(r.Context(), id); err != nil {
        logger.Error("delete tool by id %d: %v", id, err)
        http.Error(w, "failed to delete tool", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
    logger.Debug("deleteToolByID: deleted id=%d", id)
}

// deleteToolChi parses {id} from chi route and deletes
func (h *Handler) deleteToolChi(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        logger.Debug("deleteToolChi: invalid id=%q", idStr)
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    h.deleteToolByID(w, r, id)
}

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
    logger.Debug("listTools: start")
    w.Header().Set("Content-Type", "application/json")
    items, err := h.repo.ListTools(r.Context())
    if err != nil {
        logger.Error("list tools: %v", err)
        http.Error(w, "failed to list tools", http.StatusInternalServerError)
        return
    }
    if items == nil {
        items = []*repoIface.Tool{}
    }
    logger.Debug("listTools: returned %d items", len(items))
    _ = json.NewEncoder(w).Encode(items)
}

func (h *Handler) getToolByIDChi(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        logger.Debug("getToolByIDChi: invalid id=%q", idStr)
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    item, err := h.repo.GetToolByID(r.Context(), id)
    if err != nil {
        logger.Error("get tool by id %d: %v", id, err)
        http.Error(w, "failed to get tool", http.StatusInternalServerError)
        return
    }
    if item == nil {
        logger.Debug("getToolByIDChi: not found id=%d", id)
        http.NotFound(w, r)
        return
    }
    logger.Debug("getToolByIDChi: found id=%d name=%s", id, item.Name)
    _ = json.NewEncoder(w).Encode(item)
}
