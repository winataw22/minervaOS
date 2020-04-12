<template>
  <v-app dark>
    <v-navigation-drawer mini-variant app class="primary rounded" dark>
      <v-layout column fill-height justify-end>
        <div>
          <v-toolbar color="secondary darken-2 " class="py-3">
            <v-badge bottom right overlap color="primary">
              <template v-slot:badge>
                <v-icon size="12" dark>{{$route.meta.icon}}</v-icon>
              </template>
              <!--slot can be any component-->
              <v-avatar>
                <v-img src="./assets/logo.jpg" />
              </v-avatar>
            </v-badge>
          </v-toolbar>
        </div>
        <div>
          <v-list-item
            v-for="(route, i) in routes.filter(r => r.meta.position == 'top')"
            :key="i"
            :to="route"
            active-class="secondary--text"
          >
            <v-list-item-icon>
              <v-icon>{{ route.meta.icon }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title class="title text-capitalize">{{route.meta.displayName}}</v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </div>
        <v-spacer></v-spacer>
        <div>
          <v-list-item
            v-for="(route, i) in routes.filter(r => r.meta.position == 'bottom')"
            :key="i"
            :to="route"
            active-class="secondary--text"
          >
            <v-list-item-icon>
              <v-icon>{{ route.meta.icon }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title class="title text-capitalize">{{route.meta.displayName}}</v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </div>
      </v-layout>
    </v-navigation-drawer>

    <v-content class="content">
      <v-col>
        <v-row class="pa-4 mx-1">
          <h1 class="headline pt-0 pb-1 text-uppercase">
            <span>TF</span>
            <span class="font-weight-light">explorer</span>
            <span class="title font-weight-light">- {{$route.meta.displayName}}</span>
          </h1>
          <v-spacer />
        </v-row>
        <router-view></router-view>
      </v-col>
    </v-content>
    <v-bottom-navigation
      v-if="$vuetify.breakpoint.mdAndDown"
      grow
      dark
      class="primary topround"
      app
      fixed
      shift
      :value="$route.name"
    >
      <v-btn
        :value="route.name"
        icon
        v-for="(route, i) in routes"
        :key="i"
        @click="$router.push(route)"
      >
        <span>{{route.meta.displayName}}</span>
        <v-icon>{{route.meta.icon}}</v-icon>
      </v-btn>
    </v-bottom-navigation>
  </v-app>
</template>

<script>
export default {
  name: 'App',
  components: {},
  data: () => ({
    showDialog: false,
    dilogTitle: 'title',
    dialogBody: '',
    dialogActions: [],
    dialogImage: null,
    block: null,
    showBadge: true,
    menu: false
  }),
  computed: {
    routes () {
      return this.$router.options.routes
    }
  },
  mounted () {}
}
</script>

<style lang="scss">
.content {
  background: #fafafa !important;
}
.topround {
  border-radius: 10px 10px 0 0 !important;
}
.rounded {
  border-radius: 0 10px 10px 0 !important;
}
.v-menu__content,
.v-card {
  border-radius: 10px !important;
}
.v-card__title {
  font-size: 18px !important;
}
</style>
