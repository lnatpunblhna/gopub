const API_URL = import.meta.env.VITE_API_URL || ''

export const list = API_URL + '/api/get/record/list'
export const get = API_URL + '/api/get/record/get'
// 单条记录的完整输出（入库的 memo 是截断过的）
export const log = API_URL + '/api/get/record/log'
// 同一个上线单的历次发布批次
export const attempts = API_URL + '/api/get/record/attempts'
