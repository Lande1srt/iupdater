<template>
  <div class="min-h-screen bg-gradient-to-br from-primary-600 to-primary-800 flex items-center justify-center p-4">
    <div class="card w-full max-w-md p-8">
      <div class="text-center mb-8">
        <img src="/favicon.ico" alt="iUpdater" class="w-16 h-16 mx-auto mb-4" />
        <h1 class="text-2xl font-bold text-gray-800">iUpdater</h1>
        <p class="text-gray-500 mt-1">通用程序版本更新器</p>
      </div>

      <!-- Step 1: 用户名 + 密码 -->
      <form v-if="step === 1" @submit.prevent="step1Submit">
        <div class="mb-4">
          <label class="label">用户名</label>
          <input v-model="form.username" type="text" class="input" placeholder="请输入用户名" autocomplete="username" autofocus />
        </div>
        <div class="mb-6">
          <label class="label">密码</label>
          <input v-model="form.password" type="password" class="input" placeholder="请输入密码" autocomplete="current-password" />
        </div>
        <button type="submit" class="btn btn-primary w-full" :disabled="loading">
          <span v-if="loading">验证中...</span>
          <span v-else>下一步</span>
        </button>
      </form>

      <!-- Step 2: TOTP 验证 -->
      <form v-else @submit.prevent="step2Submit">
        <div class="mb-2">
          <label class="label">两步验证</label>
          <p class="text-sm text-gray-500 mb-4">请在身份验证器 App 中输入 6 位验证码</p>
          <input
            ref="totpInput"
            v-model="form.totp_code"
            type="text"
            inputmode="numeric"
            maxlength="6"
            class="input tracking-[0.5em] text-center text-lg font-mono"
            placeholder="• • • • • •"
            autocomplete="one-time-code"
          />
        </div>
        <button type="submit" class="btn btn-primary w-full mt-6" :disabled="loading">
          <span v-if="loading">登录中...</span>
          <span v-else>登 录</span>
        </button>
        <button type="button" class="btn btn-secondary w-full mt-2" :disabled="loading" @click="backToStep1">
          返回
        </button>
      </form>

      <p v-if="error" class="text-red-500 text-sm text-center mt-4">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import { login } from '../api'

const router = useRouter()
const userStore = useUserStore()

const step = ref(1)         // 1 = 密码, 2 = TOTP
const loading = ref(false)
const error = ref('')
const totpInput = ref(null)

const form = ref({
  username: '',
  password: '',
  totp_code: ''
})

// 从 Step 2 退回 Step 1：清空密码和 TOTP
function backToStep1() {
  step.value = 1
  form.value.password = ''
  form.value.totp_code = ''
  error.value = ''
}

// Step 1: 提交用户名密码
async function step1Submit() {
  if (!form.value.username || !form.value.password) {
    error.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await login({
      username: form.value.username,
      password: form.value.password
    })

    // 后端返回 step=totp → 进入 Step 2
    if (res.step === 'totp') {
      step.value = 2
      await nextTick()
      totpInput.value?.focus()
      return
    }

    // 直接拿到 token（未启用 TOTP）
    userStore.setAuth(res.token, res.user)
    router.push('/')
  } catch (err) {
    // 密码错误 → 留在 Step 1
    error.value = err.response?.data?.error || '用户名或密码错误'
  } finally {
    loading.value = false
  }
}

// Step 2: 提交 TOTP（带上用户名密码）
async function step2Submit() {
  if (!form.value.totp_code || form.value.totp_code.length !== 6) {
    error.value = '请输入 6 位验证码'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await login({
      username: form.value.username,
      password: form.value.password,
      totp_code: form.value.totp_code
    })
    userStore.setAuth(res.token, res.user)
    router.push('/')
  } catch (err) {
    // TOTP 错误 → 退回 Step 1
    error.value = err.response?.data?.error || 'TOTP 验证失败'
    backToStep1()
  } finally {
    loading.value = false
  }
}
</script>
