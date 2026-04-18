import { api } from './client'

export type Connection = {
  id: string
  name: string
  driver: string
  read_only: boolean
  statement_timeout_ms: number
  created_at: string
  created_by_email?: string
}

export type CreateConnectionInput = {
  name: string
  driver: string
  dsn: string
  read_only: boolean
  statement_timeout_ms: number
}

export async function listConnections(): Promise<Connection[]> {
  const { data } = await api.get('/connections')
  return data
}

export async function createConnection(input: CreateConnectionInput): Promise<Connection> {
  const { data } = await api.post('/connections', input)
  return data
}

export async function updateConnection(id: string, input: CreateConnectionInput): Promise<Connection> {
  const { data } = await api.put(`/connections/${id}`, input)
  return data
}

export async function deleteConnection(id: string): Promise<void> {
  await api.delete(`/connections/${id}`)
}

export async function testConnection(id: string): Promise<{ status: string }> {
  const { data } = await api.post(`/connections/${id}/test`)
  return data
}

export type SchemaColumn = {
  name: string
  data_type: string
  nullable: boolean
}

export type SchemaTable = {
  schema: string
  table: string
  columns: SchemaColumn[]
}

export async function getConnectionSchema(id: string): Promise<SchemaTable[]> {
  const { data } = await api.get(`/connections/${id}/schema`)
  return data
}