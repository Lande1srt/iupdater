<template>
  <div class="min-h-screen bg-gradient-to-br from-primary-600 to-primary-800 flex items-center justify-center p-4">
    <div class="card w-full max-w-md p-8">
      <div class="text-center mb-8">
        <img src="/favicon.ico" alt="iUpdater" class="w-16 h-16 mx-auto mb-4" />
        <h1 class="text-2xl font-bold text-gray-800">iUpdater</h1>
        <p class="text-gray-500 mt-1">通用程序版本更新器</p>
      </div>

      <form @submit.prevent="handleLogin">
        <div class="mb-4">
          <label class="label">用户名</label>
          <input v-model="form.username" type="text" class="input" placeholder="请输入用户名" />
        </div>
        <div class="mb-6">
          <label class="label">密码</label>
          <input v-model="form.password" type="password" class="input" placeholder="请输入密码" />
        </div>
        <button type="submit" class="btn btn-primary w-full" :disabled="loading">
          <span v-if="loading">登录中...</span>
          <span v-else>登 录</span>
        </button>
      </form>

      <p v-if="error" class="text-red-500 text-sm text-center mt-4">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { login } from '../api'

const router = useRouter()
const userStore = useUserStore()

const form = ref({
  username: '',
  password: ''
})
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!form.value.username || !form.value.password) {
    error.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await login(form.value)
    userStore.setAuth(res.token, res.user)
    router.push('/')
  } catch (err) {
    error.value = err.response?.data?.error || '登录失败，请检查用户名和密码'
  } finally {
    loading.value = false
  }
}
</script>
