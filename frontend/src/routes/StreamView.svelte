<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { streams as streamsApi, system as systemApi } from '../lib/api'
  import { addToast, token } from '../lib/stores'
  import type { Stream, StreamURLs, SystemInfo } from '../lib/types'
  import StreamPlayer from '../lib/components/StreamPlayer.svelte'
  import { Copy, Check, ArrowLeft } from 'lucide-svelte'
  import { copyToClipboard } from '../lib/utils'

  interface Props { params: { name: string } }
  let { params }: Props = $props()

  const name = $derived(decodeURIComponent(params.name ?? ''))

  let stream = $state<Stream | null>(null)
  let urls = $state<StreamURLs | null>(null)
  let info = $state<SystemInfo | null>(null)
  let loading = $state(true)
  let copiedKey = $state<string | null>(null)

  async function load() {
    if (!$token) { push('/login'); return }
    loading = true
    try {
      ;[stream, urls, info] = await Promise.all([
        streamsApi.get(name),
        streamsApi.urls(name),
        systemApi.info(),
      ])
    } catch (e: any) {
      addToast('error', e.message)
    } finally {
      loading = false
    }
  }

  $effect(() => { if (name) load() })

  async function copyURL(key: string, value: string) {
    await copyToClipboard(value)
    copiedKey = key
    setTimeout(() => copiedKey = null, 2000)
  }

  const urlEntries = $derived(urls ? [
    { key: 'rtsp', label: $_('stream_urls.rtsp'), value: urls.rtsp },
    { key: 'hls', label: $_('stream_urls.hls'), value: urls.hls },
    { key: 'webrtc', label: $_('stream_urls.webrtc'), value: urls.webrtc },
    { key: 'rtmp', label: $_('stream_urls.rtmp'), value: urls.rtmp },
  ].filter(e => e.value) : [])

  const publishEntries = $derived(urls?.isPublishStream ? [
    { key: 'pub_rtsp', label: $_('stream_urls.publish_rtsp'), value: urls.rtsp },
    { key: 'pub_rtmp', label: $_('stream_urls.publish_rtmp'), value: urls.publishRtmp ?? urls.rtmp },
    { key: 'pub_srt',  label: $_('stream_urls.publish_srt'),  value: urls.srt ?? '' },
  ].filter(e => e.value) : [])
</script>

<div class="flex flex-col gap-6">
  <!-- Back + title -->
  <div class="flex items-center gap-3">
    <button onclick={() => history.back()} class="btn-ghost p-2">
      <ArrowLeft class="w-4 h-4" />
    </button>
    <div class="flex items-center gap-3">
      <div class="w-2.5 h-2.5 rounded-full {stream?.ready ? 'bg-green-500' : 'bg-slate-300'}"></div>
      <h1 class="text-xl font-bold text-slate-900">{name}</h1>
      {#if stream?.ready}
        <span class="badge badge-green">{$_('dashboard.active')}</span>
      {:else if stream}
        <span class="badge badge-slate">{$_('dashboard.inactive')}</span>
      {/if}
    </div>
  </div>

  {#if loading}
    <div class="flex justify-center py-16">
      <div class="w-8 h-8 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
    </div>
  {:else if stream && info}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Player -->
      <div class="lg:col-span-2">
        {#if stream.ready}
          <StreamPlayer streamName={name} {info} username={urls?.username} streamToken={urls?.streamToken} />
        {:else}
          <div class="bg-black rounded-xl aspect-video flex items-center justify-center">
            <p class="text-white/60 text-sm">{$_('player.not_ready')}</p>
          </div>
        {/if}
      </div>

      <!-- Sidebar -->
      <div class="flex flex-col gap-4">
        <!-- Stream info -->
        <div class="card p-4 flex flex-col gap-2">
          {#if stream.description}
            <p class="text-sm text-slate-600">{stream.description}</p>
          {/if}
          <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
            <span class="text-slate-500">{$_('dashboard.readers')}</span>
            <span class="font-medium">{stream.readers}</span>
            {#if stream.tracks?.length}
              <span class="text-slate-500">{$_('dashboard.tracks')}</span>
              <span class="font-medium">{stream.tracks.join(', ')}</span>
            {/if}
            <span class="text-slate-500">Source</span>
            <span class="font-medium text-slate-600 truncate">{stream.source || '—'}</span>
          </div>
        </div>

        <!-- Publish URLs (publish streams only) -->
        {#if publishEntries.length > 0}
          <div class="card p-4 border-indigo-100">
            <h3 class="text-sm font-semibold text-slate-700 mb-1">{$_('stream_urls.publish_title')}</h3>
            <p class="text-xs text-slate-400 mb-3">{$_('stream_urls.publish_hint')}</p>
            <div class="flex flex-col gap-2">
              {#each publishEntries as entry (entry.key)}
                <div>
                  <p class="text-xs text-slate-400 mb-1">{entry.label}</p>
                  <div class="flex items-center gap-1.5">
                    <code class="flex-1 text-xs bg-indigo-50 border border-indigo-100 rounded px-2 py-1.5 truncate font-mono">
                      {entry.value}
                    </code>
                    <button
                      onclick={() => copyURL(entry.key, entry.value)}
                      class="btn-secondary p-1.5 shrink-0"
                      title={$_('common.copy')}
                    >
                      {#if copiedKey === entry.key}
                        <Check class="w-3.5 h-3.5 text-green-600" />
                      {:else}
                        <Copy class="w-3.5 h-3.5" />
                      {/if}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Stream URLs -->
        <div class="card p-4">
          <h3 class="text-sm font-semibold text-slate-700 mb-3">{$_('stream_urls.title')}</h3>
          <div class="flex flex-col gap-2">
            {#each urlEntries as entry (entry.key)}
              <div>
                <p class="text-xs text-slate-400 mb-1">{entry.label}</p>
                <div class="flex items-center gap-1.5">
                  <code class="flex-1 text-xs bg-slate-50 border border-slate-200 rounded px-2 py-1.5 truncate font-mono">
                    {entry.value}
                  </code>
                  <button
                    onclick={() => copyURL(entry.key, entry.value)}
                    class="btn-secondary p-1.5 shrink-0"
                    title={$_('common.copy')}
                  >
                    {#if copiedKey === entry.key}
                      <Check class="w-3.5 h-3.5 text-green-600" />
                    {:else}
                      <Copy class="w-3.5 h-3.5" />
                    {/if}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>
