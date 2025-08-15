# AGS: Assignments and Grades Service

Keywords: AGS, lineitems, results, scores, scoreMaximum, resource_link_id, ContextID, Location header, PUBLIC_BASE_URL

File: `be/internal/controller/http/lti/handler_ags.go`

Implements line items CRUD, posting scores, and listing results per LTI 1.3 AGS.

## Endpoints
- GET `/api/ags/contexts/{contextId}/lineitems`
  - Optional filter `resource_link_id`
  - Returns `[]apiLineItem` with `id` as URL
- POST `/api/ags/contexts/{contextId}/lineitems`
  - Body: `scores.LineItem` (server sets `ContextID`)
  - Requires `scoreMaximum`; 201 with Location header
- GET `/api/ags/contexts/{contextId}/lineitems/{lineItemId}`
- PUT `/api/ags/contexts/{contextId}/lineitems/{lineItemId}`
- DELETE `/api/ags/contexts/{contextId}/lineitems/{lineItemId}`
- POST `/api/ags/contexts/{contextId}/lineitems/{lineItemId}/scores`
  - Body: `scores.Score`; sets `Timestamp` if missing; 204
- GET `/api/ags/contexts/{contextId}/lineitems/{lineItemId}/results`

## URL building
- Uses `PUBLIC_BASE_URL` if set; else X-Forwarded headers or request Host.

## Logging
- Raw bodies logged on create/update/score for debugging.
