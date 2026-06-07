import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // dev: only the /api namespace goes to the Go service; everything else is
      // served by the SPA (so /store/:username, /invoice/:id render the page).
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
