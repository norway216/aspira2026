import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: (process.env.VITE_API_BASE_URL || 'http://localhost:8080').replace('http', 'ws'),
        ws: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    // Firefox 78+ / Chrome 87+ / Safari 14+ compatibility
    target: ['es2020', 'firefox78', 'chrome87', 'safari14'],
    cssTarget: ['firefox78', 'chrome87', 'safari14'],
  },
})