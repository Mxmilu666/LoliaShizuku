import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/pages/home/index.vue')
  },
  {
    path: '/oauth',
    name: 'oauth',
    component: () => import('@/pages/oauth/index.vue')
  },
  {
    path: '/tunnels',
    name: 'tunnels',
    component: () => import('@/pages/tunnels/index.vue')
  },
  {
    path: '/runner',
    name: 'runner',
    component: () => import('@/pages/runner/index.vue')
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

const hasOAuthToken = async () => {
  const tokenService = (window as any).go?.services?.TokenService
  if (!tokenService?.HasOAuthToken) {
    return false
  }

  try {
    return await tokenService.HasOAuthToken()
  } catch {
    return false
  }
}

router.beforeEach(async (to) => {
  if (to.path === '/oauth') {
    return true
  }

  //const ok = await hasOAuthToken()
  //if (!ok) {
  //  return { path: '/oauth', replace: true }
  //}
  //return true
})

export default router
