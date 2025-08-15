module.exports = function registerAgsRoutes(lti) {
  // Protected AGS helper routes under /tool/ags/* using ltijs Grade Service
  // Note: res.locals.token is set by ltijs for routes hit within an active launch session.
  lti.app.get('/ags/lineitems', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      const resourceLinkId = (req.query.resourceLinkId || token.platformContext?.resource?.id || token.platformContext?.resource?.linkId || '').toString();
      const opts = resourceLinkId ? { resourceLinkId } : {};
      const items = await lti.Grade.getLineItems(token, opts);
      return res.status(200).json(items);
    } catch (e) {
      return res.status(500).json({ error: 'listLineItemsFailed', message: e?.message || String(e) });
    }
  });

  lti.app.post('/ags/lineitems', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      const label = (req.query.label || 'Demo Item').toString();
      const scoreMaximum = parseFloat((req.query.scoreMaximum || '1').toString());
      const resourceLinkId = (req.query.resourceLinkId || token.platformContext?.resource?.id || token.platformContext?.resource?.linkId || '').toString();
      const payload = { label, scoreMaximum, resourceLinkId };
      const created = await lti.Grade.createLineItem(token, payload);
      return res.status(201).json(created);
    } catch (e) {
      return res.status(500).json({ error: 'createLineItemFailed', message: e?.message || String(e) });
    }
  });

  // Delete line item (accepts numeric id or full URL)
  lti.app.delete('/ags/lineitems/:id', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      const id = req.params.id;
      const base = token?.platformContext?.endpoint?.lineitems || token?.endpoint?.lineitems || '';
      const lineItemId = /^https?:\/\//.test(id) ? id : (base ? `${base.replace(/\/$/, '')}/${id}` : id);
      await lti.Grade.deleteLineItemById(token, lineItemId);
      return res.status(200).json({ deleted: true });
    } catch (e) {
      return res.status(500).json({ error: 'deleteLineItemFailed', message: e?.message || String(e) });
    }
  });

  lti.app.post('/ags/lineitems/:id/scores', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      const id = req.params.id;
      const now = new Date().toISOString();
      const scoreGiven = parseFloat((req.query.scoreGiven || '1').toString());
      const scoreMaximum = parseFloat((req.query.scoreMaximum || '1').toString());
      const activityProgress = (req.query.activityProgress || 'Completed').toString();
      const gradingProgress = (req.query.gradingProgress || 'FullyGraded').toString();
      const payload = {
        scoreGiven,
        scoreMaximum,
        activityProgress,
        gradingProgress,
        timestamp: now
      };
      // Build full line item URL from claim when numeric id provided
      const base = token?.platformContext?.endpoint?.lineitems || token?.endpoint?.lineitems || '';
      const lineItemId = /^https?:\/\//.test(id) ? id : (base ? `${base.replace(/\/$/, '')}/${id}` : id);
      const result = await lti.Grade.submitScore(token, lineItemId, payload);
      return res.status(200).json(result);
    } catch (e) {
      return res.status(500).json({ error: 'submitScoreFailed', message: e?.message || String(e) });
    }
  });

  lti.app.get('/ags/lineitems/:id/results', async (req, res) => {
    try {
      const token = res.locals.token;
      if (!token) return res.status(401).json({ error: 'noLaunch', message: 'Launch session not found. Re-launch the tool.' });
      const id = req.params.id;
      const base = token?.platformContext?.endpoint?.lineitems || token?.endpoint?.lineitems || '';
      const lineItemId = /^https?:\/\//.test(id) ? id : (base ? `${base.replace(/\/$/, '')}/${id}` : id);
      const out = await lti.Grade.getScores(token, lineItemId);
      return res.status(200).json(out);
    } catch (e) {
      return res.status(500).json({ error: 'getScoresFailed', message: e?.message || String(e) });
    }
  });
};
