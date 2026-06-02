import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // dev: proxy API calls to the Go service
      '^/(auth|me|products|categories|orders|pending_orders|customers|coupons|dashboard|store|invoice|assistant|operator|files|payments)':
        { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
