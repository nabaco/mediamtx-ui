<script lang="ts">
  import Router, { location } from 'svelte-spa-router'
  import { locale } from 'svelte-i18n'
  import './lib/i18n/index'
  import { lang, token, currentUser, logout, isAdmin } from './lib/stores'
  import { auth as authApi } from './lib/api'

  import Navbar from './lib/components/Navbar.svelte'
  import ToastContainer from './lib/components/Toast.svelte'

  import Login from './routes/Login.svelte'
  import Dashboard from './routes/Dashboard.svelte'
  import StreamView from './routes/StreamView.svelte'
  import AdminStreams from './routes/admin/Streams.svelte'
  import AdminUsers from './routes/admin/Users.svelte'
  import AdminConfig from './routes/admin/Config.svelte'
  import AdminAudit from './routes/admin/Audit.svelte'
  import Instructions from './routes/Instructions.svelte'
  import Ch14Live from './routes/Ch14Live.svelte'
  import NotFound from './routes/NotFound.svelte'

  const routes = {
    '/':             Dashboard,
    '/ch14-live':    Ch14Live,
    '/stream/:name': StreamView,
    '/admin/streams': AdminStreams,
    '/admin/users':   AdminUsers,
    '/admin/config':  AdminConfig,
    '/admin/audit':   AdminAudit,
    '/instructions':  Instructions,
    '/login':         Login,
    '*':              NotFound,
  }

  // Sync svelte-i18n locale with our lang store
  lang.subscribe(l => locale.set(l))

  // Load current user if token exists
  $effect(() => {
    if ($token && !$currentUser) {
      authApi.me().then(u => currentUser.set(u)).catch(() => logout())
    }
  })

  // Non-admins on /ch14-live get a minimal navbar (no menu links, but
  // password change and logout still available).
  const minimalNav = $derived($location === '/ch14-live' && !$isAdmin)
</script>

{#if $token}
  <div class="min-h-screen flex flex-col">
    <Navbar minimal={minimalNav} />
    <main class="flex-1 container mx-auto px-4 py-6 max-w-7xl">
      <Router {routes} />
    </main>
  </div>
{:else}
  <Router {routes} />
{/if}

<ToastContainer />
