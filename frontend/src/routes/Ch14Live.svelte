<script lang="ts">
  import { push } from 'svelte-spa-router'
  import { get } from 'svelte/store'
  import { onDestroy } from 'svelte'
  import { token as jwtStore, addToast } from '../lib/stores'
  import { JanusClient } from '../lib/janusClient'

  const CHANNELS = ['Master', 'IFB 1', 'IFB 2', 'IFB 3', 'IFB 4', 'IFB 5', 'IFB 6', 'IFB 7']
  const NUM_AUDIO = CHANNELS.length

  interface Quality { label: string; mountpointId: number }
  // mountpointId 3 is the low-rate source (ch14-lr) — non-transcoded.
  const QUALITIES: Quality[] = [
    { label: 'HD', mountpointId: 1 },
    { label: '480p', mountpointId: 3 },
  ]

  // Auth guard
  const jwt = get(jwtStore)
  if (!jwt) {
    sessionStorage.setItem('loginReturn', '/ch14-live')
    push('/login')
  }

  let videoEl = $state<HTMLVideoElement | null>(null)
  let status = $state<'idle' | 'loading' | 'playing' | 'error'>('idle')
  let errorMsg = $state('')
  let needsTap = $state(false)
  let selectedQuality = $state(1) // index into QUALITIES — start on 480p

  // Audio state
  let audioTracks = $state<(MediaStreamTrack | null)[]>(Array(NUM_AUDIO).fill(null))
  let selectedChannel = $state(0)

  let janus: JanusClient | null = null
  let videoStream: MediaStream | null = null
  let videoTrackReceived = false
  let audioIdx = 0

  function buildWsUrl(): string {
    const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
    return `${scheme}://${window.location.host}/api/v1/janus/ws?token=${encodeURIComponent(jwt ?? '')}`
  }

  // Swap whichever audio track is currently attached to the playing
  // MediaStream for the one at index i. Native video controls handle
  // mute/volume — no Web Audio context (which would need a user gesture
  // to leave the suspended state).
  function selectChannel(i: number) {
    selectedChannel = i
    if (!videoStream || !audioTracks[i]) return
    videoStream.getAudioTracks().forEach(t => videoStream!.removeTrack(t))
    videoStream.addTrack(audioTracks[i]!)
  }

  function onTrack(track: MediaStreamTrack, _streams: readonly MediaStream[]) {
    if (track.kind === 'video') {
      if (!videoTrackReceived) {
        videoTrackReceived = true
        status = 'playing'
        videoStream = new MediaStream([track])
        if (videoEl) {
          videoEl.srcObject = videoStream
          // Autoplay policy requires muted=true for unattended start.
          // User unmutes via the native video controls.
          videoEl.muted = true
          videoEl.play().catch(() => { needsTap = true })
        }
      }
    } else if (track.kind === 'audio' && audioIdx < NUM_AUDIO) {
      const idx = audioIdx++
      const newTracks = [...audioTracks]
      newTracks[idx] = track
      audioTracks = newTracks
      // Auto-attach Master once it arrives.
      if (idx === 0 && videoStream) {
        videoStream.addTrack(track)
      }
    }
  }

  function stopAll() {
    janus?.destroy()
    janus = null
    if (videoEl) videoEl.srcObject = null
    videoStream = null
    audioTracks = Array(NUM_AUDIO).fill(null)
    videoTrackReceived = false
    audioIdx = 0
    status = 'idle'
    needsTap = false
  }

  function start() {
    // Guard against the auth-redirect race: $effect fires once before the
    // route change unmounts this component. Without this, an empty-token WS
    // is dispatched and surfaces "Janus WebSocket error" toasts.
    if (!jwt) return
    stopAll()
    status = 'loading'

    janus = new JanusClient({
      wsUrl: buildWsUrl(),
      onTrack,
      onError: (msg) => {
        errorMsg = msg
        status = 'error'
        addToast('error', msg)
      },
      onConnected: async () => {
        try {
          await janus!.watchMountpoint(QUALITIES[selectedQuality].mountpointId)
        } catch (e: any) {
          errorMsg = e.message
          status = 'error'
        }
      }
    })

    janus.connect()
  }

  async function switchQuality(i: number) {
    if (i === selectedQuality) return
    selectedQuality = i
    if (!janus || status !== 'playing') return
    try {
      await janus.switchMountpoint(QUALITIES[i].mountpointId)
    } catch (e: any) {
      addToast('error', e.message)
    }
  }

  function handleTap() {
    needsTap = false
    if (videoEl) {
      videoEl.muted = false
      videoEl.play().catch(() => {})
    }
  }

  // Start once videoEl is bound. Don't read `status` here — the cleanup
  // resets it to 'idle', which would re-trigger the effect and create an
  // infinite reconnect loop.
  $effect(() => {
    if (videoEl) start()
    return () => stopAll()
  })

  onDestroy(stopAll)
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-slate-900">CH14 Live</h1>

    <!-- Quality selector -->
    <div class="flex gap-1">
      {#each QUALITIES as q, i}
        <button
          onclick={() => switchQuality(i)}
          class="px-3 py-1.5 rounded-md text-sm font-semibold transition-colors
                 {selectedQuality === i
                   ? 'bg-indigo-600 text-white'
                   : 'bg-slate-100 text-slate-600 hover:bg-slate-200'}"
        >
          {q.label}
        </button>
      {/each}
    </div>
  </div>

  <!-- Desktop: video + channel column; Mobile: video then grid -->
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
</div>
