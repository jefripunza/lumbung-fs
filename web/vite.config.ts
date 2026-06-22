import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    vueDevTools(),
  ],
    server: {
    headers: {
      'Cross-Origin-Opener-Policy': 'same-origin-allow-popups',
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: false,
        xfwd: true,
      },
      '/upload': {
        target: 'http://localhost:8080',
        changeOrigin: false,
        xfwd: true,
      },
      '/file': {
        target: 'http://localhost:8080',
        changeOrigin: false,
        xfwd: true,
      },
    },
    watch: {
      ignored: [
        '**/*.go',
        '**/go.mod',
        '**/go.sum',
        '**/vendor/**',
        '**/bin/**',
        '**/tmp/**',
        '**/*.spec.ts',
        '**/tests/**',
        'erd.*',
      ],
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
