import { writable, derived } from 'svelte/store'
import type { MeResponse } from './types'

// ---- Auth ----
export const token = writable<string | null>(localStorage.getItem('token'))
export const currentUser = writable<MeResponse | null>(null)

token.subscribe(t => {
  if (t) localStorage.setItem('token', t)
  else localStorage.removeItem('token')
})

export const isAuthenticated = derived(token, $t => !!$t)
export const isAdmin = derived(currentUser, $u => $u?.role === 'admin')

export function logout() {
  token.set(null)
  currentUser.set(null)
}

// ---- Language / i18n ----
export type Lang = 'en' | 'he'
const storedLang = (localStorage.getItem('lang') as Lang) ?? 'en'
export const lang = writable<Lang>(storedLang)

lang.subscribe(l => {
  localStorage.setItem('lang', l)
  // Update document direction for RTL support
  document.documentElement.lang = l
  document.documentElement.dir = l === 'he' ? 'rtl' : 'ltr'
})

// Apply stored lang direction immediately on load
document.documentElement.lang = storedLang
document.documentElement.dir = storedLang === 'he' ? 'rtl' : 'ltr'

// ---- Toast notifications ----
export interface Toast {
  id: number
  type: 'success' | 'error' | 'info'
  message: string
}

let toastId = 0
export const toasts = writable<Toast[]>([])

export function addToast(type: Toast['type'], message: string, durationMs = 4000) {
  const id = ++toastId
  toasts.update(t => [...t, { id, type, message }])
  setTimeout(() => toasts.update(t => t.filter(x => x.id !== id)), durationMs)
}
