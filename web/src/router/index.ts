import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/layouts/DashboardLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/overview' },
        {
          path: '/overview',
          name: 'overview',
          component: () => import('@/pages/OverviewPage.vue'),
        },
        {
          path: '/origins',
          name: 'origins',
          component: () => import('@/pages/OriginsPage.vue'),
        },
        {
          path: '/rules',
          name: 'rules',
          component: () => import('@/pages/RulesPage.vue'),
        },
        {
          path: '/explorer',
          name: 'explorer',
          component: () => import('@/pages/ExplorerPage.vue'),
        },
        {
          path: '/unknown-origins',
          name: 'unknown-origins',
          component: () => import('@/pages/UnknownOriginsPage.vue'),
        },
      ],
    },
  ],
})

// Navigation guard
router.beforeEach((to) => {
  const token = localStorage.getItem('lumbungfs_token')
  const isAuthenticated = !!token

  if (to.meta.requiresAuth !== false && !isAuthenticated) {
    return { name: 'login' }
  }

  if (to.name === 'login' && isAuthenticated) {
    return { name: 'overview' }
  }
})

export default router
