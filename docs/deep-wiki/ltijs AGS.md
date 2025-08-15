# ltijs AGS

Keywords: ltijs, AGS, lineitems, scores, results, resourceLinkId, submitScore, getLineItems, getScores, createLineItem, deleteLineItemById

File: `ltijs/routes/ags.js`

Protected helper routes (require active launch; `res.locals.token` set by ltijs). Typically exposed under `/tool/ags/*` when reverse-proxied.

## Routes
- GET `/ags/lineitems`
  - Query: `resourceLinkId` (optional; defaults from token `platformContext.resource.id|linkId`)
  - Uses `lti.Grade.getLineItems(token, { resourceLinkId? })`
  - 200 → array of line items
- POST `/ags/lineitems`
  - Query: `label` (default "Demo Item"), `scoreMaximum` (default 1), `resourceLinkId` (optional)
  - Uses `lti.Grade.createLineItem(token, { label, scoreMaximum, resourceLinkId })`
  - 201 → created item JSON
- DELETE `/ags/lineitems/:id`
  - `:id` can be numeric id or full URL; builds full URL from token lineitems base
  - Uses `lti.Grade.deleteLineItemById(token, lineItemId)`
  - 200 → `{ deleted: true }`
- POST `/ags/lineitems/:id/scores`
  - Query: `scoreGiven`, `scoreMaximum`, `activityProgress`, `gradingProgress`
  - Builds `timestamp` automatically
  - Uses `lti.Grade.submitScore(token, lineItemId, payload)`
  - 200 → result object
- GET `/ags/lineitems/:id/results`
  - Uses `lti.Grade.getScores(token, lineItemId)`
  - 200 → results array

## Notes
- `lineItemId` URL is constructed from token claims if a bare id is provided.
- Errors return JSON with `error` and `message` fields.
- These endpoints assume an authenticated tool launch context; direct cURL without session will 401.
