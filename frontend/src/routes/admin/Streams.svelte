<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { streams as streamsApi } from '../../lib/api'
  import { addToast } from '../../lib/stores'
  import type { Stream } from '../../lib/types'
  import Modal from '../../lib/components/Modal.svelte'
  import { Plus, Pencil, Trash2, RefreshCw } from 'lucide-svelte'

  let list = $state<Stream[]>([])
  let loading = $state(true)
  let showForm = $state(false)
  let showDeleteConfirm = $state<string | null>(null)

  let form = $state({ name: '', description: '', source: '', sourceOnDemand: false, record: false, maxReaders: 0 })
  let editName = $state<string | null>(null)
  let saving = $state(false)

  async function load() {
    loading = true
    try { list = await streamsApi.list() }
    catch (e: any) { addToast('error', e.message) }
    finally { loading = false }
  }

  $effect(() => { load() })

  function openCreate() {
    editName = null
    form = { name: '', description: '', source: '', sourceOnDemand: false, record: false, maxReaders: 0 }
    showForm = true
  }

  function openEdit(s: Stream) {
    editName = s.name
    form = { name: s.name, description: s.description ?? '', source: s.source ?? '', sourceOnDemand: false, record: false, maxReaders: 0 }
    showForm = true
  }

  async function save() {
    saving = true
    try {
      if (editName) {
        await streamsApi.update(editName, { description: form.description, source: form.source, sourceOnDemand: form.sourceOnDemand, record: form.record, maxReaders: form.maxReaders })
        addToast('success', $_('streams.updated'))
      } else {
        await streamsApi.create(form)
        addToast('success', $_('streams.created'))
      }
      showForm = false
      await load()
    } catch (e: any) {
      addToast('error', e.message)
    } finally {
      saving = false
    }
  }

  async function deleteStream(name: string) {
    try {
      await streamsApi.delete(name)
      addToast('success', $_('streams.deleted'))
      showDeleteConfirm = null
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
      <button onclick={openCreate} class="btn-primary">
        <Plus class="w-4 h-4" />
        {$_('streams.add')}
      </button>
    </div>
  </div>

  <div class="table-wrapper">
    <table>
      <thead>
        <tr>
          <th>{$_('streams.name')}</th>
          <th>{$_('dashboard.status')}</th>
          <th>{$_('dashboard.readers')}</th>
          <th>{$_('streams.description')}</th>
          <th class="text-end rtl:text-start"></th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="5" class="text-center text-slate-400 py-8">{$_('common.loading')}</td></tr>
        {:else if list.length === 0}
          <tr><td colspan="5" class="text-center text-slate-400 py-8">{$_('dashboard.no_streams')}</td></tr>
        {:else}
          {#each list as s (s.name)}
            <tr>
              <td class="font-medium">{s.name}</td>
              <td>
                <span class="badge {s.ready ? 'badge-green' : 'badge-slate'}">
                  {s.ready ? $_('dashboard.active') : $_('dashboard.inactive')}
                </span>
              </td>
              <td>{s.readers}</td>
              <td class="text-slate-500 max-w-xs truncate">{s.description || '—'}</td>
              <td>
                <div class="flex items-center justify-end gap-1">
                  <button onclick={() => openEdit(s)} class="btn-ghost p-1.5"><Pencil class="w-3.5 h-3.5" /></button>
                  <button onclick={() => showDeleteConfirm = s.name} class="btn-ghost p-1.5 text-red-500 hover:text-red-700"><Trash2 class="w-3.5 h-3.5" /></button>
                </div>
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

{#if showForm}
  <Modal title={editName ? $_('streams.edit') : $_('streams.add')} onclose={() => showForm = false}>
    {#snippet children()}
      <div class="flex flex-col gap-4">
        {#if !editName}
          <div>
            <label class="label" for="stream-name">{$_('streams.name')}</label>
            <input id="stream-name" class="input" bind:value={form.name} placeholder="my-stream" required />
          </div>
        {/if}
        <div>
          <label class="label" for="stream-desc">{$_('streams.description')}</label>
          <input id="stream-desc" class="input" bind:value={form.description} />
        </div>
        <div>
          <label class="label" for="stream-source">{$_('streams.source')}</label>
          <input id="stream-source" class="input" bind:value={form.source} placeholder={$_('streams.source_placeholder')} />
        </div>
        <div>
          <label class="label" for="stream-maxreaders">{$_('streams.max_readers')}</label>
          <input id="stream-maxreaders" class="input" type="number" min="0" bind:value={form.maxReaders} />
        </div>
        <div class="flex flex-col gap-2">
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" bind:checked={form.sourceOnDemand} />
            {$_('streams.on_demand')}
          </label>
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" bind:checked={form.record} />
            {$_('streams.record')}
          </label>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showForm = false} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={save} class="btn-primary" disabled={saving}>{$_('common.save')}</button>
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
