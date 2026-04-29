<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { streams as streamsApi } from '../../lib/api'
  import { addToast } from '../../lib/stores'
  import type { Stream } from '../../lib/types'
  import Modal from '../../lib/components/Modal.svelte'
  import { Plus, Trash2, RefreshCw, ChevronRight, ChevronDown, Loader, ExternalLink } from 'lucide-svelte'

  let list = $state<Stream[]>([])
  let loading = $state(true)
  let showAddForm = $state(false)
  let showDeleteConfirm = $state<string | null>(null)

  let addForm = $state({ name: '', description: '', source: '', sourceOnDemand: false, record: false, maxReaders: 0 })
  let addSaving = $state(false)

  // Inline edit
  let expandedName = $state<string | null>(null)
  let expandedForm = $state({ description: '', source: '', sourceOnDemand: false, record: false, maxReaders: 0 })
  let expandedLoading = $state(false)
  let expandedSaving = $state(false)

  async function load() {
    loading = true
    try { list = await streamsApi.list() }
    catch (e: any) { addToast('error', e.message) }
    finally { loading = false }
  }

  $effect(() => { load() })

  async function toggleExpand(name: string) {
    if (expandedName === name) {
      expandedName = null
      return
    }
    expandedName = name
    expandedLoading = true
    const meta = list.find(s => s.name === name)
    try {
      const cfg = await streamsApi.config(name)
      expandedForm = {
        description: meta?.description ?? '',
        source: cfg.source ?? '',
        sourceOnDemand: cfg.sourceOnDemand ?? false,
        record: cfg.record ?? false,
        maxReaders: cfg.maxReaders ?? 0,
      }
    } catch {
      // Stream may exist in live list but not have a persistent config entry yet
      expandedForm = {
        description: meta?.description ?? '',
        source: '',
        sourceOnDemand: false,
        record: false,
        maxReaders: 0,
      }
    } finally {
      expandedLoading = false
    }
  }

  async function saveExpanded() {
    if (!expandedName) return
    expandedSaving = true
    try {
      await streamsApi.update(expandedName, expandedForm)
      addToast('success', $_('streams.updated'))
      expandedName = null
      await load()
    } catch (e: any) {
      addToast('error', e.message)
    } finally {
      expandedSaving = false
    }
  }

  async function createStream() {
    addSaving = true
    try {
      await streamsApi.create(addForm)
      addToast('success', $_('streams.created'))
      showAddForm = false
      addForm = { name: '', description: '', source: '', sourceOnDemand: false, record: false, maxReaders: 0 }
      await load()
    } catch (e: any) {
      addToast('error', e.message)
    } finally {
      addSaving = false
    }
  }

  async function deleteStream(name: string) {
    try {
      await streamsApi.delete(name)
      addToast('success', $_('streams.deleted'))
      showDeleteConfirm = null
      if (expandedName === name) expandedName = null
      await load()
    } catch (e: any) {
      addToast('error', e.message)
    }
  }
</script>

<div class="flex flex-col gap-6">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-slate-900">{$_('nav.streams')}</h1>
    <div class="flex gap-2">
      <button onclick={load} class="btn-secondary p-2">
        <RefreshCw class="w-4 h-4 {loading ? 'animate-spin' : ''}" />
      </button>
      <button onclick={() => showAddForm = true} class="btn-primary">
        <Plus class="w-4 h-4" />
        {$_('streams.add')}
      </button>
    </div>
  </div>

  <div class="table-wrapper">
    <table>
      <thead>
        <tr>
          <th class="w-6"></th>
          <th>{$_('streams.name')}</th>
          <th>{$_('dashboard.status')}</th>
          <th>{$_('dashboard.readers')}</th>
          <th>{$_('streams.description')}</th>
          <th class="text-end rtl:text-start"></th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="6" class="text-center text-slate-400 py-8">{$_('common.loading')}</td></tr>
        {:else if list.length === 0}
          <tr><td colspan="6" class="text-center text-slate-400 py-8">{$_('dashboard.no_streams')}</td></tr>
        {:else}
          {#each list as s (s.name)}
            <tr
              class="cursor-pointer select-none transition-colors {expandedName === s.name ? 'bg-indigo-50 hover:bg-indigo-50' : 'hover:bg-slate-50'}"
              onclick={() => toggleExpand(s.name)}
            >
              <td class="pr-0 text-slate-400">
                {#if expandedName === s.name}
                  <ChevronDown class="w-3.5 h-3.5" />
                {:else}
                  <ChevronRight class="w-3.5 h-3.5" />
                {/if}
              </td>
              <td class="font-medium">
                <button
                  class="text-indigo-600 hover:underline font-medium"
                  onclick={(e) => { e.stopPropagation(); push('/streams/' + encodeURIComponent(s.name)) }}
                >
                  {s.name}
                </button>
              </td>
              <td>
                <span class="badge {s.ready ? 'badge-green' : 'badge-slate'}">
                  {s.ready ? $_('dashboard.active') : $_('dashboard.inactive')}
                </span>
              </td>
              <td>{s.readers}</td>
              <td class="text-slate-500 max-w-xs truncate">{s.description || '—'}</td>
              <td>
                <div class="flex items-center justify-end gap-1">
                  <button
                    onclick={(e) => { e.stopPropagation(); showDeleteConfirm = s.name }}
                    class="btn-ghost p-1.5 text-red-500 hover:text-red-700"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
            {#if expandedName === s.name}
              <tr class="bg-indigo-50 border-t-0">
                <td colspan="6" class="px-6 pb-4 pt-0">
                  {#if expandedLoading}
                    <div class="flex items-center gap-2 py-3 text-slate-400 text-sm">
                      <Loader class="w-4 h-4 animate-spin" />
                      {$_('common.loading')}
                    </div>
                  {:else}
                    <div class="flex flex-col gap-3 pt-1">
                      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                          <label class="label text-xs" for="ex-desc-{s.name}">{$_('streams.description')}</label>
                          <input id="ex-desc-{s.name}" class="input" bind:value={expandedForm.description} />
                        </div>
                        <div>
                          <label class="label text-xs" for="ex-src-{s.name}">{$_('streams.source')}</label>
                          <input id="ex-src-{s.name}" class="input" bind:value={expandedForm.source} placeholder={$_('streams.source_placeholder')} />
                        </div>
                        <div>
                          <label class="label text-xs" for="ex-max-{s.name}">{$_('streams.max_readers')}</label>
                          <input id="ex-max-{s.name}" class="input" type="number" min="0" bind:value={expandedForm.maxReaders} />
                        </div>
                        <div class="flex flex-col justify-end gap-1 pb-1">
                          <label class="flex items-center gap-2 text-xs cursor-pointer">
                            <input type="checkbox" bind:checked={expandedForm.sourceOnDemand} />
                            {$_('streams.on_demand')}
                          </label>
                          <label class="flex items-center gap-2 text-xs cursor-pointer">
                            <input type="checkbox" bind:checked={expandedForm.record} />
                            {$_('streams.record')}
                          </label>
                        </div>
                      </div>
                      <div class="flex gap-2 items-center">
                        <button onclick={saveExpanded} class="btn-primary py-1 px-3 text-sm" disabled={expandedSaving}>
                          {$_('common.save')}
                        </button>
                        <button onclick={() => expandedName = null} class="btn-secondary py-1 px-3 text-sm">
                          {$_('common.cancel')}
                        </button>
                        <button
                          onclick={() => push('/streams/' + encodeURIComponent(expandedName!))}
                          class="btn-ghost py-1 px-2 text-sm flex items-center gap-1.5 text-indigo-600 hover:text-indigo-800 ml-auto"
                          title={$_('dashboard.view')}
                        >
                          <ExternalLink class="w-3.5 h-3.5" />
                          {$_('dashboard.view')}
                        </button>
                      </div>
                    </div>
                  {/if}
                </td>
              </tr>
            {/if}
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

{#if showAddForm}
  <Modal title={$_('streams.add')} onclose={() => showAddForm = false}>
    {#snippet children()}
      <div class="flex flex-col gap-4">
        <div>
          <label class="label" for="stream-name">{$_('streams.name')}</label>
          <input id="stream-name" class="input" bind:value={addForm.name} placeholder="my-stream" required />
        </div>
        <div>
          <label class="label" for="stream-desc">{$_('streams.description')}</label>
          <input id="stream-desc" class="input" bind:value={addForm.description} />
        </div>
        <div>
          <label class="label" for="stream-source">{$_('streams.source')}</label>
          <input id="stream-source" class="input" bind:value={addForm.source} placeholder={$_('streams.source_placeholder')} />
        </div>
        <div>
          <label class="label" for="stream-maxreaders">{$_('streams.max_readers')}</label>
          <input id="stream-maxreaders" class="input" type="number" min="0" bind:value={addForm.maxReaders} />
        </div>
        <div class="flex flex-col gap-2">
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" bind:checked={addForm.sourceOnDemand} />
            {$_('streams.on_demand')}
          </label>
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" bind:checked={addForm.record} />
            {$_('streams.record')}
          </label>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showAddForm = false} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={createStream} class="btn-primary" disabled={addSaving}>{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}

{#if showDeleteConfirm}
  <Modal title={$_('streams.delete')} onclose={() => showDeleteConfirm = null}>
    {#snippet children()}
      <p class="text-slate-700">
        {$_('streams.delete_confirm', { values: { name: showDeleteConfirm } })}
      </p>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showDeleteConfirm = null} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={() => deleteStream(showDeleteConfirm!)} class="btn-danger">{$_('common.delete')}</button>
    {/snippet}
  </Modal>
{/if}
