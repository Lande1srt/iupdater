<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-gray-800">应用管理</h2>
        <p class="text-gray-500 mt-1">管理所有应用及其版本更新</p>
      </div>
      <button @click="showCreateModal = true" class="btn btn-primary">
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
            <input v-model="keyword" type="text" class="input" placeholder="搜索应用名称或App ID" @keyup.enter="loadApps" />
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
            <th>存储类型</th>
            <th>访问校验</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="app in apps" :key="app.id">
            <td class="text-gray-500">{{ app.id }}</td>
            <td>
              <code class="bg-gray-100 px-2 py-1 rounded text-sm text-primary-600">{{ app.app_id }}</code>
            </td>
            <td class="font-medium">{{ app.name }}</td>
            <td>
              <span class="badge" :class="getStorageBadgeClass(app.storage_type)">
                {{ getStorageTypeLabel(app.storage_type) }}
              </span>
            </td>
            <td>
              <span v-if="app.enable_auth" class="badge badge-success" :title="'AccessKey: ' + app.access_key">
                已启用
              </span>
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
            <td colspan="7" class="text-center py-12 text-gray-400">
              暂无应用数据
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

    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="text-lg font-semibold">{{ editingApp ? '编辑应用' : '新建应用' }}</h3>
          <button @click="showCreateModal = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="space-y-4">
            <div>
              <label class="label">App ID <span class="text-red-500">*</span></label>
              <input v-model="form.app_id" type="text" class="input" :disabled="!!editingApp" placeholder="如：com.example.myapp" />
              <p class="text-xs text-gray-500 mt-1">应用唯一识别码，创建后不可修改</p>
            </div>
            <div>
              <label class="label">应用名称 <span class="text-red-500">*</span></label>
              <input v-model="form.name" type="text" class="input" placeholder="请输入应用名称" />
            </div>
            <div>
              <label class="label">应用描述</label>
              <textarea v-model="form.description" class="input" rows="3" placeholder="请输入应用描述"></textarea>
            </div>
            <div>
              <label class="label">图标 URL</label>
              <input v-model="form.icon" type="text" class="input" placeholder="请输入图标链接" />
            </div>
            
            <div class="bg-blue-50 p-4 rounded-lg">
              <div class="flex items-center gap-2">
                <input v-model="form.enable_auth" type="checkbox" id="enable_auth" class="w-4 h-4" />
                <label for="enable_auth" class="text-sm font-medium text-gray-700">启用访问校验</label>
              </div>
              <p class="text-xs text-gray-600 mt-2">启用后，客户端请求必须携带 AccessKey 才能访问更新接口</p>
            </div>
            
            <div>
              <label class="label">存储类型 <span class="text-red-500">*</span></label>
              <select v-model="form.storage_type" class="input" @change="onStorageTypeChange">
                <option value="metadata">元数据（本地存储）</option>
                <option value="s3">S3 对象存储</option>
                <option value="webdav">WebDAV</option>
              </select>
            </div>
            
            <!-- 元数据存储配置 -->
            <div v-if="form.storage_type === 'metadata'" class="bg-gray-50 p-4 rounded-lg space-y-3">
              <p class="text-sm text-gray-600">元数据模式：仅存储版本信息，发布时直接填写外部下载链接</p>
            </div>
            
            <!-- S3 存储配置 -->
            <div v-if="form.storage_type === 's3'" class="bg-gray-50 p-4 rounded-lg space-y-3">
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="label">Endpoint <span class="text-red-500">*</span></label>
                  <input v-model="storageFields.s3.endpoint" type="text" class="input" placeholder="https://s3.example.com" />
                </div>
                <div>
                  <label class="label">Region</label>
                  <input v-model="storageFields.s3.region" type="text" class="input" placeholder="us-east-1" />
                </div>
              </div>
              <div>
                <label class="label">Bucket <span class="text-red-500">*</span></label>
                <input v-model="storageFields.s3.bucket" type="text" class="input" placeholder="my-bucket" />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="label">Access Key <span class="text-red-500">*</span></label>
                  <input v-model="storageFields.s3.access_key" type="text" class="input" placeholder="your-access-key" />
                </div>
                <div>
                  <label class="label">Secret Key <span class="text-red-500">*</span></label>
                  <input v-model="storageFields.s3.secret_key" type="password" class="input" placeholder="your-secret-key" />
                </div>
              </div>
              <div>
                <label class="label">下载方式</label>
                <div class="space-y-2">
                  <label class="flex items-center">
                    <input type="radio" :value="false" v-model="storageFields.s3.use_proxy" class="mr-2" />
                    <span class="text-sm">外部CDN直连</span>
                  </label>
                  <label class="flex items-center">
                    <input type="radio" :value="true" v-model="storageFields.s3.use_proxy" class="mr-2" />
                    <span class="text-sm">系统代理转发</span>
                  </label>
                </div>
              </div>
              <div v-if="!storageFields.s3.use_proxy">
                <label class="label">CDN 基础 URL</label>
                <input v-model="storageFields.s3.base_url" type="text" class="input" placeholder="https://cdn.example.com" />
                <p class="text-xs text-gray-500 mt-1">外部CDN地址，留空使用S3直连</p>
              </div>
              <div v-else class="bg-blue-50 p-3 rounded">
                <p class="text-sm text-blue-800">✓ 下载请求将通过本系统代理转发</p>
              </div>
            </div>
            
            <!-- WebDAV 存储配置 -->
            <div v-if="form.storage_type === 'webdav'" class="bg-gray-50 p-4 rounded-lg space-y-3">
              <div>
                <label class="label">WebDAV URL <span class="text-red-500">*</span></label>
                <input v-model="storageFields.webdav.url" type="text" class="input" placeholder="https://dav.example.com" />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="label">用户名 <span class="text-red-500">*</span></label>
                  <input v-model="storageFields.webdav.username" type="text" class="input" placeholder="username" />
                </div>
                <div>
                  <label class="label">密码 <span class="text-red-500">*</span></label>
                  <input v-model="storageFields.webdav.password" type="password" class="input" placeholder="password" />
                </div>
              </div>
              <div>
                <label class="label">基础路径</label>
                <input v-model="storageFields.webdav.base_path" type="text" class="input" placeholder="/uploads" />
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showCreateModal = false" class="btn btn-secondary">取消</button>
          <button @click="saveApp" class="btn btn-primary" :disabled="saving">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getApps, createApp, updateApp, deleteApp } from '../api'

const router = useRouter()

const apps = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const showCreateModal = ref(false)
const editingApp = ref(null)
const saving = ref(false)

const form = ref({
  app_id: '',
  name: '',
  description: '',
  icon: '',
  storage_type: 'metadata',
  storage_config: '',
  enable_auth: false
})

const storageFields = ref({
  metadata: {
    base_url: ''
  },
  s3: {
    endpoint: '',
    access_key: '',
    secret_key: '',
    bucket: '',
    region: 'us-east-1',
    base_url: '',
    use_proxy: false
  },
  webdav: {
    url: '',
    username: '',
    password: '',
    base_path: '/uploads'
  }
})

const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

function getStorageTypeLabel(type) {
  const map = {
    s3: 'S3 对象存储',
    webdav: 'WebDAV',
    metadata: '元数据（本地）'
  }
  return map[type] || type
}

function getStorageBadgeClass(type) {
  const map = {
    s3: 'badge-info',
    webdav: 'badge-warning',
    metadata: 'badge-success'
  }
  return map[type] || 'badge-secondary'
}

function onStorageTypeChange() {
  // 切换存储类型时重置对应配置
  storageFields.value[form.value.storage_type] = getDefaultStorageFields(form.value.storage_type)
}

function getDefaultStorageFields(type) {
  if (type === 's3') {
    return { endpoint: '', access_key: '', secret_key: '', bucket: '', region: 'us-east-1', base_url: '' }
  } else if (type === 'webdav') {
    return { url: '', username: '', password: '', base_path: '/uploads' }
  } else {
    return { base_url: '' }
  }
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

async function loadApps() {
  try {
    const res = await getApps({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value
    })
    apps.value = res.list
    total.value = res.total
  } catch (err) {
    alert('加载失败：' + (err.response?.data?.error || err.message))
  }
}

function changePage(p) {
  page.value = p
  loadApps()
}

function editApp(app) {
  editingApp.value = app
  form.value = {
    app_id: app.app_id,
    name: app.name,
    description: app.description || '',
    icon: app.icon || '',
    storage_type: app.storage_type,
    storage_config: app.storage_config || '',
    enable_auth: app.enable_auth || false
  }
  
  // 解析 storage_config JSON 到表单字段
  if (app.storage_config) {
    try {
      const config = JSON.parse(app.storage_config)
      if (app.storage_type === 's3') {
        storageFields.value.s3 = {
          endpoint: config.endpoint || '',
          access_key: config.access_key || '',
          secret_key: config.secret_key || '',
          bucket: config.bucket || '',
          region: config.region || 'us-east-1',
          base_url: config.base_url || ''
        }
      } else if (app.storage_type === 'webdav') {
        storageFields.value.webdav = {
          url: config.url || '',
          username: config.username || '',
          password: config.password || '',
          base_path: config.base_path || '/uploads'
        }
      } else if (app.storage_type === 'metadata') {
        storageFields.value.metadata = {
          base_url: config.base_url || ''
        }
      }
    } catch (e) {
      console.error('Failed to parse storage_config:', e)
    }
  }
  
  showCreateModal.value = true
}

async function saveApp() {
  if (!form.value.app_id || !form.value.name) {
    alert('请填写 App ID 和应用名称')
    return
  }

  // 将表单字段转换为 JSON 配置
  const storageConfig = storageFields.value[form.value.storage_type]
  form.value.storage_config = JSON.stringify(storageConfig)

  saving.value = true
  try {
    if (editingApp.value) {
      await updateApp(editingApp.value.id, form.value)
    } else {
      await createApp(form.value)
    }
    showCreateModal.value = false
    resetForm()
    loadApps()
  } catch (err) {
    alert('保存失败：' + (err.response?.data?.error || err.message))
  } finally {
    saving.value = false
  }
}

async function deleteAppItem(app) {
  if (!confirm(`确定要删除应用 "${app.name}" 吗？相关版本和组件也会被删除。`)) {
    return
  }

  try {
    await deleteApp(app.id)
    loadApps()
  } catch (err) {
    alert('删除失败：' + (err.response?.data?.error || err.message))
  }
}

function resetForm() {
  editingApp.value = null
  form.value = {
    app_id: '',
    name: '',
    description: '',
    icon: '',
    storage_type: 'metadata',
    storage_config: '',
    enable_auth: false
  }
  
  // 重置所有存储字段
  storageFields.value = {
    metadata: { base_url: '' },
    s3: { endpoint: '', access_key: '', secret_key: '', bucket: '', region: 'us-east-1', base_url: '' },
    webdav: { url: '', username: '', password: '', base_path: '/uploads' }
  }
}

function goToInfo(app) {
  router.push(`/apps/${app.id}/info`)
}

function goToVersions(app) {
  router.push(`/apps/${app.id}/versions`)
}

function goToComponents(app) {
  router.push(`/apps/${app.id}/components`)
}

onMounted(() => {
  loadApps()
})
</script>
