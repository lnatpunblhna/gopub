const API_URL = import.meta.env.VITE_API_URL || ''

// 全部上线单，仅管理员可调
export const list = API_URL + '/api/get/task/list'
// 我的上线单，后端按登录态过滤，不需要（也不该）由前端传 user_id
export const mylist = API_URL + '/api/get/task/mylist'
export const get = API_URL + '/api/get/task/get'
export const del = API_URL + '/api/get/task/del'
export const save = API_URL + '/api/post/task/save'
export const release = API_URL + '/api/get/walle/release'
export const rollback = API_URL + '/api/get/task/rollback'
export const flush = API_URL + '/api/get/walle/flush'
export const chart = API_URL + '/api/get/task/chart'
export const changes = API_URL + '/api/get/task/changes'
// 上线单的失败原因（task_err_log）
export const errlog = API_URL + '/api/get/task/errlog'
