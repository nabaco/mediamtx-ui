<script lang="ts">
  // TEC14 wordmark — "TEC" red, "14" near-black, matching the tec14.co.il family
  // (guac/c14/moel). Latin-only font + LTR so it renders identically on the
  // Hebrew (rtl) page. `size` is a Tailwind text-size class.
  let { size = 'text-xl' }: { size?: string } = $props()
</script>

<span
  class="{size} font-black tracking-tight leading-none select-none"
  dir="ltr"
  style="font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', 'Helvetica Neue', Arial, sans-serif;"
>
  <span class="text-[#E10E1A]">TEC</span><span class="text-slate-900">14</span>
</span>
