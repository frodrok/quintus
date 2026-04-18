import { api } from './client'
import type { Column } from './queries'

export type RunResult = {
  run_id: string
  columns: { name: string; type: string }[]
  rows: any[][]
  truncated: boolean
  row_count: number
  masked_columns?: { column: string; mask: string }[]
}
export type CreateRunInput = {
  connection_id: string
  query_id?: string
  sql: string
  parameters?: Record<string, any>
}

export async function createRun(input: CreateRunInput): Promise<RunResult> {
  const { data } = await api.post('/runs', input)
  return data
}

export function downloadCSV(result: RunResult, filename = 'export.csv') {
  const header = result.columns.map(c => c.name).join(',')
  const rows = result.rows.map(row =>
    row.map(val => {
      if (val === null || val === undefined) return ''
      const str = String(val)
      return str.includes(',') || str.includes('"') || str.includes('\n')
        ? `"${str.replace(/"/g, '""')}"`
        : str
    }).join(',')
  )
  const csv = [header, ...rows].join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' })
  triggerDownload(blob, filename)
}

export function downloadExcel(result: RunResult, filename = 'export.xlsx') {
  // Enkel TSV som Excel öppnar direkt
  const header = result.columns.map(c => c.name).join('\t')
  const rows = result.rows.map(row =>
    row.map(val => val === null || val === undefined ? '' : String(val)).join('\t')
  )
  const tsv = [header, ...rows].join('\n')
  const blob = new Blob([tsv], { type: 'application/vnd.ms-excel;charset=utf-8;' })
  triggerDownload(blob, filename)
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}