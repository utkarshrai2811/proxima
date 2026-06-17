import { useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  closeProject,
  createProject,
  deleteProject,
  fetchActiveProject,
  fetchProjects,
  openProject,
} from '../lib/api'

interface ServerConfig {
  listenAddr: string
  maxBodySize: string
  allowedHosts: string
  authEnabled: boolean
}

async function fetchConfig(): Promise<ServerConfig> {
  const res = await fetch('/api/config', { credentials: 'same-origin' })
  if (!res.ok) throw new Error(`config request failed: ${res.status}`)
  return res.json()
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="mb-4 border border-border bg-bg-surface p-4">
      <h2 className="mb-3 text-sm font-bold uppercase tracking-wide text-accent">{title}</h2>
      {children}
    </div>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between border-b border-border py-1.5 text-sm last:border-0">
      <span className="text-text-secondary">{label}</span>
      <span className="break-all">{value}</span>
    </div>
  )
}

export default function Settings() {
  const qc = useQueryClient()
  const active = useQuery({ queryKey: ['activeProject'], queryFn: fetchActiveProject })
  const projects = useQuery({ queryKey: ['projects'], queryFn: fetchProjects })
  const config = useQuery({ queryKey: ['config'], queryFn: fetchConfig, retry: 0 })
  const [newName, setNewName] = useState('')

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['activeProject'] })
    qc.invalidateQueries({ queryKey: ['projects'] })
  }

  const create = useMutation({
    mutationFn: () => createProject(newName.trim()),
    onSuccess: () => {
      setNewName('')
      invalidate()
    },
  })
  const close = useMutation({ mutationFn: closeProject, onSuccess: invalidate })
  const open = useMutation({ mutationFn: openProject, onSuccess: invalidate })
  const remove = useMutation({ mutationFn: deleteProject, onSuccess: invalidate })

  return (
    <div className="mx-auto max-w-3xl p-4">
      <h1 className="mb-4 text-lg font-bold">Settings</h1>

      <Card title="Project">
        <div className="mb-3 text-sm">
          Active project:{' '}
          <span className="font-bold">{active.data ? active.data.name : 'none'}</span>
        </div>
        {active.data && (
          <button
            onClick={() => window.confirm('Close the active project?') && close.mutate()}
            className="mb-3 border border-text-danger px-3 py-1.5 text-xs text-text-danger hover:bg-text-danger hover:text-bg-base"
          >
            Close Project
          </button>
        )}
        <div className="mb-3 flex gap-2">
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="new project name"
            className="flex-1 border border-border bg-bg-base px-2 py-1.5 text-sm outline-none focus:border-accent"
          />
          <button
            disabled={!newName.trim()}
            onClick={() => create.mutate()}
            className="bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim disabled:opacity-40"
          >
            Create
          </button>
        </div>
        <div className="text-xs text-text-secondary">Projects</div>
        <div className="mt-1 space-y-1">
          {(projects.data ?? []).map((p) => (
            <div key={p.id} className="flex items-center justify-between text-sm">
              <button
                onClick={() => open.mutate(p.id)}
                className="hover:text-accent"
                disabled={p.isActive}
              >
                {p.name} {p.isActive && <span className="text-xs text-accent">(active)</span>}
              </button>
              <button
                onClick={() => window.confirm(`Delete project "${p.name}"?`) && remove.mutate(p.id)}
                className="text-xs text-text-secondary hover:text-text-danger"
              >
                delete
              </button>
            </div>
          ))}
          {(projects.data ?? []).length === 0 && (
            <div className="text-sm text-text-secondary">No projects yet.</div>
          )}
        </div>
      </Card>

      <Card title="Certificate">
        <p className="mb-3 text-sm text-text-secondary">
          Install this certificate in your browser or OS trust store to intercept HTTPS traffic.
        </p>
        <a
          href="/api/cert/download"
          download="proxima_cert.pem"
          className="inline-block bg-accent px-3 py-1.5 text-xs font-bold text-bg-base hover:bg-accent-dim"
        >
          Download CA Certificate
        </a>
      </Card>

      <Card title="Proxy">
        {config.data ? (
          <>
            <Field label="Listen address" value={config.data.listenAddr} />
            <Field label="Max body size" value={config.data.maxBodySize} />
            <Field label="Allowed hosts" value={config.data.allowedHosts} />
          </>
        ) : (
          <div className="text-sm text-text-secondary">Configuration unavailable.</div>
        )}
      </Card>

      <Card title="Authentication">
        <Field label="Auth enabled" value={config.data?.authEnabled ? 'yes' : 'no'} />
        {config.data?.authEnabled && (
          <p className="mt-2 text-sm text-text-secondary">
            An API key is configured (the value is never shown). See the README for header and
            cookie usage.
          </p>
        )}
      </Card>
    </div>
  )
}
