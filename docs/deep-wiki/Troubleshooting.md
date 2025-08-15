# Troubleshooting

Keywords: redirect_uri mismatch, correlation cookie, lti_corr, JWKS, roles, Student, Instructor, deep_linking

- redirect_uri mismatch: strict scheme/host/path match to tool config (`TargetLinkURL` or `AuthURL`).
- Missing correlation cookie: `lti_corr` absent/expired; restart launch via `/api/launch/start`.
- Deep Link JWT not verified: tool JWKS not configured or unreachable; items may still persist; check logs.
- Roles appear wrong: PoC sets Student when `resource_link_id` present; adjust for real role mapping.
