import tfService from '../services/tfService'
import lodash from 'lodash'

export default ({
  state: {
    user: {},
    registeredNodes: [],
    nodes: undefined,
    registeredFarms: [],
    farms: [],
    nodeSpecs: {
      amountregisteredNodes: 0,
      amountregisteredFarms: 0,
      countries: 0,
      onlinenodes: 0,
      cru: 0,
      mru: 0,
      sru: 0,
      hru: 0,
      network: 0,
      volume: 0,
      container: 0,
      zdb_namespace: 0,
      k8s_vm: 0

    }
  },
  actions: {
    getName: async context => {
      var response = await tfService.getName()
      return response.data.name
    },
    getUser: async context => {
      var name = await context.dispatch('getName')
      var response = await tfService.getUser(name)
      context.commit('setUser', response.data)
    },
    getRegisteredNodes (context) {
      tfService.getNodes().then(response => {
        context.commit('setRegisteredNodes', response.data)
        context.commit('setTotalSpecs', response.data)
      })
    },
    getRegisteredFarms (context, farmId) {
      tfService.registeredfarms(farmId).then(response => {
        context.commit('setAmountOfFarms', response.data)
        context.commit('setRegisteredFarms', response.data)
      })
    },
    getFarms: context => {
      tfService.getFarms(context.getters.user.id).then(response => {
        context.commit('setFarms', response.data)
      })
    },
    resetNodes: context => {
      context.commit('setNodes', undefined)
    }
  },
  mutations: {
    setRegisteredNodes (state, value) {
      state.registeredNodes = value
    },
    setRegisteredFarms (state, value) {
      state.registeredFarms = value
    },
    setFarms (state, value) {
      state.farms = value
    },
    setNodes (state, value) {
      state.nodes = value
    },
    setUser: (state, user) => {
      state.user = user
    },
    setAmountOfFarms (state, value) {
      state.nodeSpecs.amountregisteredFarms = value.length
    },
    setTotalSpecs (state, value) {
      state.nodeSpecs.amountregisteredNodes = value.length
      state.nodeSpecs.onlinenodes = countOnlineNodes(value)
      state.nodeSpecs.countries = lodash.uniqBy(
        value,
        node => node.location.country
      ).length
      state.nodeSpecs.cru = lodash.sumBy(value, node => node.total_resources.cru)
      state.nodeSpecs.mru = lodash.sumBy(value, node => node.total_resources.mru)
      state.nodeSpecs.sru = lodash.sumBy(value, node => node.total_resources.sru)
      state.nodeSpecs.hru = lodash.sumBy(value, node => node.total_resources.hru)
      state.nodeSpecs.network = lodash.sumBy(value, node => node.workloads.network)
      state.nodeSpecs.volume = lodash.sumBy(value, node => node.workloads.volume)
      state.nodeSpecs.container = lodash.sumBy(value, node => node.workloads.container)
      state.nodeSpecs.zdb_namespace = lodash.sumBy(value, node => node.workloads.zdb_namespace)
      state.nodeSpecs.k8s_vm = lodash.sumBy(value, node => node.workloads.k8s_vm)
    }
  },
  getters: {
    user: state => state.user,
    registeredNodes: state => state.registeredNodes,
    nodes: state => state.nodes,
    registeredFarms: state => state.registeredFarms,
    farms: state => state.farms,
    nodeSpecs: state => state.nodeSpecs
  }
})

function countOnlineNodes (data) {
  let onlinecounter = 0
  data.forEach(node => {
    const timestamp = new Date().getTime() / 1000
    const minutes = (timestamp - node.updated) / 60
    if (minutes < 20) onlinecounter++
  })
  return onlinecounter
}
