<script lang="ts">
  import { toasts } from '../stores'
  import { CheckCircle, XCircle, Info, X } from 'lucide-svelte'

  const icons = { success: CheckCircle, error: XCircle, info: Info }
  const colors = {
    success: 'bg-green-50 border-green-200 text-green-900',
    error:   'bg-red-50 border-red-200 text-red-900',
    info:    'bg-blue-50 border-blue-200 text-blue-900',
  }
</script>

<div class="fixed bottom-4 end-4 z-[60] flex flex-col gap-2 max-w-sm w-full pointer-events-none">
  {#each $toasts as toast (toast.id)}
    <div
      class="flex items-start gap-3 px-4 py-3 rounded-lg border shadow-md pointer-events-auto animate-in slide-in-from-bottom-2 {colors[toast.type]}"
    >
      <svelte:component this={icons[toast.type]} class="w-5 h-5 shrink-0 mt-0.5" />
      <p class="flex-1 text-sm">{toast.message}</p>
    </div>
  {/each}
</div>
