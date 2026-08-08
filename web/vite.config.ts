import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: resolve(__dirname, '../cmd/ospanel/web-dist'),
    emptyOutDir: true,
    assetsDir: 'assets',
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('naive-ui') || id.includes('vueuc') || id.includes('css-render')) return 'naive'
            if (id.includes('vue') || id.includes('pinia') || id.includes('vue-router') || id.includes('axios')) return 'vendor'
            if (id.includes('@vueuse')) return 'utils'
            return 'vendor'
          }
        },
      },
    },
  },
})
