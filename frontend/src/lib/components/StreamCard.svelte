<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { link } from 'svelte-spa-router'
  import { Eye, Users, Radio, Copy, Check } from 'lucide-svelte'
  import type { Stream } from '../types'
  import { copyToClipboard } from '../utils'

  interface Props { stream: Stream }
  let { stream }: Props = $props()

  let copied = $state(false)

  async function copyLink() {
    const url = `#/stream/${encodeURIComponent(stream.name)}`
    await copyToClipboard(window.location.origin + window.location.pathname + url)
    copied = true
    setTimeout(() => copied = false, 2000)
  }
</script>

<div class="card p-4 flex flex-col gap-3 hover:shadow-md transition-shadow">
  <!-- Header -->
  <div class="flex items-start justify-between gap-2">
    <div class="flex items-center gap-2 min-w-0">
      <div class="w-2 h-2 rounded-full shrink-0 {stream.ready ? 'bg-green-500' : 'bg-slate-300'}"></div>
      <h3 class="font-medium text-slate-900 truncate">{stream.name}</h3>
    </div>
    <span class="badge {stream.ready ? 'badge-green' : 'badge-slate'} shrink-0">
      {stream.ready ? $_('dashboard.active') : $_('dashboard.inactive')}
    </span>
  </div>

  {#if stream.description}
    <p class="text-sm text-slate-500 line-clamp-2">{stream.description}</p>
  {/if}

  <!-- Stats -->
  {#if stream.ready}
    <div class="flex items-center gap-4 text-xs text-slate-500">
      <span class="flex items-center gap-1">
        <Users class="w-3.5 h-3.5" />
        {stream.readers} {$_('dashboard.readers')}
      </span>
      {#if stream.tracks?.length}
        <span class="flex items-center gap-1">
          <Radio class="w-3.5 h-3.5" />
          {stream.tracks.join(', ')}
        </span>
      {/if}
    </div>
  {/if}

  <!-- Actions -->
  <div class="flex items-center gap-2 mt-auto pt-2 border-t border-slate-100">
    <a href="/stream/{encodeURIComponent(stream.name)}" use:link class="btn-primary flex-1 text-center text-xs py-1.5">
      <Eye class="w-3.5 h-3.5" />
      {$_('dashboard.view')}
    </a>
    <button onclick={copyLink} class="btn-secondary px-2.5 py-1.5" title={$_('dashboard.copy_url')}>
      {#if copied}
        <Check class="w-4 h-4 text-green-600" />
      {:else}
        <Copy class="w-4 h-4" />
      {/if}
    </button>
  </div>
</div>
