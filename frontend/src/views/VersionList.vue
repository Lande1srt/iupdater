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
        <h2 class="text-2xl font-bold text-gray-800">版本管理</h2>
        <p class="text-gray-500 mt-1">{{ appName }} - 版本发布管理</p>
      </div>
      <button @click="showUploadModal = true" class="btn btn-primary">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        上传新版本
      </button>
    </div>

    <div class="card mb-6">
      <div class="card-body">
        <div class="flex gap-4">
          <div class="w-48">
            <label class="label">渠道</label>
            <select v-model="channel" class="input" @change="loadVersions">
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
            <th>版本号</th>
            <th>版本名称</th>
            <th>渠道</th>
            <th>平台</th>
            <th>架构</th>
            <th>包类型</th>
            <th>强制更新</th>
            <th>大小</th>
            <th>状态</th>
            <th>发布时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ver in versions" :key="ver.id">
            <td class="font-mono text-primary-600">{{ ver.version_code }}</td>
            <td class="font-medium">{{ ver.version_name }}</td>
            <td>
              <span class="badge" :class="ver.channel === 'stable' ? 'badge-success' : 'badge-warning'">
                {{ ver.channel === 'stable' ? '正式版' : '测试版' }}
              </span>
            </td>
            <td>{{ ver.platform }}</td>
            <td>{{ ver.architecture }}</td>
            <td>
              <span class="badge" :class="ver.package_type === 'full' ? 'badge-info' : 'badge-secondary'">
                {{ ver.package_type === 'full' ? '完整包' : '组件包' }}
              </span>
            </td>
            <td>
              <span v-if="ver.is_force_update" class="badge badge-danger">强制</span>
              <span v-else class="text-gray-400">-</span>
            </td>
            <td>{{ formatSize(ver.file_size) }}</td>
            <td>
              <button @click="toggleStatus(ver)" class="badge cursor-pointer" :class="ver.status === 1 ? 'badge-success' : 'badge-secondary'">
                {{ ver.status === 1 ? '已发布' : '已下架' }}
              </button>
            </td>
            <td class="text-gray-500 text-sm">{{ formatDate(ver.created_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button v-if="ver.download_url" @click="copyUrl(ver.download_url)" class="btn btn-sm btn-secondary" title="复制下载链接">
                  复制链接
                </button>
                <button @click="deleteVersion(ver)" class="btn btn-sm btn-danger">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="versions.length === 0">
            <td colspan="11" class="text-center py-12 text-gray-400">
              暂无版本数据
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
          <h3 class="text-lg font-semibold">上传新版本</h3>
          <button @click="showUploadModal = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="space-y-4">
            <div>
              <label class="label">版本名称 <span class="text-red-500">*</span></label>
              <input v-model="uploadForm.version_name" type="text" class="input" placeholder="如：1.0.0" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="label">渠道 <span class="text-red-500">*</span></label>
                <select v-model="uploadForm.channel" class="input">
                  <option value="stable">正式版</option>
                  <option value="beta">测试版</option>
                </select>
              </div>
              <div>
                <label class="label">平台 <span class="text-red-500">*</span></label>
                <select v-model="uploadForm.platform" class="input">
                  <option value="windows">Windows</option>
                  <option value="macos">macOS</option>
                  <option value="linux">Linux</option>
                  <option value="android">Android</option>
                  <option value="ios">iOS</option>
                  <option value="all">全平台</option>
                </select>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="label">架构</label>
                <select v-model="uploadForm.architecture" class="input">
                  <option value="all">全架构</option>
                  <option value="x86_64">x86_64</option>
                  <option value="arm64">arm64</option>
                  <option value="x86">x86</option>
                </select>
              </div>
              <div>
                <label class="label">包类型</label>
                <select v-model="uploadForm.package_type" class="input">
                  <option value="full">完整包</option>
                  <option value="component">组件包</option>
                </select>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <input v-model="uploadForm.is_force_update" type="checkbox" id="force_update" class="w-4 h-4" />
              <label for="force_update" class="text-sm text-gray-700">强制更新</label>
            </div>
            
            <!-- 元数据模式：填写外部下载链接 -->
            <div v-if="appStorageType === 'metadata'" class="space-y-3">
              <div>
                <label class="label">下载链接 <span class="text-red-500">*</span></label>
                <div class="flex gap-2">
                  <input v-model="uploadForm.download_url" type="text" class="input flex-1" placeholder="https://example.com/files/app-v1.0.0.zip" />
                  <button @click="probeFile" type="button" class="btn btn-secondary whitespace-nowrap" :disabled="!uploadForm.download_url || probing">
                    {{ probing ? '探测中...' : '📊 获取大小' }}
                  </button>
                </div>
                <p class="text-sm text-gray-500 mt-1">元数据模式：直接填写外部托管的文件下载链接</p>
              </div>
              
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="label">文件大小（字节）</label>
                  <input v-model="uploadForm.file_size" type="number" class="input" placeholder="可选，例如: 10485760" />
                  <p class="text-xs text-gray-500 mt-1">可选，用于显示文件大小</p>
                </div>
                <div>
                  <label class="label">SHA256 哈希值</label>
                  <input v-model="uploadForm.file_hash" type="text" class="input" placeholder="可选，用于校验" />
                  <p class="text-xs text-gray-500 mt-1">可选，用于文件完整性校验</p>
                </div>
              </div>
            </div>
            
            <!-- S3/WebDAV 模式：上传文件 -->
            <div v-else>
              <label class="label">更新文件 <span class="text-red-500">*</span></label>
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
            
            <div>
              <label class="label">更新说明</label>
              <textarea v-model="uploadForm.release_notes" class="input" rows="4" placeholder="请输入版本更新说明..."></textarea>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showUploadModal = false" class="btn btn-secondary">取消</button>
          <button @click="uploadVersion" class="btn btn-primary" :disabled="uploading">
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
import { success, error, info, confirm as askConfirm } from '../composables/useToast'
import { getVersions, uploadVersionFile, updateVersionStatus, deleteVersion as deleteVersionApi, getApp, probeFileInfo } from '../api'

const route = useRoute()
const router = useRouter()

const appId = route.params.id
const appName = ref('')
const appStorageType = ref('')
const versions = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const channel = ref('')
const showUploadModal = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const probing = ref(false)
const fileInput = ref(null)

const uploadForm = ref({
  version_name: '',
  channel: 'stable',
  platform: 'windows',
  architecture: 'all',
  package_type: 'full',
  is_force_update: false,
  release_notes: '',
  file: null,
  download_url: '',
  file_size: '',
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

async function loadVersions() {
  try {
    const res = await getVersions({
      app_id: appName.value ? null : '',
      page: page.value,
      page_size: pageSize.value,
      channel: channel.value
    })
    const app = await getApp(appId)
    versions.value = res.list.filter(v => v.app_id === app.app_id)
    total.value = versions.value.length
  } catch (err) {
    console.error('Failed to load versions:', err)
    versions.value = []
    total.value = 0
  }
}

function changePage(p) {
  page.value = p
  loadVersions()
}

async function probeFile() {
  if (!uploadForm.value.download_url) {
    info('请先填写下载链接')
    return
  }
  
  probing.value = true
  try {
    const result = await probeFileInfo(uploadForm.value.download_url)
    if (result.file_size) {
      uploadForm.value.file_size = result.file_size.toString()
      info(`成功获取文件大小：${formatSize(result.file_size)}`)
    } else if (result.message) {
      info(result.message)
    }
  } catch (err) {
    error('探测失败：' + (err.response?.data?.error || err.message))
  } finally {
    probing.value = false
  }
}

async function uploadVersion() {
  if (!uploadForm.value.version_name) {
    info('请输入版本名称')
    return
  }
  
  // 元数据模式验证下载链接，其他模式验证文件
  if (appStorageType.value === 'metadata') {
    if (!uploadForm.value.download_url) {
      info('请填写下载链接')
      return
    }
  } else {
    if (!uploadForm.value.file) {
      info('请选择更新文件')
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
    formData.append('version_name', uploadForm.value.version_name)
    formData.append('channel', uploadForm.value.channel)
    formData.append('platform', uploadForm.value.platform)
    formData.append('architecture', uploadForm.value.architecture)
    formData.append('package_type', uploadForm.value.package_type)
    formData.append('is_force_update', uploadForm.value.is_force_update)
    formData.append('release_notes', uploadForm.value.release_notes)
    
    // 元数据模式传递下载链接，其他模式上传文件
    if (appStorageType.value === 'metadata') {
      formData.append('download_url', uploadForm.value.download_url)
      // 可选：添加文件大小和哈希值
      if (uploadForm.value.file_size) {
        formData.append('file_size', uploadForm.value.file_size)
      }
      if (uploadForm.value.file_hash) {
        formData.append('file_hash', uploadForm.value.file_hash)
      }
    } else {
      formData.append('file', uploadForm.value.file)
    }

    await uploadVersionFile(formData, onProgress)
    showUploadModal.value = false
    resetUploadForm()
    loadVersions()
  } catch (err) {
    error('上传失败：' + (err.response?.data?.error || err.message))
  } finally {
    uploading.value = false
  }
}

async function toggleStatus(ver) {
  const newStatus = ver.status === 1 ? 0 : 1
  try {
    await updateVersionStatus(ver.id, newStatus)
    ver.status = newStatus
  } catch (err) {
    error('操作失败：' + (err.response?.data?.error || err.message))
  }
}

async function deleteVersion(ver) {
  if (!(await askConfirm(`确定要删除版本 ${ver.version_name} 吗？`))) {
    return
  }

  try {
    await deleteVersionApi(ver.id)
    loadVersions()
  } catch (err) {
    error('删除失败：' + (err.response?.data?.error || err.message))
  }
}

function copyUrl(url) {
  let fullUrl
  if (url.startsWith('http://') || url.startsWith('https://')) {
    fullUrl = url
  } else if (url.startsWith('/')) {
    // 已带 /api/download/ 前缀
    fullUrl = window.location.origin + url
  } else {
    // 裸 key（历史数据），补全代理入口
    fullUrl = window.location.origin + '/api/download/' + url
  }
  navigator.clipboard.writeText(fullUrl).then(() => {
    success('链接已复制到剪贴板')
  })
}

function resetUploadForm() {
  uploadForm.value = {
    version_name: '',
    channel: 'stable',
    platform: 'windows',
    architecture: 'all',
    package_type: 'full',
    is_force_update: false,
    release_notes: '',
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
  loadVersions()
})
</script>
