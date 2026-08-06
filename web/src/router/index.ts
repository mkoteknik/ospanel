import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { guest: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/DashboardLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue'),
        },
        {
          path: 'domains',
          name: 'Domains',
          component: () => import('@/views/domains/List.vue'),
        },
        {
          path: 'domains/:id',
          name: 'DomainDetail',
          component: () => import('@/views/domains/Detail.vue'),
        },
        {
          path: 'email',
          name: 'Email',
          component: () => import('@/views/email/List.vue'),
        },
        {
          path: 'databases',
          name: 'Databases',
          component: () => import('@/views/database/List.vue'),
        },
        {
          path: 'files',
          name: 'FileManager',
          component: () => import('@/views/files/List.vue'),
        },
        {
          path: 'ssl',
          name: 'SSL',
          component: () => import('@/views/ssl/List.vue'),
        },
        {
          path: 'dns',
          name: 'DNS',
          component: () => import('@/views/dns/List.vue'),
        },
        {
          path: 'backup',
          name: 'Backup',
          component: () => import('@/views/backup/List.vue'),
        },
        {
          path: 'monitor',
          name: 'Monitor',
          component: () => import('@/views/monitor/List.vue'),
        },
        {
          path: 'security',
          name: 'Security',
          component: () => import('@/views/security/List.vue'),
        },
        {
          path: 'cache',
          name: 'Cache',
          component: () => import('@/views/cache/List.vue'),
        },
        {
          path: 'cloudflare',
          name: 'CloudFlare',
          component: () => import('@/views/cloudflare/List.vue'),
        },
        {
          path: 'containers',
          name: 'Containers',
          component: () => import('@/views/containers/List.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

// Navigation guard
router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.meta.guest && authStore.isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router
