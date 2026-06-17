import type { LucideIcon } from 'lucide-react'

export default function StubPage({ title, icon: Icon }: { title: string; icon: LucideIcon }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 text-center">
      <Icon className="text-accent" size={48} strokeWidth={1.5} />
      <h1 className="text-xl font-bold tracking-wide">{title}</h1>
      <p className="text-sm text-text-secondary">Coming soon — this feature is being built.</p>
    </div>
  )
}
