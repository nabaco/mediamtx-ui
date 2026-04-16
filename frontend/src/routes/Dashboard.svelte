<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { token } from '../lib/stores'
  import { streams as streamsApi } from '../lib/api'
  import type { Stream } from '../lib/types'
  import StreamCard from '../lib/components/StreamCard.svelte'
  import { RefreshCw } from 'lucide-svelte'

  let streamList = $state<Stream[]>([])
  let loading = $state(true)
  let error = $state('')
  let search = $state('')

  const filtered = $derived(
    search
      ? streamList.filter(s =>
          s.name.toLowerCase().includes(search.toLowerCase()) ||
          s.description?.toLowerCase().includes(search.toLowerCase())
        )
      : streamList
  )

  async function load() {
    if (!$token) { push('/login'); return }
    loading = true
    error = ''
    try {
      streamList = await streamsApi.list()
    } catch (e: any) {
      error = e.message
    } finally {
      loading = false
    }
  }

  $effect(() => { load() })
</script>

<div class="flex flex-col gap-6">
  <!-- Header -->
  <div class="flex items-center justify-between gap-4 flex-wrap">
    <h1 class="text-2xl font-bold text-slate-900">{$_('dashboard.title')}</h1>
    <div class="flex items-center gap-3">
      <input
        class="input w-48"
        type="search"
        placeholder={$_('common.search')}
        bind:value={search}
      />
      <button onclick={load} class="btn-secondary p-2" title="Refresh">
        <RefreshCw class="w-4 h-4 {loading ? 'animate-spin' : ''}" />
      </button>
    </div>
  </div>

  {#if loading}
    <div class="flex justify-center py-16">
      <div class="w-8 h-8 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
    </div>
  {:else if error}
    <div class="card p-6 text-center text-red-600">
      <p>{error}</p>
      <button onclick={load} class="btn-secondary mt-4">{$_('common.loading')}</button>
    </div>
  {:else if filtered.length === 0}
    <div class="card p-12 text-center text-slate-400">
      {$_('dashboard.no_streams')}
    </div>
  {:else}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {#each filtered as stream (stream.name)}
        <StreamCard {stream} />
      {/each}
    </div>
  {/if}
</div>
