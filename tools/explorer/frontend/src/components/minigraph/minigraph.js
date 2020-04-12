export default {
  name: 'minigraph',
  components: {},
  props: {
    color: {
      type: String,
      default: 'secondary darken-2 '
    },
    title: {
      type: String,
      default: ''
    },
    value: {
      default: '0'
    },
    append: {
      type: String,
      default: ''
    },
    special: {
      type: Boolean
    },
    clickable: {
      type: Boolean,
      default: false
    }
  },
  watch: {
    value: function (newVal) {
      this.val = newVal
      this.byte_head = this.append
      while (this.byte_heads.includes(this.byte_head) && this.val > 9999) {
        var index = this.byte_heads.indexOf(this.byte_head)
        if (index < this.byte_heads.length) {
          this.byte_head = this.byte_heads[index + 1]
          this.val /= 1000
          this.val = Math.round((this.val + Number.EPSILON) * 100) / 100
        }
      }
    }
  },
  mounted () { },
  data () {
    return {
      data: [],
      byte_heads: ['GB', 'TB', 'PB', 'EB', 'ZB', 'YB'],
      val: '0',
      byte_head: ''
    }
  },
  methods: {
    adaptUnit () { }
  },
  computed: {}
}
