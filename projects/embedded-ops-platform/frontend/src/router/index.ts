import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/dashboard',
      name: 'Dashboard',
      component: () => import('../views/DashboardView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/devices',
      name: 'Devices',
      component: () => import('../views/DeviceListView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/devices/:deviceId',
      name: 'DeviceDetail',
      component: () => import('../views/DeviceDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/alerts',
      name: 'Alerts',
      component: () => import('../views/AlertView.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('access_token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router