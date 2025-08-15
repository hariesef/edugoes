# ltijs NRPS

Keywords: ltijs, NRPS, memberships, NamesAndRoles.getMembers, context_memberships_url, pagination, limit, offset

File: `ltijs/routes/nrps.js`

Protected helper route (requires active launch).

## Routes
- GET `/nrps/members`
  - Discovers NRPS URL from token (tries multiple mapped claim locations):
    - `token.platformContext.namesRoles.context_memberships_url`
    - `token.platformContext.endpoint.memberships`
    - `token.platformContext.nrps.context_memberships_url`
    - `token.nrps.context_memberships_url`
    - `token.namesroleservice.context_memberships_url`
  - Query: `limit` (default 50), `offset` (default 0)
  - Uses `lti.NamesAndRoles.getMembers(token, nrpsUrl, { limit, offset })`
  - 200 → `{ ...members array... }`

## Notes
- If NRPS URL not present in the token, returns `400` (`nrpsUrlMissing`).
- Errors from ltijs helper return `502` with message.
- Intended to be invoked from the tool UI while in an active LTI session.
