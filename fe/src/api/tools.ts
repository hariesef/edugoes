export type Tool = {
  id: number
  name: string
  client_id: string
  auth_url?: string
  target_link_url?: string
  target_launch_url?: string
  key_set_url?: string
  created_at?: string
}

export type CreateToolPayload = {
  name: string
  client_id: string
  auth_url?: string
  target_link_url?: string
  target_launch_url?: string
  key_set_url?: string
}

const headers = {
  'Content-Type': 'application/json',
}

export async function listTools(): Promise<Tool[]> {
  const res = await fetch('/api/tools')
  if (!res.ok) throw new Error(`listTools failed: ${res.status}`)
  return res.json()
}

export async function getTool(id: number): Promise<Tool | null> {
  const res = await fetch(`/api/tools/${id}`)
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`getTool failed: ${res.status}`)
  return res.json()
}

export async function createTool(payload: CreateToolPayload): Promise<{ id: number; created_at?: string }> {
  const res = await fetch('/api/tools', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  })
  if (!res.ok) throw new Error(`createTool failed: ${res.status}`)
  return res.json()
}

export async function deleteTool(id: number): Promise<void> {
  const res = await fetch(`/api/tools/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteTool failed: ${res.status}`)
}
