import { get } from 'svelte/store'
import { token, logout } from './stores'
import type {
  AuthResponse, MeResponse, Stream, StreamURLs,
  User, Group, ACL, SystemInfo, SystemConfig, AuditPage,
} from './types'

const BASE = '/api/v1'

class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const t = get(token)
  if (t) headers['Authorization'] = `Bearer ${t}`

  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body != null ? JSON.stringify(body) : undefined,
  })

  if (res.status === 401) {
    logout()
    throw new APIError(401, 'Session expired')
  }

  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => ({ error: res.statusText }))
  if (!res.ok) throw new APIError(res.status, data.error ?? res.statusText)
  return data as T
}

const get_ = <T>(path: string) => request<T>('GET', path)
const post = <T>(path: string, body: unknown) => request<T>('POST', path, body)
const patch = <T>(path: string, body: unknown) => request<T>('PATCH', path, body)
const del = (path: string) => request<void>('DELETE', path)

// ---- Auth ----
export const auth = {
  login: (username: string, password: string) =>
    post<AuthResponse>('/auth/login', { username, password }),
  me: () => get_<MeResponse>('/auth/me'),
  changePassword: (currentPassword: string, newPassword: string) =>
    post<void>('/auth/change-password', { currentPassword, newPassword }),
  getStreamToken: () => get_<{ token: string | null }>('/auth/stream-token'),
  regenerateStreamToken: () => post<{ token: string }>('/auth/stream-token', {}),
}

// ---- Streams ----
export const streams = {
  list: () => get_<Stream[]>('/streams'),
  get: (name: string) => get_<Stream>(`/streams/${encodeURIComponent(name)}`),
  urls: (name: string) => get_<StreamURLs>(`/streams/${encodeURIComponent(name)}/urls`),
  create: (data: { name: string; description?: string; source?: string; sourceOnDemand?: boolean; record?: boolean; maxReaders?: number }) =>
    post<{ name: string }>('/streams', data),
  update: (name: string, data: Partial<{ description: string; source: string; sourceOnDemand: boolean; record: boolean; maxReaders: number }>) =>
    patch<void>(`/streams/${encodeURIComponent(name)}`, data),
  delete: (name: string) => del(`/streams/${encodeURIComponent(name)}`),
}

// ---- Users ----
export const users = {
  list: () => get_<User[]>('/users'),
  get: (id: number) => get_<User>(`/users/${id}`),
  create: (data: { username: string; password: string; role?: string }) =>
    post<User>('/users', data),
  update: (id: number, data: Partial<{ password: string; role: string; enabled: boolean }>) =>
    patch<User>(`/users/${id}`, data),
  delete: (id: number) => del(`/users/${id}`),
  regenerateToken: (id: number) =>
    post<{ token: string }>(`/users/${id}/stream-token`, {}),
}

// ---- Groups ----
export const groups = {
  list: () => get_<Group[]>('/groups'),
  create: (name: string) => post<Group>('/groups', { name }),
  rename: (id: number, name: string) => patch<Group>(`/groups/${id}`, { name }),
  delete: (id: number) => del(`/groups/${id}`),
  members: (id: number) => get_<User[]>(`/groups/${id}/members`),
  addMember: (groupId: number, userId: number) =>
    post<void>(`/groups/${groupId}/members`, { userId }),
  removeMember: (groupId: number, userId: number) =>
    del(`/groups/${groupId}/members/${userId}`),
}

// ---- ACLs ----
export const acls = {
  list: () => get_<ACL[]>('/acls'),
  create: (data: { subjectType: string; subjectId: number; streamPattern: string; action: string }) =>
    post<ACL>('/acls', data),
  delete: (id: number) => del(`/acls/${id}`),
}

// ---- System ----
export const system = {
  info: () => get_<SystemInfo>('/system/info'),
  config: () => get_<SystemConfig>('/system/config'),
}

// ---- Audit ----
export const audit = {
  list: (params?: { username?: string; stream?: string; limit?: number; offset?: number }) => {
    const q = new URLSearchParams()
    if (params?.username) q.set('username', params.username)
    if (params?.stream) q.set('stream', params.stream)
    if (params?.limit) q.set('limit', String(params.limit))
    if (params?.offset) q.set('offset', String(params.offset))
    return get_<AuditPage>(`/audit?${q}`)
  },
}

export { APIError }
