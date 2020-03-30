import nodeInfo from '../nodeinfo'
import { mapGetters, mapActions } from 'vuex'
import moment from 'moment'
import momentDurationFormatSetup from 'moment-duration-format'
import { find } from 'lodash'

momentDurationFormatSetup(moment)

export default {
  name: 'nodestable',
  props: ['farmselected', 'registerednodes'],

  components: { nodeInfo },
  data () {
    return {
      searchnodes: undefined,
      showOffline: false,
      storeName: '',
      showDialog: false,
      dilogTitle: 'title',
      dialogBody: '',
      dialogActions: [],
      dialogImage: null,
      block: null,
      showBadge: true,
      menu: false,
      loadedNodes: false,
      othersHidden: false,
      itemsPerPage: 4,
      expanded: [],

      headers: [
        { text: 'ID', value: 'id' },
        { text: 'Uptime', value: 'uptime' },
        { text: 'Version', value: 'version' },
        { text: 'Farmer', value: 'farm_name' },
        { text: 'Status', value: 'status', align: 'center' }
      ]
    }
  },
  computed: {
    ...mapGetters(['registeredFarms', 'nodes']),
    // Parse nodelist to table format here
    parsedNodesList: function () {
      const nodeList = this.nodes ? this.nodes : this.registerednodes
      const parsedNodes = nodeList.filter(node => this.showNode(node)).map(node => {
        const farm = find(this.registeredFarms, farmer => {
          return farmer.id === node.farm_id
        })

        return {
          uptime: moment.duration(node.uptime, 'seconds').format(),
          version: node.os_version,
          id: node.node_id,
          farm_name: farm ? farm.name : node.farm_id,
          farm_id: node.farm_id,
          name: 'node ' + node.node_id,
          totalResources: node.total_resources,
          reservedResources: node.reserved_resources,
          usedResources: node.used_resources,
          workloads: node.workloads,
          updated: new Date(node.updated * 1000),
          status: this.getStatus(node),
          location: node.location,
          freeToUse: node.free_to_use
        }
      })
      return parsedNodes
    }
  },
  mounted () {
    this.resetNodes()
  },
  methods: {
    ...mapActions(['resetNodes']),
    getStatus (node) {
      const { updated } = node
      const startTime = moment()
      const end = moment.unix(updated)
      const minutes = startTime.diff(end, 'minutes')

      // if updated difference in minutes with now is less then 10 minutes, node is up
      if (minutes < 15) return { color: 'green', status: 'up' }
      else if (minutes > 16 && minutes < 20) { return { color: 'orange', status: 'likely down' } } else return { color: 'red', status: 'down' }
    },
    showNode (node) {
      if (this.farmselected && this.farmselected.id !== node.farm_id) {
        return false
      }
      if (!this.showOffline && this.getStatus(node)['status'] === 'down') {
        return false
      }

      return true
    },
    truncateString (str) {
      // do not truncate in full screen mode
      if (this.othersHidden === true) {
        return str
      }
      str = str.toString()
      if (str.length < 10) return str
      return str.substr(0, 10) + '...'
    },
    openNodeDetails (node) {
      const index = this.expanded.indexOf(node)
      if (index > -1) this.expanded.splice(index, 1)
      else this.expanded.push(node)
    },
    hideOthers () {
      var all = document.getElementsByClassName('others')
      for (var i = 0; i < all.length; i++) {
        all[i].style.display = 'none'
        all[i].classList.remove('flex')
      }
      this.othersHidden = true
    },
    showOthers () {
      var all = document.getElementsByClassName('others')
      for (var i = 0; i < all.length; i++) {
        all[i].style.display = 'block'
        all[i].classList.add('flex')
      }
      this.othersHidden = false
    }
  }
}
