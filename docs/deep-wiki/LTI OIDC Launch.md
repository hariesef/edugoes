# LTI OIDC Launch

Keywords: OIDC, id_token, LtiResourceLinkRequest, LtiDeepLinkingRequest, target_link_uri, resource_link_id, login_hint, nonce, state, deep_linking, AGS claim, NRPS claim, services claim, issuer, audience

File: `be/internal/controller/http/lti/handler_oidc.go`

Accepts OIDC auth request from Tool, validates state, issues LTI 1.3 `id_token`, and form_post backs to the tool.

## Flow
1. Tool redirects user to platform auth with `client_id`, `redirect_uri`, `state`, `nonce`, optional `lti_message_hint`, `login_hint`.
2. Platform validates correlation cookie `lti_corr` and consumes stored state via `validationRepo.ConsumeOIDCState()`.
3. Validates `redirect_uri` host/path against tool `TargetLinkURL` (fallback `AuthURL`).
4. Build `id_token` (RS256):
   - `iss` = platform issuer
   - `aud` = tool `client_id`
   - `exp` ~ 5 min
   - LTI claims:
     - `.../claim/message_type`: `LtiResourceLinkRequest` or `LtiDeepLinkingRequest` (when `lti_message_hint == deep_linking`)
     - `.../claim/target_link_uri`: from state
     - `.../claim/resource_link` with `id` (resource launch)
     - `.../lti-ags/claim/endpoint`: AGS endpoints + scopes
     - `.../lti-nrps/claim/namesroleservice`: NRPS endpoint
     - `.../spec/lti/claim/service`: advertised token endpoints and scopes
   - Roles: defaults Instructor; PoC sets Student when `resource_link_id` present
   - Subject/user: `sub` from `login_hint`; email/name filled for PoC

## Response
- HTML form auto-submits (`form_post`) to tool `redirect_uri` with `id_token` and `state`.

## Notes
- Requires correlation cookie; cookie cleared after use.
- `PUBLIC_BASE_URL` overrides URLs in claims.
