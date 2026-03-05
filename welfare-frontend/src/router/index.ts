import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'
import AuthCallbackView from '../views/AuthCallbackView.vue'
import AdminView from '../views/AdminView.vue'
import { useAuthStore } from '../store/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView },
    { path: '/auth/callback', component: AuthCallbackView },
    { path: '/', component: HomeView, meta: { requireAuth: true } },
    { path: '/admin', component: AdminView, meta: { requireAuth: true, requireAdmin: true } }
  ]
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.state.loaded) {
    await auth.loadProfile()
  }
  if (to.meta.requireAuth && !auth.state.token) {
    return '/login'
  }
  if (to.meta.requireAdmin && !auth.state.profile?.is_admin) {
    return '/'
  }
  return true
})

export default router
