import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { AUTH_EXPIRED_EVENT } from './api/client'
import { useAuthStore } from './store/auth'
import { sanitizeAuthRedirect } from './utils/authRedirect'

const auth = useAuthStore()

window.addEventListener(AUTH_EXPIRED_EVENT, () => {
  auth.clearSession()
  const currentRoute = router.currentRoute.value
  if (currentRoute.path === '/login') {
    return
  }
  const redirect = sanitizeAuthRedirect(currentRoute.fullPath || '/')
  void router.replace({ path: '/login', query: { redirect } })
})

createApp(App).use(router).mount('#app')
