<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { X } from 'lucide-svelte'

  interface Props {
    title: string
    onclose: () => void
    children?: import('svelte').Snippet
    footer?: import('svelte').Snippet
  }
  let { title, onclose, children, footer }: Props = $props()

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Backdrop -->
<!-- svelte-ignore a11y_click_events_have_key_events a11y_interactive_supports_focus -->
<div
  class="fixed inset-0 bg-black/40 z-50 flex items-center justify-center p-4"
  onclick={e => e.target === e.currentTarget && onclose()}
  role="dialog"
  aria-modal="true"
  tabindex="-1"
>
  <div
    class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col"
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-200">
      <h2 class="text-base font-semibold text-slate-900">{title}</h2>
      <button onclick={onclose} class="btn-ghost p-1.5 -me-1.5">
        <X class="w-4 h-4" />
      </button>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto px-5 py-4">
      {@render children?.()}
    </div>

    <!-- Footer -->
    {#if footer}
      <div class="flex items-center justify-end gap-3 px-5 py-4 border-t border-slate-200">
        {@render footer?.()}
      </div>
    {/if}
  </div>
</div>
