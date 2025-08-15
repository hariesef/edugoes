export function ToolRegistration() {
  return (
    <ToolRegistrationImpl />
  )
}

import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { createTool, deleteTool, listTools, type Tool } from '../api/tools'

function ToolRegistrationImpl() {
  const [name, setName] = useState('')
  const [clientId, setClientId] = useState('')
  const [authUrl, setAuthUrl] = useState('')
  const [tokenUrl, setTokenUrl] = useState('')
  const [targetLinkUrl, setTargetLinkUrl] = useState('')
  const [keySetUrl, setKeySetUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [tools, setTools] = useState<Tool[] | null>([])

  const canSubmit = useMemo(() => name.trim() !== '' && clientId.trim() !== '', [name, clientId])

  async function refresh() {
    try {
      setError(null)
      const data = await listTools()
      setTools(Array.isArray(data) ? data : [])
    } catch (e: any) {
      setError(e?.message ?? 'Failed to load tools')
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    setLoading(true)
    setError(null)
    try {
      await createTool({
        name: name.trim(),
        client_id: clientId.trim(),
        auth_url: authUrl.trim() || undefined,
        target_link_url: targetLinkUrl.trim() || undefined,
        token_url: tokenUrl.trim() || undefined,
        key_set_url: keySetUrl.trim() || undefined,
      } as any)
      setName('')
      setClientId('')
      setAuthUrl('')
      setTargetLinkUrl('')
      setTokenUrl('')
      setKeySetUrl('')
      await refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Failed to register tool')
    } finally {
      setLoading(false)
    }
  }

  async function onDelete(id: number) {
    try {
      await deleteTool(id)
      await refresh()
    } catch (e: any) {
      setError(e?.message ?? 'Failed to delete tool')
    }
  }

  return (
    <section>
      <h1>Tool Registration</h1>
      <form onSubmit={onSubmit} style={{ display: 'grid', gap: 8, maxWidth: 520 }}>
        <label>
          <div style={{ color: 'var(--text)' }}>Name *</div>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Tool name" />
        </label>
        <label>
          <div style={{ color: 'var(--text)' }}>Client ID *</div>
          <input value={clientId} onChange={(e) => setClientId(e.target.value)} placeholder="Client ID" />
        </label>
        <label>
          <div style={{ color: 'var(--text)' }}>Auth URL</div>
          <input value={authUrl} onChange={(e) => setAuthUrl(e.target.value)} placeholder="https://..." />
        </label>
        <label>
          <div style={{ color: 'var(--text)' }}>Target Link URL</div>
          <input value={targetLinkUrl} onChange={(e) => setTargetLinkUrl(e.target.value)} placeholder="https://... (if different from Auth URL)" />
        </label>
        <label>
          <div style={{ color: 'var(--text)' }}>Token URL</div>
          <input value={tokenUrl} onChange={(e) => setTokenUrl(e.target.value)} placeholder="https://..." />
        </label>
        <label>
          <div style={{ color: 'var(--text)' }}>JWKS URL</div>
          <input value={keySetUrl} onChange={(e) => setKeySetUrl(e.target.value)} placeholder="https://..." />
        </label>
        <div>
          <button disabled={!canSubmit || loading} type="submit">{loading ? 'Saving...' : 'Register Tool'}</button>
        </div>
      </form>

      {error && <p style={{ color: 'red' }}>{error}</p>}

      <h2 style={{ marginTop: 24 }}>Registered Tools</h2>
      <div style={{ display: 'grid', gap: 8 }}>
        {(tools?.length ?? 0) === 0 && <p>No tools yet.</p>}
        {tools?.map(t => (
          <div key={t.id} style={{ border: '1px solid rgba(255,255,255,0.12)', padding: 12, borderRadius: 8 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <strong>#{t.id} {t.name}</strong>
              <button onClick={() => onDelete(t.id)}>Delete</button>
            </div>
            <div style={{ fontSize: 12, color: 'var(--text)' }}>Client ID: {t.client_id}</div>
            {t.auth_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>Auth: {t.auth_url}</div>}
            {t.target_link_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>Target Link: {t.target_link_url}</div>}
            {t.token_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>Token: {t.token_url}</div>}
            {t.key_set_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>JWKS: {t.key_set_url}</div>}
          </div>
        ))}
      </div>
    </section>
  )
}
