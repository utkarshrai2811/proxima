import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Activity, PauseCircle, Crosshair, Send, Zap, Radio, Puzzle, Settings,
  type LucideIcon,
} from 'lucide-react'
import { fetchActiveProject, fetchInterceptedRequests, fetchRequestLogs } from '../lib/api'
import { cx } from './ui'

interface NavItem {
  label: string
  icon: LucideIcon
  path: string
  badge?: boolean
}

const navItems: NavItem[] = [
  { label: 'Proxy', icon: Activity, path: '/' },
  { label: 'Intercept', icon: PauseCircle, path: '/intercept', badge: true },
  { label: 'Scope', icon: Crosshair, path: '/scope' },
  { label: 'HTTP Client', icon: Send, path: '/client' },
  { label: 'Fuzzer', icon: Zap, path: '/fuzzer' },
  { label: 'WebSockets', icon: Radio, path: '/websockets' },
  { label: 'Plugins', icon: Puzzle, path: '/plugins' },
  { label: 'Settings', icon: Settings, path: '/settings' },
]

export default function Layout({ children }: { children: ReactNode }) {
  const intercepted = useQuery({
    queryKey: ['interceptedRequests'],
    queryFn: fetchInterceptedRequests,
    refetchInterval: 2000,
  })
  const project = useQuery({ queryKey: ['activeProject'], queryFn: fetchActiveProject })
  const logs = useQuery({ queryKey: ['requestLogs'], queryFn: fetchRequestLogs })

  const interceptCount = intercepted.data?.length ?? 0
  const requestCount = logs.data?.length ?? 0

  return (
    <div className="flex h-screen flex-col">
      <div className="flex flex-1 overflow-hidden">
        <aside className="flex w-60 shrink-0 flex-col border-r border-border bg-bg-surface">
          <div className="px-5 py-5">
            <div className="text-lg font-bold tracking-widest text-accent">PROXIMA</div>
            <div className="text-xs text-text-secondary">security toolkit</div>
          </div>
          <div className="mx-5 border-t border-border" />
          <nav className="flex flex-col py-2">
            {navItems.map(({ label, icon: Icon, path, badge }) => (
              <NavLink
                key={path}
                to={path}
                end={path === '/'}
                className={({ isActive }) =>
                  cx(
                    'flex items-center gap-3 border-l-2 px-5 py-2 text-sm transition-colors',
                    isActive
                      ? 'border-accent text-accent'
                      : 'border-transparent text-text-secondary hover:text-text-primary',
                  )
                }
              >
                <Icon size={16} />
                <span className="flex-1">{label}</span>
                {badge && interceptCount > 0 && (
                  <span className="rounded bg-accent px-1.5 text-xs font-bold text-bg-base">
                    {interceptCount}
                  </span>
                )}
              </NavLink>
            ))}
          </nav>
        </aside>

        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>

      <footer className="flex h-8 shrink-0 items-center justify-between border-t border-border bg-bg-surface px-4 text-xs">
        <div className="flex items-center gap-2">
          <span className="inline-block h-2 w-2 rounded-full bg-accent" />
          <span className="text-text-secondary">Proxy running</span>
        </div>
        <div className="italic text-text-secondary">
          {project.data ? project.data.name : 'no active project'}
        </div>
        <div className="flex items-center gap-3 text-text-secondary">
          <span>{requestCount} requests</span>
          <span className={interceptCount > 0 ? 'text-text-warn' : undefined}>
            {interceptCount} intercepted
          </span>
        </div>
      </footer>
    </div>
  )
}
