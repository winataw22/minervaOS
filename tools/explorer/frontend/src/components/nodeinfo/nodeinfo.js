export default {
  name: 'nodeinfo',
  props: ['node'],
  data () {
    return {
      freeIcon: this.node.freeToUse === true ? { icon: 'fa-check', color: 'green' } : { icon: 'fa-times', color: 'red' }
    }
  },
  mounted () {
    console.log(this.node)
  },
  methods: {
    getPercentage (type) {
      return (this.node.reservedResources[type] / this.node.totalResources[type]) * 100
    }
  }
}
