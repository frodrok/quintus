import { api } from './client'

export type Parameter = {
  name: string
  type: 'string' | 'int' | 'float' | 'date' | 'bool' | 'enum'
  required?: boolean
  default?: any
  values?: string[] // for enum
}

export type ColumnMask = {
  column: string
  visible_to_groups: string[]
  mask: 'redacted' | 'partial' | 'null' | 'hash'
}

export type RowMask = {
  condition_column: string
  visible_to_groups: string[]
}

export type Query = {
  id: string
  name: string
  description?: string
  connection_id: string
  sql: string
  parameters: Parameter[]
  column_masks: ColumnMask[]
  row_mask?: RowMask | null
  owner_email: string
  created_at: string
  updated_at: string
}

export type CreateQueryInput = {
  name: string
  description?: string
  connection_id: string
  sql: string
  parameters: Parameter[]
  column_masks: ColumnMask[]
  row_mask?: RowMask | null
}

export async function listQueries(): Promise<Query[]> {
  const { data } = await api.get('/queries')
  return data
}

export async function getQuery(id: string): Promise<Query> {
  const { data } = await api.get(`/queries/${id}`)
  return data
}

export async function createQuery(input: CreateQueryInput): Promise<Query> {
  const { data } = await api.post('/queries', input)
  return data
}

export async function updateQuery(id: string, input: CreateQueryInput): Promise<Query> {
  const { data } = await api.put(`/queries/${id}`, input)
  return data
}

