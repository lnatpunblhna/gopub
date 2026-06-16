const API_URL = import.meta.env.VITE_API_URL || ''

export const list = API_URL + '/api/get/record/list'
export const get = API_URL + '/api/get/record/get'
