const API_URL = import.meta.env.VITE_API_URL || ''

export const branch = API_URL + '/api/get/git/branch'
export const getTag = API_URL + '/api/get/git/tag'
export const commit = API_URL + '/api/get/git/commit'
export const gitlog = API_URL + '/api/get/git/gitlog'
export const gitpull = API_URL + '/api/get/git/gitpull'
export const jenkinsBranch = API_URL + '/api/get/jenkins/commit'
