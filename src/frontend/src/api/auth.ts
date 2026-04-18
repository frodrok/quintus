import { api } from './client'

export type Me = {
  sub: string
  email: string
  name: string
  groups: string[]
  role: string
  can_pii: boolean
  can_adhoc: boolean
}

export async function getMe(): Promise<Me> {
  const { data } = await api.get('/me')
  return data
}

export async function getLogout(): Promise<{ url: string }> {
  const { data } = await api.get('/logout')
  return data
}