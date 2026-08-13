import 'normalize.css'
import 'element-plus/dist/index.css'
// 用预编译 CSS 而非 SCSS 源码:font-awesome 4.x 的 scss 仍是 @import / $a / $b 写法,
// 会在 Dart Sass 里刷弃用警告,而两者产物等价
import 'font-awesome/css/font-awesome.css'

import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import {
  Calendar,
  Delete,
  Document,
  Edit,
  Menu,
  Plus,
  Refresh,
  Search,
  Setting,
  Share
} from '@element-plus/icons-vue'

import router from './router/index.js'
import request from './request/index.js'
import store from 'store'
import Plugins from 'plugins'
import App from './App.vue'

const app = createApp(App)

const legacyIconAliases = {
  Plus,
  plus: Plus,
  Edit,
  edit: Edit,
  Delete,
  delete: Delete,
  Search,
  search: Search,
  Setting,
  setting: Setting,
  Document,
  document: Document,
  Share,
  share: Share,
  Refresh,
  refresh: Refresh,
  Calendar,
  date: Calendar,
  Menu
}

Object.entries(legacyIconAliases).forEach(([name, component]) => {
  app.component(name, component)
})

app.use(ElementPlus)
app.use(Plugins)
app.use(request)
app.use(store)
app.use(router)

app.mount('#app')
