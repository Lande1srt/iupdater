<template>
  <div class="min-h-screen bg-gray-50 flex">
    <aside class="w-64 bg-white border-r border-gray-200 fixed h-full">
      <div class="p-6 border-b border-gray-100">
        <div class="flex items-center gap-3">
          <img src="/favicon.ico" alt="iUpdater" class="w-10 h-10" />
          <div>
            <h1 class="font-bold text-gray-800">iUpdater</h1>
            <p class="text-xs text-gray-500">版本更新管理</p>
          </div>
        </div>
      </div>

      <nav class="p-4">
        <router-link
          to="/apps"
          class="flex items-center gap-3 px-4 py-3 rounded-lg mb-1 transition-colors"
          :class="isActive('/apps') ? 'bg-primary-50 text-primary-600' : 'text-gray-600 hover:bg-gray-50'"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <span class="font-medium">应用管理</span>
        </router-link>
      </nav>

      <div class="absolute bottom-0 left-0 right-0 p-4 border-t border-gray-100">
        <div class="flex items-center justify-between px-4 py-2">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-gray-200 rounded-full flex items-center justify-center">
              <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
            </div>
            <span class="text-sm text-gray-700">{{ userStore.user?.username }}</span>
          </div>
          <button @click="handleLogout" class="text-gray-400 hover:text-red-500 transition-colors" title="退出登录">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
            </svg>
          </button>
        </div>
      </div>
    </aside>

    <main class="ml-64 flex-1 p-8">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

function isActive(path) {
  return route.path.startsWith(path)
}

function handleLogout() {
  userStore.logout()
  router.push('/login')
}
</script>
