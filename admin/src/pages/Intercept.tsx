import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  cancelRequest,
  fetchActiveProject,
  fetchInterceptedRequests,
  modifyRequest,
  updateInterceptSettings,
  type HttpMethod,
  type HttpProtocol,
  type InterceptedRequest,
} from '../lib/api'
import { MethodChip, cx } from '../components/ui'

const protoLabel: Record<HttpProtocol, string> = {
  HTTP10: 'HTTP/1.0',
  HTTP11: 'HTTP/1.1',
  HTTP20: 'HTTP/2.0',
}

function serialize(req: InterceptedRequest): string {
  const head = `${req.method} ${req.url} ${protoLabel[req.proto]}`
  const headers = req.headers.map((h) => `${h.key}: ${h.value}`).join('\n')
  return `${head}\n${headers}\n\n${req.body ?? ''}`
}

function parse(raw: string, original: InterceptedRequest) {
  const sep = raw.indexOf('\n\n')
  const headPart = sep === -1 ? raw : raw.slice(0, sep)
  const body = sep === -1 ? '' : raw.slice(sep + 2)
  const lines = headPart.split('\n')
  const reqLine = (lines[0] ?? '').trim().split(/\s+/)
  const headers = lines
    .slice(1)
    .filter((l) => l.includes(':'))
    .map((l) => {
      const idx = l.indexOf(':')
      return { key: l.slice(0, idx).trim(), value: l.slice(idx + 1).trim() }
    })

  return {
    id: original.id,
    method: (reqLine[0] || original.method) as HttpMethod,
    url: reqLine[1] || original.url,
    proto: original.proto,
    headers,
    body: body.length > 0 ? body : null,
  }
}

export default function Intercept() {
  const qc = useQueryClient()
  const queue = useQuery({
    queryKey: ['interceptedRequests'],
    queryFn: fetchInterceptedRequests,
    refetchInterval: 2000,
  })
  const project = useQuery({ queryKey: ['activeProject'], queryFn: fetchActiveProject })

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState('')

  const items = queue.data ?? []
  const selected = items.find((r) => r.id === selectedId) ?? null
  const settings = project.data?.settings.intercept
  const interceptOn = settings?.requestsEnabled ?? false

  useEffect(() => {
    if (selected) setDraft(serialize(selected))
  }, [selected])

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ['interceptedRequests'] })
    setSelectedId(null)
  }

  const toggle = useMutation({
    mutationFn: () =>
      updateInterceptSettings({
        requestsEnabled: !interceptOn,
        responsesEnabled: settings?.responsesEnabled ?? false,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['activeProject'] }),
  })

  const forward = useMutation({
    mutationFn: (modified: boolean) => {
      if (!selected) return Promise.resolve()
      const payload = modified
        ? parse(draft, selected)
        : {
            id: selected.id,
            url: selected.url,
            method: selected.method,
            proto: selected.proto,
            headers: selected.headers,
            body: selected.body,
          }
      return modifyRequest(payload)
    },
    onSuccess: refresh,
  })

  const drop = useMutation({
    mutationFn: () => (selected ? cancelRequest(selected.id) : Promise.resolve()),
    onSuccess: refresh,
  })

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-border p-3">
        <div>
          <h1 className="text-lg font-bold">Intercept</h1>
          <p className="text-xs text-text-secondary">
            {interceptOn
              ? 'Interception is ON — matching requests pause here for review.'
              : 'Interception is OFF — requests pass through automatically.'}
          </p>
        </div>
        <button
          onClick={() => toggle.mutate()}
          className={cx(
            'px-4 py-1.5 text-xs font-bold',
            interceptOn ? 'bg-accent text-bg-base' : 'border border-border text-text-secondary',
          )}
        >
          Intercept {interceptOn ? 'ON' : 'OFF'}
        </button>
      </div>

      <div className="max-h-64 overflow-y-auto border-b border-border">
        <table className="w-full text-left text-sm">
          <tbody>
            {items.map((r) => (
              <tr
                key={r.id}
                onClick={() => setSelectedId(r.id)}
                className={cx(
                  'cursor-pointer border-b border-border hover:bg-bg-surface',
                  selectedId === r.id && 'bg-bg-elevated',
                )}
              >
                <td className="px-3 py-1.5">
                  <MethodChip method={r.method} />
                </td>
                <td className="truncate px-3 py-1.5">{r.url}</td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td className="px-3 py-8 text-center text-text-secondary">Queue is empty.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="flex flex-1 flex-col overflow-hidden p-3">
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            spellCheck={false}
            className="flex-1 resize-none border border-border bg-bg-base p-2 text-xs outline-none focus:border-accent"
          />
          <div className="mt-3 flex gap-2">
            <button
              onClick={() => forward.mutate(false)}
              className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
            >
              Forward
            </button>
            <button
              onClick={() => forward.mutate(true)}
              className="bg-text-warn px-3 py-1.5 text-xs font-bold text-bg-base"
            >
              Forward Modified
            </button>
            <button
              onClick={() => drop.mutate()}
              className="border border-text-danger px-3 py-1.5 text-xs text-text-danger hover:bg-text-danger hover:text-bg-base"
            >
              Drop
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
