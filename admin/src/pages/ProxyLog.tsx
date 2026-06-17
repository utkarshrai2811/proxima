import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import {
  clearRequestLogs,
  createSenderFromLog,
  fetchRequestLogs,
  setRequestLogFilter,
  type HttpHeader,
  type HttpRequestLog,
} from '../lib/api'
import { MethodChip, StatusChip, cx } from '../components/ui'

const PAGE_SIZE = 50

function byteLen(s: string | null): string {
  if (!s) return '0'
  const n = new TextEncoder().encode(s).length
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function HeaderList({ headers }: { headers: HttpHeader[] }) {
  return (
    <div className="text-xs">
      {headers.map((h, i) => (
        <div key={i} className="break-all">
          <span className="text-accent-dim">{h.key}</span>
          <span className="text-text-secondary">: </span>
          <span>{h.value}</span>
        </div>
      ))}
    </div>
  )
}

function DetailPanel({ log, onClose }: { log: HttpRequestLog; onClose: () => void }) {
  const [tab, setTab] = useState<'request' | 'response'>('request')
  const navigate = useNavigate()

  const sendToClient = useMutation({
    mutationFn: () => createSenderFromLog(log.id),
    onSuccess: (sender) => navigate('/client', { state: { sender } }),
  })

  return (
    <div className="flex w-1/2 shrink-0 flex-col border-l border-border bg-bg-surface">
      <div className="flex items-center gap-2 border-b border-border px-4 py-2">
        <MethodChip method={log.method} />
        {log.response && <StatusChip code={log.response.statusCode} />}
        <span className="flex-1 truncate text-xs text-text-secondary">{log.url}</span>
        <button onClick={onClose} className="text-text-secondary hover:text-text-primary">
          <X size={16} />
        </button>
      </div>

      <div className="flex gap-4 border-b border-border px-4 text-xs">
        {(['request', 'response'] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cx(
              'border-b-2 py-2 uppercase tracking-wide',
              tab === t ? 'border-accent text-accent' : 'border-transparent text-text-secondary',
            )}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        {tab === 'request' ? (
          <div className="space-y-3">
            <div className="text-xs text-text-secondary">
              {log.method} {log.url} {log.proto}
            </div>
            <HeaderList headers={log.headers} />
            {log.body && (
              <pre className="whitespace-pre-wrap break-all bg-bg-base p-2 text-xs">{log.body}</pre>
            )}
          </div>
        ) : log.response ? (
          <div className="space-y-3">
            <div className="text-xs text-text-secondary">
              {log.response.proto} {log.response.statusCode} {log.response.statusReason}
            </div>
            <HeaderList headers={log.response.headers} />
            {log.response.body && (
              <pre className="whitespace-pre-wrap break-all bg-bg-base p-2 text-xs">
                {log.response.body}
              </pre>
            )}
          </div>
        ) : (
          <div className="text-xs text-text-secondary">No response recorded.</div>
        )}
      </div>

      <div className="flex gap-2 border-t border-border p-3">
        <button
          onClick={() => sendToClient.mutate()}
          className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
        >
          Send to HTTP Client
        </button>
        <button
          onClick={() => navigate('/fuzzer')}
          className="border border-border px-3 py-1.5 text-xs text-text-secondary hover:text-text-primary"
        >
          Send to Fuzzer
        </button>
      </div>
    </div>
  )
}

export default function ProxyLog() {
  const qc = useQueryClient()
  const [selected, setSelected] = useState<HttpRequestLog | null>(null)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)

  const logs = useQuery({ queryKey: ['requestLogs'], queryFn: fetchRequestLogs })

  const applyFilter = useMutation({
    mutationFn: () => setRequestLogFilter(search, false),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['requestLogs'] }),
  })

  const clearAll = useMutation({
    mutationFn: clearRequestLogs,
    onSuccess: () => {
      setSelected(null)
      qc.invalidateQueries({ queryKey: ['requestLogs'] })
    },
  })

  const all = logs.data ?? []
  const total = all.length
  const start = page * PAGE_SIZE
  const rows = all.slice(start, start + PAGE_SIZE)

  return (
    <div className="flex h-full">
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="border-b border-border p-3">
          <div className="flex gap-2">
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && applyFilter.mutate()}
              placeholder='filter: url contains "login"'
              className="flex-1 border border-border bg-bg-surface px-2 py-1.5 text-sm outline-none focus:border-accent"
            />
            <button
              onClick={() => applyFilter.mutate()}
              className="bg-accent px-3 text-xs font-bold text-bg-base hover:bg-accent-dim"
            >
              Filter
            </button>
            <button
              onClick={() => clearAll.mutate()}
              className="border border-text-danger px-3 text-xs text-text-danger hover:bg-text-danger hover:text-bg-base"
            >
              Clear
            </button>
          </div>
          <div className="mt-1 text-xs text-text-secondary">
            search expression: <code>url | method | statusCode | body =~ "..."</code>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          <table className="w-full text-left text-sm">
            <thead className="sticky top-0 bg-bg-surface text-xs uppercase text-text-secondary">
              <tr>
                <th className="px-3 py-2">Method</th>
                <th className="px-3 py-2">URL</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Size</th>
                <th className="px-3 py-2">Time</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((log) => (
                <tr
                  key={log.id}
                  onClick={() => setSelected(log)}
                  className={cx(
                    'cursor-pointer border-b border-border hover:bg-bg-surface',
                    selected?.id === log.id && 'bg-bg-elevated',
                  )}
                >
                  <td className="px-3 py-1.5">
                    <MethodChip method={log.method} />
                  </td>
                  <td className="max-w-md truncate px-3 py-1.5">{log.url}</td>
                  <td className="px-3 py-1.5">
                    {log.response ? <StatusChip code={log.response.statusCode} /> : '—'}
                  </td>
                  <td className="px-3 py-1.5 text-text-secondary">
                    {log.response ? byteLen(log.response.body) : '—'}
                  </td>
                  <td className="px-3 py-1.5 text-text-secondary">
                    {new Date(log.timestamp).toLocaleTimeString()}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-3 py-8 text-center text-text-secondary">
                    {logs.isLoading ? 'Loading…' : 'No requests logged yet.'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="flex items-center justify-between border-t border-border px-3 py-2 text-xs text-text-secondary">
          <span>
            {total === 0 ? 'Showing 0' : `Showing ${start + 1}–${Math.min(start + PAGE_SIZE, total)} of ${total}`}
          </span>
          <div className="flex gap-2">
            <button
              disabled={page === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              className="border border-border px-2 py-1 disabled:opacity-40"
            >
              Previous
            </button>
            <button
              disabled={start + PAGE_SIZE >= total}
              onClick={() => setPage((p) => p + 1)}
              className="border border-border px-2 py-1 disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      </div>

      {selected && <DetailPanel log={selected} onClose={() => setSelected(null)} />}
    </div>
  )
}
