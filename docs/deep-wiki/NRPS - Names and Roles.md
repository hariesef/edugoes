# NRPS: Names and Roles

Keywords: NRPS, memberships, contextmembership.readonly, Authorization Bearer, scopes, pagination, Link rel="next", roster

Files:
- `be/internal/controller/http/lti/handler_nrps.go`
- `be/internal/controller/http/lti/handler_nrps_auth.go`

Provides context memberships. PoC includes upsert/delete helpers.

## Auth
- Middleware `nrpsRequireScopes()` validates `Authorization: Bearer <JWT>` and required scopes against platform key.

## Endpoints
- GET `/api/nrps/contexts/{contextId}/members`
  - Query: `limit`, `offset`
  - Link header `rel="next"` if more pages
  - Response: `{ id, context: { id }, members: [] }`
- POST `/api/nrps/contexts/{contextId}/members` (PoC helper)
  - Body: `roster.Member`
- DELETE `/api/nrps/contexts/{contextId}/members/{userId}` (PoC helper)

## Notes
- Absolute URLs computed from forwarded headers when present; otherwise host/TLS.
- Members array is `[]` when empty (never null).
