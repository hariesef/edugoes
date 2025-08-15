package lti

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/quipper/poc/lti/be/pkg/common/jwkscache"
	"github.com/quipper/poc/lti/be/pkg/common/keys"
	repoIface "github.com/quipper/poc/lti/be/pkg/repositories/lti"
	rosterRepo "github.com/quipper/poc/lti/be/pkg/repositories/roster"
	scoresRepo "github.com/quipper/poc/lti/be/pkg/repositories/scores"
	vRepoIface "github.com/quipper/poc/lti/be/pkg/repositories/validation"
)

type Handler struct {
	repo           repoIface.Repository
	scores         scoresRepo.Repository
	roster         rosterRepo.Repository
	issuer         string
	jwksCache      jwkscache.Cache
	validationRepo vRepoIface.Repository
}

// NewHandler constructs a Handler with explicit tools, scores and validation repositories.
// Useful when these come from different backends or databases.
func NewHandler(tools repoIface.Repository, scores scoresRepo.Repository, validation vRepoIface.Repository, roster rosterRepo.Repository) *Handler {
	iss := os.Getenv("PLATFORM_ISSUER")
	if iss == "" {
		iss = "https://monarch-legal-admittedly.ngrok-free.app"
	}
	return &Handler{
		repo:           tools,
		scores:         scores,
		roster:         roster,
		issuer:         iss,
		jwksCache:      jwkscache.Default(),
		validationRepo: validation,
	}
}

// Router returns a chi-based router for the /api endpoints.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/api/health", h.health)
	r.Get("/api/hello", h.hello)

	// Platform metadata/JWKS
	r.Get("/.well-known/jwks.json", h.jwks)
	r.Get("/api/.well-known/jwks.json", h.jwks)

	// LTI launch (3rd-party initiated login)
	r.Post("/api/launch/start", h.launchStart)
	// OIDC auth endpoint (issues id_token via form_post)
	r.Get("/api/oidc/auth", h.oidcAuth)
	r.Post("/api/oidc/auth", h.oidcAuth)
	// Deep Linking return endpoint (form_post from tool)
	r.Get("/api/deeplink/return", h.deeplinkReturn)
	r.Post("/api/deeplink/return", h.deeplinkReturn)
	r.Post("/api/oauth2/token", h.oauth2Token)
	r.Get("/api/tools", h.listTools)
	r.Get("/api/tools/{id}", h.getToolByIDChi)
	r.Post("/api/tools", h.createTool)
	r.Delete("/api/tools/{id}", h.deleteToolChi)

	// Deep link selections CRUD (list/get/delete)
	r.Get("/api/deeplink/selections", h.listSelections)
	r.Get("/api/deeplink/selections/{id}", h.getSelectionByID)
	r.Delete("/api/deeplink/selections/{id}", h.deleteSelectionByID)

	// AGS endpoints (context-scoped)
	r.Route("/api/ags/contexts/{contextId}", func(r chi.Router) {
		// Line items
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/lineitem.readonly", "https://purl.imsglobal.org/spec/lti-ags/scope/lineitem")).Get("/lineitems", h.agsListLineItems)
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/lineitem")).Post("/lineitems", h.agsCreateLineItem)
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/lineitem.readonly", "https://purl.imsglobal.org/spec/lti-ags/scope/lineitem")).Get("/lineitems/{lineItemId}", h.agsGetLineItem)
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/lineitem")).Put("/lineitems/{lineItemId}", h.agsUpdateLineItem)
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/lineitem")).Delete("/lineitems/{lineItemId}", h.agsDeleteLineItem)
		// Scores and Results
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/score")).Post("/lineitems/{lineItemId}/scores", h.agsPostScore)
		r.With(h.agsRequireScopes("https://purl.imsglobal.org/spec/lti-ags/scope/result.readonly")).Get("/lineitems/{lineItemId}/results", h.agsListResults)
	})

	// NRPS endpoints (context-scoped)
	r.Route("/api/nrps/contexts/{contextId}", func(r chi.Router) {
		// Memberships list (readonly scope per spec)
		r.With(h.nrpsRequireScopes("https://purl.imsglobal.org/spec/lti-nrps/scope/contextmembership.readonly")).Get("/members", h.nrpsListMembers)
		// Sandbox helpers to manage a local roster (no official write scope in spec; keep unscoped or protect with same scope)
		r.Post("/members", h.nrpsUpsertMember)
		r.Delete("/members/{userId}", h.nrpsDeleteMember)
	})
	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Health(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "unhealthy", "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Hello from Go backend"})
}

// jwks serves the platform JWKS (public keys for JWT verification by tools).
func (h *Handler) jwks(w http.ResponseWriter, r *http.Request) {
	if err := keys.Init(); err != nil {
		http.Error(w, "failed to initialize JWKS", http.StatusInternalServerError)
		return
	}
	data, err := keys.JWKSJSON()
	if err != nil {
		http.Error(w, "failed to get JWKS", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
