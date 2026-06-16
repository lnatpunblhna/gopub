import 'normalize.css'
import 'element-plus/dist/index.css'
import 'font-awesome/scss/font-awesome.scss'

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
