<script lang="ts">
  import { _ } from 'svelte-i18n'
  import Hls from 'hls.js'
  import { onDestroy } from 'svelte'
  import type { SystemInfo } from '../types'

  interface Props {
    streamName: string
    info: SystemInfo
    username?: string
    streamToken?: string
  }
  let { streamName, info, username, streamToken }: Props = $props()

  type Format = 'webrtc' | 'hls'
  let format = $state<Format>('webrtc')
  let videoEl = $state<HTMLVideoElement | null>(null)
  let status = $state<'loading' | 'playing' | 'error' | 'idle'>('idle')
  let errorMsg = $state('')

  let hlsInstance: Hls | null = null
  let rtcPeer: RTCPeerConnection | null = null

  // If the server returns localhost/127.0.0.1, use the browser's hostname instead
  // so stream URLs work when accessing the UI from a remote machine.
  const resolvedHost = $derived(
    (info.mediamtxHost === 'localhost' || info.mediamtxHost === '127.0.0.1')
      ? window.location.hostname
      : info.mediamtxHost
  )

  const hlsUrl = $derived(
    `http://${resolvedHost}:${info.hlsPort}/${streamName}/index.m3u8`
  )
  const webrtcUrl = $derived(
    `http://${resolvedHost}:${info.webrtcPort}/${streamName}`
  )

  function stopAll() {
    if (hlsInstance) { hlsInstance.destroy(); hlsInstance = null }
    if (rtcPeer) { rtcPeer.close(); rtcPeer = null }
    if (videoEl) { videoEl.srcObject = null; videoEl.src = '' }
    status = 'idle'
  }

  async function startWebRTC() {
    if (!videoEl) return
    stopAll()
    status = 'loading'
    try {
      const pc = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
      })
      rtcPeer = pc

      // Helper: bail out silently if this pc was superseded by stopAll()
      const stale = () => pc.signalingState === 'closed'

      pc.addTransceiver('video', { direction: 'recvonly' })
      pc.addTransceiver('audio', { direction: 'recvonly' })

      pc.ontrack = e => {
        if (e.streams[0] && videoEl && !stale()) {
          videoEl.srcObject = e.streams[0]
          videoEl.play().catch(() => {})
          status = 'playing'
        }
      }

      pc.oniceconnectionstatechange = () => {
        if (stale()) return
        if (pc.iceConnectionState === 'failed' || pc.iceConnectionState === 'disconnected') {
          status = 'error'
          errorMsg = 'WebRTC connection lost'
        }
      }

      const offer = await pc.createOffer()
      if (stale()) return
      await pc.setLocalDescription(offer)

      // Wait for ICE gathering to complete so all candidates are in the SDP
      await new Promise<void>(resolve => {
        if (pc.iceGatheringState === 'complete') { resolve(); return }
        const onStateChange = () => {
          if (pc.iceGatheringState === 'complete') {
            pc.removeEventListener('icegatheringstatechange', onStateChange)
            resolve()
          }
        }
        pc.addEventListener('icegatheringstatechange', onStateChange)
        // Safety timeout: send after 5s even if gathering isn't done
        setTimeout(resolve, 5000)
      })

      if (stale()) return

      const whepHeaders: Record<string, string> = { 'Content-Type': 'application/sdp' }
      if (username && streamToken) {
        whepHeaders['Authorization'] = 'Basic ' + btoa(`${username}:${streamToken}`)
      }

      const resp = await fetch(webrtcUrl + '/whep', {
        method: 'POST',
        headers: whepHeaders,
        body: pc.localDescription!.sdp,
      })

      if (stale()) return
      if (!resp.ok) throw new Error(`WHEP ${resp.status}: ${await resp.text()}`)

      const sdpAnswer = await resp.text()
      if (stale()) return
      await pc.setRemoteDescription({ type: 'answer', sdp: sdpAnswer })
    } catch (e: any) {
      errorMsg = e.message
      status = 'error'
    }
  }

  async function startHLS() {
    if (!videoEl) return
    stopAll()
    status = 'loading'
    if (Hls.isSupported()) {
      const hlsConfig: Partial<Hls['config']> = { enableWorker: false }
      if (username && streamToken) {
        const authHeader = 'Basic ' + btoa(`${username}:${streamToken}`)
        hlsConfig.xhrSetup = (xhr: XMLHttpRequest) => {
          xhr.setRequestHeader('Authorization', authHeader)
        }
      }
      const hls = new Hls(hlsConfig)
      hlsInstance = hls
      hls.loadSource(hlsUrl)
      hls.attachMedia(videoEl)
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        videoEl?.play().catch(() => {})
        status = 'playing'
      })
      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal) {
          errorMsg = 'HLS error: ' + data.details
          status = 'error'
        }
      })
    } else if (videoEl.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS — credentials must be embedded in the URL
      const safariUrl = username && streamToken
        ? `http://${username}:${streamToken}@${resolvedHost}:${info.hlsPort}/${streamName}/index.m3u8`
        : hlsUrl
      videoEl.src = safariUrl
      videoEl.play().then(() => status = 'playing').catch(e => {
        errorMsg = e.message; status = 'error'
      })
    } else {
      errorMsg = 'HLS not supported in this browser'
      status = 'error'
    }
  }

  async function start() {
    if (format === 'webrtc') {
      await startWebRTC()
      // Auto-fallback: if WebRTC fails, silently try HLS
      if (status === 'error') {
        format = 'hls'
        await startHLS()
      }
    } else {
      await startHLS()
    }
  }

  function switchFormat(f: Format) {
    format = f
    start()
  }

  // Auto-start when videoEl is available
  $effect(() => {
    if (videoEl) start()
    return () => stopAll()
  })

  onDestroy(stopAll)
</script>

<div class="flex flex-col gap-3">
  <!-- Format switcher -->
  <div class="flex items-center gap-2">
    <span class="text-sm text-slate-500">{$_('player.format')}:</span>
    {#each (['webrtc', 'hls'] as Format[]) as f}
      <button
        onclick={() => switchFormat(f)}
        class="px-3 py-1 text-xs rounded-full font-medium transition-colors
               {format === f ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'}"
      >
        {$_(`player.${f}`)}
      </button>
    {/each}
  </div>

  <!-- Video element -->
  <div class="relative bg-black rounded-xl overflow-hidden aspect-video">
    <!-- svelte-ignore a11y_media_has_caption -->
    <video
      bind:this={videoEl}
      class="w-full h-full object-contain"
      controls
      playsinline
      muted
    ></video>

    {#if status === 'loading'}
      <div class="absolute inset-0 flex items-center justify-center bg-black/60">
        <div class="text-white text-sm flex flex-col items-center gap-2">
          <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
          {$_('player.loading')}
        </div>
      </div>
    {:else if status === 'error'}
      <div class="absolute inset-0 flex items-center justify-center bg-black/80">
        <div class="text-center text-white">
          <p class="font-medium">{$_('player.error')}</p>
          <p class="text-xs text-white/60 mt-1">{errorMsg}</p>
          <button onclick={start} class="mt-3 btn-primary text-xs">Retry</button>
        </div>
      </div>
    {/if}
  </div>
</div>
