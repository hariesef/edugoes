require('dotenv').config({ override: true });
const express = require('express');
const path = require('path');
const lti = require('ltijs').Provider; // note: this is an instance, not a class
const Keyset = require('ltijs/dist/Utils/Keyset');

// Env
const PORT = process.env.PORT || 4000;
const ENCRYPTION_KEY = process.env.ENCRYPTION_KEY || 'dev-encryption-key-change-me';
const LTI_KEY = process.env.LTI_KEY || 'dev-lti-key-change-me';
const TOOL_URL = process.env.TOOL_URL || `http://localhost:${PORT}`;
const MONGO_URL = process.env.MONGO_URL || 'mongodb://localhost:27017/ltijs';
const BASE_PLATFORM_URL = process.env.BASE_PLATFORM_URL || 'https://monarch-legal-admittedly.ngrok-free.app';
console.log('[ltijs] Using Mongo URL:', MONGO_URL);
console.log('[ltijs] Using Base Platform URL:', BASE_PLATFORM_URL);

// Setup ltijs instance
lti.setup(ENCRYPTION_KEY, { url: MONGO_URL }, {
  appRoute: '/launch',
  loginRoute: '/login',
  logger: true,
  staticPath: path.join(__dirname, 'public'),
  cookies: { secure: false, sameSite: 'Lax' }
});

// Whitelist public routes so they bypass Ltijs auth
lti.whitelist('/healthz', '/keys', '/.well-known/jwks.json');

(async () => {
  try {
    // Deploy provider
    await lti.deploy({
      serverless: false,
      port: PORT,
      serverAddon: (app) => {
        // Additional express routes can be added here if needed
      }
    });


    // Public health endpoint (must be added via lti.app to bypass LTI auth)
    lti.app.get('/healthz', (_, res) => res.send('ok'));

    // Set up basic routes/handlers
    lti.onConnect(async (token, req, res) => {
      const resourceLinkId = token.platformContext?.resource?.id || token.platformContext?.resource?.linkId || '';
      const userId = token.user;
      // AGS Sandbox UI
      const ltik = (req.query?.ltik || '').toString();
      // Derive NRPS context memberships URL from mapped claims (no fallbacks)
      const namesRoles = token?.platformContext?.namesRoles;
      let nrpsUrl = namesRoles?.context_memberships_url
        || token?.platformContext?.endpoint?.memberships
        || token?.platformContext?.nrps?.context_memberships_url
        || token?.nrps?.context_memberships_url
        || token?.namesroleservice?.context_memberships_url
        || '';
      return res.status(200).send(`
        <html>
          <head>
            <meta charset="utf-8" />
            <title>ltijs launch - AGS Sandbox</title>
            <style>
              body { font-family: system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif; margin: 20px; }
              h1 { margin-bottom: 8px; }
              .row { margin: 8px 0; }
              button { margin-right: 8px; }
              input, select { padding: 4px 6px; }
              fieldset { margin: 12px 0; }
              pre { background: #f6f8fa; padding: 12px; border-radius: 6px; max-height: 420px; overflow: auto; }
              label { display: inline-block; min-width: 150px; }
            </style>
          </head>
          <body>
            <h1>ltijs launch</h1>
            <p><strong>User:</strong> ${userId}</p>
            <p><strong>Roles:</strong> ${JSON.stringify(token.platformContext?.roles || [])}</p>
            <p><strong>Resource Link ID:</strong> <code id="resId">${resourceLinkId}</code></p>

            <fieldset>
              <legend>Line Items</legend>
              <div class="row">
                <button id="btnList">List line items</button>
                <label>resourceLinkId</label>
                <input id="rlid" value="${resourceLinkId}" />
              </div>
              <div class="row">
                <label>Label</label>
                <input id="liLabel" value="Demo Item" />
                <label>Score Maximum</label>
                <input id="liMax" type="number" step="0.1" value="1" />
                <button id="btnCreate">Create line item</button>
              </div>
              <div class="row">
                <label>Line Item ID</label>
                <input id="liId" placeholder="numeric id" />
                <button id="btnDelete" style="background:#fee">Delete line item</button>
              </div>
            </fieldset>

            <fieldset>
              <legend>Scores & Results</legend>
              <div class="row">
                <label>Line Item ID</label>
                <input id="scLiId" placeholder="numeric id" />
              </div>
              <div class="row">
                <label>scoreGiven</label>
                <input id="scoreGiven" type="number" step="0.1" value="1" />
                <label>scoreMaximum</label>
                <input id="scoreMax" type="number" step="0.1" value="1" />
              </div>
              <div class="row">
                <label>activityProgress</label>
                <select id="actProg">
                  <option>Initialized</option>
                  <option>InProgress</option>
                  <option>Submitted</option>
                  <option selected>Completed</option>
                </select>
                <label>gradingProgress</label>
                <select id="gradProg">
                  <option>NotReady</option>
                  <option selected>FullyGraded</option>
                </select>
                <button id="btnScore">Post score</button>
                <button id="btnResults">List scores</button>
              </div>
            </fieldset>

            <fieldset>
              <legend>Roster (NRPS)</legend>
              <div class="row">
                <label>Context ID</label>
                <code id="ctxId">${token.platformContext?.context?.id || ''}</code>
              </div>
              <div class="row">
                <label>limit</label>
                <input id="nrpsLimit" type="number" value="50" />
                <label>offset</label>
                <input id="nrpsOffset" type="number" value="0" />
                <button id="btnNrpsList">List members (ltijs)</button>
              </div>
              <div class="row">
                <label>userId</label>
                <input id="nrpsUserId" placeholder="user id" />
              </div>
              <div class="row">
                <label>name</label>
                <input id="nrpsName" placeholder="Full Name" />
                <label>given_name</label>
                <input id="nrpsGiven" placeholder="Given" />
                <label>family_name</label>
                <input id="nrpsFamily" placeholder="Family" />
                <label>role</label>
                <select id="nrpsRole">
                  <option value="http://purl.imsglobal.org/vocab/lis/v2/membership#Learner">Learner</option>
                  <option value="http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor">Instructor</option>
                </select>
              </div>
              <div class="row">
                <button id="btnNrpsUpsert">Upsert member (calls backend)</button>
                <button id="btnNrpsDelete" style="background:#fee">Delete member (calls backend)</button>
              </div>
            </fieldset>

            <h3>Response</h3>
            <pre id="out">(no calls yet)</pre>

            <script>
              const LTIK = ${JSON.stringify(ltik)};
              const out = document.getElementById('out');
              function log(data) { out.textContent = typeof data === 'string' ? data : JSON.stringify(data, null, 2); }
              const CTX = ${JSON.stringify((token.platformContext?.context?.id || ''))};
              const NRPS_URL = ${JSON.stringify(nrpsUrl)};
              function withLtik(url) {
                if (!LTIK) return url;
                const sep = url.includes('?') ? '&' : '?';
                return url + sep + 'ltik=' + encodeURIComponent(LTIK);
              }
              async function fetchJSON(url, opts) {
                const r = await fetch(withLtik(url), { credentials: 'same-origin', ...(opts||{}) });
                const text = await r.text();
                try { return { status: r.status, json: JSON.parse(text) }; } catch { return { status: r.status, text } }
              }

              document.getElementById('btnList').onclick = async () => {
                const rlid = document.getElementById('rlid').value;
                const url = '/tool/ags/lineitems' + (rlid ? ('?resourceLinkId=' + encodeURIComponent(rlid)) : '');
                const res = await fetchJSON(url);
                log(res);
              };
              document.getElementById('btnCreate').onclick = async () => {
                const label = encodeURIComponent(document.getElementById('liLabel').value);
                const max = encodeURIComponent(document.getElementById('liMax').value);
                const rlid = encodeURIComponent(document.getElementById('rlid').value);
                const res = await fetchJSON('/tool/ags/lineitems?label=' + label + '&scoreMaximum=' + max + '&resourceLinkId=' + rlid, { method: 'POST' });
                log(res);
              };
              document.getElementById('btnDelete').onclick = async () => {
                const id = document.getElementById('liId').value;
                const res = await fetchJSON('/tool/ags/lineitems/' + encodeURIComponent(id), { method: 'DELETE' });
                log(res);
              };
              document.getElementById('btnScore').onclick = async () => {
                const li = document.getElementById('scLiId').value || document.getElementById('liId').value;
                const sg = encodeURIComponent(document.getElementById('scoreGiven').value);
                const sm = encodeURIComponent(document.getElementById('scoreMax').value);
                const ap = encodeURIComponent(document.getElementById('actProg').value);
                const gp = encodeURIComponent(document.getElementById('gradProg').value);
                const res = await fetchJSON('/tool/ags/lineitems/' + encodeURIComponent(li) + '/scores?scoreGiven=' + sg + '&scoreMaximum=' + sm + '&activityProgress=' + ap + '&gradingProgress=' + gp, { method: 'POST' });
                log(res);
              };
              document.getElementById('btnResults').onclick = async () => {
                const li = document.getElementById('scLiId').value || document.getElementById('liId').value;
                const res = await fetchJSON('/tool/ags/lineitems/' + encodeURIComponent(li) + '/results');
                log(res);
              };

              // NRPS helpers
              document.getElementById('btnNrpsList').onclick = async () => {
                const limit = parseInt(document.getElementById('nrpsLimit').value || '50', 10);
                const offset = parseInt(document.getElementById('nrpsOffset').value || '0', 10);
                const url = '/tool/nrps/members' + '?limit=' + encodeURIComponent(limit) + '&offset=' + encodeURIComponent(offset);
                const res = await fetchJSON(url);
                log(res);
              };

              document.getElementById('btnNrpsUpsert').onclick = async () => {
                const userId = document.getElementById('nrpsUserId').value;
                const name = document.getElementById('nrpsName').value;
                const given = document.getElementById('nrpsGiven').value;
                const family = document.getElementById('nrpsFamily').value;
                const role = document.getElementById('nrpsRole').value;
                if (!userId) { log('userId is required'); return; }
                const payload = { user_id: userId, name, given_name: given, family_name: family, roles: [role] };
                if (!NRPS_URL) { log('NRPS URL missing from launch token'); return; }
                const res = await fetchJSON(NRPS_URL, {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify(payload)
                });
                log(res);
              };

              document.getElementById('btnNrpsDelete').onclick = async () => {
                const userId = document.getElementById('nrpsUserId').value;
                if (!userId) { log('userId is required'); return; }
                if (!NRPS_URL) { log('NRPS URL missing from launch token'); return; }
                const res = await fetchJSON(NRPS_URL + '/' + encodeURIComponent(userId), { method: 'DELETE' });
                log(res);
              };
            </script>
          </body>
        </html>
      `);
    });

    // Deep linking handler (selection page)
    lti.onDeepLinking(async (token, req, res) => {
      // Simple picker with one item
      const returnUrl = token.platformContext.deepLinkingSettings?.deep_link_return_url;
      const data = token.platformContext.deepLinkingSettings?.data;
      // Build a simple content item (link)
      const items = [
        {
          type: 'ltiResourceLink',
          title: 'Sample Content',
          url: `${BASE_PLATFORM_URL}/tool/launch`,
          text: 'Sample link',
        }
      ];
      const html = await lti.DeepLinking.createDeepLinkingForm(token, items, { data });
      return res.status(200).send(html);
    });


    // Well-known JWKS (mirror of /keys)
    lti.app.get('/.well-known/jwks.json', async (req, res, next) => {
      try {
        const keyset = await Keyset.build(lti.Database, ENCRYPTION_KEY);
        res.setHeader('Content-Type', 'application/json');
        return res.status(200).send(keyset);
      } catch (e) {
        return next(e);
      }
    });

    // Register external route modules
    require(path.join(__dirname, 'routes', 'nrps'))(lti);
    require(path.join(__dirname, 'routes', 'ags'))(lti);

    // registration default 
    await lti.registerPlatform({
      name: 'haries-platform',
      url: BASE_PLATFORM_URL,
      clientId: 'haries-client-id',
      authenticationEndpoint: `${BASE_PLATFORM_URL}/api/oidc/auth`,
      accesstokenEndpoint: `${BASE_PLATFORM_URL}/api/oauth2/token`,
      authConfig: { method: 'JWK_SET', key: `${BASE_PLATFORM_URL}/.well-known/jwks.json` }
    }).catch(() => {});
    console.log(`ltijs listening on ${TOOL_URL}`);
  } catch (err) {
    console.error('ltijs error:', err);
    process.exit(1);
  }
})();
