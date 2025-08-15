# ltijs Overview

Keywords: ltijs, Node, Express, tool, launch, login, keys, jwks, ngrok, nginx, proxy, /tool

- Directory: `ltijs/`
- Purpose: Minimal LTI 1.3 Tool implementation using `ltijs` for local testing against the Go Platform.
- Entrypoint: `ltijs/index.js` (Express app powered by ltijs). Public routes typically exposed under `/tool/*` via reverse proxy.
- Docs: `ltijs/README.md`

## Public Endpoints (via reverse proxy)
- `/tool/login` — Login initiation
- `/tool/launch` — Tool launch
- `/tool/keys` — JWKS keys
- `/tool/.well-known/jwks.json` — JWKS (alternative path)

See `ltijs/README.md` for the Nginx snippet that maps `/tool/` to the Node server.
