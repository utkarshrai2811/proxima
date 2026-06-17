// WebSocket session API. The backend exposes these as REST + SSE (not GraphQL)
// on the same origin, so cookies (the auth session) are sent automatically.

export interface WsSession {
  id: string
  requestId: string
  url: string
  remoteAddr: string
  startTime: string
  endTime: string | null
  open: boolean
  frameCount: number
}

export type WsDirection = 'CLIENT_TO_SERVER' | 'SERVER_TO_CLIENT'

export interface WsFrame {
  id: string
  sessionId: string
  direction: WsDirection
  timestamp: string
  opcode: string
  payload: string
  payloadHex: string
  size: number
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: 'same-origin' })
  if (!res.ok) throw new Error(`request failed: ${res.status}`)

  return res.json()
}

export function fetchWsSessions(activeOnly: boolean): Promise<WsSession[]> {
  return getJSON(`/api/ws/sessions${activeOnly ? '?active=true' : ''}`)
}

export function fetchWsFrames(sessionId: string): Promise<WsFrame[]> {
  return getJSON(`/api/ws/sessions/${sessionId}/frames`)
}

export async function sendWsFrame(
  sessionId: string,
  payload: string,
  opcode: 'TEXT' | 'BINARY',
): Promise<WsFrame> {
  const res = await fetch(`/api/ws/sessions/${sessionId}/frames`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ payload, opcode }),
  })

  if (!res.ok) throw new Error((await res.text()) || `request failed: ${res.status}`)

  return res.json()
}

export function wsStreamUrl(sessionId: string): string {
  return `/api/ws/sessions/${sessionId}/stream`
}
