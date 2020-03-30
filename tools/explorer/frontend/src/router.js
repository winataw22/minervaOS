import Vue from 'vue'
import Router from 'vue-router'
import { publicPath } from '../vue.config'

Vue.use(Router)

export default new Router({
  mode: 'history',
  base: publicPath,
  routes: [
    {
      path: '/',
      name: 'Capacity directory',
      component: () => import(/* webpackChunkName: "capacity-page" */ './views/capacity'),
      meta: {
        icon: 'fas fa-server',
        position: 'top',
        displayName: 'Capacity'
      }
    }
  ]
})
