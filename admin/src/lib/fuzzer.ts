// Fuzzer API (REST + SSE on the admin router, same-origin so cookies are sent).

export type AttackType = 'SNIPER' | 'BATTERING_RAM' | 'PITCHFORK' | 'CLUSTER_BOMB'
export type AttackStatus = 'PENDING' | 'RUNNING' | 'PAUSED' | 'DONE' | 'CANCELLED'
export type BuiltInList =
  | 'SQLI_BASIC' | 'XSS_BASIC' | 'COMMON_PASSWORDS' | 'DIR_NAMES' | 'NUMERIC_RANGE'

export interface FuzzAttack {
  id: string
  name: string
  type: AttackType
  baseRequest: string
  status: AttackStatus
  createdAt: string
  startedAt: string | null
  finishedAt: string | null
  totalRequests: number
  completedCount: number
  errorCount: number
}

export interface FuzzResult {
  id: string
  attackId: string
  requestIndex: number
  payloadValues: Record<string, string>
  rawRequest: string
  rawResponse: string
  statusCode: number
  responseSize: number
  responseTimeMs: number
  isError: boolean
  errorMessage: string
}

export interface PayloadSourceInput {
  type: 'INLINE' | 'BUILT_IN'
  values?: string[]
  builtIn?: BuiltInList
  rangeMin?: number
  rangeMax?: number
}

export interface CreateAttackInput {
  name: string
  type: AttackType
  baseRequest: string
  concurrency: number
  payloadSources: PayloadSourceInput[]
}

async function req<T>(url: string, method = 'GET', body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    credentials: 'same-origin',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) throw new Error((await res.text()) || `request failed: ${res.status}`)

  return res.json()
}

export const createAttack = (input: CreateAttackInput) =>
  req<FuzzAttack>('/api/fuzzer/attacks', 'POST', input)
export const getAttack = (id: string) => req<FuzzAttack>(`/api/fuzzer/attacks/${id}`)
export const startAttack = (id: string) => req<FuzzAttack>(`/api/fuzzer/attacks/${id}/start`, 'POST')
export const pauseAttack = (id: string) => req<FuzzAttack>(`/api/fuzzer/attacks/${id}/pause`, 'POST')
export const cancelAttack = (id: string) => req<FuzzAttack>(`/api/fuzzer/attacks/${id}/cancel`, 'POST')
export const listResults = (id: string) => req<FuzzResult[]>(`/api/fuzzer/attacks/${id}/results`)
export const fuzzStreamUrl = (id: string) => `/api/fuzzer/attacks/${id}/stream`

export function detectPositions(template: string): string[] {
  const names: string[] = []

  for (const m of template.matchAll(/§([^§]+)§/g)) {
    if (!names.includes(m[1])) names.push(m[1])
  }

  return names
}
