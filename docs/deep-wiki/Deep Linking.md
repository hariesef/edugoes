# Deep Linking

Keywords: deep_linking, content_items, JWT, JWKS, client_id, contextId, lineItem, label, scoreMaximum, resourceLinkID, CreateDeepLinkSelection, CreateLineItem

File: `be/internal/controller/http/lti/handler_deeplink.go`

Receives the Tool's Deep Linking Response (`JWT` form field), verifies via Tool JWKS, persists selections, and may create AGS line items.

## Endpoint
- POST to platform return URL (provided in deep_linking_settings).
- Reads `JWT` (fallback `id_token`).
- Verifies against each tool `KeySetURL` (best-effort).

## Behavior
- Decodes payload and extracts:
  - `aud` → client_id
  - `.../lti-dl/claim/data` → contextId
  - `.../lti-dl/claim/content_items` → items array
- For each content item:
  - Persist via `repo.CreateDeepLinkSelection` with `client_id`, `tool_name`, `url`, `content_item_json`.
  - If `lineItem` present with `label`, `scoreMaximum`, and we have `contextId`:
    - Create AGS line item (`scores.CreateLineItem`) with `ContextID=contextId`, `ResourceLinkID=selection.id`.
    - Create mapping selection.id ↔ lineitem.id.

## Response
- Renders HTML with verification status and pretty-printed JWT claims.
