module.exports = function registerNrpsRoutes(lti) {
  // Protected NRPS helper: list members via ltijs using the token's NRPS endpoint
  lti.app.get('/nrps/members', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      // Pull NRPS URL from claim(s)
      const namesRoles = token?.platformContext?.namesRoles;
      // Try to derive directly from token-exposed props first
      let nrpsUrl = namesRoles?.context_memberships_url
        || token?.platformContext?.endpoint?.memberships
        || token?.platformContext?.nrps?.context_memberships_url
        || token?.nrps?.context_memberships_url
        || token?.namesroleservice?.context_memberships_url
        || '';
      // No more fallbacks: if not present in mapped claims, error out
      console.debug('[NRPS] using nrpsUrl =', nrpsUrl);
      if (!nrpsUrl) return res.status(400).json({ error: 'nrpsUrlMissing', message: 'NRPS URL not found in token.' });
      const limit = parseInt((req.query.limit || '50').toString(), 10);
      const offset = parseInt((req.query.offset || '0').toString(), 10);
      const params = { limit, offset };
      console.debug('[NRPS] params =', params);
      // Use ltijs helper (handles token acquisition internally)
      try {
        const members = await lti.NamesAndRoles.getMembers(token, nrpsUrl, params);
        return res.status(200).json(members);
      } catch (e1) {
        console.error('[NRPS] NamesAndRoles.getMembers failed:', e1);
        return res.status(502).json({ error: 'nrpsMembersFailed', message: e1?.message || String(e1) });
      }
    } catch (e) {
      console.error('[NRPS] listMembers error:', e);
      return res.status(500).json({ error: 'listMembersFailed', message: e?.message || String(e) });
    }
  });
};
