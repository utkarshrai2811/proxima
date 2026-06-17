import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Plus, X } from 'lucide-react'
import {
  createOrUpdateSenderRequest,
  sendSenderRequest,
  type HttpHeader,
  type HttpMethod,
  type SenderRequest,
} from '../lib/api'
import { Spinner, StatusChip, cx } from '../components/ui'

const METHODS: HttpMethod[] = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']

interface HeaderRow {
  key: string
  value: string
}

export default function HttpClient() {
  const location = useLocation()
  const navigate = useNavigate()
  const seed = (location.state as { sender?: SenderRequest } | null)?.sender

  const [id, setId] = useState<string | undefined>(seed?.id)
  const [method, setMethod] = useState<HttpMethod>(seed?.method ?? 'GET')
  const [url, setUrl] = useState(seed?.url ?? '')
  const [headers, setHeaders] = useState<HeaderRow[]>(
    seed?.headers?.map((h) => ({ key: h.key, value: h.value })) ?? [{ key: '', value: '' }],
  )
  const [body, setBody] = useState(seed?.body ?? '')
  const [tab, setTab] = useState<'headers' | 'body'>('headers')
  const [respTab, setRespTab] = useState<'headers' | 'body'>('body')
  const [response, setResponse] = useState<SenderRequest['response']>(seed?.response ?? null)
  const [elapsed, setElapsed] = useState<number | null>(null)

  const send = useMutation({
    mutationFn: async () => {
      const cleanHeaders: HttpHeader[] = headers.filter((h) => h.key.trim() !== '')
      const saved = await createOrUpdateSenderRequest({
        id,
        url,
        method,
        headers: cleanHeaders,
        body: body || null,
      })
      setId(saved.id)
      const startedAt = performance.now()
      const sent = await sendSenderRequest(saved.id)
      setElapsed(Math.round(performance.now() - startedAt))
      return sent
    },
    onSuccess: (sent) => setResponse(sent.response),
  })

  const setHeader = (i: number, field: keyof HeaderRow, val: string) =>
    setHeaders((hs) => hs.map((h, j) => (j === i ? { ...h, [field]: val } : h)))

  const formatJson = () => {
    try {
      setBody(JSON.stringify(JSON.parse(body), null, 2))
    } catch {
      /* leave body unchanged if it is not valid JSON */
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-border p-3">
        <div className="flex gap-2">
          <select
            value={method}
            onChange={(e) => setMethod(e.target.value as HttpMethod)}
            className="border border-border bg-bg-surface px-2 py-1.5 text-sm outline-none focus:border-accent"
          >
            {METHODS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
          <input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://example.com/path"
            className="flex-1 border border-border bg-bg-surface px-2 py-1.5 text-sm outline-none focus:border-accent"
          />
          <button
            disabled={!url || send.isPending}
            onClick={() => send.mutate()}
            className="flex items-center gap-2 bg-accent px-4 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim disabled:opacity-40"
          >
            {send.isPending && <Spinner />} Send
          </button>
        </div>

        <div className="mt-3 flex gap-4 text-xs">
          {(['headers', 'body'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={cx(
                'border-b-2 py-1 uppercase tracking-wide',
                tab === t ? 'border-accent text-accent' : 'border-transparent text-text-secondary',
              )}
            >
              {t}
            </button>
          ))}
        </div>

        {tab === 'headers' ? (
          <div className="mt-2 space-y-1">
            {headers.map((h, i) => (
              <div key={i} className="flex gap-2">
                <input
                  value={h.key}
                  onChange={(e) => setHeader(i, 'key', e.target.value)}
                  placeholder="Header"
                  className="flex-1 border border-border bg-bg-base px-2 py-1 text-xs outline-none focus:border-accent"
                />
                <input
                  value={h.value}
                  onChange={(e) => setHeader(i, 'value', e.target.value)}
                  placeholder="value"
                  className="flex-1 border border-border bg-bg-base px-2 py-1 text-xs outline-none focus:border-accent"
                />
                <button
                  onClick={() => setHeaders((hs) => hs.filter((_, j) => j !== i))}
                  className="text-text-secondary hover:text-text-danger"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
            <button
              onClick={() => setHeaders((hs) => [...hs, { key: '', value: '' }])}
              className="flex items-center gap-1 text-xs text-text-secondary hover:text-accent"
            >
              <Plus size={12} /> Add header
            </button>
          </div>
        ) : (
          <div className="mt-2">
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              spellCheck={false}
              rows={6}
              className="w-full resize-y border border-border bg-bg-base p-2 text-xs outline-none focus:border-accent"
            />
            <button
              onClick={formatJson}
              className="mt-1 border border-border px-2 py-1 text-xs text-text-secondary hover:text-accent"
            >
              Format JSON
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-3">
        {send.isError && (
          <div className="text-sm text-text-danger">Request failed: {String(send.error)}</div>
        )}
        {response ? (
          <>
            <div className="mb-3 flex items-center gap-3 text-sm">
              <StatusChip code={response.statusCode} />
              <span>{response.statusReason}</span>
              {elapsed !== null && <span className="text-text-secondary">{elapsed} ms</span>}
            </div>
            <div className="mb-2 flex gap-4 text-xs">
              {(['headers', 'body'] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => setRespTab(t)}
                  className={cx(
                    'border-b-2 py-1 uppercase tracking-wide',
                    respTab === t
                      ? 'border-accent text-accent'
                      : 'border-transparent text-text-secondary',
                  )}
                >
                  {t}
                </button>
              ))}
            </div>
            {respTab === 'headers' ? (
              <div className="text-xs">
                {response.headers.map((h, i) => (
                  <div key={i} className="break-all">
                    <span className="text-accent-dim">{h.key}</span>
                    <span className="text-text-secondary">: </span>
                    <span>{h.value}</span>
                  </div>
                ))}
              </div>
            ) : (
              <pre className="whitespace-pre-wrap break-all bg-bg-base p-2 text-xs">
                {response.body}
              </pre>
            )}
            <button
              onClick={() => navigate('/fuzzer')}
              className="mt-3 border border-border px-3 py-1.5 text-xs text-text-secondary hover:text-text-primary"
            >
              Send to Fuzzer
            </button>
          </>
        ) : (
          <div className="text-sm text-text-secondary">
            Build a request and press Send to see the response here.
          </div>
        )}
      </div>
    </div>
  )
}
