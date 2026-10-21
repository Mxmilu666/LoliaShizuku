import { createRouter, createWebHistory } from 'vue-router'
import { TokenService } from '../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services'

const routes = [
  {
    path: '/',
    name: 'home',
    meta: { title: '首页' },
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
    meta: { title: '隧道' },
    component: () => import('@/pages/tunnels/index.vue')
  },
  {
    path: '/runner',
    name: 'runner',
    meta: { title: '运行' },
    component: () => import('@/pages/runner/index.vue')
  },
  {
    path: '/settings',
    name: 'settings',
    meta: { title: '设置' },
    component: () => import('@/pages/settings/index.vue')
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

const hasOAuthToken = async () => {
  try {
    return await TokenService.HasOAuthToken()
  } catch {
    return false
  }
}

router.beforeEach(async (to) => {
  const ok = await hasOAuthToken()

  if (to.path === '/oauth') {
    if (ok) {
      return { path: '/', replace: true }
    }
    return true
  }

  if (!ok) {
    return { path: '/oauth', replace: true }
  }
  return true
})

export default router
