import react from '@astrojs/react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'astro/config'

export default defineConfig({
  integrations: [react()],
  output: 'static',
  server: {
    port: 5173
  },
  vite: {
    plugins: [tailwindcss()],
    optimizeDeps: {
      include: ['react-dom/client']
    },
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true
        },
        '/auth': {
          target: 'http://localhost:8080',
          changeOrigin: true
        }
      }
    }
  }
})
