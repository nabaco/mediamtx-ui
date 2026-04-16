<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { users as usersApi, groups as groupsApi, acls as aclsApi } from '../../lib/api'
  import { addToast } from '../../lib/stores'
  import type { User, Group, ACL } from '../../lib/types'
  import Modal from '../../lib/components/Modal.svelte'
  import { Plus, Trash2, RefreshCw, KeyRound, Copy, Check, Shield, Pencil, UserPlus, ChevronDown, ChevronRight } from 'lucide-svelte'
  import { copyToClipboard } from '../../lib/utils'

  let activeTab = $state<'users' | 'groups' | 'acls'>('users')

  // ---- Users ----
  let userList = $state<User[]>([])
  let showUserForm = $state(false)
  let showTokenModal = $state<{ username: string; token: string } | null>(null)
  let showResetPwModal = $state<User | null>(null)
  let userForm = $state({ username: '', password: '', role: 'user', groupId: 0 })
  let resetPwValue = $state('')
  let savingUser = $state(false)
  let tokenCopied = $state(false)

  // ---- Groups ----
  let groupList = $state<Group[]>([])
  let showGroupForm = $state(false)
  let groupName = $state('')
  let expandedGroup = $state<number | null>(null)
  let groupMembers = $state<Record<number, User[]>>({})
  let renamingGroup = $state<number | null>(null)
  let renameValue = $state('')
  let pendingMemberChecks = $state<Record<number, Record<number, boolean>>>({})
  let newPermForm = $state<Record<number, { streamPattern: string; action: string }>>({})

  // ---- ACLs ----
  let aclList = $state<ACL[]>([])
  let showACLForm = $state(false)
  let aclForm = $state({ subjectType: 'user', subjectId: 0, streamPattern: '', action: 'read' })

  async function loadAll() {
    const [u, g, a] = await Promise.all([usersApi.list(), groupsApi.list(), aclsApi.list()])
    userList = u
    groupList = g
    aclList = a
  }

  $effect(() => { loadAll().catch((e: any) => addToast('error', e.message)) })

  // Users
  async function createUser() {
    savingUser = true
    try {
      const newUser = await usersApi.create({ username: userForm.username, password: userForm.password, role: userForm.role })
      if (userForm.groupId && newUser?.id) {
        await groupsApi.addMember(userForm.groupId, newUser.id)
      }
      addToast('success', $_('users.created'))
      showUserForm = false
      userList = await usersApi.list()
    } catch (e: any) { addToast('error', e.message) }
    finally { savingUser = false }
  }

  async function toggleUser(u: User) {
    try {
      await usersApi.update(u.id, { enabled: !u.enabled })
      userList = await usersApi.list()
    } catch (e: any) { addToast('error', e.message) }
  }

  async function deleteUser(id: number) {
    try {
      await usersApi.delete(id)
      addToast('success', $_('users.deleted'))
      userList = await usersApi.list()
    } catch (e: any) { addToast('error', e.message) }
  }

  async function regenToken(u: User) {
    try {
      const res = await usersApi.regenerateToken(u.id)
      showTokenModal = { username: u.username, token: res.token }
      userList = await usersApi.list()
    } catch (e: any) { addToast('error', e.message) }
  }

  async function resetPassword() {
    if (!showResetPwModal || !resetPwValue) return
    try {
      await usersApi.update(showResetPwModal.id, { password: resetPwValue })
      addToast('success', $_('users.password_reset'))
      showResetPwModal = null
      resetPwValue = ''
    } catch (e: any) { addToast('error', e.message) }
  }

  // Groups
  async function createGroup() {
    try {
      await groupsApi.create(groupName)
      groupName = ''
      showGroupForm = false
      groupList = await groupsApi.list()
      addToast('success', $_('groups.created'))
    } catch (e: any) { addToast('error', e.message) }
  }

  async function deleteGroup(id: number) {
    try {
      await groupsApi.delete(id)
      if (expandedGroup === id) expandedGroup = null
      groupList = await groupsApi.list()
      addToast('success', $_('groups.deleted'))
    } catch (e: any) { addToast('error', e.message) }
  }

  async function expandGroup(id: number) {
    if (expandedGroup === id) { expandedGroup = null; return }
    expandedGroup = id
    if (!newPermForm[id]) newPermForm[id] = { streamPattern: '', action: 'read' }
    if (!pendingMemberChecks[id]) pendingMemberChecks[id] = {}
    try {
      groupMembers[id] = await groupsApi.members(id)
    } catch (e: any) { addToast('error', e.message) }
  }

  async function removeFromGroup(groupId: number, userId: number) {
    try {
      await groupsApi.removeMember(groupId, userId)
      groupMembers[groupId] = await groupsApi.members(groupId)
    } catch (e: any) { addToast('error', e.message) }
  }

  function togglePendingMember(groupId: number, userId: number, checked: boolean) {
    if (!pendingMemberChecks[groupId]) pendingMemberChecks[groupId] = {}
    pendingMemberChecks[groupId][userId] = checked
  }

  function pendingCount(groupId: number): number {
    return Object.values(pendingMemberChecks[groupId] ?? {}).filter(Boolean).length
  }

  function nonMembers(groupId: number): User[] {
    return userList.filter(u => !groupMembers[groupId]?.find(m => m.id === u.id))
  }

  async function addCheckedMembers(groupId: number) {
    const checks = pendingMemberChecks[groupId] ?? {}
    const ids = Object.entries(checks).filter(([, v]) => v).map(([k]) => Number(k))
    if (!ids.length) return
    try {
      await Promise.all(ids.map(uid => groupsApi.addMember(groupId, uid)))
      groupMembers[groupId] = await groupsApi.members(groupId)
      pendingMemberChecks[groupId] = {}
    } catch (e: any) { addToast('error', e.message) }
  }

  function groupACLs(groupId: number): ACL[] {
    return aclList.filter(a => a.subjectType === 'group' && a.subjectId === groupId)
  }

  async function addGroupACL(groupId: number) {
    const form = newPermForm[groupId]
    if (!form?.streamPattern.trim()) return
    try {
      await aclsApi.create({ subjectType: 'group', subjectId: groupId, streamPattern: form.streamPattern.trim(), action: form.action })
      aclList = await aclsApi.list()
      newPermForm[groupId] = { streamPattern: '', action: 'read' }
      addToast('success', $_('acls.created'))
    } catch (e: any) { addToast('error', e.message) }
  }

  function focusEl(el: HTMLElement) { el.focus() }

  function startRename(g: Group) {
    renamingGroup = g.id
    renameValue = g.name
  }

  async function saveRename(id: number) {
    if (!renameValue.trim()) { renamingGroup = null; return }
    try {
      const updated = await groupsApi.rename(id, renameValue.trim())
      groupList = groupList.map(g => g.id === id ? { ...g, name: updated.name } : g)
      addToast('success', $_('groups.renamed'))
    } catch (e: any) { addToast('error', e.message) }
    finally { renamingGroup = null }
  }

  // ACLs
  async function createACL() {
    try {
      await aclsApi.create(aclForm)
      aclList = await aclsApi.list()
      showACLForm = false
      addToast('success', $_('acls.created'))
    } catch (e: any) { addToast('error', e.message) }
  }

  async function deleteACL(id: number) {
    try {
      await aclsApi.delete(id)
      aclList = await aclsApi.list()
      addToast('success', $_('acls.deleted'))
    } catch (e: any) { addToast('error', e.message) }
  }
</script>

<div class="flex flex-col gap-6">
  <h1 class="text-2xl font-bold text-slate-900">{$_('users.title')}</h1>

  <!-- Tabs -->
  <div class="flex gap-1 border-b border-slate-200">
    {#each [['users', $_('nav.users')], ['groups', $_('nav.groups')], ['acls', $_('nav.acls')]] as [tab, label]}
      <button
        onclick={() => activeTab = tab as any}
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
               {activeTab === tab ? 'border-indigo-600 text-indigo-700' : 'border-transparent text-slate-500 hover:text-slate-700'}"
      >{label}</button>
    {/each}
  </div>

  <!-- Users tab -->
  {#if activeTab === 'users'}
    <div class="flex justify-end">
      <button onclick={() => { userForm = { username: '', password: '', role: 'user', groupId: 0 }; showUserForm = true }} class="btn-primary">
        <Plus class="w-4 h-4" />{$_('users.add')}
      </button>
    </div>
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th>{$_('users.username')}</th>
            <th>{$_('users.role')}</th>
            <th>{$_('common.status') ?? 'Status'}</th>
            <th>Token</th>
            <th class="text-end rtl:text-start"></th>
          </tr>
        </thead>
        <tbody>
          {#each userList as u (u.id)}
            <tr>
              <td class="font-medium">{u.username}</td>
              <td>
                <span class="badge {u.role === 'admin' ? 'badge-indigo' : 'badge-slate'}">
                  {u.role === 'admin' ? $_('users.role_admin') : $_('users.role_user')}
                </span>
              </td>
              <td>
                <button onclick={() => toggleUser(u)} class="badge {u.enabled ? 'badge-green' : 'badge-red'} cursor-pointer">
                  {u.enabled ? $_('users.enabled') : $_('users.disabled')}
                </button>
              </td>
              <td>
                <span class="text-xs {u.hasToken ? 'text-green-600' : 'text-slate-400'}">
                  {u.hasToken ? $_('users.has_token') : $_('users.no_token')}
                </span>
              </td>
              <td>
                <div class="flex items-center justify-end gap-1">
                  <button onclick={() => regenToken(u)} class="btn-ghost p-1.5" title={$_('users.regen_token')}>
                    <KeyRound class="w-3.5 h-3.5" />
                  </button>
                  <button onclick={() => { showResetPwModal = u; resetPwValue = '' }} class="btn-ghost p-1.5" title={$_('users.reset_password')}>
                    <Shield class="w-3.5 h-3.5" />
                  </button>
                  <button onclick={() => deleteUser(u.id)} class="btn-ghost p-1.5 text-red-500">
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Groups tab -->
  {#if activeTab === 'groups'}
    <div class="flex justify-end">
      <button onclick={() => { groupName = ''; showGroupForm = true }} class="btn-primary">
        <Plus class="w-4 h-4" />{$_('groups.add')}
      </button>
    </div>
    <div class="flex flex-col gap-3">
      {#each groupList as g (g.id)}
        <div class="card">
          <!-- Header -->
          <div class="flex items-center justify-between px-2 py-2 gap-2">
            <!-- Expand button -->
            <button
              onclick={() => expandGroup(g.id)}
              class="flex items-center gap-3 flex-1 min-w-0 px-2 py-1 rounded-lg hover:bg-slate-100 transition-colors text-start"
            >
              {#if expandedGroup === g.id}
                <ChevronDown class="w-4 h-4 text-slate-400 shrink-0" />
              {:else}
                <ChevronRight class="w-4 h-4 text-slate-400 shrink-0" />
              {/if}
              {#if renamingGroup === g.id}
                <input
                  class="input py-1 text-sm font-medium w-48"
                  bind:value={renameValue}
                  onkeydown={(e) => { if (e.key === 'Enter') saveRename(g.id); if (e.key === 'Escape') renamingGroup = null }}
                  onblur={() => saveRename(g.id)}
                  use:focusEl
                />
              {:else}
                <span class="font-medium text-slate-900 truncate">{g.name}</span>
                <span class="text-xs text-slate-400 shrink-0">
                  {groupMembers[g.id]?.length ?? '—'} {$_('groups.members_count')}
                  · {groupACLs(g.id).length} {$_('groups.permissions_count')}
                </span>
              {/if}
            </button>
            <!-- Action buttons -->
            <div class="flex items-center gap-1 shrink-0">
              <button onclick={() => startRename(g)} class="btn-ghost p-1.5" title={$_('groups.rename')}>
                <Pencil class="w-3.5 h-3.5" />
              </button>
              <button onclick={() => deleteGroup(g.id)} class="btn-ghost p-1.5 text-red-500">
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {#if expandedGroup === g.id}
            <div class="border-t border-slate-100 divide-y divide-slate-100">

              <!-- Members section -->
              <div class="px-4 py-3 flex flex-col gap-3">
                <h4 class="text-xs font-semibold text-slate-500 uppercase tracking-wide">{$_('groups.members')}</h4>

                <!-- Current members -->
                {#if groupMembers[g.id]?.length}
                  <div class="flex flex-wrap gap-2">
                    {#each groupMembers[g.id] as m (m.id)}
                      <span class="badge badge-slate gap-1">
                        {m.username}
                        <button onclick={() => removeFromGroup(g.id, m.id)} class="text-slate-400 hover:text-red-600 leading-none">×</button>
                      </span>
                    {/each}
                  </div>
                {:else}
                  <p class="text-sm text-slate-400">{$_('groups.no_members')}</p>
                {/if}

                <!-- Add multiple members via checkboxes -->
                {#if nonMembers(g.id).length > 0}
                  <div class="flex flex-col gap-2">
                    <p class="text-xs text-slate-500">{$_('groups.add_members')}:</p>
                    <div class="grid grid-cols-2 sm:grid-cols-3 gap-0.5 max-h-36 overflow-y-auto rounded border border-slate-200 p-2 bg-slate-50">
                      {#each nonMembers(g.id) as u (u.id)}
                        <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white px-2 py-1 rounded">
                          <input
                            type="checkbox"
                            checked={pendingMemberChecks[g.id]?.[u.id] ?? false}
                            onchange={(e) => togglePendingMember(g.id, u.id, e.currentTarget.checked)}
                          />
                          {u.username}
                        </label>
                      {/each}
                    </div>
                    {#if pendingCount(g.id) > 0}
                      <button onclick={() => addCheckedMembers(g.id)} class="btn-primary self-start text-sm py-1 px-3">
                        <UserPlus class="w-3.5 h-3.5" />
                        {$_('groups.add_selected', { values: { count: pendingCount(g.id) } })}
                      </button>
                    {/if}
                  </div>
                {:else if !groupMembers[g.id]?.length}
                  <!-- no users at all -->
                {:else}
                  <p class="text-xs text-slate-400">{$_('groups.no_users_to_add')}</p>
                {/if}
              </div>

              <!-- Permissions section -->
              <div class="px-4 py-3 flex flex-col gap-3">
                <h4 class="text-xs font-semibold text-slate-500 uppercase tracking-wide">{$_('groups.permissions')}</h4>

                {#if groupACLs(g.id).length}
                  <div class="flex flex-col gap-1.5">
                    {#each groupACLs(g.id) as acl (acl.id)}
                      <div class="flex items-center gap-2">
                        <code class="text-xs bg-slate-100 border border-slate-200 px-2 py-1 rounded flex-1 font-mono">{acl.streamPattern}</code>
                        <span class="badge shrink-0 {acl.action === 'read' ? 'badge-indigo' : acl.action === 'publish' ? 'badge-green' : 'badge-slate'}">
                          {acl.action === 'read' ? $_('acls.action_read') : acl.action === 'publish' ? $_('acls.action_publish') : $_('acls.action_both')}
                        </span>
                        <button onclick={() => deleteACL(acl.id)} class="btn-ghost p-1 text-red-500 hover:text-red-700 shrink-0">
                          <Trash2 class="w-3.5 h-3.5" />
                        </button>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <p class="text-sm text-slate-400">{$_('groups.no_permissions')}</p>
                {/if}

                <!-- Inline add permission -->
                {#if newPermForm[g.id]}
                  <div class="flex items-center gap-2">
                    <input
                      class="input py-1 text-sm flex-1 font-mono"
                      placeholder="cameras/* or stream-name"
                      bind:value={newPermForm[g.id].streamPattern}
                      onkeydown={(e) => { if (e.key === 'Enter') addGroupACL(g.id) }}
                    />
                    <select class="input py-1 text-sm w-36 shrink-0" bind:value={newPermForm[g.id].action}>
                      <option value="read">{$_('acls.action_read')}</option>
                      <option value="publish">{$_('acls.action_publish')}</option>
                      <option value="both">{$_('acls.action_both')}</option>
                    </select>
                    <button
                      onclick={() => addGroupACL(g.id)}
                      disabled={!newPermForm[g.id]?.streamPattern.trim()}
                      class="btn-primary py-1 px-3 text-sm shrink-0"
                    >
                      <Plus class="w-3.5 h-3.5" />
                    </button>
                  </div>
                {/if}
              </div>

            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}

  <!-- ACLs tab -->
  {#if activeTab === 'acls'}
    <div class="flex justify-end">
      <button onclick={() => { aclForm = { subjectType: 'user', subjectId: 0, streamPattern: '', action: 'read' }; showACLForm = true }} class="btn-primary">
        <Plus class="w-4 h-4" />{$_('acls.add')}
      </button>
    </div>
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th>{$_('acls.subject_type')}</th>
            <th>{$_('acls.subject')}</th>
            <th>{$_('acls.stream_pattern')}</th>
            <th>{$_('acls.action')}</th>
            <th class="text-end rtl:text-start"></th>
          </tr>
        </thead>
        <tbody>
          {#each aclList as a (a.id)}
            <tr>
              <td><span class="badge badge-slate">{a.subjectType}</span></td>
              <td class="font-medium">{a.subjectName}</td>
              <td><code class="text-xs bg-slate-100 px-1.5 py-0.5 rounded">{a.streamPattern}</code></td>
              <td>
                <span class="badge {a.action === 'read' ? 'badge-indigo' : a.action === 'publish' ? 'badge-green' : 'badge-slate'}">
                  {a.action}
                </span>
              </td>
              <td>
                <div class="flex justify-end">
                  <button onclick={() => deleteACL(a.id)} class="btn-ghost p-1.5 text-red-500">
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- Create user modal -->
{#if showUserForm}
  <Modal title={$_('users.add')} onclose={() => showUserForm = false}>
    {#snippet children()}
      <div class="flex flex-col gap-4">
        <div>
          <label class="label" for="user-username">{$_('users.username')}</label>
          <input id="user-username" class="input" bind:value={userForm.username} required />
        </div>
        <div>
          <label class="label" for="user-password">{$_('users.password')}</label>
          <input id="user-password" class="input" type="password" bind:value={userForm.password} required />
        </div>
        <div>
          <label class="label" for="user-role">{$_('users.role')}</label>
          <select id="user-role" class="input" bind:value={userForm.role}>
            <option value="user">{$_('users.role_user')}</option>
            <option value="admin">{$_('users.role_admin')}</option>
          </select>
        </div>
        <div>
          <label class="label" for="user-group">{$_('users.group_optional')}</label>
          <select id="user-group" class="input" bind:value={userForm.groupId}>
            <option value={0}>{$_('users.group_none')}</option>
            {#each groupList as g (g.id)}
              <option value={g.id}>{g.name}</option>
            {/each}
          </select>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showUserForm = false} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={createUser} class="btn-primary" disabled={savingUser}>{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}

<!-- Reset password modal (admin) -->
{#if showResetPwModal}
  <Modal title={$_('users.reset_password')} onclose={() => showResetPwModal = null}>
    {#snippet children()}
      <div class="flex flex-col gap-3">
        <p class="text-sm text-slate-600">
          {$_('users.reset_password_for', { values: { username: showResetPwModal.username } })}
        </p>
        <div>
          <label class="label" for="reset-pw">{$_('users.new_password')}</label>
          <input id="reset-pw" class="input" type="password" bind:value={resetPwValue} required />
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showResetPwModal = null} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={resetPassword} class="btn-primary" disabled={!resetPwValue}>{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}

<!-- Stream token modal -->
{#if showTokenModal}
  <Modal title={$_('users.regen_token')} onclose={() => showTokenModal = null}>
    {#snippet children()}
      <div class="flex flex-col gap-3">
        <div class="bg-amber-50 border border-amber-200 rounded-lg p-3 text-sm text-amber-800">
          {$_('users.token_warning')}
        </div>
        <div>
          <p class="label">{$_('users.token_label')} — {showTokenModal.username}</p>
          <div class="flex gap-2">
            <code class="input flex-1 font-mono text-xs">{showTokenModal.token}</code>
            <button
              onclick={async () => { await copyToClipboard(showTokenModal!.token); tokenCopied = true; setTimeout(() => tokenCopied = false, 2000) }}
              class="btn-secondary px-3"
            >
              {#if tokenCopied}<Check class="w-4 h-4 text-green-600" />{:else}<Copy class="w-4 h-4" />{/if}
            </button>
          </div>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showTokenModal = null} class="btn-primary">{$_('common.close')}</button>
    {/snippet}
  </Modal>
{/if}

<!-- Create group modal -->
{#if showGroupForm}
  <Modal title={$_('groups.add')} onclose={() => showGroupForm = false}>
    {#snippet children()}
      <div>
        <label class="label" for="group-name">{$_('groups.name')}</label>
        <input id="group-name" class="input" bind:value={groupName} required />
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showGroupForm = false} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={createGroup} class="btn-primary">{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}

<!-- Create ACL modal -->
{#if showACLForm}
  <Modal title={$_('acls.add')} onclose={() => showACLForm = false}>
    {#snippet children()}
      <div class="flex flex-col gap-4">
        <div>
          <label class="label" for="acl-subject-type">{$_('acls.subject_type')}</label>
          <select id="acl-subject-type" class="input" bind:value={aclForm.subjectType}>
            <option value="user">{$_('acls.type_user')}</option>
            <option value="group">{$_('acls.type_group')}</option>
          </select>
        </div>
        <div>
          <label class="label" for="acl-subject">{$_('acls.subject')}</label>
          <select id="acl-subject" class="input" bind:value={aclForm.subjectId}>
            {#if aclForm.subjectType === 'user'}
              {#each userList as u (u.id)}
                <option value={u.id}>{u.username}</option>
              {/each}
            {:else}
              {#each groupList as g (g.id)}
                <option value={g.id}>{g.name}</option>
              {/each}
            {/if}
          </select>
        </div>
        <div>
          <label class="label" for="acl-pattern">{$_('acls.stream_pattern')}</label>
          <input id="acl-pattern" class="input" bind:value={aclForm.streamPattern} placeholder="cameras/* or live/stream1" />
          <p class="text-xs text-slate-400 mt-1">{$_('acls.stream_pattern_help')}</p>
        </div>
        <div>
          <label class="label" for="acl-action">{$_('acls.action')}</label>
          <select id="acl-action" class="input" bind:value={aclForm.action}>
            <option value="read">{$_('acls.action_read')}</option>
            <option value="publish">{$_('acls.action_publish')}</option>
            <option value="both">{$_('acls.action_both')}</option>
          </select>
        </div>
      </div>
    {/snippet}
    {#snippet footer()}
      <button onclick={() => showACLForm = false} class="btn-secondary">{$_('common.cancel')}</button>
      <button onclick={createACL} class="btn-primary">{$_('common.save')}</button>
    {/snippet}
  </Modal>
{/if}
