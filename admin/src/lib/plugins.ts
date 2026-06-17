export interface Plugin {
  name: string
  version: string
  description: string
  author: string
  enabled: boolean
  lastError: string
  loadedAt: string
}

async function jreq<T>(url: string, method = 'GET'): Promise<T> {
  const res = await fetch(url, { method, credentials: 'same-origin' })
  if (!res.ok) throw new Error((await res.text()) || `request failed: ${res.status}`)

  return res.json()
}

export const listPlugins = () => jreq<Plugin[]>('/api/plugins')
export const enablePlugin = (name: string) =>
  jreq<Plugin>(`/api/plugins/${encodeURIComponent(name)}/enable`, 'POST')
export const disablePlugin = (name: string) =>
  jreq<Plugin>(`/api/plugins/${encodeURIComponent(name)}/disable`, 'POST')
export const reloadPlugin = (name: string) =>
  jreq<Plugin>(`/api/plugins/${encodeURIComponent(name)}/reload`, 'POST')
export const openPluginFolder = () => jreq<{ ok: boolean }>('/api/plugins/open-folder', 'POST')
