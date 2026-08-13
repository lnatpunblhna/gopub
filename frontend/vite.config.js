import { fileURLToPath, URL } from 'node:url'
import { copyFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ command }) => ({
  base: '/static/',
  // 生产构建剥离 console/debugger,开发环境保留
  esbuild: {
    drop: command === 'build' ? ['console', 'debugger'] : []
  },
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
    emptyOutDir: true,
    // 业务 chunk 都在 30 kB 量级;放宽阈值是为了两个已单独拆出的第三方 vendor:
    // element-plus(全量注册,约 970 kB)与 echarts(路由内懒加载,约 1.1 MB)。
    // 阈值只覆盖它们,业务代码或其他依赖膨胀时仍会告警。
    chunkSizeWarningLimit: 1200,
    // 第三方大库单独成 chunk,避免业务代码与 vendor 混在一个巨型 index.js 里
    rolldownOptions: {
      // @vueuse/core(element-plus 的传递依赖)的产物里有 Rolldown 无法识别位置的
      // /* #__PURE__ */ 注释。第三方源码改不了,且只影响它自身的 DCE 粒度,
      // 因此只关掉这项检查;其余诊断照常输出。
      checks: {
        invalidAnnotation: false
      },
      output: {
        codeSplitting: {
          groups: [
            { name: 'echarts', test: /[\\/]node_modules[\\/](echarts|zrender)[\\/]/ },
            { name: 'element-plus', test: /[\\/]node_modules[\\/](element-plus|@element-plus|@popperjs|@floating-ui|@ctrl[\\/]tinycolor|async-validator|@vueuse)[\\/]/ },
            { name: 'vue-vendor', test: /[\\/]node_modules[\\/](vue|vue-router|vuex|@vue)[\\/]/ },
            { name: 'vendor', test: /[\\/]node_modules[\\/]/ }
          ]
        }
      }
    }
  },
  server: {
    host: '0.0.0.0',
    port: 8080,
    proxy: {
      '/api': 'http://127.0.0.1:8192'
    }
  }
}))
