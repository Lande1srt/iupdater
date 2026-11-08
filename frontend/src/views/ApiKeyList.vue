<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-gray-800">API 密钥管理</h2>
        <p class="text-sm text-gray-500 mt-1">
          用于基座代理、CI/脚本调用 Admin API。明文只会在「创建 / 重新生成」时返回一次，请立即保存。
        </p>
      </div>
      <button
        @click="openCreateModal"
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        创建新密钥
      </button>
    </div>

    <!-- 提示条 -->
    <div class="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800 flex items-start gap-2">
      <svg class="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      <span>
        API Key 只存 SHA256 哈希，数据库里拿不到明文。丢失只能「重新生成」一把新的（旧的立即失效）。
      </span>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && keys.length === 0" class="bg-white border border-gray-200 rounded-xl p-12 text-center">
      <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
      <h3 class="text-lg font-medium text-gray-700 mb-1">还没有 API 密钥</h3>
      <p class="text-gray-500 text-sm mb-4">创建第一把密钥，用于基座代理或脚本调用 Admin API</p>
      <button @click="openCreateModal" class="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700">
        创建密钥
      </button>
    </div>

    <!-- 列表 -->
    <div v-else class="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <table class="w-full">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr>
            <th class="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">名称</th>
            <th class="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">权限</th>
            <th class="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">最后使用</th>
            <th class="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">创建时间</th>
            <th class="text-center px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">状态</th>
            <th class="text-right px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="k in keys" :key="k.id" class="hover:bg-gray-50 transition-colors">
            <td class="px-6 py-4">
              <div class="font-medium text-gray-900">{{ k.name }}</div>
              <div class="text-xs text-gray-400 font-mono mt-0.5">id:{{ k.id }}</div>
            </td>
            <td class="px-6 py-4">
              <span class="inline-flex px-2 py-1 rounded text-xs bg-blue-50 text-blue-700">
                {{ k.permissions || '*' }}
              </span>
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">
              {{ k.last_used_at ? formatTime(k.last_used_at) : '—' }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">{{ formatTime(k.created_at) }}</td>
            <td class="px-6 py-4 text-center">
              <span
                :class="k.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'"
                class="inline-flex px-2 py-1 rounded text-xs font-medium"
              >
                {{ k.enabled ? '启用' : '禁用' }}
              </span>
            </td>
            <td class="px-6 py-4 text-right">
              <div class="inline-flex items-center gap-1">
                <button
                  @click="openEditModal(k)"
                  class="px-2 py-1 text-xs text-gray-600 hover:text-primary-600 hover:bg-primary-50 rounded transition-colors"
                >编辑</button>
                <button
                  @click="handleRegenerate(k)"
                  class="px-2 py-1 text-xs text-gray-600 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                >重新生成</button>
                <button
                  @click="handleToggle(k)"
                  class="px-2 py-1 text-xs rounded transition-colors"
                  :class="k.enabled ? 'text-gray-600 hover:text-orange-600 hover:bg-orange-50' : 'text-gray-600 hover:text-green-600 hover:bg-green-50'"
                >{{ k.enabled ? '禁用' : '启用' }}</button>
                <button
                  @click="handleDelete(k)"
                  class="px-2 py-1 text-xs text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                >删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 加载遮罩 -->
    <div v-if="loading" class="fixed inset-0 bg-black/20 flex items-center justify-center z-40">
      <div class="bg-white rounded-xl px-6 py-4 shadow-lg flex items-center gap-3">
        <svg class="animate-spin w-5 h-5 text-primary-600" fill="none" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" class="opacity-25" />
          <path fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" class="opacity-75" />
        </svg>
        <span class="text-gray-700 text-sm">{{ loadingText }}</span>
      </div>
    </div>

    <!-- 创建/编辑模态框 -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl w-full max-w-md shadow-2xl">
        <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-800">
            {{ editingKey ? '编辑 API 密钥' : '创建新 API 密钥' }}
          </h3>
          <button @click="closeModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">名称 *</label>
            <input
              v-model="form.name"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none"
              placeholder="例如 SeaStack 基座 / GitHub CI / 运维脚本"
              maxlength="100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">权限范围</label>
            <input
              v-model="form.permissions"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none font-mono text-sm"
              placeholder="*（全部）或 apps:read,versions:write"
            />
            <p class="text-xs text-gray-400 mt-1">留空默认为 <code>*</code>（全部权限）。当前后端尚未细分 scope，预留字段。</p>
          </div>
        </div>
        <div class="px-6 py-4 border-t border-gray-100 flex justify-end gap-2">
          <button @click="closeModal" class="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50">
            取消
          </button>
          <button
            @click="submitModal"
            :disabled="!form.name.trim() || submitting"
            class="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ submitting ? '保存中...' : (editingKey ? '保存' : '创建') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 明文 Key 展示（创建 / 重新生成 后弹一次） -->
    <div v-if="plainKeyModal.show" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl w-full max-w-lg shadow-2xl">
        <div class="px-6 py-4 border-b border-gray-100">
          <h3 class="text-lg font-semibold text-gray-800 flex items-center gap-2">
            <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            密钥已生成
          </h3>
        </div>
        <div class="p-6 space-y-3">
          <div class="p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800">
            立即复制保存！这是数据库里唯一一次返回明文，刷新后再也看不到。
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-500 mb-1">明文 API Key</label>
            <div class="flex gap-2">
              <input
                :value="plainKeyModal.key"
                readonly
                class="flex-1 px-3 py-2 bg-gray-50 border border-gray-300 rounded-lg font-mono text-sm text-gray-800 select-all"
              />
              <button
                @click="copyPlainKey"
                class="px-4 py-2 bg-gray-800 text-white rounded-lg hover:bg-gray-900 text-sm"
                :class="{ 'bg-green-600': plainKeyModal.copied }"
              >
                {{ plainKeyModal.copied ? '✓ 已复制' : '复制' }}
              </button>
            </div>
          </div>
          <div class="text-xs text-gray-500 space-y-1 pt-2 border-t">
            <div><strong>curl 示例：</strong></div>
            <code class="block px-3 py-2 bg-gray-900 text-green-300 rounded font-mono text-xs overflow-x-auto">
              curl -H "X-API-Key: {{ plainKeyModal.key }}" http://localhost:8081/api/admin/apps
            </code>
          </div>
        </div>
        <div class="px-6 py-4 border-t border-gray-100 flex justify-end">
          <button @click="plainKeyModal.show = false" class="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700">
            我已保存，关闭
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  listAPIKeys, createAPIKey, updateAPIKey,
  deleteAPIKey, regenerateAPIKey, toggleAPIKey
} from '../api'
import { success as toastSuccess, error as toastError, confirm as toastConfirm } from '../composables/useToast'

const keys = ref([])
const loading = ref(false)
const loadingText = ref('加载中...')

const showModal = ref(false)
const editingKey = ref(null)
const form = ref({ name: '', permissions: '*' })
const submitting = ref(false)

const plainKeyModal = ref({ show: false, key: '', copied: false })

async function loadKeys() {
  loading.value = true
  loadingText.value = '加载中...'
  try {
    const resp = await listAPIKeys()
    keys.value = resp.keys || []
  } catch (e) {
    toastError('加载 API 密钥列表失败')
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  editingKey.value = null
  form.value = { name: '', permissions: '*' }
  showModal.value = true
}

function openEditModal(k) {
  editingKey.value = k
  form.value = { name: k.name || '', permissions: k.permissions || '*' }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingKey.value = null
}

async function submitModal() {
  if (!form.value.name.trim()) return
  submitting.value = true
  try {
    if (editingKey.value) {
      await updateAPIKey(editingKey.value.id, form.value)
      toastSuccess('已更新')
    } else {
      const resp = await createAPIKey(form.value)
      toastSuccess('密钥已创建 — 请在弹窗中保存明文')
      plainKeyModal.value = { show: true, key: resp.key.plain, copied: false }
    }
    closeModal()
    await loadKeys()
  } catch (e) {
    toastError(e.response?.data?.error || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleRegenerate(k) {
  const ok = await toastConfirm(
    '确认重新生成「' + k.name + '」？\n旧密钥立即失效，已用它的脚本/代理会报错。'
  )
  if (!ok) return
  loading.value = true
  loadingText.value = '正在重新生成...'
  try {
    const resp = await regenerateAPIKey(k.id)
    toastSuccess('已重新生成')
    plainKeyModal.value = { show: true, key: resp.key.plain, copied: false }
    await loadKeys()
  } catch (e) {
    toastError(e.response?.data?.error || '重新生成失败')
  } finally {
    loading.value = false
  }
}

async function handleToggle(k) {
  loading.value = true
  loadingText.value = k.enabled ? '正在禁用...' : '正在启用...'
  try {
    await toggleAPIKey(k.id)
    toastSuccess(k.enabled ? '已禁用' : '已启用')
    await loadKeys()
  } catch (e) {
    toastError(e.response?.data?.error || '操作失败')
  } finally {
    loading.value = false
  }
}

async function handleDelete(k) {
  const ok = await toastConfirm('删除后不可恢复，确认删除「' + k.name + '」？')
  if (!ok) return
  loading.value = true
  loadingText.value = '正在删除...'
  try {
    await deleteAPIKey(k.id)
    toastSuccess('已删除')
    await loadKeys()
  } catch (e) {
    toastError(e.response?.data?.error || '删除失败')
  } finally {
    loading.value = false
  }
}

async function copyPlainKey() {
  try {
    await navigator.clipboard.writeText(plainKeyModal.value.key)
    plainKeyModal.value.copied = true
    setTimeout(() => { plainKeyModal.value.copied = false }, 2000)
  } catch {
    toastError('复制失败，请手动选择文本')
  }
}

function formatTime(t) {
  if (!t) return '—'
  try {
    const d = new Date(t)
    return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch { return String(t) }
}

onMounted(loadKeys)
</script>
