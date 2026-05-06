<script lang="ts">
  import { push } from 'svelte-spa-router'
  import { get } from 'svelte/store'
  import { onDestroy } from 'svelte'
  import { token as jwtStore, currentUser, addToast } from '../lib/stores'
  import { system as systemApi } from '../lib/api'
  import type { SystemInfo } from '../lib/types'

  const STREAM = 'ch14'
  const CHANNELS = ['Master', 'IFB 1', 'IFB 2', 'IFB 3', 'IFB 4', 'IFB 5', 'IFB 6', 'IFB 7']
  const NUM_AUDIO = CHANNELS.length

  // Auth guard
  if (!get(jwtStore)) {
    sessionStorage.setItem('loginReturn', '/ch14-live')
    push('/login')
  }

  let info = $state<SystemInfo | null>(null)
  let videoEl = $state<HTMLVideoElement | null>(null)
  let status = $state<'idle' | 'loading' | 'playing' | 'error'>('idle')
  let errorMsg = $state('')
  let needsTap = $state(false)

  // Audio state
  let audioTracks = $state<MediaTrack[]>(Array(NUM_AUDIO).fill(null))
  let selectedChannel = $state(0)

  interface MediaTrack { track: MediaStreamTrack }

  let rtcPeer: RTCPeerConnection | null = null
  let videoStream: MediaStream | null = null
  let audioCtx: AudioContext | null = null
  let activeGain: GainNode | null = null
  let channelSources: (MediaStreamAudioSourceNode | null)[] = Array(NUM_AUDIO).fill(null)

  const resolvedHost = $derived(
    info
      ? ((info.mediamtxHost === 'localhost' || info.mediamtxHost === '127.0.0.1')
          ? window.location.hostname
          : info.mediamtxHost)
      : ''
  )
  const scheme = $derived(window.location.protocol === 'https:' ? 'https' : 'http')
  const webrtcUrl = $derived(info ? `${scheme}://${resolvedHost}:${info.webrtcPort}/${STREAM}` : '')

  async function loadInfo() {
    try { info = await systemApi.info() }
    catch (e: any) { addToast('error', e.message) }
  }

  function stopAll() {
    activeGain?.disconnect()
    activeGain = null
    channelSources.forEach(s => s?.disconnect())
    channelSources = Array(NUM_AUDIO).fill(null)
    audioCtx?.close()
    audioCtx = null
    if (rtcPeer) { rtcPeer.close(); rtcPeer = null }
    if (videoEl) { videoEl.srcObject = null }
    videoStream = null
    audioTracks = Array(NUM_AUDIO).fill(null)
    status = 'idle'
    needsTap = false
  }

  function ensureAudioCtx() {
    if (!audioCtx || audioCtx.state === 'closed') {
      audioCtx = new AudioContext()
      // Re-create sources for any already-received tracks
      audioTracks.forEach((t, i) => {
        if (t) {
          channelSources[i] = audioCtx!.createMediaStreamSource(new MediaStream([t.track]))
        }
      })
    }
    if (audioCtx.state === 'suspended') audioCtx.resume()
  }

  function activateChannel(i: number) {
    ensureAudioCtx()
    activeGain?.disconnect()
    activeGain = null
    if (!audioTracks[i] || !channelSources[i]) return
    activeGain = audioCtx!.createGain()
    channelSources[i]!.connect(activeGain)
    activeGain.connect(audioCtx!.destination)
  }

  function selectChannel(i: number) {
    selectedChannel = i
    activateChannel(i)
  }

  async function start() {
    if (!info || !webrtcUrl || !videoEl) return
    stopAll()
    status = 'loading'
    needsTap = false

    const newVideoStream = new MediaStream()
    videoStream = newVideoStream
    videoEl.srcObject = newVideoStream

    const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] })
    rtcPeer = pc
    const stale = () => pc.signalingState === 'closed'

    pc.addTransceiver('video', { direction: 'recvonly' })
    for (let i = 0; i < NUM_AUDIO; i++) {
      pc.addTransceiver('audio', { direction: 'recvonly' })
    }

    let audioIdx = 0
    let playStarted = false

    pc.ontrack = e => {
      if (stale()) return
      if (e.track.kind === 'video') {
        newVideoStream.addTrack(e.track)
        if (!playStarted) {
          playStarted = true
          status = 'playing'
          videoEl!.muted = false
          videoEl!.play().then(() => {
            needsTap = false
          }).catch(() => {
            // Autoplay blocked — show tap overlay, play muted, audio via AudioContext
            videoEl!.muted = true
            videoEl!.play().catch(() => {})
            needsTap = true
          })
        }
      } else if (e.track.kind === 'audio' && audioIdx < NUM_AUDIO) {
        const idx = audioIdx++
        const newTracks = [...audioTracks]
        newTracks[idx] = { track: e.track }
        audioTracks = newTracks
        // Create AudioContext source for this track
        if (!audioCtx || audioCtx.state === 'closed') {
          audioCtx = new AudioContext()
        }
        channelSources[idx] = audioCtx.createMediaStreamSource(new MediaStream([e.track]))
        // Auto-activate channel 0 (Master) once it arrives
        if (idx === 0) activateChannel(0)
      }
    }

    pc.oniceconnectionstatechange = () => {
      if (stale()) return
      if (pc.iceConnectionState === 'failed' || pc.iceConnectionState === 'disconnected') {
        status = 'error'
        errorMsg = 'Connection lost'
      }
    }

    try {
      const offer = await pc.createOffer()
      if (stale()) return
      await pc.setLocalDescription(offer)

      await new Promise<void>(resolve => {
        if (pc.iceGatheringState === 'complete') { resolve(); return }
        const fn = () => {
          if (pc.iceGatheringState === 'complete') {
            pc.removeEventListener('icegatheringstatechange', fn); resolve()
          }
        }
        pc.addEventListener('icegatheringstatechange', fn)
        setTimeout(resolve, 5000)
      })

      if (stale()) return

      const jwt = get(jwtStore)
      const username = get(currentUser)?.username
      const headers: Record<string, string> = { 'Content-Type': 'application/sdp' }
      if (username && jwt) headers['Authorization'] = 'Basic ' + btoa(`${username}:${jwt}`)

      const resp = await fetch(webrtcUrl + '/whep', { method: 'POST', headers, body: pc.localDescription!.sdp })
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

  function handleTap() {
    needsTap = false
    ensureAudioCtx()
    if (videoEl) {
      videoEl.muted = false
      videoEl.play().catch(() => {})
    }
    activateChannel(selectedChannel)
  }

  // Start once info and videoEl are both ready
  $effect(() => {
    if (info && videoEl && status === 'idle') start()
    return () => stopAll()
  })

  $effect(() => { loadInfo() })

  onDestroy(stopAll)
</script>

<div class="flex flex-col gap-4">
  <h1 class="text-2xl font-bold text-slate-900">CH14 Live</h1>

  {#if !info}
    <div class="flex justify-center py-16">
      <div class="w-8 h-8 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Desktop: video + channel column side by side; Mobile: video then channel grid -->
    <div class="flex flex-col md:flex-row gap-3">

      <!-- Video -->
      <div class="relative bg-black rounded-xl overflow-hidden flex-1 aspect-video">
        <!-- svelte-ignore a11y_media_has_caption -->
        <video bind:this={videoEl} class="w-full h-full object-contain" controls playsinline></video>

        {#if status === 'loading'}
          <div class="absolute inset-0 flex items-center justify-center bg-black/60">
            <div class="text-white text-sm flex flex-col items-center gap-2">
              <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              Connecting...
            </div>
          </div>
        {:else if status === 'error'}
          <div class="absolute inset-0 flex items-center justify-center bg-black/80">
            <div class="text-center text-white">
              <p class="font-medium">Stream unavailable</p>
              <p class="text-xs text-white/60 mt-1">{errorMsg}</p>
              <button onclick={start} class="mt-3 btn-primary text-xs">Retry</button>
            </div>
          </div>
        {:else if needsTap}
          <!-- Autoplay blocked overlay -->
          <button
            onclick={handleTap}
            class="absolute inset-0 flex items-center justify-center bg-black/50 cursor-pointer"
          >
            <div class="flex flex-col items-center gap-3 text-white">
              <div class="w-16 h-16 rounded-full bg-white/20 flex items-center justify-center">
                <svg class="w-8 h-8 ml-1" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              </div>
              <span class="text-sm font-medium">Tap to watch</span>
            </div>
          </button>
        {/if}
      </div>

      <!-- Audio channel buttons -->
      <div class="flex flex-row flex-wrap md:flex-col gap-2 md:w-32">
        {#each CHANNELS as label, i}
          {@const available = !!audioTracks[i]}
          <button
            onclick={() => selectChannel(i)}
            disabled={!available}
            class="flex-1 md:flex-none min-w-[4rem] md:w-full py-3 md:py-4 px-2 rounded-lg text-sm font-semibold
                   transition-colors text-center leading-tight
                   {selectedChannel === i && available
                     ? 'bg-indigo-600 text-white shadow-md'
                     : available
                       ? 'bg-slate-100 text-slate-700 hover:bg-slate-200 active:bg-slate-300'
                       : 'bg-slate-50 text-slate-300 cursor-not-allowed'}"
          >
            {label}
          </button>
        {/each}
      </div>
    </div>
  {/if}
</div>
