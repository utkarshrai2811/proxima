import { useEffect, useRef, useState } from 'react'
import { X } from 'lucide-react'
import {
  cancelAttack,
  createAttack,
  detectPositions,
  fuzzStreamUrl,
  getAttack,
  pauseAttack,
  startAttack,
  type AttackType,
  type BuiltInList,
  type FuzzAttack,
  type FuzzResult,
  type PayloadSourceInput,
} from '../lib/fuzzer'
import { Spinner, StatusChip, cx } from '../components/ui'

const ATTACK_TYPES: { value: AttackType; label: string }[] = [
  { value: 'SNIPER', label: 'Sniper' },
  { value: 'BATTERING_RAM', label: 'Battering Ram' },
  { value: 'PITCHFORK', label: 'Pitchfork' },
  { value: 'CLUSTER_BOMB', label: 'Cluster Bomb' },
]

const BUILT_INS: BuiltInList[] = [
  'SQLI_BASIC', 'XSS_BASIC', 'COMMON_PASSWORDS', 'DIR_NAMES', 'NUMERIC_RANGE',
]

interface SourceState {
  mode: 'INLINE' | 'BUILT_IN'
  inline: string
  builtIn: BuiltInList
  rangeMin: number
  rangeMax: number
}

const defaultSource = (): SourceState => ({
  mode: 'INLINE', inline: '', builtIn: 'SQLI_BASIC', rangeMin: 1, rangeMax: 100,
})

function toSourceInput(s: SourceState): PayloadSourceInput {
  if (s.mode === 'BUILT_IN') {
    return { type: 'BUILT_IN', builtIn: s.builtIn, rangeMin: s.rangeMin, rangeMax: s.rangeMax }
  }

  return { type: 'INLINE', values: s.inline.split('\n').filter((l) => l.length > 0) }
}

function SourceEditor({
  label,
  source,
  onChange,
}: {
  label: string
  source: SourceState
  onChange: (s: SourceState) => void
}) {
  const count =
    source.mode === 'INLINE'
      ? source.inline.split('\n').filter((l) => l.length > 0).length
      : source.builtIn === 'NUMERIC_RANGE'
        ? Math.max(0, source.rangeMax - source.rangeMin + 1)
        : null

  return (
    <div className="mb-2 border border-border p-2">
      <div className="mb-1 flex items-center justify-between">
        <span className="text-xs text-accent">{label}</span>
        <div className="flex gap-1 text-xs">
          {(['INLINE', 'BUILT_IN'] as const).map((m) => (
            <button
              key={m}
              onClick={() => onChange({ ...source, mode: m })}
              className={cx('px-1.5', source.mode === m ? 'text-accent' : 'text-text-secondary')}
            >
              {m === 'INLINE' ? 'Inline' : 'Built-in'}
            </button>
          ))}
        </div>
      </div>
      {source.mode === 'INLINE' ? (
        <>
          <textarea
            value={source.inline}
            onChange={(e) => onChange({ ...source, inline: e.target.value })}
            rows={3}
            spellCheck={false}
            placeholder="one payload per line"
            className="w-full resize-y border border-border bg-bg-base p-1 text-xs outline-none focus:border-accent"
          />
          <div className="text-xs text-text-secondary">{count} payloads</div>
        </>
      ) : (
        <>
          <select
            value={source.builtIn}
            onChange={(e) => onChange({ ...source, builtIn: e.target.value as BuiltInList })}
            className="w-full border border-border bg-bg-base px-1 py-1 text-xs"
          >
            {BUILT_INS.map((b) => (
              <option key={b} value={b}>
                {b}
              </option>
            ))}
          </select>
          {source.builtIn === 'NUMERIC_RANGE' && (
            <div className="mt-1 flex items-center gap-2 text-xs">
              <input
                type="number"
                value={source.rangeMin}
                onChange={(e) => onChange({ ...source, rangeMin: Number(e.target.value) })}
                className="w-20 border border-border bg-bg-base px-1 py-0.5"
              />
              <span className="text-text-secondary">to</span>
              <input
                type="number"
                value={source.rangeMax}
                onChange={(e) => onChange({ ...source, rangeMax: Number(e.target.value) })}
                className="w-20 border border-border bg-bg-base px-1 py-0.5"
              />
              {count !== null && <span className="text-text-secondary">({count})</span>}
            </div>
          )}
        </>
      )}
    </div>
  )
}

function ResultDrawer({ result, onClose }: { result: FuzzResult; onClose: () => void }) {
  return (
    <div className="flex w-1/2 shrink-0 flex-col border-l border-border bg-bg-surface">
      <div className="flex items-center gap-2 border-b border-border px-3 py-2">
        <span className="text-xs text-text-secondary">#{result.requestIndex}</span>
        {!result.isError && <StatusChip code={result.statusCode} />}
        <span className="flex-1" />
        <button onClick={onClose} className="text-text-secondary hover:text-text-primary">
          <X size={16} />
        </button>
      </div>
      <div className="flex-1 overflow-y-auto p-3 text-xs">
        <div className="mb-1 text-text-secondary">Request</div>
        <pre className="mb-3 whitespace-pre-wrap break-all bg-bg-base p-2">{result.rawRequest}</pre>
        <div className="mb-1 text-text-secondary">Response</div>
        <pre className="whitespace-pre-wrap break-all bg-bg-base p-2">
          {result.isError ? result.errorMessage : result.rawResponse}
        </pre>
      </div>
    </div>
  )
}

export default function Fuzzer() {
  const [name, setName] = useState('Attack 1')
  const [type, setType] = useState<AttackType>('SNIPER')
  const [baseRequest, setBaseRequest] = useState(
    'POST https://example.com/login HTTP/1.1\nContent-Type: application/json\n\n{"user":"§user§","password":"§pass§"}',
  )
  const [concurrency, setConcurrency] = useState(10)
  const [sources, setSources] = useState<SourceState[]>([defaultSource()])
  const baseRef = useRef<HTMLTextAreaElement>(null)

  const [attack, setAttack] = useState<FuzzAttack | null>(null)
  const [results, setResults] = useState<FuzzResult[]>([])
  const [error, setError] = useState('')
  const [highlightDiffs, setHighlightDiffs] = useState(false)
  const [selected, setSelected] = useState<FuzzResult | null>(null)
  const [filters, setFilters] = useState({ status: '', minSize: '', maxSize: '', minTime: '', maxTime: '' })

  const positions = detectPositions(baseRequest)
  const perPosition = type === 'PITCHFORK' || type === 'CLUSTER_BOMB'
  const sourceCount = perPosition ? positions.length : 1

  const getSource = (i: number): SourceState => sources[i] ?? defaultSource()
  const setSource = (i: number, s: SourceState) =>
    setSources((prev) => {
      const next = [...prev]
      while (next.length <= i) next.push(defaultSource())
      next[i] = s

      return next
    })

  useEffect(() => {
    if (!attack) return

    const es = new EventSource(fuzzStreamUrl(attack.id))
    es.onmessage = (e) => {
      const r = JSON.parse(e.data) as FuzzResult
      setResults((prev) => (prev.some((x) => x.id === r.id) ? prev : [...prev, r]))
    }

    const poll = setInterval(async () => {
      const a = await getAttack(attack.id)
      setAttack(a)
      if (a.status === 'DONE' || a.status === 'CANCELLED') clearInterval(poll)
    }, 1000)

    return () => {
      es.close()
      clearInterval(poll)
    }
  }, [attack?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const start = async () => {
    setError('')
    try {
      const created = await createAttack({
        name,
        type,
        baseRequest,
        concurrency,
        payloadSources: Array.from({ length: Math.max(1, sourceCount) }, (_, i) =>
          toSourceInput(getSource(i)),
        ),
      })
      setResults([])
      const started = await startAttack(created.id)
      setAttack(started)
    } catch (e) {
      setError(String(e))
    }
  }

  const addPosition = () => {
    const ta = baseRef.current
    if (!ta) return

    const { selectionStart: s, selectionEnd: e, value } = ta
    const posName = `pos${positions.length + 1}`
    const next = `${value.slice(0, s)}§${posName}§${value.slice(e)}`
    setBaseRequest(next)
  }

  const running = attack?.status === 'RUNNING'
  const paused = attack?.status === 'PAUSED'
  const progress = attack && attack.totalRequests > 0 ? attack.completedCount / attack.totalRequests : 0

  const baseline = results[0]?.responseSize ?? 0
  const filtered = results.filter((r) => {
    if (filters.status && String(r.statusCode) !== filters.status) return false
    if (filters.minSize && r.responseSize < Number(filters.minSize)) return false
    if (filters.maxSize && r.responseSize > Number(filters.maxSize)) return false
    if (filters.minTime && r.responseTimeMs < Number(filters.minTime)) return false
    if (filters.maxTime && r.responseTimeMs > Number(filters.maxTime)) return false

    return true
  })

  return (
    <div className="flex h-full">
      {/* Setup */}
      <div className="flex w-80 shrink-0 flex-col overflow-y-auto border-r border-border p-3">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mb-2 border border-border bg-bg-surface px-2 py-1 text-sm outline-none focus:border-accent"
        />
        <div className="mb-2 grid grid-cols-2 gap-1">
          {ATTACK_TYPES.map((t) => (
            <button
              key={t.value}
              onClick={() => setType(t.value)}
              className={cx(
                'border px-1 py-1 text-xs',
                type === t.value ? 'border-accent text-accent' : 'border-border text-text-secondary',
              )}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="mb-1 flex items-center justify-between text-xs text-text-secondary">
          <span>Base request (full URL in the request line)</span>
          <button onClick={addPosition} className="text-accent hover:underline">
            + Position
          </button>
        </div>
        <textarea
          ref={baseRef}
          value={baseRequest}
          onChange={(e) => setBaseRequest(e.target.value)}
          rows={8}
          spellCheck={false}
          className="mb-1 resize-y border border-border bg-bg-base p-2 text-xs outline-none focus:border-accent"
        />
        <div className="mb-3 text-xs text-text-secondary">
          Positions:{' '}
          {positions.length ? (
            positions.map((p) => (
              <span key={p} className="text-accent">
                §{p}§{' '}
              </span>
            ))
          ) : (
            <span>none — wrap text in §…§</span>
          )}
        </div>

        {Array.from({ length: Math.max(1, sourceCount) }, (_, i) => (
          <SourceEditor
            key={i}
            label={perPosition ? `Position §${positions[i] ?? `pos${i + 1}`}§` : 'Payload set (all positions)'}
            source={getSource(i)}
            onChange={(s) => setSource(i, s)}
          />
        ))}

        <label className="mb-3 mt-2 block text-xs text-text-secondary">
          Concurrency: {concurrency}
          <input
            type="range"
            min={1}
            max={50}
            value={concurrency}
            onChange={(e) => setConcurrency(Number(e.target.value))}
            className="w-full"
          />
        </label>

        {!running && !paused ? (
          <button
            onClick={start}
            className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
          >
            Start Attack
          </button>
        ) : (
          <div className="flex gap-2">
            {running ? (
              <button
                onClick={() => attack && pauseAttack(attack.id).then(setAttack)}
                className="bg-text-warn px-3 py-1.5 text-xs font-bold text-bg-base"
              >
                Pause
              </button>
            ) : (
              <button
                onClick={() => attack && startAttack(attack.id).then(setAttack)}
                className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base"
              >
                Resume
              </button>
            )}
            <button
              onClick={() => attack && cancelAttack(attack.id).then(setAttack)}
              className="border border-text-danger px-3 py-1.5 text-xs text-text-danger"
            >
              Cancel
            </button>
          </div>
        )}

        {attack && (
          <div className="mt-3">
            <div className="mb-1 flex justify-between text-xs text-text-secondary">
              <span>{attack.status}</span>
              <span>
                {attack.completedCount}/{attack.totalRequests}
                {attack.errorCount > 0 && ` · ${attack.errorCount} err`}
              </span>
            </div>
            <div className="h-1.5 w-full bg-bg-elevated">
              <div className="h-full bg-accent" style={{ width: `${progress * 100}%` }} />
            </div>
          </div>
        )}

        {error && <div className="mt-2 text-xs text-text-danger">{error}</div>}
      </div>

      {/* Results */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-2 text-xs">
          <label className="flex items-center gap-1 text-text-secondary">
            <input
              type="checkbox"
              checked={highlightDiffs}
              onChange={(e) => setHighlightDiffs(e.target.checked)}
            />
            Highlight diffs
          </label>
          <input
            value={filters.status}
            onChange={(e) => setFilters({ ...filters, status: e.target.value })}
            placeholder="status"
            className="w-16 border border-border bg-bg-base px-1 py-0.5"
          />
          <input
            value={filters.minSize}
            onChange={(e) => setFilters({ ...filters, minSize: e.target.value })}
            placeholder="min len"
            className="w-20 border border-border bg-bg-base px-1 py-0.5"
          />
          <input
            value={filters.maxSize}
            onChange={(e) => setFilters({ ...filters, maxSize: e.target.value })}
            placeholder="max len"
            className="w-20 border border-border bg-bg-base px-1 py-0.5"
          />
        </div>

        <div className="flex flex-1 overflow-hidden">
          <div className="flex-1 overflow-y-auto">
            <table className="w-full text-left text-sm">
              <thead className="sticky top-0 bg-bg-surface text-xs uppercase text-text-secondary">
                <tr>
                  <th className="px-3 py-2">#</th>
                  <th className="px-3 py-2">Payload(s)</th>
                  <th className="px-3 py-2">Status</th>
                  <th className="px-3 py-2">Length</th>
                  <th className="px-3 py-2">Time</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r) => {
                  const deviates =
                    highlightDiffs && baseline > 0 && Math.abs(r.responseSize - baseline) / baseline > 0.1

                  return (
                    <tr
                      key={r.id}
                      onClick={() => setSelected(r)}
                      className={cx(
                        'cursor-pointer border-b border-border hover:bg-bg-surface',
                        deviates && 'bg-text-warn/10',
                        selected?.id === r.id && 'bg-bg-elevated',
                      )}
                    >
                      <td className="px-3 py-1.5 text-text-secondary">{r.requestIndex}</td>
                      <td className="max-w-xs truncate px-3 py-1.5">
                        {Object.values(r.payloadValues).filter(Boolean).join(', ')}
                      </td>
                      <td className="px-3 py-1.5">
                        {r.isError ? (
                          <span className="text-xs text-text-secondary">error</span>
                        ) : (
                          <StatusChip code={r.statusCode} />
                        )}
                      </td>
                      <td className="px-3 py-1.5 text-text-secondary">{r.responseSize}</td>
                      <td className="px-3 py-1.5 text-text-secondary">{r.responseTimeMs}ms</td>
                    </tr>
                  )
                })}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-3 py-8 text-center text-text-secondary">
                      {running ? (
                        <span className="flex items-center justify-center gap-2">
                          <Spinner /> running…
                        </span>
                      ) : (
                        'No results yet. Configure an attack and press Start.'
                      )}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {selected && <ResultDrawer result={selected} onClose={() => setSelected(null)} />}
        </div>
      </div>
    </div>
  )
}
