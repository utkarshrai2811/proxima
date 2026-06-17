import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Radio, X } from 'lucide-react'
import {
  fetchWsFrames,
  fetchWsSessions,
  sendWsFrame,
  wsStreamUrl,
  type WsFrame,
} from '../lib/ws'
import { cx } from '../components/ui'

function FrameRow({ frame }: { frame: WsFrame }) {
  const [expanded, setExpanded] = useState(false)
  const [hex, setHex] = useState(false)
  const c2s = frame.direction === 'CLIENT_TO_SERVER'

  return (
    <div className="border-b border-border">
      <div
        onClick={() => setExpanded((e) => !e)}
        className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-xs hover:bg-bg-surface"
      >
        {c2s ? (
          <ArrowUp size={14} className="text-accent" />
        ) : (
          <ArrowDown size={14} className="text-blue-400" />
        )}
        <span className="text-text-secondary">{new Date(frame.timestamp).toLocaleTimeString()}</span>
        <span className="rounded border border-border px-1">{frame.opcode}</span>
        <span className="text-text-secondary">{frame.size}B</span>
        <span className="flex-1 truncate">{frame.payload.slice(0, 80)}</span>
      </div>
      {expanded && (
        <div className="bg-bg-base p-2">
          <button
            onClick={(e) => {
              e.stopPropagation()
              setHex((h) => !h)
            }}
            className="mb-1 border border-border px-2 py-0.5 text-xs text-text-secondary hover:text-accent"
          >
            {hex ? 'Text' : 'Hex'}
          </button>
          <pre className="whitespace-pre-wrap break-all text-xs">
            {hex ? frame.payloadHex : frame.payload}
          </pre>
        </div>
      )}
    </div>
  )
}

function InjectModal({ sessionId, onClose }: { sessionId: string; onClose: () => void }) {
  const [payload, setPayload] = useState('')
  const [opcode, setOpcode] = useState<'TEXT' | 'BINARY'>('TEXT')
  const [error, setError] = useState('')

  const submit = async () => {
    try {
      await sendWsFrame(sessionId, payload, opcode)
      onClose()
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div className="w-96 border border-border bg-bg-surface p-4" onClick={(e) => e.stopPropagation()}>
        <h2 className="mb-3 text-sm font-bold">Inject frame (client → server)</h2>
        <textarea
          value={payload}
          onChange={(e) => setPayload(e.target.value)}
          rows={5}
          spellCheck={false}
          placeholder={opcode === 'BINARY' ? 'base64 payload' : 'text payload'}
          className="w-full border border-border bg-bg-base p-2 text-xs outline-none focus:border-accent"
        />
        <div className="mt-2 flex items-center gap-3">
          <select
            value={opcode}
            onChange={(e) => setOpcode(e.target.value as 'TEXT' | 'BINARY')}
            className="border border-border bg-bg-base px-2 py-1 text-xs"
          >
            <option value="TEXT">Text</option>
            <option value="BINARY">Binary</option>
          </select>
          <button
            onClick={submit}
            className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
          >
            Send
          </button>
          <button
            onClick={onClose}
            className="border border-border px-3 py-1.5 text-xs text-text-secondary"
          >
            Cancel
          </button>
        </div>
        {error && <div className="mt-2 text-xs text-text-danger">{error}</div>}
      </div>
    </div>
  )
}

export default function WebSockets() {
  const [activeOnly, setActiveOnly] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [frames, setFrames] = useState<WsFrame[]>([])
  const [showInject, setShowInject] = useState(false)

  const sessions = useQuery({
    queryKey: ['wsSessions', activeOnly],
    queryFn: () => fetchWsSessions(activeOnly),
    refetchInterval: 2000,
  })
  const selected = sessions.data?.find((s) => s.id === selectedId) ?? null

  useEffect(() => {
    if (!selectedId) {
      setFrames([])

      return
    }

    let cancelled = false

    fetchWsFrames(selectedId).then((fs) => {
      if (!cancelled) setFrames(fs)
    })

    const es = new EventSource(wsStreamUrl(selectedId))
    es.onmessage = (e) => {
      const frame = JSON.parse(e.data) as WsFrame
      setFrames((prev) => (prev.some((x) => x.id === frame.id) ? prev : [...prev, frame]))
    }

    return () => {
      cancelled = true
      es.close()
    }
  }, [selectedId])

  return (
    <div className="flex h-full">
      <div className="flex w-80 shrink-0 flex-col border-r border-border">
        <div className="flex items-center justify-between border-b border-border p-3">
          <h1 className="font-bold">WebSockets</h1>
          <label className="flex items-center gap-1 text-xs text-text-secondary">
            <input
              type="checkbox"
              checked={activeOnly}
              onChange={(e) => setActiveOnly(e.target.checked)}
            />
            Active only
          </label>
        </div>
        <div className="flex-1 overflow-y-auto">
          {(sessions.data ?? []).map((s) => (
            <div
              key={s.id}
              onClick={() => setSelectedId(s.id)}
              className={cx(
                'cursor-pointer border-b border-border px-3 py-2 text-xs hover:bg-bg-surface',
                selectedId === s.id && 'bg-bg-elevated',
              )}
            >
              <div className="truncate">{s.url}</div>
              <div className="mt-1 flex items-center gap-2 text-text-secondary">
                <span className={cx('rounded px-1', s.open ? 'bg-accent text-bg-base' : 'border border-border')}>
                  {s.open ? 'open' : 'closed'}
                </span>
                <span>{s.frameCount} frames</span>
                <span>{new Date(s.startTime).toLocaleTimeString()}</span>
              </div>
            </div>
          ))}
          {(sessions.data ?? []).length === 0 && (
            <div className="p-4 text-center text-xs text-text-secondary">
              No WebSocket sessions yet.
            </div>
          )}
        </div>
      </div>

      {selected ? (
        <div className="flex flex-1 flex-col overflow-hidden">
          <div className="flex items-center gap-2 border-b border-border p-3">
            <span
              className={cx(
                'rounded px-1.5 text-xs',
                selected.open ? 'bg-accent text-bg-base' : 'border border-border text-text-secondary',
              )}
            >
              {selected.open ? 'OPEN' : 'CLOSED'}
            </span>
            <span className="flex-1 truncate text-sm">{selected.url}</span>
            <button
              onClick={() => setSelectedId(null)}
              className="text-text-secondary hover:text-text-primary"
            >
              <X size={16} />
            </button>
          </div>
          <div className="flex-1 overflow-y-auto">
            {frames.map((f) => (
              <FrameRow key={f.id} frame={f} />
            ))}
            {frames.length === 0 && (
              <div className="p-4 text-center text-xs text-text-secondary">No frames.</div>
            )}
          </div>
          <div className="flex items-center gap-2 border-t border-border p-3">
            <button
              disabled={!selected.open}
              onClick={() => setShowInject(true)}
              className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim disabled:opacity-40"
            >
              Inject Frame
            </button>
            {!selected.open && <span className="text-xs text-text-secondary">session closed</span>}
          </div>
        </div>
      ) : (
        <div className="flex flex-1 items-center justify-center text-text-secondary">
          <div className="text-center">
            <Radio className="mx-auto mb-2 text-accent" size={40} />
            <div className="text-sm">Select a session to view frames</div>
          </div>
        </div>
      )}

      {showInject && selected && (
        <InjectModal sessionId={selected.id} onClose={() => setShowInject(false)} />
      )}
    </div>
  )
}
