<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { streams as streamsApi, users as usersApi, groups as groupsApi, acls as aclsApi } from '../../lib/api'
  import { addToast } from '../../lib/stores'
  import type { Stream, User, Group } from '../../lib/types'
  import Modal from '../../lib/components/Modal.svelte'
  import { Plus, Trash2, RefreshCw, ChevronRight, ChevronDown, Loader, ExternalLink, Copy, Check, Link, UserPlus } from 'lucide-svelte'
  import { copyToClipboard } from '../../lib/utils'

  let list = $state<Stream[]>([])
  let loading = $state(true)
  let showAddForm = $state(false)
  let showDeleteConfirm = $state<string | null>(null)

  let addForm = $state({ name: '', description: '', source: '', isPublisher: false, sourceOnDemand: false, record: false, maxReaders: 0 })
  let addSaving = $state(false)

  // Inline edit
  let expandedName = $state<string | null>(null)
  let expandedForm = $state({ description: '', source: '', isPublisher: false, sourceOnDemand: false, record: false, maxReaders: 0 })
  let expandedLoading = $state(false)
  let expandedSaving = $state(false)

  // Copy URLs popover
  let urlsPopover = $state<{ name: string; urls: { label: string; value: string }[] } | null>(null)
  let urlsLoading = $state(false)
  let copiedKey = $state<string | null>(null)

  // Assign access modal
  let assignModal = $state<{ streamName: string } | null>(null)
  let userList = $state<User[]>([])
  let groupList = $state<Group[]>([])
  let assignForm = $state({ subjectType: 'user', subjectId: 0, action: 'read' })
  let assignSaving = $state(false)

  // Context menu
  let ctxMenu = $state<{ x: number; y: number; streamName: string } | null>(null)

  async function load() {
    loading = true
    try { list = await streamsApi.list() }
    catch (e: any) { addToast('error', e.message) }
    finally { loading = false }
  }

  $effect(() => { load() })

  async function toggleExpand(name: string) {
    if (expandedName === name) { expandedName = null; return }
    expandedName = name
    expandedLoading = true
    const meta = list.find(s => s.name === name)
    try {
      const cfg = await streamsApi.config(name)
      const src = cfg.source ?? ''
      expandedForm = {
        description: meta?.description ?? '',
        source: src === 'publisher' ? '' : src,
        isPublisher: !src || src === 'publisher',
        sourceOnDemand: cfg.sourceOnDemand ?? false,
        record: cfg.record ?? false,
        maxReaders: cfg.maxReaders ?? 0,
      }
    } catch {
      expandedForm = { description: meta?.description ?? '', source: '', isPublisher: true, sourceOnDemand: false, record: false, maxReaders: 0 }
    } finally {
      expandedLoading = false
    }
  }

  async function saveExpanded() {
    if (!expandedName) return
    expandedSaving = true
    try {
      const { isPublisher, ...fields } = expandedForm
      await streamsApi.update(expandedName, { ...fields, source: isPublisher ? 'publisher' : fields.source })
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
      const { isPublisher, ...fields } = addForm
      await streamsApi.create({ ...fields, source: isPublisher ? 'publisher' : fields.source })
      addToast('success', $_('streams.created'))
      showAddForm = false
      addForm = { name: '', description: '', source: '', isPublisher: false, sourceOnDemand: false, record: false, maxReaders: 0 }
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

  async function showURLs(name: string) {
    ctxMenu = null
    urlsLoading = true
    urlsPopover = { name, urls: [] }
    try {
      const u = await streamsApi.urls(name)
      const entries: { label: string; value: string }[] = []
      if (u.rtsp) entries.push({ label: 'RTSP', value: u.rtsp })
      if (u.hls) entries.push({ label: 'HLS', value: u.hls })
      if (u.webrtc) entries.push({ label: 'WebRTC', value: u.webrtc })
      if (u.rtmp) entries.push({ label: 'RTMP', value: u.rtmp })
      urlsPopover = { name, urls: entries }
    } catch (e: any) {
      addToast('error', e.message)
      urlsPopover = null
    } finally {
      urlsLoading = false
    }
  }

  async function openAssign(name: string) {
    ctxMenu = null
    assignModal = { streamName: name }
    assignForm = { subjectType: 'user', subjectId: 0, action: 'read' }
    const [u, g] = await Promise.all([usersApi.list(), groupsApi.list()])
    userList = u
    groupList = g
    if (u.length) assignForm.subjectId = u[0].id
  }

  async function saveAssign() {
    if (!assignModal) return
    assignSaving = true
    try {
      await aclsApi.create({
        subjectType: assignForm.subjectType,
        subjectId: assignForm.subjectId,
        streamPattern: assignModal.streamName,
        action: assignForm.action,
      })
      addToast('success', $_('acls.created'))
      assignModal = null
    } catch (e: any) {
      addToast('error', e.message)
    } finally {
      assignSaving = false
    }
  }

  function openCtxMenu(e: MouseEvent, name: string) {
    e.preventDefault()
    ctxMenu = { x: e.clientX, y: e.clientY, streamName: name }
  }

  function closeCtxMenu() { ctxMenu = null }

  async function copyURL(key: string, value: string) {
    await copyToClipboard(value)
    copiedKey = key
    setTimeout(() => copiedKey = null, 2000)
  }
</script>

<svelte:window onclick={closeCtxMenu} onkeydown={(e) => { if (e.key === 'Escape') { ctxMenu = null; urlsPopover = null } }} />

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
              oncontextmenu={(e) => openCtxMenu(e, s.name)}
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
                  onclick={(e) => { e.stopPropagation(); push('/stream/' + encodeURIComponent(s.name)) }}
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
                <div class="flex items-center justify-end gap-0.5" onclick={(e) => e.stopPropagation()}>
                  <button
                    onclick={() => push('/stream/' + encodeURIComponent(s.name))}
                    class="btn-ghost p-1.5" title={$_('dashboard.view')}
                  ><ExternalLink class="w-3.5 h-3.5" /></button>
                  <button
                    onclick={() => showURLs(s.name)}
                    class="btn-ghost p-1.5" title={$_('streams.copy_urls')}
                  ><Link class="w-3.5 h-3.5" /></button>
                  <button
                    onclick={() => openAssign(s.name)}
                    class="btn-ghost p-1.5" title={$_('streams.assign_access')}
                  ><UserPlus class="w-3.5 h-3.5" /></button>
                  <button
                    onclick={() => showDeleteConfirm = s.name}
                    class="btn-ghost p-1.5 text-red-500 hover:text-red-700"
                  ><Trash2 class="w-3.5 h-3.5" /></button>
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
                          <div class="flex items-center justify-between mb-1">
                            <label class="label text-xs mb-0" for="ex-src-{s.name}">{$_('streams.source')}</label>
                            <label class="flex items-center gap-1.5 text-xs cursor-pointer text-slate-500">
                              <input type="checkbox" bind:checked={expandedForm.isPublisher}
                                onchange={() => { if (expandedForm.isPublisher) expandedForm.source = '' }} />
                              {$_('streams.publisher')}
                            </label>
                          </div>
                          <input id="ex-src-{s.name}" class="input" bind:value={expandedForm.source}
                            placeholder={$_('streams.source_placeholder')}
                            disabled={expandedForm.isPublisher} />
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

<!-- Context menu -->
{#if ctxMenu}
  <div
    class="fixed z-50 bg-white border border-slate-200 rounded-lg shadow-lg py-1 min-w-40"
    style="left:{ctxMenu.x}px; top:{ctxMenu.y}px"
    onclick={(e) => e.stopPropagation()}
  >
    <button onclick={() => push('/stream/' + encodeURIComponent(ctxMenu!.streamName))}
      class="w-full text-start px-4 py-2 text-sm hover:bg-slate-50 flex items-center gap-2">
      <ExternalLink class="w-3.5 h-3.5 text-slate-400" />{$_('dashboard.view')}
    </button>
    <button onclick={() => showURLs(ctxMenu!.streamName)}
      class="w-full text-start px-4 py-2 text-sm hover:bg-slate-50 flex items-center gap-2">
      <Link class="w-3.5 h-3.5 text-slate-400" />{$_('streams.copy_urls')}
    </button>
    <button onclick={() => openAssign(ctxMenu!.streamName)}
      class="w-full text-start px-4 py-2 text-sm hover:bg-slate-50 flex items-center gap-2">
      <UserPlus class="w-3.5 h-3.5 text-slate-400" />{$_('streams.assign_access')}
    </button>
    <div class="border-t border-slate-100 my-1"></div>
    <button onclick={() => { showDeleteConfirm = ctxMenu!.streamName; ctxMenu = null }}
      class="w-full text-start px-4 py-2 text-sm hover:bg-red-50 text-red-600 flex items-center gap-2">
      <Trash2 class="w-3.5 h-3.5" />{$_('streams.delete')}
    </button>
  </div>
{/if}

<!-- URL copy panel -->
{#if urlsPopover}
  <div class="fixed inset-0 z-40 flex items-center justify-center bg-black/30" onclick={() => urlsPopover = null}>
    <div class="bg-white rounded-xl shadow-xl p-5 w-full max-w-md mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="flex items-center justify-between mb-3">
        <h3 class="font-semibold text-slate-900">{urlsPopover.name} — {$_('stream_urls.title')}</h3>
        <button onclick={() => urlsPopover = null} class="btn-ghost p-1 text-slate-400">✕</button>
      </div>
      {#if urlsLoading}
        <div class="flex justify-center py-6">
          <div class="w-6 h-6 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
        </div>
      {:else}
        <div class="flex flex-col gap-2">
          {#each urlsPopover.urls as entry (entry.label)}
            <div>
              <p class="text-xs text-slate-400 mb-1">{entry.label}</p>
              <div class="flex items-center gap-1.5">
                <code class="flex-1 text-xs bg-slate-50 border border-slate-200 rounded px-2 py-1.5 truncate font-mono">{entry.value}</code>
                <button
                  onclick={() => copyURL(entry.label, entry.value)}
                  class="btn-secondary p-1.5 shrink-0"
                >
                  {#if copiedKey === entry.label}
                    <Check class="w-3.5 h-3.5 text-green-600" />
                  {:else}
                    <Copy class="w-3.5 h-3.5" />
                  {/if}
                </button>
              </div>
            </div>
          {/each}
          <p class="text-xs text-slate-400 mt-1">{$_('streams.auth_note')}</p>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Assign access modal -->
{#if assignModal}
  <Modal title="{$_('streams.assign_access')}: {assignModal.streamName}" onclose={() => assignModal = null}>
    {#snippet children()}
      <div class="flex flex-col gap-4">
        <div>
          <label class="label" for="assign-type">{$_('acls.subject_type')}</label>
          <select id="assign-type" class="input" bind:value={assignForm.subjectType}
            onchange={() => {
              assignForm.subjectId = assignForm.subjectType === 'user'
                ? (userList[0]?.id ?? 0)
                : (groupList[0]?.id ?? 0)
            }}>
            <option value="user">{$_('acls.type_user')}</option>
            <option value="group">{$_('acls.type_group')}</option>
          </select>
        </div>
        <div>
          <label class="label" for="assign-subject">{$_('acls.subject')}</label>
          <select id="assign-subject" class="input" bind:value={assignForm.subjectId}>
            {#if assignForm.subjectType === 'user'}
              {#each userList as u (u.id)}<option value={u.id}>{u.username}</option>{/each}
            {:else}
              {#each groupList as g (g.id)}<option value={g.id}>{g.name}</option>{/each}
            {/if}
          </select>
        </div>
        <div>
          <label class="label" for="assign-action">{$_('acls.action')}</label>
          <select id="assign-action" class="input" bind:value={assignForm.action}>
            <option value="read">{$_('acls.action_read')}</option>
            <option value="publish">{$_('acls.action_publish')}</option>
            <option value="both">{$_('acls.action_both')}</option>
          </select>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => assignModal = null} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={saveAssign} class="btn-primary" disabled={assignSaving}>{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}

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
          <div class="flex items-center justify-between mb-1">
            <label class="label mb-0" for="stream-source">{$_('streams.source')}</label>
            <label class="flex items-center gap-1.5 text-xs cursor-pointer text-slate-500">
              <input type="checkbox" bind:checked={addForm.isPublisher}
                onchange={() => { if (addForm.isPublisher) addForm.source = '' }} />
              {$_('streams.publisher')}
            </label>
          </div>
          <input id="stream-source" class="input" bind:value={addForm.source}
            placeholder={$_('streams.source_placeholder')}
            disabled={addForm.isPublisher} />
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
