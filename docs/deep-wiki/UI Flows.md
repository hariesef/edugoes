# UI Flows

Keywords: Deep Link flow, Resource Launch flow, OIDC, id_token, nrps, ags

## Deep Link Flow (Instructor)
1. FE posts to `/api/launch/start` with `lti_message_hint=deep_linking`.
2. BE sets correlation state and redirects to Tool auth.
3. Tool posts Deep Linking Response to BE return URL.
4. BE verifies JWT, persists selections, optionally creates AGS line items.
5. FE fetches selections from `/api/deeplink/selections`.

## Resource Launch Flow (Student)
1. FE posts to `/api/launch/start` with `target_link_uri` and `resource_link_id`.
2. BE OIDC issues `id_token` with `LtiResourceLinkRequest`, roles=Student, AGS/NRPS claims.
3. Form auto-posts to tool; tool launches content.

## NRPS / AGS
- NRPS: `GET /api/nrps/contexts/{contextId}/members` (requires scope).
- AGS: line items CRUD, scores, results under `/api/ags/contexts/{contextId}`.
