<script lang="ts">
  import { _ } from 'svelte-i18n'
  import { system as systemApi } from '../lib/api'
  import { isAdmin } from '../lib/stores'
  import type { SystemInfo } from '../lib/types'
  import { Terminal, RefreshCw, FileText, ArrowUpCircle, Tv2, Radio, Video, Globe, Upload } from 'lucide-svelte'

  let info = $state<SystemInfo | null>(null)
  $effect(() => { systemApi.info().then(i => info = i).catch(() => {}) })

  const deployType = $derived(info?.deployType ?? 'unknown')
  const host = $derived(info?.mediamtxHost ?? window.location.hostname)
  const rtspPort = $derived(info?.rtspPort ?? 8554)
  const hlsPort = $derived(info?.hlsPort ?? 8888)
  const webrtcPort = $derived(info?.webrtcPort ?? 8889)
  const rtmpPort = $derived(info?.rtmpPort ?? 1935)

  type Step = { title: string; commands: string[] }
  type Section = { icon: any; title: string; steps: Step[] }

  const adminSections = $derived<Section[]>([
    {
      icon: Terminal,
      title: $_('instructions.sections.logs'),
      steps: [{
        title: 'View live logs',
        commands: deployType === 'docker' ? ['docker logs -f mediamtx']
          : deployType === 'podman' ? ['podman logs -f mediamtx']
          : deployType === 'compose' ? ['docker compose logs -f mediamtx']
          : deployType === 'quadlets' ? ['journalctl -u mediamtx.service -f']
          : deployType === 'systemd' ? ['journalctl -u mediamtx -f']
          : ['# Check your mediamtx log destination in mediamtx.yml'],
      }],
    },
    {
      icon: RefreshCw,
      title: $_('instructions.sections.restart'),
      steps: [{
        title: 'Restart mediamtx',
        commands: deployType === 'docker' ? ['docker restart mediamtx']
          : deployType === 'podman' ? ['podman restart mediamtx']
          : deployType === 'compose' ? ['docker compose restart mediamtx']
          : deployType === 'quadlets' ? ['systemctl restart mediamtx.service']
          : deployType === 'systemd' ? ['systemctl restart mediamtx']
          : ['# Restart mediamtx according to your deployment method'],
      }],
    },
    {
      icon: FileText,
      title: $_('instructions.sections.config'),
      steps: [{
        title: 'Edit configuration',
        commands: deployType === 'docker' || deployType === 'podman' ? [
            '# Configuration is usually mounted into the container',
            '# Edit the host file and restart:',
            'nano /etc/mediamtx/mediamtx.yml',
            deployType === 'docker' ? 'docker restart mediamtx' : 'podman restart mediamtx',
          ]
          : deployType === 'compose' ? [
            'nano /etc/mediamtx/mediamtx.yml',
            'docker compose restart mediamtx',
          ]
          : deployType === 'quadlets' ? [
            'nano /var/lib/mediamtx/mediamtx.yml',
            'systemctl restart mediamtx.service',
          ]
          : deployType === 'systemd' ? [
            'nano /etc/mediamtx/mediamtx.yml',
            'systemctl restart mediamtx',
          ]
          : ['nano /path/to/mediamtx.yml'],
      }],
    },
    {
      icon: ArrowUpCircle,
      title: $_('instructions.sections.update'),
      steps: [{
        title: 'Update to latest version',
        commands: deployType === 'docker' ? [
            'docker pull bluenviron/mediamtx:latest',
            'docker stop mediamtx && docker rm mediamtx',
            '# Re-run your original docker run command',
          ]
          : deployType === 'podman' ? [
            'podman pull docker.io/bluenviron/mediamtx:latest',
            'podman stop mediamtx && podman rm mediamtx',
            '# Re-run your original podman run command',
          ]
          : deployType === 'compose' ? [
            'docker compose pull',
            'docker compose up -d',
          ]
          : deployType === 'quadlets' ? [
            '# Update the image tag in your .container file, then:',
            'systemctl daemon-reload',
            'systemctl restart mediamtx.service',
          ]
          : deployType === 'systemd' ? [
            '# Download latest mediamtx binary from GitHub releases',
            'systemctl stop mediamtx',
            'cp mediamtx /usr/local/bin/mediamtx',
            'systemctl start mediamtx',
          ]
          : ['# See https://github.com/bluenviron/mediamtx/releases'],
      }],
    },
  ])
</script>

<div class="flex flex-col gap-8">
  <!-- Header -->
  <div>
    <h1 class="text-2xl font-bold text-slate-900">{$_('instructions.title')}</h1>
    <p class="text-slate-500 mt-1">{$_('instructions.subtitle')}</p>
  </div>

  <!-- ── USER SECTION: How to watch streams ── -->
  <div class="flex flex-col gap-4">
    <div>
      <h2 class="text-lg font-semibold text-slate-900">{$_('instructions.user_title')}</h2>
      <p class="text-sm text-slate-500 mt-0.5">{$_('instructions.user_subtitle')}</p>
    </div>

    <!-- VLC -->
    <div class="card">
      <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
        <Tv2 class="w-5 h-5 text-indigo-600" />
        <h3 class="font-semibold text-slate-900">{$_('instructions.sections.vlc')}</h3>
      </div>
      <div class="px-5 py-4 flex flex-col gap-3">
        <p class="text-sm text-slate-600">Media → Open Network Stream (Ctrl+N)</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          <div class="text-slate-500 text-xs mb-1"># RTSP (recommended)</div>
          rtsp://username:stream-token@{host}:{rtspPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'
          }
        </div>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          <div class="text-slate-500 text-xs mb-1"># HLS (no auth required for read access)</div>
          http://{host}:{hlsPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}/index.m3u8
        </div>
        <p class="text-xs text-slate-400">{$_('instructions.replace_stream', { values: { stream: `{${$_('streams.name').toLowerCase()}}` } })} <code class="bg-slate-100 px-1 rounded">cam1</code></p>
      </div>
    </div>

    <!-- OBS -->
    <div class="card">
      <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
        <Video class="w-5 h-5 text-indigo-600" />
        <h3 class="font-semibold text-slate-900">{$_('instructions.sections.obs')}</h3>
      </div>
      <div class="px-5 py-4 flex flex-col gap-3">
        <p class="text-sm text-slate-600">Sources → + → Media Source → Network → uncheck "Local File"</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          rtsp://username:stream-token@{host}:{rtspPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}
        </div>
        <p class="text-sm text-slate-600 mt-1">To push a stream from OBS to this server (Settings → Stream):</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm">
          <div class="text-slate-400">Service: Custom</div>
          <div class="text-green-300">Server: rtmp://{host}:{rtmpPort}/</div>
          <div class="text-green-300">Stream Key: {'{'}{$_('streams.name').toLowerCase()}{'}'}?token=your-token-slug</div>
        </div>
        <p class="text-xs text-slate-400">Copy the ready-made RTMP publish URL from the stream detail page — it includes your personal token slug automatically.</p>
      </div>
    </div>

    <!-- vMix -->
    <div class="card">
      <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
        <Radio class="w-5 h-5 text-indigo-600" />
        <h3 class="font-semibold text-slate-900">{$_('instructions.sections.vmix')}</h3>
      </div>
      <div class="px-5 py-4 flex flex-col gap-3">
        <p class="text-sm text-slate-600">Add Input → Stream / SRT</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          rtsp://username:stream-token@{host}:{rtspPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}
        </div>
        <p class="text-sm text-slate-600 mt-1">To output from vMix (External Output → RTMP):</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          rtmp://{host}:{rtmpPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}
        </div>
      </div>
    </div>

    <!-- Browser -->
    <div class="card">
      <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
        <Globe class="w-5 h-5 text-indigo-600" />
        <h3 class="font-semibold text-slate-900">{$_('instructions.sections.browser')}</h3>
      </div>
      <div class="px-5 py-4 flex flex-col gap-3">
        <p class="text-sm text-slate-600">Use the built-in player on the Dashboard or stream view — it supports WebRTC (low latency) and HLS fallback.</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm text-green-300">
          <div class="text-slate-400 text-xs mb-1"># WebRTC (WHEP) — lowest latency</div>
          http://{host}:{webrtcPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}/whep
          <div class="text-slate-400 text-xs mt-2 mb-1"># HLS — wider compatibility</div>
          http://{host}:{hlsPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}/index.m3u8
        </div>
      </div>
    </div>

    <!-- RTMP/SRT push -->
    <div class="card">
      <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
        <Upload class="w-5 h-5 text-indigo-600" />
        <h3 class="font-semibold text-slate-900">{$_('instructions.sections.rtmp_push')}</h3>
      </div>
      <div class="px-5 py-4 flex flex-col gap-3">
        <p class="text-sm text-slate-600">Push a stream to a path configured as <code class="bg-slate-100 px-1 rounded">source: publisher</code>:</p>
        <div class="bg-slate-950 rounded-lg px-4 py-3 font-mono text-sm">
          <div class="text-slate-400 text-xs mb-1"># RTMP</div>
          <div class="text-green-300">ffmpeg -re -i input.mp4 -c copy -f flv rtmp://{host}:{rtmpPort}/{'{'}{$_('streams.name').toLowerCase()}{'}'}</div>
          <div class="text-slate-400 text-xs mt-3 mb-1"># SRT</div>
          <div class="text-green-300">ffmpeg -re -i input.mp4 -c copy -f mpegts "srt://{host}:9999?streamid=publish:{'{'}{$_('streams.name').toLowerCase()}{'}'}"</div>
        </div>
      </div>
    </div>
  </div>

  <!-- ── ADMIN SECTION ── -->
  {#if $isAdmin}
    <div class="flex flex-col gap-4">
      <div>
        <h2 class="text-lg font-semibold text-slate-900">{$_('instructions.admin_title')}</h2>
        <p class="text-sm text-slate-500 mt-0.5">{$_('instructions.admin_subtitle')}</p>
        {#if info}
          <div class="flex items-center gap-2 mt-2">
            <span class="text-sm text-slate-500">{$_('instructions.deploy_type')}:</span>
            <span class="badge badge-indigo">{deployType}</span>
          </div>
        {/if}
      </div>

      {#each adminSections as section}
        {@const SectionIcon = section.icon}
        <div class="card">
          <div class="flex items-center gap-2 px-5 py-4 border-b border-slate-100">
            <SectionIcon class="w-5 h-5 text-indigo-600" />
            <h3 class="font-semibold text-slate-900">{section.title}</h3>
          </div>
          <div class="px-5 py-4 flex flex-col gap-4">
            {#each section.steps as step}
              <div>
                <p class="text-sm text-slate-600 mb-2">{step.title}</p>
                <div class="bg-slate-950 rounded-lg px-4 py-3">
                  {#each step.commands as cmd}
                    <div class="flex items-start gap-2 font-mono text-sm leading-relaxed">
                      {#if !cmd.startsWith('#')}
                        <span class="text-indigo-400 select-none shrink-0">$</span>
                        <span class="text-green-300">{cmd}</span>
                      {:else}
                        <span class="text-slate-500">{cmd}</span>
                      {/if}
                    </div>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}

      <!-- Auth callback -->
      <div class="card p-5 bg-blue-50 border-blue-100">
        <h3 class="font-semibold text-blue-900 mb-2">MediaMTX Auth Callback</h3>
        <p class="text-sm text-blue-800">
          For this UI to control stream access, add the following to your <code class="bg-blue-100 px-1 rounded">mediamtx.yml</code>:
        </p>
        <pre class="bg-blue-950 text-blue-200 text-xs rounded-lg p-3 mt-3 overflow-x-auto font-mono">authMethod: http
authHTTPAddress: http://{window.location.host}/api/v1/mediamtx/auth</pre>
        <p class="text-xs text-blue-600 mt-2">
          The address above is auto-filled based on where you're accessing this UI from.
        </p>
      </div>
    </div>
  {/if}
</div>
