export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(' ')
}

const methodClasses: Record<string, string> = {
  GET: 'bg-accent text-bg-base',
  POST: 'bg-text-warn text-bg-base',
  PUT: 'text-blue-400 border border-blue-400',
  DELETE: 'text-text-danger border border-text-danger',
}

export function MethodChip({ method }: { method: string }) {
  const cls = methodClasses[method] ?? 'text-text-secondary border border-border'

  return (
    <span className={cx('inline-block rounded px-1.5 py-0.5 text-xs font-bold', cls)}>
      {method}
    </span>
  )
}

export function StatusChip({ code }: { code: number }) {
  let cls = 'text-text-secondary border-border'

  if (code >= 200 && code < 300) cls = 'text-accent border-accent'
  else if (code >= 300 && code < 400) cls = 'text-blue-400 border-blue-400'
  else if (code >= 400 && code < 500) cls = 'text-text-warn border-text-warn'
  else if (code >= 500) cls = 'text-text-danger border-text-danger'

  return <span className={cx('inline-block rounded border px-1.5 py-0.5 text-xs', cls)}>{code}</span>
}

export function Spinner() {
  return (
    <span className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
  )
}
