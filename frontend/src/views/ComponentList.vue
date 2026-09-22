<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <button @click="goBack" class="text-gray-500 hover:text-gray-700 mb-2 flex items-center gap-1">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
          返回应用列表
        </button>
        <h2 class="text-2xl font-bold text-gray-800">组件管理</h2>
        <p class="text-gray-500 mt-1">{{ appName }} - 组件包管理</p>
      </div>
      <button @click="showUploadModal = true" class="btn btn-primary">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        上传新组件
      </button>
    </div>

    <div class="card mb-6">
      <div class="card-body">
        <div class="flex gap-4">
          <div class="w-48">
            <label class="label">渠道</label>
            <select v-model="channel" class="input" @change="loadComponents">
              <option value="">全部渠道</option>
              <option value="stable">正式版</option>
              <option value="beta">测试版</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th>组件Key</th>
            <th>组件名称</th>
            <th>版本号</th>
            <th>版本名称</th>
            <th>渠道</th>
            <th>平台</th>
            <th>架构</th>
            <th>大小</th>
            <th>状态</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="comp in components" :key="comp.id">
            <td>
              <code class="bg-gray-100 px-2 py-1 rounded text-sm text-primary-600">{{ comp.component_key }}</code>
            </td>
            <td class="font-medium">{{ comp.name }}</td>
            <td class="font-mono text-primary-600">{{ comp.version_code }}</td>
            <td>{{ comp.version_name }}</td>
            <td>
              <span class="badge" :class="comp.channel === 'stable' ? 'badge-success' : 'badge-warning'">
                {{ comp.channel === 'stable' ? '正式版' : '测试版' }}
              </span>
            </td>
            <td>{{ comp.platform }}</td>
            <td>{{ comp.architecture }}</td>
            <td>{{ formatSize(comp.file_size) }}</td>
            <td>
              <button @click="toggleStatus(comp)" class="badge cursor-pointer" :class="comp.status === 1 ? 'badge-success' : 'badge-secondary'">
                {{ comp.status === 1 ? '已发布' : '已下架' }}
              </button>
            </td>
            <td class="text-gray-500 text-sm">{{ formatDate(comp.created_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button v-if="comp.download_url" @click="copyUrl(comp.download_url)" class="btn btn-sm btn-secondary">
                  复制链接
                </button>
                <button @click="deleteComponent(comp)" class="btn btn-sm btn-danger">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="components.length === 0">
            <td colspan="11" class="text-center py-12 text-gray-400">
              暂无组件数据
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > pageSize" class="flex justify-center mt-6 gap-2">
      <button @click="changePage(page - 1)" :disabled="page <= 1" class="btn btn-sm btn-secondary">
        上一页
      </button>
      <span class="px-4 py-2 text-gray-600">
        第 {{ page }} / {{ totalPages }} 页，共 {{ total }} 条
      </span>
      <button @click="changePage(page + 1)" :disabled="page >= totalPages" class="btn btn-sm btn-secondary">
        下一页
      </button>
    </div>

    <div v-if="showUploadModal" class="modal-overlay" @click.self="showUploadModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="text-lg font-semibold">上传新组件</h3>
          <button @click="showUploadModal = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="space-y-4">
            <div>
              <label class="label">组件Key <span class="text-red-500">*</span></label>
              <input v-model="uploadForm.component_key" type="text" class="input" placeholder="如：core-module" />
              <p class="text-xs text-gray-500 mt-1">组件唯一标识，用于客户端识别</p>
            </div>
            <div>
              <label class="label">组件名称 <span class="text-red-500">*</span></label>
              <input v-model="uploadForm.name" type="text" class="input" placeholder="如：核心模块" />
            </div>
            <div>
              <label class="label">组件描述</label>
              <textarea v-model="uploadForm.description" class="input" rows="2" placeholder="组件功能描述"></textarea>
            </div>
            <div>
              <label class="label">版本名称 <span class="text-red-500">*</span></label>
              <input v-model="uploadForm.version_name" type="text" class="input" placeholder="如：1.0.0" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="label">渠道</label>
                <select v-model="uploadForm.channel" class="input">
                  <option value="stable">正式版</option>
                  <option value="beta">测试版</option>
                </select>
              </div>
              <div>
                <label class="label">平台</label>
                <select v-model="uploadForm.platform" class="input">
                  <option value="all">全平台</option>
                  <option value="windows">Windows</option>
                  <option value="macos">macOS</option>
                  <option value="linux">Linux</option>
                  <option value="android">Android</option>
                  <option value="ios">iOS</option>
                </select>
              </div>
            </div>
            <div>
              <label class="label">架构</label>
              <select v-model="uploadForm.architecture" class="input">
                <option value="all">全架构</option>
                <option value="x86_64">x86_64</option>
                <option value="arm64">arm64</option>
                <option value="x86">x86</option>
              </select>
            </div>
            <div v-if="appStorageType === 'metadata'">
              <label class="label">下载链接 <span class="text-red-500">*</span></label>
              <input v-model="uploadForm.download_url" type="text" class="input" placeholder="https://example.com/file.zip" />
            </div>
            <div v-if="appStorageType === 'metadata'">
              <label class="label">文件大小（字节）</label>
              <input v-model.number="uploadForm.file_size" type="number" class="input" placeholder="可选，如：1048576" />
            </div>
            <div v-if="appStorageType === 'metadata'">
              <label class="label">SHA256 哈希值</label>
              <input v-model="uploadForm.file_hash" type="text" class="input" placeholder="可选，文件SHA256哈希" />
            </div>
            <div v-else>
              <label class="label">组件文件 <span class="text-red-500">*</span></label>
              <input ref="fileInput" type="file" class="input" @change="handleFileChange" />
              <p v-if="uploadForm.file" class="text-sm text-gray-500 mt-1">
                已选择：{{ uploadForm.file.name }} ({{ formatSize(uploadForm.file.size) }})
              </p>
              <!-- 上传进度条 -->
              <div v-if="uploading" class="mt-3">
                <div class="flex justify-between text-xs text-gray-600 mb-1">
                  <span>上传中...</span>
                  <span>{{ uploadProgress }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2.5 overflow-hidden">
                  <div class="bg-blue-500 h-2.5 rounded-full transition-all duration-200" :style="{ width: uploadProgress + '%' }"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showUploadModal = false" class="btn btn-secondary">取消</button>
          <button @click="uploadComponent" class="btn btn-primary" :disabled="uploading">
            {{ uploading ? '上传中...' : '上传发布' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getComponents, uploadComponentFile, updateComponentStatus, deleteComponent as deleteComponentApi, getApp } from '../api'
import { success, error, info, confirm as askConfirm } from '../composables/useToast'
const route = useRoute()
const router = useRouter()

const appId = route.params.id
const appName = ref('')
const appStorageType = ref('metadata')
const components = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const channel = ref('')
const showUploadModal = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const fileInput = ref(null)

const uploadForm = ref({
  component_key: '',
  name: '',
  description: '',
  version_name: '',
  channel: 'stable',
  platform: 'all',
  architecture: 'all',
  file: null,
  download_url: '',
  file_size: null,
  file_hash: ''
})

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function formatSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

function handleFileChange(e) {
  uploadForm.value.file = e.target.files[0]
}

async function loadAppInfo() {
  try {
    const app = await getApp(appId)
    appName.value = app.name
    appStorageType.value = app.storage_type
  } catch (err) {
    console.error('Failed to load app info:', err)
  }
}

async function loadComponents() {
  try {
    const res = await getComponents({
      page: page.value,
      page_size: pageSize.value,
      channel: channel.value
    })
    const app = await getApp(appId)
    components.value = res.list.filter(c => c.app_id === app.app_id)
    total.value = components.value.length
  } catch (err) {
    console.error('Failed to load components:', err)
    components.value = []
    total.value = 0
  }
}

function changePage(p) {
  page.value = p
  loadComponents()
}

async function uploadComponent() {
  if (!uploadForm.value.component_key || !uploadForm.value.name || !uploadForm.value.version_name) {
    info('请填写组件Key、组件名称和版本名称')
    return
  }
  
  if (appStorageType.value === 'metadata') {
    if (!uploadForm.value.download_url) {
      info('请填写下载链接')
      return
    }
  } else {
    if (!uploadForm.value.file) {
      info('请选择组件文件')
      return
    }
  }

  uploading.value = true
  uploadProgress.value = 0
  const onProgress = (e) => {
    if (e.total > 0) {
      uploadProgress.value = Math.round((e.loaded * 100) / e.total)
    }
  }
  try {
    const app = await getApp(appId)
    const formData = new FormData()
    formData.append('app_id', app.app_id)
    formData.append('component_key', uploadForm.value.component_key)
    formData.append('name', uploadForm.value.name)
    formData.append('description', uploadForm.value.description)
    formData.append('version_name', uploadForm.value.version_name)
    formData.append('channel', uploadForm.value.channel)
    formData.append('platform', uploadForm.value.platform)
    formData.append('architecture', uploadForm.value.architecture)
    
    if (appStorageType.value === 'metadata') {
      formData.append('download_url', uploadForm.value.download_url)
      if (uploadForm.value.file_size) {
        formData.append('file_size', uploadForm.value.file_size)
      }
      if (uploadForm.value.file_hash) {
        formData.append('file_hash', uploadForm.value.file_hash)
      }
    } else {
      formData.append('file', uploadForm.value.file)
    }

    await uploadComponentFile(formData, onProgress)
    showUploadModal.value = false
    resetUploadForm()
    loadComponents()
  } catch (err) {
    error('上传失败：' + (err.response?.data?.error || err.message))
  } finally {
    uploading.value = false
  }
}

async function toggleStatus(comp) {
  const newStatus = comp.status === 1 ? 0 : 1
  try {
    await updateComponentStatus(comp.id, newStatus)
    comp.status = newStatus
  } catch (err) {
    error('操作失败：' + (err.response?.data?.error || err.message))
  }
}

async function deleteComponent(comp) {
  if (!(await askConfirm(`确定要删除组件 "${comp.name}" 吗？`))) {
    return
  }

  try {
    await deleteComponentApi(comp.id)
    loadComponents()
  } catch (err) {
    error('删除失败：' + (err.response?.data?.error || err.message))
  }
}

function copyUrl(url) {
  let fullUrl
  if (url.startsWith('http://') || url.startsWith('https://')) {
    fullUrl = url
  } else if (url.startsWith('/')) {
    fullUrl = window.location.origin + url
  } else {
    fullUrl = window.location.origin + '/api/download/' + url
  }
  navigator.clipboard.writeText(fullUrl).then(() => {
    success('链接已复制到剪贴板')
  })
}

function resetUploadForm() {
  uploadForm.value = {
    component_key: '',
    name: '',
    description: '',
    version_name: '',
    channel: 'stable',
    platform: 'all',
    architecture: 'all',
    file: null
  }
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function goBack() {
  router.push('/apps')
}

onMounted(() => {
  loadAppInfo()
  loadComponents()
})
</script>
