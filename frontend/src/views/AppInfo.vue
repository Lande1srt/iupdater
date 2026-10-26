<template>
  <div>
    <div class="mb-6">
      <button @click="goBack" class="btn btn-secondary mb-4">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        返回应用列表
      </button>
      <h2 class="text-2xl font-bold text-gray-800">应用信息</h2>
      <p class="text-gray-500 mt-1">查看应用详细配置和访问密钥</p>
    </div>

    <div v-if="loading" class="text-center py-12">
      <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      <p class="text-gray-500 mt-2">加载中...</p>
    </div>

    <div v-else-if="app" class="space-y-6">
      <!-- 基本信息 -->
      <div class="card">
        <div class="card-header">
          <h3 class="text-lg font-semibold">基本信息</h3>
        </div>
        <div class="card-body">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label class="text-sm font-medium text-gray-500">应用名称</label>
              <p class="mt-1 text-gray-900 font-medium">{{ app.name }}</p>
            </div>
            <div>
              <label class="text-sm font-medium text-gray-500">App ID</label>
              <p class="mt-1">
                <code class="bg-gray-100 px-3 py-1 rounded text-sm text-primary-600">{{ app.app_id }}</code>
              </p>
            </div>
            <div>
              <label class="text-sm font-medium text-gray-500">应用描述</label>
              <p class="mt-1 text-gray-700">{{ app.description || '暂无描述' }}</p>
            </div>
            <div>
              <label class="text-sm font-medium text-gray-500">图标 URL</label>
              <p class="mt-1 text-gray-700 break-all">{{ app.icon || '暂未设置' }}</p>
            </div>
            <div>
              <label class="text-sm font-medium text-gray-500">存储档案</label>
              <p class="mt-1">
                <template v-if="app.storage_profile">
                  <span class="badge" :class="storageProfileBadge(app.storage_profile.storage_type)">
                    {{ app.storage_profile.name }}
                  </span>
                  <span v-if="app.storage_path" class="text-xs text-gray-500 ml-1">/{{ app.storage_path }}</span>
                </template>
                <template v-else>
                  <span class="badge badge-secondary">元数据（无存储档案）</span>
                </template>
              </p>
            </div>
            <div>
              <label class="text-sm font-medium text-gray-500">创建时间</label>
              <p class="mt-1 text-gray-700">{{ formatDate(app.created_at) }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 访问密钥 -->
      <div class="card border-2" :class="app.enable_auth ? 'border-green-200 bg-green-50' : 'border-gray-200'">
        <div class="card-header flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h3 class="text-lg font-semibold">访问密钥 (Access Key)</h3>
            <span v-if="app.enable_auth" class="badge badge-success">已启用</span>
            <span v-else class="badge badge-secondary">未启用</span>
          </div>
        </div>
        <div class="card-body">
          <div v-if="app.enable_auth">
            <p class="text-sm text-gray-600 mb-4">
              客户端请求更新接口时必须携带此密钥，用于身份验证和访问控制。
            </p>
            <div class="bg-white p-4 rounded-lg border border-gray-200">
              <label class="text-sm font-medium text-gray-500 block mb-2">Access Key</label>
              <div class="flex items-center gap-3">
                <code class="flex-1 bg-gray-50 px-4 py-3 rounded border border-gray-300 text-base font-mono text-gray-800 break-all">
                  {{ showAccessKey ? app.access_key : maskAccessKey(app.access_key) }}
                </code>
                <button @click="toggleAccessKey" class="btn btn-sm btn-secondary whitespace-nowrap">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path v-if="!showAccessKey" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path v-if="!showAccessKey" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    <path v-if="showAccessKey" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                  </svg>
                  {{ showAccessKey ? '隐藏' : '显示' }}
                </button>
                <button @click="copyAccessKey" class="btn btn-sm btn-primary whitespace-nowrap">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  {{ copied ? '已复制' : '复制' }}
                </button>
              </div>
            </div>
            <div class="mt-4 bg-blue-50 p-4 rounded-lg">
              <h4 class="text-sm font-semibold text-blue-900 mb-2">使用方法</h4>
              <p class="text-sm text-blue-800 mb-2">客户端请求时需要在 JSON 数据中包含此密钥：</p>
              <pre class="bg-blue-100 p-3 rounded text-xs overflow-x-auto"><code>{
  "app_id": "{{ app.app_id }}",
  "version_code": 1000,
  "channel": "stable",
  "platform": "windows",
  "architecture": "x86_64",
  "access_key": "{{ app.access_key }}"
}</code></pre>
            </div>
          </div>
          <div v-else class="text-center py-8">
            <svg class="w-16 h-16 mx-auto text-gray-400 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
            <p class="text-gray-500 mb-2">此应用未启用访问校验</p>
            <p class="text-sm text-gray-400">客户端可以直接访问更新接口，无需提供密钥</p>
          </div>
        </div>
      </div>

      <!-- 下载链接生成器 -->
      <div class="card">
        <div class="card-header">
          <h3 class="text-lg font-semibold">最新版本下载链接</h3>
        </div>
        <div class="card-body space-y-4">
          <p class="text-sm text-gray-500">
            按不同平台/架构生成公开访问的最新版本直链（302 重定向）或元数据 JSON。多平台版本组必须选择 platform 参数。
          </p>

          <!-- 生成器控制条 -->
          <div class="bg-gray-50 rounded-lg p-4">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div>
                <label class="text-xs font-medium text-gray-500 block mb-1">发布渠道</label>
                <select v-model="linkChannel" class="form-control">
                  <option value="stable">stable 正式</option>
                  <option value="beta">beta 测试</option>
                  <option value="dev">dev 开发</option>
                </select>
              </div>
              <div>
                <label class="text-xs font-medium text-gray-500 block mb-1">平台</label>
                <select v-model="linkPlatform" class="form-control">
                  <option value="">自动（仅单平台版本可用）</option>
                  <option value="all">all</option>
                  <option value="windows">windows</option>
                  <option value="linux">linux</option>
                  <option value="darwin">darwin</option>
                  <option value="android">android</option>
                  <option value="ios">ios</option>
                </select>
              </div>
              <div>
                <label class="text-xs font-medium text-gray-500 block mb-1">架构</label>
                <select v-model="linkArch" class="form-control">
                  <option value="">自动</option>
                  <option value="all">all</option>
                  <option value="x86_64">x86_64</option>
                  <option value="amd64">amd64</option>
                  <option value="aarch64">aarch64</option>
                  <option value="arm64">arm64</option>
                  <option value="i386">i386</option>
                </select>
              </div>
            </div>
          </div>

          <!-- 链接输出 -->
          <div class="space-y-3">
            <!-- getlink 302 -->
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-xs font-medium text-gray-600">① 直接下载（302 重定向）</label>
                <button @click="copyText(publicLink('getlink'))" class="btn btn-xs btn-secondary">复制</button>
              </div>
              <code class="block bg-gray-900 text-green-300 px-3 py-2 rounded text-xs font-mono break-all select-all">{{ publicLink('getlink') }}</code>
            </div>

            <!-- 最新版本元数据 JSON -->
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-xs font-medium text-gray-600">② 最新版本元数据</label>
                <button @click="copyText(publicLink('info'))" class="btn btn-xs btn-secondary">复制</button>
              </div>
              <code class="block bg-gray-900 text-green-300 px-3 py-2 rounded text-xs font-mono break-all select-all">{{ publicLink('info') }}</code>
            </div>

            <!-- 带 Access Key -->
            <div v-if="app?.enable_auth">
              <div class="flex items-center justify-between mb-1">
                <label class="text-xs font-medium text-orange-600">③ 带 Access Key 版本</label>
                <button @click="copyText(publicLink('info', true))" class="btn btn-xs bg-orange-50 text-orange-700 hover:bg-orange-100">复制</button>
              </div>
              <code class="block bg-orange-50 text-orange-900 border border-orange-200 px-3 py-2 rounded text-xs font-mono break-all select-all">{{ publicLink('info', true) }}</code>
            </div>

            <!-- 列出全部版本元数据（新） -->
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-xs font-medium text-gray-600">④ 列出所有版本元数据</label>
                <button @click="copyText(publicLink('listmeta'))" class="btn btn-xs btn-secondary">复制</button>
              </div>
              <code class="block bg-gray-900 text-green-300 px-3 py-2 rounded text-xs font-mono break-all select-all">{{ publicLink('listmeta') }}</code>
            </div>
          </div>

          <!-- 多平台警告 -->
          <div v-if="!linkPlatform && !linkArch" class="bg-yellow-50 border border-yellow-200 rounded p-3">
            <div class="text-xs text-yellow-800">
              ⚠ 未选择 platform / architecture 参数时，若最新版本组只有一个平台会直接返回；
              若包含多平台，接口返回 <code class="bg-yellow-100 px-1 rounded">409 Conflict</code> 并提示所有可用平台，请在上表选择 platform。
            </div>
          </div>
        </div>
      </div>

      <!-- 存储配置 -->
      <div class="card">
        <div class="card-header">
          <h3 class="text-lg font-semibold">存储配置</h3>
        </div>
        <div class="card-body">
          <template v-if="app.storage_profile">
            <div class="mb-4">
              <div class="flex items-center gap-2 mb-2">
                <span class="badge" :class="storageProfileBadge(app.storage_profile.storage_type)">{{ storageProfileLabel(app.storage_profile.storage_type) }}</span>
                <span class="text-sm font-medium text-gray-700">{{ app.storage_profile.name }}</span>
              </div>
              <p v-if="app.storage_profile.description" class="text-sm text-gray-500">{{ app.storage_profile.description }}</p>
            </div>

            <!-- S3 -->
            <div v-if="app.storage_profile.storage_type === 's3'" class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="text-sm font-medium text-gray-500">Endpoint</label>
                <p class="mt-1 text-gray-700 break-all">{{ s3Cfg.endpoint || '-' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">Region</label>
                <p class="mt-1 text-gray-700">{{ s3Cfg.region || '-' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">Bucket</label>
                <p class="mt-1 text-gray-700">{{ s3Cfg.bucket || '-' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">下载方式</label>
                <p class="mt-1 text-gray-700">{{ s3Cfg.download_mode === 'redirect' ? '302 重定向' : '系统代理转发' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">CDN 基础 URL</label>
                <p class="mt-1 text-gray-700 break-all">{{ s3Cfg.base_url || '未设置' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">基础路径</label>
                <p class="mt-1 text-gray-700">
                  {{ app.storage_path || s3Cfg.base_path || '根目录' }}
                  <span v-if="app.storage_path && s3Cfg.base_path" class="text-xs text-gray-400">（覆盖档案默认 {{ s3Cfg.base_path }}）</span>
                </p>
              </div>
            </div>

            <!-- WebDAV -->
            <div v-else-if="app.storage_profile.storage_type === 'webdav'" class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="text-sm font-medium text-gray-500">WebDAV URL</label>
                <p class="mt-1 text-gray-700 break-all">{{ webdavCfg.url || '-' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">用户名</label>
                <p class="mt-1 text-gray-700">{{ webdavCfg.username || '-' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">下载方式</label>
                <p class="mt-1 text-gray-700">{{ webdavCfg.download_mode === 'redirect' ? '302 重定向' : '系统代理转发' }}</p>
              </div>
              <div>
                <label class="text-sm font-medium text-gray-500">基础路径</label>
                <p class="mt-1 text-gray-700">
                  {{ app.storage_path || webdavCfg.base_path || '/' }}
                  <span v-if="app.storage_path && webdavCfg.base_path" class="text-xs text-gray-400">（覆盖档案默认 {{ webdavCfg.base_path }}）</span>
                </p>
              </div>
            </div>
          </template>
          <div v-else class="text-center py-6 text-gray-500">
            <svg class="w-10 h-10 mx-auto mb-2 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <p>此应用为元数据模式</p>
            <p class="text-xs text-gray-400 mt-1">发布版本时填写外部下载链接即可</p>
          </div>
        </div>
      </div>

      <!-- 操作区 -->
      <div class="card">
        <div class="card-body">
          <div class="flex gap-3">
            <button @click="goToVersions" class="btn btn-primary">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
              </svg>
              版本管理
            </button>
            <button @click="goToComponents" class="btn btn-primary">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
              组件管理
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="card">
      <div class="card-body text-center py-12">
        <p class="text-gray-500">应用不存在</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getApp } from '../api'
import { success, error, info } from '../composables/useToast'
const router = useRouter()
const route = useRoute()

const app = ref(null)
const loading = ref(true)
const showAccessKey = ref(false)
const copied = ref(false)

// 下载链接生成器状态
const linkChannel = ref('stable')
const linkPlatform = ref('')
const linkArch = ref('')

// 通过 window.location 取主控绝对 URL（前端是主控静态文件，同源）
const siteOrigin = computed(() => window.location.origin)

function buildQuery(includeAccessKey) {
  const params = new URLSearchParams()
  if (linkChannel.value && linkChannel.value !== 'stable') params.set('channel', linkChannel.value)
  if (linkPlatform.value) params.set('platform', linkPlatform.value)
  if (linkArch.value) params.set('architecture', linkArch.value)
  if (includeAccessKey && app.value?.access_key) params.set('access_key', app.value.access_key)
  const q = params.toString()
  return q ? '?' + q : ''
}

function publicLink(kind, includeAccessKey = false) {
  if (!app.value) return '...'
  const appid = encodeURIComponent(app.value.app_id)
  let path
  switch (kind) {
    case 'getlink':  path = `/u/getlink/${appid}`; break       // 302 重定向下载
    case 'listmeta': path = `/u/listmeta/${appid}`; break      // 列出所有版本元数据
    default:         path = `/u/getmeta/${appid}`              // 最新版本元数据 JSON（重写旧 /u/:appid）
  }
  return siteOrigin.value + path + buildQuery(includeAccessKey)
}

function copyText(text) {
  navigator.clipboard.writeText(text).then(() => {
    success('已复制到剪贴板')
  }).catch(err => {
    error('复制失败：' + err.message)
  })
}

const storageProfileConfig = computed(() => {
  if (!app.value?.storage_profile?.config) return null
  try { return JSON.parse(app.value.storage_profile.config) } catch { return null }
})
const s3Cfg = computed(() => storageProfileConfig.value || {})
const webdavCfg = computed(() => storageProfileConfig.value || {})

function storageProfileLabel(type) {
  return { s3: 'S3 对象存储', webdav: 'WebDAV', metadata: '元数据' }[type] || type
}
function storageProfileBadge(type) {
  return { s3: 'bg-blue-100 text-blue-800', webdav: 'bg-purple-100 text-purple-800', metadata: 'bg-gray-100 text-gray-700' }[type] || 'bg-gray-100'
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

function maskAccessKey(key) {
  if (!key) return ''
  if (key.length <= 8) return '********'
  return key.substring(0, 4) + '********' + key.substring(key.length - 4)
}

function toggleAccessKey() {
  showAccessKey.value = !showAccessKey.value
}

function copyAccessKey() {
  if (!app.value?.access_key) return
  navigator.clipboard.writeText(app.value.access_key).then(() => {
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }).catch(err => {
    error('复制失败：' + err.message)
  })
}

function goBack() {
  router.push('/apps')
}

function goToVersions() {
  router.push(`/apps/${app.value.id}/versions`)
}

function goToComponents() {
  router.push(`/apps/${app.value.id}/components`)
}

async function loadApp() {
  loading.value = true
  try {
    const res = await getApp(route.params.id)
    app.value = res
  } catch (err) {
    error('加载失败：' + (err.response?.data?.error || err.message))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadApp()
})
</script>
