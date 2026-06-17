import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FolderOpen, RefreshCw } from 'lucide-react'
import {
  disablePlugin,
  enablePlugin,
  listPlugins,
  openPluginFolder,
  reloadPlugin,
} from '../lib/plugins'
import { cx } from '../components/ui'

export default function Plugins() {
  const qc = useQueryClient()
  const plugins = useQuery({ queryKey: ['plugins'], queryFn: listPlugins })
  const invalidate = () => qc.invalidateQueries({ queryKey: ['plugins'] })

  const toggle = useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      enabled ? disablePlugin(name) : enablePlugin(name),
    onSuccess: invalidate,
  })
  const reload = useMutation({ mutationFn: reloadPlugin, onSuccess: invalidate })
  const openFolder = useMutation({ mutationFn: openPluginFolder })

  return (
    <div className="p-4">
      <div className="mb-4 flex items-start justify-between">
        <div>
          <h1 className="text-lg font-bold">Plugins</h1>
          <p className="text-xs text-text-secondary">
            Drop <code>.js</code> files into the plugins folder, then reload. See PLUGINS.md for the
            hook and <code>proxima.*</code> API reference.
          </p>
        </div>
        <button
          onClick={() => openFolder.mutate()}
          className="flex items-center gap-1 border border-border px-3 py-1.5 text-xs hover:text-accent"
        >
          <FolderOpen size={14} /> Open plugins folder
        </button>
      </div>

      <table className="w-full text-left text-sm">
        <thead className="text-xs uppercase text-text-secondary">
          <tr>
            <th className="px-2 py-2">Name</th>
            <th className="px-2 py-2">Version</th>
            <th className="px-2 py-2">Author</th>
            <th className="px-2 py-2">Description</th>
            <th className="px-2 py-2">Status</th>
            <th className="px-2 py-2">Last Error</th>
            <th className="px-2 py-2" />
          </tr>
        </thead>
        <tbody>
          {(plugins.data ?? []).map((p) => (
            <tr key={p.name} className="border-b border-border">
              <td className="px-2 py-1.5 font-bold">{p.name}</td>
              <td className="px-2 py-1.5 text-text-secondary">{p.version || '—'}</td>
              <td className="px-2 py-1.5 text-text-secondary">{p.author || '—'}</td>
              <td className="max-w-xs truncate px-2 py-1.5 text-text-secondary">{p.description}</td>
              <td className="px-2 py-1.5">
                <button
                  onClick={() => toggle.mutate({ name: p.name, enabled: p.enabled })}
                  className={cx(
                    'rounded px-2 py-0.5 text-xs font-bold',
                    p.enabled ? 'bg-accent text-bg-base' : 'border border-border text-text-secondary',
                  )}
                >
                  {p.enabled ? 'enabled' : 'disabled'}
                </button>
              </td>
              <td className="max-w-xs truncate px-2 py-1.5 text-text-warn">{p.lastError}</td>
              <td className="px-2 py-1.5">
                <button
                  onClick={() => reload.mutate(p.name)}
                  className="text-text-secondary hover:text-accent"
                  title="Reload"
                >
                  <RefreshCw size={14} />
                </button>
              </td>
            </tr>
          ))}
          {(plugins.data ?? []).length === 0 && (
            <tr>
              <td colSpan={7} className="px-2 py-8 text-center text-text-secondary">
                No plugins installed. Drop a <code>.js</code> file into the plugins folder and
                reload.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
