# Frontend Overview

Keywords: Vite, React, TypeScript, dev server 5173, proxy /api, ToolLaunch, HMR, ngrok, wss

- Location: `fe/`
- Stack: Vite + React + TypeScript (Node 16, Vite 4)
- Dev server: http://localhost:5173 (proxy `/api` → http://localhost:8080)
- Notable pages/components:
  - `src/pages/ToolLaunch.tsx` — LTI Deep Link and Resource Launch UI
  - `src/api/*` — API calls
  - `src/components/*` — shared UI

## Menus
- Tool Registration
- Tool Launch

## Getting Started
1. Install deps

   npm install

2. Run dev server

   npm run dev

## Notes
- Vite config allows ngrok domain for HMR over wss.
- Keep `login_hint` and `context_id` consistent with BE expectations.
- No routing yet; sidebar toggles views via local state.
- Style is minimal but modern, easy to extend.
