export interface User {
  id: number
  username: string
  role: 'admin' | 'user'
  enabled: boolean
  hasToken: boolean
  createdAt: string
}

export interface Group {
  id: number
  name: string
  members?: User[]
  createdAt: string
}

export interface ACL {
  id: number
  subjectType: 'user' | 'group'
  subjectId: number
  subjectName: string
  streamPattern: string
  action: 'read' | 'publish' | 'both'
  createdAt: string
}

export interface Stream {
  name: string
  description: string
  source: string
  ready: boolean
  tracks: string[]
  readers: number
  bytesReceived: number
  bytesSent: number
}

export interface StreamURLs {
  rtsp: string
  rtsps?: string
  hls: string
  webrtc: string
  rtmp: string
  streamToken?: string
  username?: string
}

export interface SystemInfo {
  version: string
  deployType: 'docker' | 'podman' | 'compose' | 'quadlets' | 'systemd' | 'unknown'
  mediamtxHost: string
  rtspPort: number
  hlsPort: number
  webrtcPort: number
  rtmpPort: number
  mediamtxOnline: boolean
}

export interface SystemConfig {
  available: boolean
  resolvedPath?: string
  rawYaml?: string
}

export interface AuditEntry {
  id: number
  username: string
  streamPath: string
  action: string
  protocol: string
  remoteAddr: string
  allowed: boolean
  createdAt: string
}

export interface AuditPage {
  total: number
  entries: AuditEntry[]
}

export interface AuthResponse {
  token: string
  username: string
  role: string
}

export interface MeResponse {
  id: number
  username: string
  role: string
}
