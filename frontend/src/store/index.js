import { createStore } from 'vuex'

import actions from './actions/index.js'
import getters from './getters/index.js'
import mutations from './mutations/index.js'
import state from './states/index.js'

export default createStore({
  state,
  getters,
  actions,
  mutations
})
