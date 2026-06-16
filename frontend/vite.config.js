import { fileURLToPath, URL } from 'node:url'
import { copyFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/static/',
  plugins: [
    vue(),
    {
      name: 'beego-template-output',
      closeBundle() {
        copyFileSync(
          fileURLToPath(new URL('../src/static/index.html', import.meta.url)),
          fileURLToPath(new URL('../src/views/index.tpl', import.meta.url))
        )
      }
    }
  ],
  resolve: {
    alias: [
      { find: /^store$/, replacement: fileURLToPath(new URL('./src/store/index.js', import.meta.url)) },
      { find: /^plugins$/, replacement: fileURLToPath(new URL('./src/plugins/index.js', import.meta.url)) },
      { find: /^plugins\/date$/, replacement: fileURLToPath(new URL('./src/plugins/date/index.js', import.meta.url)) },
      { find: /^components$/, replacement: fileURLToPath(new URL('./src/components/index.js', import.meta.url)) },
      { find: /^common\/port_uri$/, replacement: fileURLToPath(new URL('./src/common/port_uri/index.js', import.meta.url)) },
      { find: /^common\/tools$/, replacement: fileURLToPath(new URL('./src/common/tools/index.js', import.meta.url)) },
      { find: /^common\/storage$/, replacement: fileURLToPath(new URL('./src/common/storage/index.js', import.meta.url)) },
      { find: 'src', replacement: fileURLToPath(new URL('./src', import.meta.url)) },
      { find: 'assets', replacement: fileURLToPath(new URL('./src/assets', import.meta.url)) },
      { find: 'common', replacement: fileURLToPath(new URL('./src/common', import.meta.url)) },
      { find: 'components', replacement: fileURLToPath(new URL('./src/components', import.meta.url)) },
      { find: 'pages', replacement: fileURLToPath(new URL('./src/pages', import.meta.url)) },
      { find: 'plugins', replacement: fileURLToPath(new URL('./src/plugins', import.meta.url)) },
      { find: 'request', replacement: fileURLToPath(new URL('./src/request', import.meta.url)) },
      { find: 'store', replacement: fileURLToPath(new URL('./src/store', import.meta.url)) }
    ]
  },
  build: {
    outDir: '../src/static',
    emptyOutDir: true
  },
  server: {
    host: '0.0.0.0',
    port: 8080,
    proxy: {
      '/api': 'http://127.0.0.1:8192'
    }
  }
})
