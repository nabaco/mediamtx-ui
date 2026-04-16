<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { link, location } from 'svelte-spa-router'
  import { currentUser, isAdmin, logout, addToast } from '../stores'
  import { auth as authApi } from '../api'
  import LanguageSwitcher from './LanguageSwitcher.svelte'
  import { Radio, Users, ShieldCheck, Settings, BookOpen, LogOut, Activity, UserCircle2 } from 'lucide-svelte'
  import Modal from './Modal.svelte'
  import { copyToClipboard } from '../utils'

  let mobileOpen = $state(false)

  // Account modal
  let showAccount = $state(false)
  let activeSection = $state<'password' | 'token'>('password')

  // Change password
  let changePwForm = $state({ current: '', next: '', confirm: '' })
  let changePwSaving = $state(false)
  let changePwError = $state('')
  let changePwSuccess = $state(false)

  // Stream token
  let streamToken = $state<string | null>(null)
  let tokenLoading = $state(false)
  let tokenRegenSaving = $state(false)
  let tokenCopied = $state(false)
  let tokenRegenSuccess = $state(false)
  let showRegenWarning = $state(false)

  async function openAccount(section: 'password' | 'token' = 'password') {
    activeSection = section
    changePwForm = { current: '', next: '', confirm: '' }
    changePwError = ''
    changePwSuccess = false
    streamToken = null
    tokenCopied = false
    tokenRegenSuccess = false
    showRegenWarning = false
    showAccount = true
    if (section === 'token') await loadToken()
  }

  async function switchSection(section: 'password' | 'token') {
    activeSection = section
    if (section === 'token' && streamToken === null && !tokenLoading) await loadToken()
  }

  async function loadToken() {
    tokenLoading = true
    try {
      const res = await authApi.getStreamToken()
      streamToken = res.token
    } catch { streamToken = null }
    finally { tokenLoading = false }
  }

  async function submitChangePw() {
    changePwError = ''
    if (changePwForm.next !== changePwForm.confirm) {
      changePwError = $_('account.passwords_mismatch')
      return
    }
    changePwSaving = true
    try {
      await authApi.changePassword(changePwForm.current, changePwForm.next)
      changePwSuccess = true
      setTimeout(() => { showAccount = false }, 1500)
    } catch (e: any) {
      changePwError = e.message ?? $_('common.error')
    } finally {
      changePwSaving = false
    }
  }

  async function copyToken() {
    if (!streamToken) return
    await copyToClipboard(streamToken)
    tokenCopied = true
    setTimeout(() => tokenCopied = false, 2000)
  }

  async function regenToken() {
    tokenRegenSaving = true
    showRegenWarning = false
    try {
      const res = await authApi.regenerateStreamToken()
      streamToken = res.token
      tokenRegenSuccess = true
      setTimeout(() => tokenRegenSuccess = false, 3000)
    } catch (e: any) {
      addToast('error', e.message ?? $_('common.error'))
    } finally {
      tokenRegenSaving = false
    }
  }

  const navLinks = $derived([
    { href: '/', label: $_('nav.dashboard'), icon: Radio, always: true },
    { href: '/admin/streams', label: $_('nav.streams'), icon: Activity, adminOnly: true },
    { href: '/admin/users', label: $_('nav.users'), icon: Users, adminOnly: true },
    { href: '/admin/config', label: $_('nav.config'), icon: Settings, adminOnly: true },
    { href: '/admin/audit', label: $_('nav.audit'), icon: ShieldCheck, adminOnly: true },
    { href: '/instructions', label: $_('nav.instructions'), icon: BookOpen, always: true },
  ])

  const visible = $derived(navLinks.filter(l => l.always || ($isAdmin && l.adminOnly)))

  const currentPath = $derived($location)
  $effect(() => { currentPath; showAccount = false })

  function isActive(href: string) {
    return currentPath === href || (href === '/' && (currentPath === '' || currentPath === '/'))
  }
</script>

<nav class="bg-white border-b border-slate-200 sticky top-0 z-40">
  <div class="container mx-auto px-4 max-w-7xl">
    <div class="flex h-14 items-center justify-between gap-4">

      <!-- Logo -->
      <a href="/" use:link class="flex items-center gap-2 font-semibold text-slate-900 shrink-0">
        <Radio class="w-5 h-5 text-indigo-600" />
        <span class="hidden sm:inline">{$_('app.name')}</span>
      </a>

      <!-- Desktop nav links -->
      <div class="hidden md:flex items-center gap-1">
        {#each visible as nav}
          <a
            href={nav.href}
            use:link
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors
                   {isActive(nav.href) ? 'bg-indigo-50 text-indigo-700' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'}"
          >
            <nav.icon class="w-4 h-4" />
            {nav.label}
          </a>
        {/each}
      </div>

      <!-- Right side -->
      <div class="flex items-center gap-2 shrink-0">
        <LanguageSwitcher />

        <span class="hidden sm:inline text-sm text-slate-500">{$currentUser?.username}</span>

        <button
          onclick={() => openAccount('password')}
          class="btn-ghost p-2"
          title={$_('account.title')}
        >
          <UserCircle2 class="w-4 h-4" />
        </button>

        <button
          onclick={() => { logout(); window.location.hash = '#/login' }}
          class="btn-ghost p-2"
          title={$_('nav.logout')}
        >
          <LogOut class="w-4 h-4" />
        </button>

        <!-- Mobile menu toggle -->
        <button
          class="md:hidden btn-ghost p-2"
          onclick={() => mobileOpen = !mobileOpen}
          aria-label="menu"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d={mobileOpen ? 'M6 18L18 6M6 6l12 12' : 'M4 6h16M4 12h16M4 18h16'} />
          </svg>
        </button>
      </div>
    </div>
  </div>

  <!-- Mobile menu -->
  {#if mobileOpen}
    <div class="md:hidden border-t border-slate-100 bg-white pb-2">
      {#each visible as nav}
        <a
          href={nav.href}
          use:link
          onclick={() => mobileOpen = false}
          class="flex items-center gap-2 px-4 py-2.5 text-sm font-medium
                 {isActive(nav.href) ? 'text-indigo-700 bg-indigo-50' : 'text-slate-700 hover:bg-slate-50'}"
        >
          <nav.icon class="w-4 h-4" />
          {nav.label}
        </a>
      {/each}
    </div>
  {/if}
</nav>

<!-- Account Modal -->
{#if showAccount}
  <Modal title={$_('account.title')} onclose={() => showAccount = false}>
    {#snippet children()}
      <!-- Section tabs -->
      <div class="flex gap-1 border-b border-slate-200 mb-4 -mx-5 px-5">
        <button
          onclick={() => switchSection('password')}
          class="px-3 py-2 text-sm font-medium border-b-2 transition-colors
                 {activeSection === 'password' ? 'border-indigo-600 text-indigo-700' : 'border-transparent text-slate-500 hover:text-slate-700'}"
        >{$_('account.change_password')}</button>
        <button
          onclick={() => switchSection('token')}
          class="px-3 py-2 text-sm font-medium border-b-2 transition-colors
                 {activeSection === 'token' ? 'border-indigo-600 text-indigo-700' : 'border-transparent text-slate-500 hover:text-slate-700'}"
        >{$_('account.stream_token')}</button>
      </div>

      <!-- Change password section -->
      {#if activeSection === 'password'}
        {#if changePwSuccess}
          <p class="text-green-600 text-sm font-medium py-4 text-center">{$_('account.password_changed')}</p>
        {:else}
          <div class="flex flex-col gap-4">
            <div>
              <label class="label" for="cp-current">{$_('account.current_password')}</label>
              <input id="cp-current" type="password" class="input" bind:value={changePwForm.current}
                disabled={changePwSaving} autocomplete="current-password" />
            </div>
            <div>
              <label class="label" for="cp-new">{$_('account.new_password')}</label>
              <input id="cp-new" type="password" class="input" bind:value={changePwForm.next}
                disabled={changePwSaving} autocomplete="new-password" />
            </div>
            <div>
              <label class="label" for="cp-confirm">{$_('account.confirm_password')}</label>
              <input id="cp-confirm" type="password" class="input" bind:value={changePwForm.confirm}
                disabled={changePwSaving} autocomplete="new-password"
                onkeydown={(e) => { if (e.key === 'Enter') submitChangePw() }} />
            </div>
            {#if changePwError}
              <p class="text-red-600 text-sm">{changePwError}</p>
            {/if}
          </div>
        {/if}
      {/if}

      <!-- Stream token section -->
      {#if activeSection === 'token'}
        <div class="flex flex-col gap-4">
          <p class="text-sm text-slate-500">{$_('account.stream_token_desc')}</p>

          {#if tokenLoading}
            <p class="text-sm text-slate-400">{$_('common.loading')}</p>
          {:else if streamToken}
            <div>
              <p class="label">{$_('account.stream_token')}</p>
              <div class="flex gap-2">
                <code class="input flex-1 font-mono text-xs truncate">{streamToken}</code>
                <button onclick={copyToken} class="btn-secondary px-3 shrink-0">
                  {tokenCopied ? $_('account.token_copied') : $_('account.copy_token')}
                </button>
              </div>
              {#if tokenRegenSuccess}
                <p class="text-amber-600 text-xs mt-2">{$_('account.token_regenerated')}</p>
              {/if}
            </div>
          {:else}
            <p class="text-sm text-slate-400">{$_('account.no_token')}</p>
          {/if}

          <!-- Regen warning + confirm -->
          {#if showRegenWarning}
            <div class="bg-amber-50 border border-amber-200 rounded-lg p-3 flex flex-col gap-3">
              <p class="text-sm text-amber-800">{$_('account.regen_warning')}</p>
              <div class="flex gap-2">
                <button onclick={() => showRegenWarning = false} class="btn-secondary text-sm">
                  {$_('common.cancel')}
                </button>
                <button onclick={regenToken} disabled={tokenRegenSaving} class="btn-primary text-sm">
                  {tokenRegenSaving ? $_('common.loading') : $_('account.regen_token')}
                </button>
              </div>
            </div>
          {:else}
            <button onclick={() => showRegenWarning = true} class="btn-secondary self-start text-sm">
              {$_('account.regen_token')}
            </button>
          {/if}
        </div>
      {/if}
    {/snippet}
    {#snippet footer()}
      {#if activeSection === 'password' && !changePwSuccess}
        <button class="btn-secondary" onclick={() => showAccount = false} disabled={changePwSaving}>
          {$_('common.cancel')}
        </button>
        <button class="btn-primary" onclick={submitChangePw} disabled={changePwSaving}>
          {changePwSaving ? $_('common.loading') : $_('common.save')}
        </button>
      {:else}
        <button class="btn-secondary" onclick={() => showAccount = false}>{$_('common.close')}</button>
      {/if}
    {/snippet}
  </Modal>
{/if}
