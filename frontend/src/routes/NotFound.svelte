<script lang="ts">
  import { link } from 'svelte-spa-router'
</script>

<div class="flex flex-col items-center justify-center py-24 gap-4 text-center">
  <p class="text-6xl font-bold text-slate-200">404</p>
  <p class="text-slate-600">Page not found</p>
  <a href="/" use:link class="btn-primary">Go home</a>
</div>
