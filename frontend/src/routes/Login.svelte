<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { auth as authApi } from '../lib/api'
  import { token, currentUser, addToast } from '../lib/stores'
  import LanguageSwitcher from '../lib/components/LanguageSwitcher.svelte'
  import Wordmark from '../lib/components/Wordmark.svelte'

  let username = $state('')
  let password = $state('')
  let loading = $state(false)
  let error = $state('')

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    loading = true
    error = ''
    try {
      const res = await authApi.login(username, password)
      token.set(res.token)
      currentUser.set({ id: 0, username: res.username, role: res.role })
      const returnTo = sessionStorage.getItem('loginReturn') ?? '/'
      sessionStorage.removeItem('loginReturn')
      push(returnTo)
    } catch (err: any) {
      error = $_('auth.invalid_credentials')
    } finally {
      loading = false
    }
  }
</script>

<div class="min-h-screen bg-white flex items-center justify-center p-4">
  <div class="w-full max-w-sm">

    <!-- Language switcher top-right -->
    <div class="flex justify-end mb-6">
      <LanguageSwitcher />
    </div>

    <!-- Card -->
    <div class="bg-white rounded-2xl shadow-md border border-slate-200 px-8 py-10">
      <!-- Logo: large TEC14 wordmark, matching the tec14.co.il family. -->
      <div class="flex flex-col items-center mb-8 select-none">
        <Wordmark size="text-7xl" />
        <h1 class="text-lg font-bold text-slate-900 mt-3">{$_('app.name')}</h1>
        <p class="text-sm text-slate-500 mt-1">{$_('app.tagline')}</p>
      </div>

      <form onsubmit={submit} class="flex flex-col gap-4">
        <div>
          <label class="label" for="username">{$_('auth.username')}</label>
          <input
            id="username"
            class="input border-2 border-slate-300 focus:border-slate-900 focus:ring-0"
            type="text"
            autocomplete="username"
            bind:value={username}
            required
            disabled={loading}
          />
        </div>

        <div>
          <label class="label" for="password">{$_('auth.password')}</label>
          <input
            id="password"
            class="input border-2 border-slate-300 focus:border-slate-900 focus:ring-0"
            type="password"
            autocomplete="current-password"
            bind:value={password}
            required
            disabled={loading}
          />
        </div>

        {#if error}
          <p class="text-sm text-red-600 text-center">{error}</p>
        {/if}

        <button
          type="submit"
          class="w-full mt-2 py-2.5 rounded-md font-semibold text-white bg-slate-800 hover:bg-slate-900 disabled:bg-slate-400 transition-colors"
          disabled={loading}
        >
          {loading ? $_('auth.signing_in') : $_('auth.login')}
        </button>
      </form>
    </div>
  </div>
</div>
