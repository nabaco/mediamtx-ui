// Janus WebRTC Gateway client — streaming plugin only.
// Connects via the Go proxy at /api/v1/janus/ws?token=<jwt>.

function randomTxn(): string {
  return Math.random().toString(36).slice(2, 10)
}

export interface JanusClientOptions {
  wsUrl: string
  onTrack: (track: MediaStreamTrack, streams: readonly MediaStream[]) => void
  onError: (msg: string) => void
  onConnected: () => void
}

export class JanusClient {
  private ws: WebSocket | null = null
  private sessionId: number | null = null
  private handleId: number | null = null
  private pc: RTCPeerConnection | null = null
  private opts: JanusClientOptions
  private pending = new Map<string, (data: any) => void>()
  private keepaliveTimer: ReturnType<typeof setInterval> | null = null
  private currentMountpointId: number | null = null

  constructor(opts: JanusClientOptions) {
    this.opts = opts
  }

  connect(): void {
    const ws = new WebSocket(this.opts.wsUrl, 'janus-protocol')
    this.ws = ws

    ws.onopen = async () => {
      try {
        await this.createSession()
        await this.attachPlugin()
        this.startKeepalive()
        this.opts.onConnected()
      } catch (e: any) {
        this.opts.onError(e.message ?? 'Janus setup failed')
      }
    }

    ws.onmessage = (ev) => this.onMessage(JSON.parse(ev.data))
    ws.onerror = () => this.opts.onError('Janus WebSocket error')
    ws.onclose = (ev) => {
      this.stopKeepalive()
      if (!ev.wasClean) this.opts.onError('Janus connection closed')
    }
  }

  private sendRaw(obj: any): void {
    this.ws?.send(JSON.stringify(obj))
  }

  private request(obj: any): Promise<any> {
    return new Promise((resolve, reject) => {
      const txn = randomTxn()
      obj.transaction = txn
      this.pending.set(txn, resolve)
      try {
        this.ws!.send(JSON.stringify(obj))
      } catch (e) {
        this.pending.delete(txn)
        reject(e)
        return
      }
      setTimeout(() => {
        if (this.pending.has(txn)) {
          this.pending.delete(txn)
          reject(new Error('Janus request timeout'))
        }
      }, 10_000)
    })
  }

  private onMessage(msg: any): void {
    if (msg.transaction && this.pending.has(msg.transaction)) {
      // Janus may send an ack then an event for the same transaction.
      // Resolve on success/error; leave pending for "ack" so the event resolves it.
      if (msg.janus === 'ack') return
      const resolve = this.pending.get(msg.transaction)!
      this.pending.delete(msg.transaction)
      resolve(msg)
      // If the event also carries a JSEP, handle it now
      if (msg.jsep) this.applyRemoteSdp(msg.jsep)
      return
    }

    // Async push events (no pending transaction)
    if (msg.janus === 'event' && msg.jsep) {
      this.applyRemoteSdp(msg.jsep)
    }
    // webrtcup / media / slowlink / hangup — handled implicitly via ICE state
  }

  private async applyRemoteSdp(jsep: { type: string; sdp: string }): Promise<void> {
    if (!this.pc) return
    try {
      await this.pc.setRemoteDescription({ type: jsep.type as RTCSdpType, sdp: jsep.sdp })
      if (jsep.type === 'offer') {
        const answer = await this.pc.createAnswer()
        await this.pc.setLocalDescription(answer)
        this.sendRaw({
          janus: 'message',
          session_id: this.sessionId,
          handle_id: this.handleId,
          transaction: randomTxn(),
          body: { request: 'start' },
          jsep: { type: 'answer', sdp: this.pc.localDescription!.sdp }
        })
      }
    } catch (e: any) {
      this.opts.onError('SDP error: ' + (e.message ?? e))
    }
  }

  private async createSession(): Promise<void> {
    const resp = await this.request({ janus: 'create' })
    if (resp.janus !== 'success') throw new Error('Failed to create Janus session')
    this.sessionId = resp.data.id
  }

  private async attachPlugin(): Promise<void> {
    const resp = await this.request({
      janus: 'attach',
      session_id: this.sessionId,
      plugin: 'janus.plugin.streaming'
    })
    if (resp.janus !== 'success') throw new Error('Failed to attach streaming plugin')
    this.handleId = resp.data.id
    this.setupPeerConnection()
  }

  private setupPeerConnection(): void {
    const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] })
    this.pc = pc

    pc.ontrack = (e) => this.opts.onTrack(e.track, e.streams)

    pc.onicecandidate = (e) => {
      if (!this.sessionId || !this.handleId) return
      this.sendRaw({
        janus: 'trickle',
        session_id: this.sessionId,
        handle_id: this.handleId,
        transaction: randomTxn(),
        candidate: e.candidate ?? { completed: true }
      })
    }

    pc.oniceconnectionstatechange = () => {
      if (pc.iceConnectionState === 'failed') {
        this.opts.onError('ICE connection failed')
      }
    }
  }

  async watchMountpoint(mountpointId: number): Promise<void> {
    if (!this.sessionId || !this.handleId || !this.pc) throw new Error('Not connected')
    this.currentMountpointId = mountpointId

    // The response carries a JSEP offer, but onMessage() already routes any
    // jsep on a matching transaction into applyRemoteSdp. Don't apply it
    // again here — the second pass hits PC.signalingState === 'stable'
    // and throws "Failed to set local answer sdp".
    await this.request({
      janus: 'message',
      session_id: this.sessionId,
      handle_id: this.handleId,
      body: { request: 'watch', id: mountpointId }
    })
  }

  async switchMountpoint(mountpointId: number): Promise<void> {
    if (mountpointId === this.currentMountpointId) return
    if (!this.sessionId || !this.handleId) throw new Error('Not connected')
    // Janus streaming plugin's "switch" reuses the existing PeerConnection and
    // swaps the underlying source — no SDP renegotiation. Requires the new
    // mountpoint to use the same codecs (H264 + PCMA in our case).
    const resp = await this.request({
      janus: 'message',
      session_id: this.sessionId,
      handle_id: this.handleId,
      body: { request: 'switch', id: mountpointId }
    })
    if (resp.plugindata?.data?.error) {
      throw new Error('switch failed: ' + resp.plugindata.data.error)
    }
    this.currentMountpointId = mountpointId
  }

  async getStats(): Promise<RTCStatsReport | null> {
    return this.pc ? await this.pc.getStats() : null
  }

  private startKeepalive(): void {
    this.keepaliveTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN && this.sessionId) {
        this.sendRaw({
          janus: 'keepalive',
          session_id: this.sessionId,
          transaction: randomTxn()
        })
      }
    }, 25_000)
  }

  private stopKeepalive(): void {
    if (this.keepaliveTimer) {
      clearInterval(this.keepaliveTimer)
      this.keepaliveTimer = null
    }
  }

  destroy(): void {
    this.stopKeepalive()
    this.pc?.close()
    this.pc = null
    this.ws?.close()
    this.ws = null
    this.sessionId = null
    this.handleId = null
    this.currentMountpointId = null
    this.pending.clear()
  }
}
