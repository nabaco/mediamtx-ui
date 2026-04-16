<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { audit as auditApi } from '../../lib/api'
  import type { AuditEntry } from '../../lib/types'
  import { Search, ChevronLeft, ChevronRight } from 'lucide-svelte'

  const PAGE_SIZE = 50

  let entries = $state<AuditEntry[]>([])
  let total = $state(0)
  let page = $state(0)
  let loading = $state(true)
  let filterUser = $state('')
  let filterStream = $state('')

  const totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)))

  async function load() {
    loading = true
    try {
      const res = await auditApi.list({
        username: filterUser || undefined,
        stream: filterStream || undefined,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      })
      entries = res.entries ?? []
      total = res.total
    } catch {
      entries = []
    } finally {
      loading = false
    }
  }

  $effect(() => { load() })

  function applyFilter() { page = 0; load() }
  function clearFilter() { filterUser = ''; filterStream = ''; page = 0; load() }

  function formatDate(s: string) {
    return new Date(s).toLocaleString()
  }
</script>

<div class="flex flex-col gap-6">
  <h1 class="text-2xl font-bold text-slate-900">{$_('audit.title')}</h1>

  <!-- Filters -->
  <div class="card p-4 flex flex-wrap items-end gap-3">
    <div>
      <label class="label" for="audit-user">{$_('audit.username')}</label>
      <input id="audit-user" class="input w-40" bind:value={filterUser} onkeydown={e => e.key === 'Enter' && applyFilter()} />
    </div>
    <div>
      <label class="label" for="audit-stream">{$_('audit.stream')}</label>
      <input id="audit-stream" class="input w-40" bind:value={filterStream} onkeydown={e => e.key === 'Enter' && applyFilter()} />
    </div>
    <button onclick={applyFilter} class="btn-primary">
      <Search class="w-4 h-4" />{$_('audit.filter')}
    </button>
    <button onclick={clearFilter} class="btn-secondary">{$_('audit.clear')}</button>
    <span class="text-sm text-slate-500 ms-auto">{$_('common.total')}: {total}</span>
  </div>

  <div class="table-wrapper">
    <table>
      <thead>
        <tr>
          <th>{$_('audit.date')}</th>
          <th>{$_('audit.username')}</th>
          <th>{$_('audit.stream')}</th>
          <th>{$_('audit.action')}</th>
          <th>{$_('audit.protocol')}</th>
          <th>{$_('audit.ip')}</th>
          <th>{$_('audit.allowed')}</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="7" class="text-center text-slate-400 py-8">{$_('common.loading')}</td></tr>
        {:else if entries.length === 0}
          <tr><td colspan="7" class="text-center text-slate-400 py-8">{$_('audit.no_entries')}</td></tr>
        {:else}
          {#each entries as e (e.id)}
            <tr>
              <td class="text-xs text-slate-500 whitespace-nowrap">{formatDate(e.createdAt)}</td>
              <td class="font-medium">{e.username}</td>
              <td class="font-mono text-xs">{e.streamPath}</td>
              <td>{e.action}</td>
              <td>{e.protocol || '—'}</td>
              <td class="font-mono text-xs text-slate-500">{e.remoteAddr || '—'}</td>
              <td>
                <span class="badge {e.allowed ? 'badge-green' : 'badge-red'}">
                  {e.allowed ? $_('audit.allowed') : $_('audit.denied')}
                </span>
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>

  <!-- Pagination -->
  {#if totalPages > 1}
    <div class="flex items-center justify-center gap-2">
      <button onclick={() => { page--; load() }} disabled={page === 0} class="btn-secondary p-2">
        <ChevronLeft class="w-4 h-4" />
      </button>
      <span class="text-sm text-slate-600">{page + 1} {$_('common.of')} {totalPages}</span>
      <button onclick={() => { page++; load() }} disabled={page >= totalPages - 1} class="btn-secondary p-2">
        <ChevronRight class="w-4 h-4" />
      </button>
    </div>
  {/if}
</div>
