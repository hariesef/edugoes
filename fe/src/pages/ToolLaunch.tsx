import { useEffect, useState } from 'react'
import { listTools, type Tool } from '../api/tools'
import { PLATFORM_ISSUER } from '../config'

export function ToolLaunch() {
  const [tools, setTools] = useState<Tool[]>([])
  const [selections, setSelections] = useState<any[]>([])
  const [expanded, setExpanded] = useState<Record<number, boolean>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  // Controls for resource launch
  const [email, setEmail] = useState('student@efrika.net')
  const [contextId, setContextId] = useState('dev-context')

  useEffect(() => {
    listTools().then(setTools).catch(console.error)
    fetch('/api/deeplink/selections')
      .then(r => r.json())
      .then((data) => {
        if (Array.isArray(data)) setSelections(data)
        else setSelections([])
      })
      .catch((e) => {
        console.error(e)
        setSelections([])
      })
  }, [])

  useEffect(() => {
    let mounted = true
    listTools()
      .then((items) => {
        if (mounted) setTools(items)
      })
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false))
    return () => {
      mounted = false
    }
  }, [])

  return (
    <div style={{ padding: 24 }}>
      <h2>Tool Launch</h2>
      {/* Global context controls */}
      <div style={{ display: 'flex', gap: 12, alignItems: 'center', margin: '8px 0 16px' }}>
        <label style={{ display: 'flex', flexDirection: 'column', fontSize: 12 }}>
          <span style={{ marginBottom: 4 }}>contextId</span>
          <input
            type="text"
            value={contextId}
            onChange={(e) => setContextId(e.target.value)}
            placeholder="dev-context"
            style={{ padding: 6, minWidth: 240 }}
          />
        </label>
      </div>
      {loading && <p>Loading tools…</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {!loading && tools.length === 0 && <p>No tools registered yet.</p>}

      <ul style={{ listStyle: 'none', padding: 0 }}>
        {tools.map((t) => (
          <li key={t.id} style={{ border: '1px solid #ddd', padding: '12px', marginBottom: '8px', borderRadius: 6 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontWeight: 600 }}>{t.name}</div>
                <div style={{ fontSize: 12, color: 'var(--text)' }}>client_id: {t.client_id}</div>
                {t.auth_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>login_initiation_url: {t.auth_url}</div>}
                {t.target_link_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>target_link_url: {t.target_link_url}</div>}
                {t.target_launch_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>target_launch_url: {t.target_launch_url}</div>}
                {t.key_set_url && <div style={{ fontSize: 12, color: 'var(--text)' }}>jwks: {t.key_set_url}</div>}
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <form method="post" action="/api/launch/start">
                  <input type="hidden" name="issuer" value={PLATFORM_ISSUER} />
                  <input type="hidden" name="client_id" value={t.client_id} />
                  <input type="hidden" name="login_initiation_url" value={t.auth_url || ''} />
                  <input type="hidden" name="target_link_uri" value={t.target_link_url || t.auth_url || ''} />
                  <input type="hidden" name="lti_message_hint" value="deep_linking" />
                  <input type="hidden" name="login_hint" value="haries@efrika.net" />
                  <input type="hidden" name="context_id" value={contextId} />
                  <button type="submit" disabled={!t.auth_url}>
                    Deep Link
                  </button>
                </form>
              </div> 
            </div>
          </li>
        ))}
      </ul>

      <div style={{ marginTop: 36 }}>
        <h2>Deep Link Selections</h2>
        {/* Launch parameters */}
        <div style={{ display: 'flex', gap: 12, alignItems: 'center', margin: '8px 0 16px' }}>
          <label style={{ display: 'flex', flexDirection: 'column', fontSize: 12 }}>
            <span style={{ marginBottom: 4 }}>email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="student@efrika.net"
              style={{ padding: 6, minWidth: 240 }}
            />
          </label>
        </div>
        {(Array.isArray(selections) ? selections.length : 0) === 0 ? (
          <p>No selections yet.</p>
        ) : (
          <ul>
            {(Array.isArray(selections) ? selections : []).map((s:any) => {
              let parsed: any = null
              try {
                parsed = s.content_item_json ? JSON.parse(s.content_item_json) : null
              } catch {}
              const li = parsed && parsed.lineItem ? parsed.lineItem : null
              const pretty = parsed
                ? JSON.stringify(parsed, null, 2)
                : (s.content_item_json || '(no content_item_json)')
              const open = !!expanded[s.id]
              const targetContentUrl: string = s.url || parsed?.url || ''
              let toolFor = tools.find((t) => t.client_id === s.client_id)
              if (!toolFor && s.tool_name) {
                toolFor = tools.find((t) => (t.name || '').toLowerCase() === String(s.tool_name || '').toLowerCase())
              }
              if (!toolFor && targetContentUrl) {
                try {
                  const u = new URL(targetContentUrl)
                  toolFor = tools.find((t) => {
                    try {
                      const au = t.auth_url ? new URL(t.auth_url) : null
                      return !!au && au.hostname === u.hostname
                    } catch { return false }
                  })
                } catch {}
              }
              const launchUrl = toolFor?.target_link_url || toolFor?.auth_url || ''
              const disabledReason = !toolFor
                ? 'Tool not found for this selection (client_id mismatch).'
                : (!toolFor.auth_url ? 'Tool has no login_initiation_url (auth_url) configured.' : (!launchUrl ? 'Missing tool launch URL.' : ''))
              const isDisabled = !toolFor || !toolFor.auth_url || !launchUrl
              return (
                <li key={s.id} style={{ marginBottom: 12, border:'1px solid #eee', borderRadius:8, padding:12 }}>
                  <div style={{ display:'flex', gap:12, alignItems:'center', justifyContent:'space-between' }}>
                    <div style={{ flex:1 }}>
                      <strong>{s.title || parsed?.title || '(untitled)'}</strong>
                      {s.tool_name ? <span style={{ marginLeft:8, color:'var(--text)' }}>from {s.tool_name}</span> : null}
                      {(s.url || parsed?.url) ? (
                        <div><a href={s.url || parsed?.url} target="_blank" rel="noreferrer">{s.url || parsed?.url}</a></div>
                      ) : null}
                      {li ? (
                        <div style={{ fontSize:12, color:'var(--text)', marginTop:4 }}>
                          <em>LineItem:</em> {li.label || '(no label)'} · max {li.scoreMaximum ?? '—'}
                        </div>
                      ) : null}
                    </div>
                    <div style={{ display:'flex', gap:8 }}>
                      <form method="post" action="/api/launch/start">
                        <input type="hidden" name="issuer" value={PLATFORM_ISSUER} />
                        <input type="hidden" name="client_id" value={toolFor?.client_id || ''} />
                        <input type="hidden" name="login_initiation_url" value={toolFor?.auth_url || ''} />
                        {/* Send the content URL as target_link_uri for backend to forward to tool launch */}
                        <input type="hidden" name="target_link_uri" value={targetContentUrl} />
                        <input type="hidden" name="resource_link_id" value={s.id} />
                        {/* Use configured email for login_hint and include context_id */}
                        <input type="hidden" name="login_hint" value={email} />
                        <input type="hidden" name="context_id" value={contextId} />
                        <button type="submit" disabled={isDisabled} title={disabledReason} formTarget="_blank">
                          Resource Launch
                        </button>
                      </form>
                      <div style={{ fontSize: 10, opacity: 0.8 }}>
                        <div>client_id: {toolFor?.client_id || '(none)'}</div>
                        <div>launch_url: {launchUrl || '(none)'}</div>
                        <div>target_link_uri: {targetContentUrl || '(none)'}</div>
                        <div>resource_link_id: {s.id || '(none)'}
                        </div>
                        <div>email (login_hint): {email || '(none)'}
                        </div>
                        <div>context_id: {contextId || '(none)'}
                        </div>
                      </div>
                      <button onClick={()=> setExpanded((e)=> ({...e, [s.id]: !open}))}>{open ? 'Hide JSON' : 'Show JSON'}</button>
                      <button onClick={async ()=>{
                        if(!confirm('Delete this selection?')) return;
                        const res = await fetch(`/api/deeplink/selections/${s.id}`, { method:'DELETE' })
                        if(res.ok){
                          const updated = await fetch('/api/deeplink/selections').then(r=>r.json()).catch(()=>[] as any[])
                          setSelections(Array.isArray(updated) ? updated : [])
                        } else {
                          alert('Failed to delete')
                        }
                      }}>Delete</button>
                    </div>
                  </div>
                  {open && (
                    <div style={{ marginTop:8 }}>
                      <textarea
                        readOnly
                        value={pretty}
                        style={{
                          width: '100%',
                          maxWidth: '100%',
                          minHeight: 160,
                          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                          fontSize: 12,
                          lineHeight: 1.4,
                          color: '#111',
                          background: '#fafafa',
                          border: '1px solid #eee',
                          borderRadius: 6,
                          padding: 12,
                          overflowX: 'auto',
                          resize: 'vertical',
                        }}
                      />
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

export default ToolLaunch
