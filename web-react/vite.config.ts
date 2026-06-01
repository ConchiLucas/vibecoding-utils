import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: path.resolve(__dirname, '../server/frontend/dist'),
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  optimizeDeps: {
    include: ['zustand'],
  },
  server: {
    host: '0.0.0.0',
    port: 5175,
    hmr: {
      overlay: false,
    },
    proxy: {
       '/api': {
          target: 'http://localhost:8008',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
       }
    },
  },
})
