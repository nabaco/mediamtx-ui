<script lang="ts">
  import { _ } from 'svelte-i18n'
  import Hls from 'hls.js'
  import { get } from 'svelte/store'
  import { onDestroy } from 'svelte'
  import { token as jwtStore, currentUser } from '../stores'
  import type { SystemInfo } from '../types'

  // Factory returns a hls.js pLoader that rewrites absolute URLs in manifests to use
  // the known-good public host, fixing mediamtx embedding its internal hostname in HLS output.
  function makeManifestRewriteLoader(host: string, port: number, scheme: string) {
    const DefaultLoader = Hls.DefaultConfig.loader as any
    return class extends DefaultLoader {
      load(context: any, config: any, callbacks: any) {
        const origSuccess = callbacks.onSuccess
        callbacks = {
          ...callbacks,
          onSuccess: (response: any, stats: any, ctx: any, nd: any) => {
            if (typeof response.data === 'string') {
              response.data = response.data.replace(/https?:\/\/[^\s"]+/g, (m: string) => {
                try {
                  const u = new URL(m)
                  u.protocol = scheme + ':'
                  u.hostname = host
                  u.port = String(port)
                  return u.toString()
                } catch { return m }
              })
            }
            origSuccess(response, stats, ctx, nd)
          }
        }
        super.load(context, config, callbacks)
      }
    }
  }

  interface Props {
    streamName: string
    info: SystemInfo
  }
  let { streamName, info }: Props = $props()

  type Mode = 'auto' | 'webrtc' | 'hls'
  let mode = $state<Mode>('auto')
  let autoFallback = $state(false)
  let videoEl = $state<HTMLVideoElement | null>(null)
  let status = $state<'loading' | 'playing' | 'error' | 'idle'>('idle')
  let errorMsg = $state('')

  let hlsInstance: Hls | null = null
  let rtcPeer: RTCPeerConnection | null = null
  let qualityTimer: ReturnType<typeof setInterval> | null = null
  let lastPktReceived = 0
  let lastPktLost = 0
  let lastVideoTime = 0
  let frozenTicks = 0

  const resolvedHost = $derived(
    (info.mediamtxHost === 'localhost' || info.mediamtxHost === '127.0.0.1')
      ? window.location.hostname
      : info.mediamtxHost
  )
  const scheme = $derived(window.location.protocol === 'https:' ? 'https' : 'http')
  const hlsUrl = $derived(`${scheme}://${resolvedHost}:${info.hlsPort}/${streamName}/index.m3u8`)
  const webrtcUrl = $derived(`${scheme}://${resolvedHost}:${info.webrtcPort}/${streamName}`)

  function stopQualityMonitor() {
    if (qualityTimer) { clearInterval(qualityTimer); qualityTimer = null }
  }

  function startQualityMonitor() {
    stopQualityMonitor()
    lastPktReceived = 0; lastPktLost = 0
    lastVideoTime = videoEl?.currentTime ?? 0
    frozenTicks = 0

    qualityTimer = setInterval(async () => {
      if (!rtcPeer || mode !== 'auto' || status !== 'playing') return

      // Video freeze detection: if currentTime hasn't advanced for two ticks (10s), bail out
      const ct = videoEl?.currentTime ?? 0
      if (ct === lastVideoTime) {
        if (++frozenTicks >= 2) { fallbackToHLS(); return }
      } else {
        frozenTicks = 0
      }
      lastVideoTime = ct

      // Packet loss detection via WebRTC stats
      try {
        const stats = await rtcPeer!.getStats()
        stats.forEach(report => {
          if (report.type !== 'inbound-rtp' || report.kind !== 'video') return
          const dr = report.packetsReceived - lastPktReceived
          const dl = report.packetsLost - lastPktLost
          lastPktReceived = report.packetsReceived
          lastPktLost = report.packetsLost
          const total = dr + dl
          // Only act if we have a meaningful sample and >5% loss
          if (total > 10 && dl / total > 0.05) fallbackToHLS()
        })
      } catch {}
    }, 5000)
  }

  function fallbackToHLS() {
    stopQualityMonitor()
    autoFallback = true
    startHLS()
  }

  function stopAll() {
    stopQualityMonitor()
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
      const stale = () => pc.signalingState === 'closed'

      pc.addTransceiver('video', { direction: 'recvonly' })
      pc.addTransceiver('audio', { direction: 'recvonly' })

      const remoteStream = new MediaStream()
      videoEl.srcObject = remoteStream
      let playStarted = false

      pc.ontrack = e => {
        if (stale() || !videoEl) return
        remoteStream.addTrack(e.track)
        if (!playStarted) {
          playStarted = true
          videoEl.muted = true
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

      await new Promise<void>(resolve => {
        if (pc.iceGatheringState === 'complete') { resolve(); return }
        const onStateChange = () => {
          if (pc.iceGatheringState === 'complete') {
            pc.removeEventListener('icegatheringstatechange', onStateChange)
            resolve()
          }
        }
        pc.addEventListener('icegatheringstatechange', onStateChange)
        setTimeout(resolve, 5000)
      })

      if (stale()) return

      const jwt = get(jwtStore)
      const username = get(currentUser)?.username
      const whepHeaders: Record<string, string> = { 'Content-Type': 'application/sdp' }
      if (username && jwt) {
        whepHeaders['Authorization'] = 'Basic ' + btoa(`${username}:${jwt}`)
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
      const jwt = get(jwtStore)
      const username = get(currentUser)?.username
      const hlsConfig: any = {
        enableWorker: false,
        pLoader: makeManifestRewriteLoader(resolvedHost, info.hlsPort, scheme),
      }
      if (username && jwt) {
        const authHeader = 'Basic ' + btoa(`${username}:${jwt}`)
        hlsConfig.xhrSetup = (xhr: XMLHttpRequest) => {
          xhr.setRequestHeader('Authorization', authHeader)
        }
      }
      const hls = new Hls(hlsConfig)
      hlsInstance = hls
      hls.loadSource(hlsUrl)
      hls.attachMedia(videoEl)
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        videoEl!.muted = true
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
      videoEl.src = hlsUrl
      videoEl.muted = true
      videoEl.play().then(() => status = 'playing').catch(e => {
        errorMsg = e.message; status = 'error'
      })
    } else {
      errorMsg = 'HLS not supported in this browser'
      status = 'error'
    }
  }

  async function start() {
    autoFallback = false
    stopQualityMonitor()
    if (mode === 'hls') {
      await startHLS()
      return
    }
    // 'webrtc' or 'auto'
    await startWebRTC()
    if (status === 'error' && mode === 'auto') {
      autoFallback = true
      await startHLS()
    } else if (status === 'playing' && mode === 'auto') {
      startQualityMonitor()
    }
  }

  function switchMode(m: Mode) {
    mode = m
    start()
  }

  $effect(() => {
    if (videoEl) start()
    return () => stopAll()
  })

  onDestroy(stopAll)
</script>

<div class="flex flex-col gap-3">
  <!-- Mode switcher -->
  <div class="flex items-center gap-2 flex-wrap">
    <span class="text-sm text-slate-500">{$_('player.quality')}:</span>
    {#each (['auto', 'webrtc', 'hls'] as Mode[]) as m}
      <button
        onclick={() => switchMode(m)}
        class="px-3 py-1 text-xs rounded-full font-medium transition-colors
               {mode === m ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'}"
      >
        {$_(`player.mode_${m}`)}
      </button>
    {/each}
    {#if autoFallback}
      <span class="text-xs text-amber-600 italic">{$_('player.auto_downgraded')}</span>
    {/if}
  </div>

  <!-- Video element -->
  <div class="relative bg-black rounded-xl overflow-hidden aspect-video group">
    <!-- svelte-ignore a11y_media_has_caption -->
    <video
      bind:this={videoEl}
      class="w-full h-full object-contain"
      controls
      playsinline
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
