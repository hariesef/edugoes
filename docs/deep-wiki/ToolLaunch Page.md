# ToolLaunch Page

Keywords: Resource Launch, Deep Link, formTarget, target_link_uri, login_hint, context_id, client_id

File: `fe/src/pages/ToolLaunch.tsx`

## Features
- List registered tools and trigger Deep Link flow (POST `/api/launch/start` with `lti_message_hint=deep_linking`).
- Show Deep Link selections with metadata.
- Resource Launch per selection (opens in new tab via `formTarget="_blank"`).

## Hidden fields used
- issuer (platform)
- client_id
- login_initiation_url
- target_link_uri
- resource_link_id (on resource launches)
- login_hint (email)
- context_id

## Tips
- Ensure `tool.auth_url` (login initiation) and `tool.target_link_url` are configured in BE tool registry.
- For selection deletion, page refreshes list from `/api/deeplink/selections`.
