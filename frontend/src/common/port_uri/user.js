const API_URL = import.meta.env.VITE_API_URL || ''

export const info = API_URL + '/api/get/user/info'
export const login = API_URL + '/login'
export const logout = API_URL + '/logout'
export const changepasswd = API_URL + '/changePasswd'
export const register = API_URL + '/register'
export const users = API_URL + '/api/get/user'
export const usersProject = API_URL + '/api/get/user/project'
