<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { link, location } from 'svelte-spa-router'
  import { currentUser, isAdmin, logout } from '../stores'
  import { auth as authApi } from '../api'
  import LanguageSwitcher from './LanguageSwitcher.svelte'
  import { Radio, Users, ShieldCheck, Settings, BookOpen, LogOut, Activity, UserCircle2, Tv } from 'lucide-svelte'
  import Modal from './Modal.svelte'

  // When `minimal` is true, nav links and the mobile menu toggle are hidden.
  // Right-side controls (language, account, logout) stay visible.
  let { minimal = false }: { minimal?: boolean } = $props()

  let mobileOpen = $state(false)

  // Account modal
  let showAccount = $state(false)

  // Change password
  let changePwForm = $state({ current: '', next: '', confirm: '' })
  let changePwSaving = $state(false)
  let changePwError = $state('')
  let changePwSuccess = $state(false)

  function openAccount() {
    changePwForm = { current: '', next: '', confirm: '' }
    changePwError = ''
    changePwSuccess = false
    showAccount = true
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

  const navLinks = $derived([
    { href: '/', label: $_('nav.dashboard'), icon: Radio, always: true },
    { href: '/admin/streams', label: $_('nav.streams'), icon: Activity, adminOnly: true },
    { href: '/admin/users', label: $_('nav.users'), icon: Users, adminOnly: true },
    { href: '/admin/config', label: $_('nav.config'), icon: Settings, adminOnly: true },
    { href: '/admin/audit', label: $_('nav.audit'), icon: ShieldCheck, adminOnly: true },
    { href: '/ch14-live', label: $_('nav.ch14_live'), icon: Tv, adminOnly: true },
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

      <!-- Logo: link when normal, plain span in minimal mode (no navigation) -->
      {#if minimal}
        <div class="flex items-center gap-2 font-semibold text-slate-900 shrink-0">
          <Radio class="w-5 h-5 text-indigo-600" />
          <span class="hidden sm:inline">{$_('app.name')}</span>
        </div>
      {:else}
        <a href="/" use:link class="flex items-center gap-2 font-semibold text-slate-900 shrink-0">
          <Radio class="w-5 h-5 text-indigo-600" />
          <span class="hidden sm:inline">{$_('app.name')}</span>
        </a>
      {/if}

      <!-- Desktop nav links -->
      {#if !minimal}
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
      {/if}

      <!-- Right side -->
      <div class="flex items-center gap-2 shrink-0">
        {#if !minimal}
          <LanguageSwitcher />
        {/if}

        <span class="hidden sm:inline text-sm text-slate-500">{$currentUser?.username}</span>

        <button
          onclick={openAccount}
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
        {#if !minimal}
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
        {/if}
      </div>
    </div>
  </div>

  <!-- Mobile menu -->
  {#if mobileOpen && !minimal}
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
    {/snippet}
    {#snippet footer()}
      {#if !changePwSuccess}
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
