import axios from 'axios'
import { ElMessage } from 'element-plus'
import { port_code } from 'common/port_uri'
import router from 'src/router'
import store from 'store'
import { SET_USER_INFO } from 'store/actions/type'

const setUserInfo = function (user) {
  store.dispatch(SET_USER_INFO, user)
}

const install = function (app) {
  if (install.installed) return
  install.installed = true

  axios.defaults.baseURL = '/'
  axios.defaults.timeout = 600000

  axios.interceptors.request.use(
    config => {
      if (store.state.user_info && store.state.user_info.user && store.state.user_info.user.AuthKey) {
        config.headers.Authorization = `token ${store.state.user_info.user.AuthKey}`
      }
      return config
    },
    err => Promise.reject(err)
  )

  axios.interceptors.response.use(
    response => {
      const resData = response.data
      const dataCode = resData.code
      const datamsg = resData.msg

      if (dataCode === port_code.success) {
        return Promise.resolve(response)
      }

      if (dataCode === port_code.unlogin) {
        setUserInfo(null)
        router.replace({ name: 'login' })
      }

      if (datamsg == null || datamsg === '') {
        return Promise.resolve(response)
      }

      ElMessage({
        message: datamsg,
        type: 'warning'
      })

      return Promise.reject({ code: dataCode, msg: datamsg })
    },
    error => {
      if (error.response) {
        const resCode = error.response.status
        const resMsg = error.message
        ElMessage({
          message: '操作失败！错误原因 ' + resMsg,
          type: 'error'
        })
        return Promise.reject({ code: resCode, msg: resMsg })
      }
      return Promise.reject(error)
    }
  )

  app.config.globalProperties.axios = axios
  app.config.globalProperties.$http = axios
}

export default {
  install
}
