<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { push } from 'svelte-spa-router'
  import { auth as authApi } from '../lib/api'
  import { token, currentUser, addToast } from '../lib/stores'
  import LanguageSwitcher from '../lib/components/LanguageSwitcher.svelte'
  import { Radio } from 'lucide-svelte'

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
      push('/')
    } catch (err: any) {
      error = $_('auth.invalid_credentials')
    } finally {
      loading = false
    }
  }
</script>

<div class="min-h-screen bg-slate-50 flex items-center justify-center p-4">
  <div class="w-full max-w-sm">

    <!-- Language switcher top-right -->
    <div class="flex justify-end mb-6">
      <LanguageSwitcher />
    </div>

    <!-- Card -->
    <div class="card px-8 py-8">
      <!-- Logo -->
      <div class="flex flex-col items-center mb-8">
        <div class="w-12 h-12 bg-indigo-600 rounded-xl flex items-center justify-center mb-3">
          <Radio class="w-6 h-6 text-white" />
        </div>
        <h1 class="text-xl font-bold text-slate-900">{$_('app.name')}</h1>
        <p class="text-sm text-slate-500 mt-1">{$_('app.tagline')}</p>
      </div>

      <form onsubmit={submit} class="flex flex-col gap-4">
        <div>
          <label class="label" for="username">{$_('auth.username')}</label>
          <input
            id="username"
            class="input"
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
            class="input"
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

        <button type="submit" class="btn-primary w-full mt-2" disabled={loading}>
          {loading ? $_('auth.signing_in') : $_('auth.login')}
        </button>
      </form>
    </div>
  </div>
</div>
