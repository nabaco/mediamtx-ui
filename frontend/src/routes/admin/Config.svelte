<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { system as systemApi } from '../../lib/api'
  import type { SystemConfig, SystemInfo } from '../../lib/types'
  import { FileText, Wifi, WifiOff } from 'lucide-svelte'

  let config = $state<SystemConfig | null>(null)
  let info = $state<SystemInfo | null>(null)
  let loading = $state(true)

  $effect(() => {
    Promise.all([systemApi.config(), systemApi.info()])
      .then(([c, i]) => { config = c; info = i })
      .catch(() => {})
      .finally(() => loading = false)
  })
</script>

<div class="flex flex-col gap-6">
  <h1 class="text-2xl font-bold text-slate-900">{$_('config.title')}</h1>

  {#if loading}
    <div class="flex justify-center py-16">
      <div class="w-8 h-8 border-2 border-indigo-200 border-t-indigo-600 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- System info card -->
    {#if info}
      <div class="card p-4 grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div>
          <p class="text-xs text-slate-400 uppercase tracking-wide">{$_('system.version')}</p>
          <p class="font-medium mt-0.5">{info.version}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400 uppercase tracking-wide">MediaMTX</p>
          <p class="flex items-center gap-1.5 font-medium mt-0.5">
            {#if info.mediamtxOnline}
              <Wifi class="w-4 h-4 text-green-500" />{$_('system.online')}
            {:else}
              <WifiOff class="w-4 h-4 text-red-500" />{$_('system.offline')}
            {/if}
          </p>
        </div>
        <div>
          <p class="text-xs text-slate-400 uppercase tracking-wide">Host</p>
          <p class="font-medium mt-0.5 font-mono text-sm">{info.mediamtxHost}</p>
        </div>
        <div>
          <p class="text-xs text-slate-400 uppercase tracking-wide">Ports</p>
          <p class="font-mono text-sm mt-0.5">
            RTSP:{info.rtspPort} HLS:{info.hlsPort} WebRTC:{info.webrtcPort}
          </p>
        </div>
      </div>
    {/if}

    <!-- Config file display -->
    {#if config?.available}
      <div class="card">
        <div class="flex items-center gap-2 px-4 py-3 border-b border-slate-100">
          <FileText class="w-4 h-4 text-slate-400" />
          <span class="text-sm font-medium">{$_('config.path')}:</span>
          <code class="text-xs text-slate-600 font-mono">{config.resolvedPath}</code>
        </div>
        <div class="p-4">
          <p class="label">{$_('config.raw_config')}</p>
          <pre class="bg-slate-950 text-green-400 text-xs rounded-lg p-4 overflow-x-auto max-h-[60vh] leading-relaxed font-mono">{config.rawYaml}</pre>
        </div>
      </div>
    {:else}
      <div class="card p-8 text-center text-slate-400">
        <FileText class="w-8 h-8 mx-auto mb-3 opacity-40" />
        <p>{$_('config.not_available')}</p>
      </div>
    {/if}
  {/if}
</div>
