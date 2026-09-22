import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { guest: true }
  },
  {
    path: '/',
    component: () => import('../views/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/apps'
      },
      {
        path: 'apps',
        name: 'Apps',
        component: () => import('../views/AppList.vue')
      },
      {
        path: 'apps/:id/info',
        name: 'AppInfo',
        component: () => import('../views/AppInfo.vue')
      },
      {
        path: 'apps/:id/versions',
        name: 'Versions',
        component: () => import('../views/VersionList.vue')
      },
      {
        path: 'apps/:id/components',
        name: 'Components',
        component: () => import('../views/ComponentList.vue')
      },
      {
        path: 'storage',
        name: 'Storage',
        component: () => import('../views/StorageProfileList.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.isLoggedIn()) {
    next('/login')
  } else if (to.meta.guest && userStore.isLoggedIn()) {
    next('/')
  } else {
    next()
  }
})

export default router
