<script lang="ts">
  import { lang, type Lang } from '../stores'
  import { locale } from 'svelte-i18n'

  function toggle() {
    const next: Lang = $lang === 'en' ? 'he' : 'en'
    lang.set(next)
    locale.set(next)
  }
</script>

<button
  onclick={toggle}
  class="btn-secondary px-2.5 py-1.5 text-xs font-semibold min-w-[2.5rem]"
  title="Switch language / החלף שפה"
>
  {$lang === 'en' ? 'עב' : 'EN'}
</button>
