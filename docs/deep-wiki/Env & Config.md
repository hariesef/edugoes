# Env & Config

Keywords: keys, RS256, issuer, PUBLIC_BASE_URL, JWKS, client_id, auth_url, target_link_url, key_set_url

- Keys: `be/pkg/common/keys` must be initialized; platform signs `id_token` and validates Bearer tokens.
- Issuer: `Handler.issuer` must be set to platform issuer (e.g., `https://<host>`).
- `PUBLIC_BASE_URL`: override for URLs embedded in tokens and API responses.
- Registered Tools (repository): `client_id`, `auth_url`, `target_link_url`, `key_set_url` required for proper flows.
