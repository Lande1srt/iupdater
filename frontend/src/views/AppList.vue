<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-gray-800">应用管理</h2>
        <p class="text-gray-500 mt-1">管理所有应用及其版本更新</p>
      </div>
      <button @click="openCreate" class="btn btn-primary">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        新建应用
      </button>
    </div>

    <div class="card mb-6">
      <div class="card-body">
        <div class="flex gap-4">
          <div class="flex-1">
            <input v-model="keyword" type="text" class="input" placeholder="搜索应用名称或 App ID" @keyup.enter="loadApps" />
          </div>
          <button @click="loadApps" class="btn btn-secondary">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            搜索
          </button>
        </div>
      </div>
    </div>

    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>App ID</th>
            <th>应用名称</th>
            <th>存储档案</th>
            <th>访问校验</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="app in apps" :key="app.id">
            <td class="text-gray-500">{{ app.id }}</td>
            <td><code class="bg-gray-100 px-2 py-1 rounded text-sm text-primary-600">{{ app.app_id }}</code></td>
            <td class="font-medium">{{ app.name }}</td>
            <td>
              <span v-if="app.storage_profile" class="badge badge-info">{{ app.storage_profile.name }}</span>
              <span v-else class="badge badge-secondary">元数据</span>
              <span v-if="app.storage_path" class="text-xs text-gray-500 ml-1">/{{ app.storage_path }}</span>
            </td>
            <td>
              <span v-if="app.enable_auth" class="badge badge-success" :title="'AccessKey: ' + app.access_key">已启用</span>
              <span v-else class="badge badge-secondary">未启用</span>
            </td>
            <td class="text-gray-500">{{ formatDate(app.created_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button @click="goToInfo(app)" class="btn btn-sm btn-primary">详情</button>
                <button @click="goToVersions(app)" class="btn btn-sm btn-secondary">版本</button>
                <button @click="goToComponents(app)" class="btn btn-sm btn-secondary">组件</button>
                <button @click="editApp(app)" class="btn btn-sm btn-secondary">编辑</button>
                <button @click="deleteAppItem(app)" class="btn btn-sm btn-danger">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="apps.length === 0">
            <td colspan="7" class="text-center py-12 text-gray-400">暂无应用数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > pageSize" class="flex justify-center mt-6 gap-2">
      <button @click="changePage(page - 1)" :disabled="page <= 1" class="btn btn-sm btn-secondary">上一页</button>
      <span class="px-4 py-2 text-gray-600">第 {{ page }} / {{ totalPages }} 页，共 {{ total }} 条</span>
      <button @click="changePage(page + 1)" :disabled="page >= totalPages" class="btn btn-sm btn-secondary">下一页</button>
    </div>

    <!-- 新建/编辑弹窗 -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-white rounded-lg shadow-xl w-[520px] max-h-[90vh] overflow-hidden flex flex-col">
        <div class="p-5 border-b border-gray-200 flex items-center justify-between">
          <h3 class="text-lg font-semibold">{{ editingId !== null ? '编辑应用' : '新建应用' }}</h3>
          <button @click="showModal = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
        <div class="p-5 space-y-4 overflow-y-auto flex-1">
          <div>
            <label class="label">App ID <span class="text-red-500">*</span></label>
            <input v-model="form.app_id" type="text" class="input" :disabled="editingId !== null" placeholder="如：com.example.myapp" />
            <p class="text-xs text-gray-500 mt-1">应用唯一识别码，创建后不可修改</p>
          </div>
          <div>
            <label class="label">应用名称 <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="input" placeholder="请输入应用名称" />
          </div>
          <div>
            <label class="label">应用描述</label>
            <textarea v-model="form.description" class="input" rows="2" placeholder="可选"></textarea>
          </div>
          <div>
            <label class="label">图标 URL</label>
            <input v-model="form.icon" type="text" class="input" placeholder="可选" />
          </div>

          <hr class="border-gray-200" />

          <div>
            <label class="label">存储档案 <span class="text-red-500">*</span></label>
            <select v-model="form.storage_profile_id" class="input" :disabled="profiles.length === 0">
              <option :value="null">元数据（仅填 URL，无实际存储）</option>
              <option v-for="p in profiles" :key="p.id" :value="p.id">
                {{ p.name }} ({{ storageTypeLabel(p.storage_type) }})
              </option>
            </select>
            <p v-if="profiles.length === 0" class="text-xs text-orange-500 mt-1">
              还没有存储档案，请先到 <router-link to="/storage" class="text-primary-600 hover:underline">存储管理</router-link> 创建
            </p>
          </div>
          <div v-if="form.storage_profile_id !== null" class="bg-gray-50 p-3 rounded-lg">
            <label class="label">自定义基础路径（可选）</label>
            <input v-model="form.storage_path" type="text" class="input" placeholder="如 coldsea/app 或留空使用档案默认" />
            <p class="text-xs text-gray-500 mt-1">覆盖档案的基础路径，多个 App 共用一个档案但存放到不同子目录</p>
          </div>

          <div class="bg-blue-50 p-4 rounded-lg">
            <div class="flex items-center gap-2">
              <input v-model="form.enable_auth" type="checkbox" id="enable_auth" class="w-4 h-4" />
              <label for="enable_auth" class="text-sm font-medium text-gray-700">启用访问校验</label>
            </div>
            <p class="text-xs text-gray-600 mt-2">启用后，客户端请求必须携带 AccessKey 才能访问更新接口</p>
          </div>
        </div>
        <div class="p-4 border-t border-gray-200 flex justify-end gap-2">
          <button @click="showModal = false" class="btn btn-secondary">取消</button>
          <button @click="saveApp" :disabled="saving" class="btn btn-primary">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getApps, createApp, updateApp, deleteApp, getStorageProfiles } from '../api'
import { success, error, info, confirm as askConfirm } from '../composables/useToast'

const router = useRouter()

const apps = ref([])
const profiles = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const showModal = ref(false)
const editingId = ref(null)
const saving = ref(false)

const form = ref({
  app_id: '', name: '', description: '', icon: '',
  storage_profile_id: null, storage_path: '',
  enable_auth: false
})

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

function storageTypeLabel(t) {
  return { s3: 'S3', webdav: 'WebDAV', metadata: '元数据' }[t] || t
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleString('zh-CN')
}

async function loadApps() {
  try {
    const res = await getApps({ page: page.value, page_size: pageSize.value, keyword: keyword.value })
    apps.value = res.list || []
    total.value = res.total || 0
  } catch (err) {
    error('加载失败')
  }
}

async function loadProfiles() {
  try {
    const res = await getStorageProfiles({ page: 1, page_size: 100 })
    profiles.value = res.list || []
  } catch {}
}

function changePage(p) { page.value = p; loadApps() }

function openCreate() {
  editingId.value = null
  form.value = { app_id: '', name: '', description: '', icon: '', storage_profile_id: null, storage_path: '', enable_auth: false }
  showModal.value = true
}

function editApp(app) {
  editingId.value = app.id
  form.value = {
    app_id: app.app_id,
    name: app.name,
    description: app.description || '',
    icon: app.icon || '',
    storage_profile_id: app.storage_profile_id,
    storage_path: app.storage_path || '',
    enable_auth: app.enable_auth || false
  }
  showModal.value = true
}

async function saveApp() {
  if (!form.value.app_id || !form.value.name) { info('请填写 App ID 和应用名称'); return }
  saving.value = true
  try {
    if (editingId.value !== null) {
      await updateApp(editingId.value, form.value)
      success('更新成功')
    } else {
      await createApp(form.value)
      success('创建成功')
    }
    showModal.value = false
    loadApps()
  } catch (err) {
    error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function deleteAppItem(app) {
  if (!(await askConfirm(`确定要删除应用 "${app.name}" 吗？相关版本和组件也会被删除。`))) return
  try { await deleteApp(app.id); success('删除成功'); loadApps() } catch { error('删除失败') }
}

function goToInfo(a) { router.push(`/apps/${a.id}/info`) }
function goToVersions(a) { router.push(`/apps/${a.id}/versions`) }
function goToComponents(a) { router.push(`/apps/${a.id}/components`) }

onMounted(() => { loadApps(); loadProfiles() })
</script>
