import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { fetchScope, setScope, type ScopeRule } from '../lib/api'

interface Row {
  url: string
  headerKey: string
  headerValue: string
  body: string
}

function toRows(rules: ScopeRule[]): Row[] {
  return rules.map((r) => ({
    url: r.url ?? '',
    headerKey: r.header?.key ?? '',
    headerValue: r.header?.value ?? '',
    body: r.body ?? '',
  }))
}

function Cell({
  value,
  onChange,
  placeholder,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full border border-border bg-bg-surface px-2 py-1 text-xs outline-none focus:border-accent"
    />
  )
}

export default function Scope() {
  const qc = useQueryClient()
  const scope = useQuery({ queryKey: ['scope'], queryFn: fetchScope })
  const [rows, setRows] = useState<Row[]>([])

  useEffect(() => {
    if (scope.data) setRows(toRows(scope.data))
  }, [scope.data])

  const save = useMutation({
    mutationFn: () => {
      const rules: ScopeRule[] = rows
        .filter((r) => r.url || r.headerKey || r.headerValue || r.body)
        .map((r) => ({
          url: r.url || null,
          body: r.body || null,
          header: { key: r.headerKey || null, value: r.headerValue || null },
        }))
      return setScope(rules)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['scope'] }),
  })

  const update = (i: number, field: keyof Row, val: string) =>
    setRows((rs) => rs.map((r, j) => (j === i ? { ...r, [field]: val } : r)))
  const addRow = () =>
    setRows((rs) => [...rs, { url: '', headerKey: '', headerValue: '', body: '' }])
  const removeRow = (i: number) => setRows((rs) => rs.filter((_, j) => j !== i))

  return (
    <div className="p-4">
      <div className="mb-4 flex items-start justify-between">
        <div>
          <h1 className="text-lg font-bold">Scope</h1>
          <p className="text-xs text-text-secondary">
            Regex rules that focus modules on in-scope traffic. Within a rule, every pattern set
            must match. The header <span className="text-accent">value</span> pattern is enforced
            (fixes upstream hetty#142).
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={addRow}
            className="flex items-center gap-1 border border-border px-3 py-1.5 text-xs hover:text-accent"
          >
            <Plus size={14} /> Add Rule
          </button>
          <button
            onClick={() => save.mutate()}
            className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
          >
            {save.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>

      <table className="w-full text-left text-sm">
        <thead className="text-xs uppercase text-text-secondary">
          <tr>
            <th className="px-2 py-2">URL Pattern</th>
            <th className="px-2 py-2">Header Name</th>
            <th className="px-2 py-2">Header Value</th>
            <th className="px-2 py-2">Body Pattern</th>
            <th className="w-10 px-2 py-2" />
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-b border-border">
              <td className="p-1">
                <Cell value={r.url} onChange={(v) => update(i, 'url', v)} placeholder="regex" />
              </td>
              <td className="p-1">
                <Cell
                  value={r.headerKey}
                  onChange={(v) => update(i, 'headerKey', v)}
                  placeholder="regex"
                />
              </td>
              <td className="p-1">
                <Cell
                  value={r.headerValue}
                  onChange={(v) => update(i, 'headerValue', v)}
                  placeholder="regex"
                />
              </td>
              <td className="p-1">
                <Cell value={r.body} onChange={(v) => update(i, 'body', v)} placeholder="regex" />
              </td>
              <td className="p-1 text-center">
                <button
                  onClick={() => removeRow(i)}
                  className="text-text-secondary hover:text-text-danger"
                >
                  <Trash2 size={14} />
                </button>
              </td>
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td colSpan={5} className="px-2 py-8 text-center text-text-secondary">
                No scope rules. Add one to begin.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
