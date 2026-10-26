import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    tailwindcss(),
    svelte(),
  ],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 600,
    rolldownOptions: {
      output: {
        // Vite 8 (rolldown) only accepts the function form of manualChunks.
        manualChunks(id: string) {
          if (!id.includes('/node_modules/')) return undefined
          if (id.includes('/node_modules/hls.js/')) return 'hls'
          if (id.includes('/node_modules/@lucide/svelte/')) return 'icons'
          if (/\/node_modules\/(svelte|svelte-spa-router|svelte-i18n)\//.test(id)) return 'vendor'
          return undefined
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:9996',
      '/metrics': 'http://localhost:9996',
    },
  },
})
