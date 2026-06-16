const API_URL = import.meta.env.VITE_API_URL || ''

export const check = API_URL + '/api/get/p2p/check'
export const send = API_URL + '/api/get/p2p/send'
export const agent = API_URL + '/api/post/p2p/agent'
