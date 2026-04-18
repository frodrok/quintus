import { createRouter, createWebHistory } from 'vue-router'
import HomeView from './views/HomeView.vue'
import ConnectionsView from './views/ConnectionsView.vue'
import QueriesView from './views/QueriesView.vue'
import QueryEditorView from './views/QueryEditorView.vue'
import ExploreView from './views/ExploreView.vue'
import AiExploreView from './views/AiExploreView.vue'


export default createRouter({
  history: createWebHistory(),
  routes: [
  { path: '/', component: HomeView },
  { path: '/connections', component: ConnectionsView },
  { path: '/queries', component: QueriesView },
{ path: '/queries/new', component: QueryEditorView },
{ path: '/queries/:id/edit', component: QueryEditorView },
{ path: '/explore', component: ExploreView },
{ path: '/ai-explore', component: AiExploreView },
]
})