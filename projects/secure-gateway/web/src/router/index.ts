import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { ElMessage } from 'element-plus'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    meta: { requiresAuth: true },
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘', icon: 'Monitor' }
      },
      {
        path: 'nodes',
        name: 'Nodes',
        component: () => import('@/views/Nodes.vue'),
        meta: { title: '节点管理', icon: 'Connection' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { title: '用户管理', icon: 'User', requiresAdmin: true }
      },
      {
        path: 'traffic',
        name: 'Traffic',
        component: () => import('@/views/Traffic.vue'),
        meta: { title: '流量统计', icon: 'DataLine' }
      },
      {
        path: 'audit-logs',
        name: 'AuditLogs',
        component: () => import('@/views/AuditLogs.vue'),
        meta: { title: '审计日志', icon: 'Document' }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: '系统设置', icon: 'Setting' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// Navigation guard
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('access_token')

  if (to.meta.requiresAuth !== false && !token) {
    next('/login')
    return
  }

  if (to.path === '/login' && token) {
    next('/dashboard')
    return
  }

  // Check admin requirement
  if (to.meta.requiresAdmin) {
    const role = localStorage.getItem('user_role')
    if (role !== 'admin') {
      ElMessage.warning('需要管理员权限')
      next('/dashboard')
      return
    }
  }

  next()
})

export default router