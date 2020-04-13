import capacityselector from '../capacityselector'
import nodeinfo from '../nodeinfo'
import { mapGetters, mapMutations } from 'vuex'
import { groupBy, map } from 'lodash'

export default {
  name: 'capacitymap',
  components: { capacityselector, nodeinfo },
  props: [],
  data () {
    return {
      select: { text: 'All', value: 'All' }
    }
  },
  computed: {
    ...mapGetters(['registeredFarms', 'registeredNodes']),
    allFarmsList: function () {
      const allFarmers = this.registeredFarms.map(farm => {
        return {
          value: farm,
          text: farm.name
        }
      })
      allFarmers.push({ text: 'All', value: 'All' })
      return allFarmers
    },
    nodeLocation: function () {
      // Group nodes by country
      const groupedNodeLocations = groupBy(
        this.registeredNodes,
        node => node.location.country
      )

      const nodeLocations = []
      // Map expect type [[country, count], ...]
      map(groupedNodeLocations, (groupedLocation, key) => {
        const numberOfNodesInLocation = []
        const count = groupedLocation.length
        numberOfNodesInLocation.push(key, count)
        nodeLocations.push(numberOfNodesInLocation)
      })

      return nodeLocations
    }
  },
  mounted () { },
  methods: {
    setSelected (value) {
      if (value === 'All') {
        this.$emit('custom-event-input-changed', '')
        return this.setNodes(this.registeredNodes)
      }
      const filteredNodes = this.registeredNodes.filter(
        node => node.farm_id.toString() === value.id.toString()
      )
      this.setNodes(filteredNodes)
      this.$emit('custom-event-input-changed', value.name.toString())
    },
    ...mapMutations(['setNodes'])
  }
}
