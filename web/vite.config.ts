import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const environment = loadEnv(mode, process.cwd(), '')
  const backend = environment.WEB_API_PROXY || 'http://127.0.0.1:8080'

  return {
    plugins: [vue()],
    server: {
      host: '127.0.0.1',
      port: 5173,
      strictPort: true,
      proxy: { '/api': { target: backend, changeOrigin: false } }
    },
    build: { outDir: 'dist', emptyOutDir: true, sourcemap: false },
    test: { environment: 'node', restoreMocks: true }
  }
})
