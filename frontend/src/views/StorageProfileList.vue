<template>
  <div class="space-y-6">
    <!-- 头部 -->
    <div class="flex justify-between items-center">
      <div>
        <h2 class="text-2xl font-bold text-gray-800">存储管理</h2>
        <p class="text-sm text-gray-500 mt-1">管理可复用的 S3 / WebDAV 存储配置档案，创建应用时可直接选择</p>
      </div>
      <button @click="openCreateModal" class="btn btn-primary">+ 新建档案</button>
    </div>

    <!-- 列表 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200">
      <table class="w-full">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">类型</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">配置摘要</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">创建时间</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="p in profiles" :key="p.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 font-medium text-gray-800">{{ p.name }}</td>
            <td class="px-6 py-4">
              <span :class="storageTypeBadge(p.storage_type)" class="px-2 py-1 text-xs rounded font-medium">
                {{ storageTypeLabel(p.storage_type) }}
              </span>
            </td>
            <td class="px-6 py-4 text-sm text-gray-500 max-w-xs truncate" :title="summarize(p)">
              {{ summarize(p) }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500">{{ formatDate(p.created_at) }}</td>
            <td class="px-6 py-4 text-right space-x-2">
              <button @click="doCheckConnection(p)" :disabled="checkingId === p.id" class="btn btn-sm btn-secondary">
                {{ checkingId === p.id ? '测试中...' : '🔌 连通性' }}
              </button>
              <button @click="openEditModal(p)" class="btn btn-sm btn-secondary">编辑</button>
              <button @click="handleDelete(p)" class="btn btn-sm btn-danger">删除</button>
            </td>
          </tr>
          <tr v-if="profiles.length === 0">
            <td colspan="5" class="px-6 py-12 text-center text-gray-400">暂无存储档案，点击右上角新建</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>

  <!-- 创建/编辑弹窗 -->
  <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="closeModal">
    <div class="bg-white rounded-lg shadow-xl w-[700px] max-h-[92vh] overflow-hidden flex flex-col">
      <div class="p-5 border-b border-gray-200">
        <h3 class="text-lg font-semibold">{{ editingId !== null ? '编辑存储档案' : '新建存储档案' }}</h3>
      </div>
      <div class="p-5 space-y-4 overflow-y-auto flex-1">
        <div>
          <label class="label">档案名称 <span class="text-red-500">*</span></label>
          <input v-model="form.name" type="text" class="input" placeholder="例如：Cloudflare R2 生产桶" />
        </div>
        <div>
          <label class="label">描述</label>
          <textarea v-model="form.description" class="input min-h-[50px]" placeholder="可选，备注用途"></textarea>
        </div>
        <div>
          <label class="label">存储类型 <span class="text-red-500">*</span></label>
          <select v-model="form.storage_type" class="input" @change="onStorageTypeChange">
            <option value="s3">S3 / OSS / COS / R2 对象存储</option>
            <option value="webdav">WebDAV</option>
            <option value="metadata">元数据（仅 URL，无实际存储）</option>
          </select>
        </div>

        <!-- S3 完整表单 -->
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
                <input type="radio" value="redirect" v-model="storageFields.s3.download_mode" class="mr-2" />
                <span class="text-sm">302 重定向（客户端直连 S3）</span>
              </label>
              <label class="flex items-center">
                <input type="radio" value="proxy" v-model="storageFields.s3.download_mode" class="mr-2" />
                <span class="text-sm">系统代理转发（通过后端下载）</span>
              </label>
            </div>
          </div>
          <div v-if="storageFields.s3.download_mode === 'redirect'">
            <label class="label">CDN 基础 URL（可选）</label>
            <input v-model="storageFields.s3.base_url" type="text" class="input" placeholder="https://cdn.example.com" />
            <p class="text-xs text-gray-500 mt-1">填了 CDN 域名后会优先走 CDN；留空则 302 到 S3 原始地址</p>
          </div>
          <div v-else class="bg-blue-50 p-3 rounded">
            <p class="text-sm text-blue-800">✓ 下载请求将通过本系统代理转发</p>
          </div>
          <div>
            <label class="label">路径风格</label>
            <div class="space-y-2">
              <label class="flex items-center">
                <input type="radio" :value="false" v-model="storageFields.s3.use_path_style" class="mr-2" />
                <span class="text-sm">Virtual-Hosted（默认）— bucket 在域名前，OSS/COS/七牛/R2/官方 S3 用这个</span>
              </label>
              <label class="flex items-center">
                <input type="radio" :value="true" v-model="storageFields.s3.use_path_style" class="mr-2" />
                <span class="text-sm">Path-Style — bucket 在路径中，MinIO/Ceph/自建 S3 用这个</span>
              </label>
            </div>
          </div>
          <button type="button" @click="doCheckStorageFields" :disabled="checkingStorage"
            class="w-full py-2 border border-gray-300 rounded text-gray-700 hover:bg-gray-50 disabled:opacity-50">
            {{ checkingStorage ? '测试中...' : '🔌 测试连通性' }}
          </button>
        </div>

        <!-- WebDAV 完整表单 -->
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
            <label class="label">下载方式</label>
            <div class="space-y-2">
              <label class="flex items-center">
                <input type="radio" value="redirect" v-model="storageFields.webdav.download_mode" class="mr-2" />
                <span class="text-sm">302 重定向（客户端直连 WebDAV）</span>
              </label>
              <label class="flex items-center">
                <input type="radio" value="proxy" v-model="storageFields.webdav.download_mode" class="mr-2" />
                <span class="text-sm">系统代理转发（后端代理 BasicAuth）</span>
              </label>
            </div>
            <p v-if="storageFields.webdav.download_mode === 'redirect'" class="text-xs text-orange-600 mt-1">
              ⚠️ 重定向需要 WebDAV 开启匿名读取权限，否则客户端无法下载
            </p>
            <p v-else class="text-xs text-green-700 mt-1">
              ✓ 后端携带 BasicAuth 代理转发，安全但消耗后端带宽
            </p>
          </div>
          <button type="button" @click="doCheckStorageFields" :disabled="checkingStorage"
            class="w-full py-2 border border-gray-300 rounded text-gray-700 hover:bg-gray-50 disabled:opacity-50">
            {{ checkingStorage ? '测试中...' : '🔌 测试连通性' }}
          </button>
        </div>

        <!-- 元数据 -->
        <div v-if="form.storage_type === 'metadata'" class="bg-gray-50 p-4 rounded-lg">
          <p class="text-sm text-gray-600">元数据模式：仅存储版本信息，发布时直接填写外部下载链接。无需额外配置。</p>
        </div>
      </div>
      <div class="p-4 border-t border-gray-200 flex justify-end gap-2">
        <button @click="closeModal" class="btn btn-secondary">取消</button>
        <button @click="handleSave" :disabled="saving" class="btn btn-primary">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import {
  getStorageProfiles, createStorageProfile, updateStorageProfile, deleteStorageProfile,
  checkStorageProfileConnection, checkStorageConnection
} from '../api'
import { success, error, confirm as askConfirm } from '../composables/useToast'

const profiles = ref([])
const showModal = ref(false)
const editingId = ref(null)
const saving = ref(false)
const checkingId = ref(null)
const checkingStorage = ref(false)

const form = reactive({ name: '', description: '', storage_type: 's3' })
const storageFields = reactive({
  s3: { endpoint: '', region: '', bucket: '', access_key: '', secret_key: '', download_mode: 'redirect', base_url: '', use_path_style: false },
  webdav: { url: '', username: '', password: '', download_mode: 'redirect' }
})

function storageTypeLabel(t) {
  return { s3: 'S3 对象存储', webdav: 'WebDAV', metadata: '元数据' }[t] || t
}
function storageTypeBadge(t) {
  return {
    s3: 'bg-blue-100 text-blue-800',
    webdav: 'bg-purple-100 text-purple-800',
    metadata: 'bg-gray-100 text-gray-700'
  }[t] || 'bg-gray-100'
}
function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleString('zh-CN')
}
function summarize(p) {
  if (p.storage_type === 'metadata') return '元数据模式（无配置）'
  try {
    const c = JSON.parse(p.config)
    if (p.storage_type === 's3') {
      return `${c.endpoint || ''} / ${c.bucket || ''}${c.base_url ? ' | CDN:' + c.base_url : ''}`
    }
    if (p.storage_type === 'webdav') return c.url || ''
  } catch {}
  return ''
}

function onStorageTypeChange() {
  // 类型切换时不重置，让用户来回切换保留输入
}

function buildConfigJSON() {
  if (form.storage_type === 's3') return JSON.stringify(storageFields.s3)
  if (form.storage_type === 'webdav') return JSON.stringify(storageFields.webdav)
  return '{}'
}

function parseConfigJSON(json) {
  if (!json) return
  try {
    const raw = JSON.parse(json)
    if (form.storage_type === 's3') Object.assign(storageFields.s3, raw)
    else if (form.storage_type === 'webdav') Object.assign(storageFields.webdav, raw)
  } catch {}
}

function resetFormFields() {
  form.name = ''
  form.description = ''
  form.storage_type = 's3'
  Object.assign(storageFields.s3, { endpoint: '', region: '', bucket: '', access_key: '', secret_key: '', download_mode: 'redirect', base_url: '', use_path_style: false })
  Object.assign(storageFields.webdav, { url: '', username: '', password: '', download_mode: 'redirect' })
}

async function load() {
  try {
    const res = await getStorageProfiles({ page: 1, page_size: 100 })
    profiles.value = res.list || res.data?.list || []
  } catch (e) {
    error('加载失败')
  }
}

function openCreateModal() {
  editingId.value = null
  resetFormFields()
  showModal.value = true
}

function openEditModal(p) {
  editingId.value = p.id
  form.name = p.name
  form.description = p.description || ''
  form.storage_type = p.storage_type
  resetFormFields()
  parseConfigJSON(p.config)
  showModal.value = true
}

function closeModal() { showModal.value = false }

async function handleSave() {
  if (!form.name.trim()) return error('请填写档案名称')
  saving.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description,
      storage_type: form.storage_type,
      config: buildConfigJSON()
    }
    if (editingId.value !== null) {
      await updateStorageProfile(editingId.value, payload)
      success('更新成功')
    } else {
      await createStorageProfile(payload)
      success('创建成功')
    }
    closeModal()
    load()
  } catch (e) {
    error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(p) {
  if (!(await askConfirm(`确定要删除档案 "${p.name}" 吗？（不会影响已使用此档案的 App）`))) return
  try {
    await deleteStorageProfile(p.id)
    success('删除成功')
    load()
  } catch (e) {
    error('删除失败')
  }
}

async function doCheckConnection(p) {
  checkingId.value = p.id
  try {
    const res = await checkStorageProfileConnection(p.id)
    if (res.success) success(res.message || '连通正常')
    else error(res.message || '连通失败')
  } catch (e) {
    error('请求失败')
  } finally {
    checkingId.value = null
  }
}

async function doCheckStorageFields() {
  checkingStorage.value = true
  try {
    const res = await checkStorageConnection({
      storage_type: form.storage_type,
      storage_config: buildConfigJSON()
    })
    if (res.success) success(res.message || '连通正常')
    else error(res.message || '连通失败')
  } catch (e) {
    error('请求失败')
  } finally {
    checkingStorage.value = false
  }
}

onMounted(load)
</script>
