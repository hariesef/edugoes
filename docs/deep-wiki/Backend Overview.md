# Backend Overview

- Keywords: LTI, OIDC, id_token, deep_linking, NRPS, AGS, lineitems, scores, results, memberships, resource_link_id, context_id, login_hint, target_link_uri, PUBLIC_BASE_URL, issuer, JWKS

- Location: `be/`
- Primary package: `be/internal/controller/http/lti/`
- Key handlers:
  - `handler_oidc.go` — OIDC auth + LTI id_token issuance
  - `handler_deeplink.go` — Deep Linking response intake and persistence
  - `handler_nrps.go` — NRPS memberships API
  - `handler_nrps_auth.go` — Bearer token + scope enforcement (NRPS)
  - `handler_ags.go` — AGS line items, scores, and results
- Common utilities:
  - `be/pkg/common/keys` — signing keys (RS256)
  - `be/pkg/common/logger` — logging
  - `be/pkg/repositories/*` — tools, deep link selections, scores, roster

## HTTP base
- Platform issuer is configured on `Handler` (server init).
- `PUBLIC_BASE_URL` can override self-referential URLs in tokens and APIs.
