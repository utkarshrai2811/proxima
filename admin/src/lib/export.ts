export type ExportFormat = 'BURP_XML' | 'CURL' | 'OPENAPI'

export interface ExportResult {
  content: string
  filename: string
  mimeType: string
}

export async function exportRequests(ids: string[], format: ExportFormat): Promise<ExportResult> {
  const res = await fetch('/api/export', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids, format }),
  })

  if (!res.ok) throw new Error((await res.text()) || `export failed: ${res.status}`)

  return res.json()
}

export function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
