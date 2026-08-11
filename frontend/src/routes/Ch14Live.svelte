<script lang="ts">
  import { push } from 'svelte-spa-router'
  import { get } from 'svelte/store'
  import { system as systemApi } from '../lib/api'
  import { token as jwtStore, addToast } from '../lib/stores'
  import type { SystemInfo } from '../lib/types'
  import StreamPlayer from '../lib/components/StreamPlayer.svelte'

  // Feedback plays the mediamtx paths directly (WHEP/HLS), same code path as
  // the Streams page. The old Janus mountpoint player was removed on
  // 2026-08-11: it required a GStreamer RTP bridge whose depay/pay pair had to
  // be kept in lockstep with the encoder codec, and it broke silently whenever
  // they drifted. IFB audio selection went with it — it will come back as a
  // separate audio-only source, not as extra tracks on this stream.
  interface Quality { label: string; stream: string }
  const QUALITIES: Quality[] = [
    { label: 'HD', stream: 'ch14' },
    { label: '480p', stream: 'ch14-lr' },
  ]

  // Auth guard
  const jwt = get(jwtStore)
  if (!jwt) {
    sessionStorage.setItem('loginReturn', '/ch14-live')
    push('/login')
  }

  let info = $state<SystemInfo | null>(null)
  let selectedQuality = $state(1) // start on 480p

  const streamName = $derived(QUALITIES[selectedQuality].stream)

  $effect(() => {
    if (!jwt || info) return
    systemApi.info()
      .then(i => info = i)
      .catch((e: any) => addToast('error', e.message))
  })
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-slate-900">CH14 Live</h1>

    <!-- Quality selector -->
    <div class="flex gap-1">
      {#each QUALITIES as q, i}
        <button
          onclick={() => selectedQuality = i}
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

  {#if info}
    <!-- Keyed so a quality change remounts the player on the new path. -->
    {#key streamName}
      <StreamPlayer {streamName} info={info} />
    {/key}
  {:else}
    <div class="relative bg-black rounded-xl overflow-hidden aspect-video">
      <div class="absolute inset-0 flex items-center justify-center">
        <div class="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
      </div>
    </div>
  {/if}
</div>
