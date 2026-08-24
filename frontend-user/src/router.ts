import { createRouter, createWebHistory } from 'vue-router'
import StreamView from './views/StreamView.vue'
import QueryView from './views/QueryView.vue'
import ClusterView from './views/ClusterView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'stream', component: StreamView },
    { path: '/query', name: 'query', component: QueryView },
    { path: '/cluster', name: 'cluster', component: ClusterView },
  ],
})
