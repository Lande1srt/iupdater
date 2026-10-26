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
      <div class="flex gap-2">
        <button @click="loadData" class="btn btn-secondary">刷新</button>
        <button @click="openUploadModal()" class="btn btn-primary">
          <svg class="w-5 h-5 inline -mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          上传新版本
        </button>
      </div>
    </div>

    <div class="card mb-6">
      <div class="card-body">
        <div class="flex gap-4">
          <div class="w-48">
            <label class="label">渠道</label>
            <select v-model="channel" class="input" @change="loadGroups">
              <option value="">全部渠道</option>
              <option value="stable">正式版</option>
              <option value="beta">测试版</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- 分组显示 -->
    <div class="card">
      <table class="table">
        <thead>
          <tr>
            <th class="w-8"></th>
            <th>版本组</th>
            <th>最新版本号</th>
            <th>渠道</th>
            <th>包类型</th>
            <th>平台数量</th>
            <th>各平台列表</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="g in groups" :key="g.version_name + '|' + g.channel + '|' + g.package_type">
            <!-- 组主行（不再折叠，改为点按钮弹窗） -->
            <tr class="group-header hover:bg-gray-50">
              <td>
                <button @click="openGroupDetail(g)" class="btn btn-sm btn-secondary whitespace-nowrap">查看详情</button>
              </td>
              <td class="font-semibold text-gray-800 text-base">
                {{ g.version_name }}
                <span class="text-xs text-gray-400 ml-2">({{ g.versions.length }} 个平台)</span>
              </td>
              <td class="font-mono text-primary-600">{{ g.latest_version_code }}</td>
              <td>
                <span class="badge" :class="g.channel === 'stable' ? 'badge-success' : 'badge-warning'">
                  {{ g.channel === 'stable' ? '正式版' : '测试版' }}
                </span>
              </td>
              <td>
                <span class="badge" :class="g.package_type === 'full' ? 'badge-info' : 'badge-secondary'">
                  {{ g.package_type === 'full' ? '完整包' : '组件包' }}
                </span>
              </td>
              <td>{{ g.versions.length }}</td>
              <td class="text-xs text-gray-500">
                <span v-for="(v, i) in g.versions" :key="v.id">
                  {{ v.platform }}/{{ v.architecture }}
                  <span v-if="i < g.versions.length - 1" class="text-gray-300">, </span>
                </span>
              </td>
              <td>
                <button @click="openUploadModal(g)" class="btn btn-sm btn-secondary">+ 追加平台</button>
              </td>
            </tr>
          </template>
          <tr v-if="groups.length === 0">
            <td colspan="8" class="text-center py-12 text-gray-400">
              暂无版本数据，点击右上角「上传新版本」开始
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 新建/追加弹窗 -->
    <div v-if="showUploadModal" class="modal-overlay" @click.self="showUploadModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3 class="text-lg font-semibold">
            {{ appendToGroup ? `向「${appendToGroup.version_name}」追加平台` : '上传新版本' }}
          </h3>
          <button @click="showUploadModal = false" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="space-y-4">

            <!-- 模式切换（只有非追加场景才显示） -->
            <div v-if="!appendToGroup" class="flex gap-2 border border-gray-200 rounded p-1">
              <button
                @click="form.mode = 'new'"
                class="flex-1 py-2 text-sm rounded"
                :class="form.mode === 'new' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-50'"
              >
                🆕 新建版本组
              </button>
              <button
                @click="form.mode = 'existing'"
                class="flex-1 py-2 text-sm rounded"
                :class="form.mode === 'existing' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-50'"
              >
                📚 追加到已有版本
              </button>
            </div>

            <!-- 版本名称：两种模式 -->
            <div>
              <label class="label">版本名称 <span class="text-red-500">*</span></label>
              <!-- 追加模式：锁定已选择的组 -->
              <input v-if="appendToGroup" :value="appendToGroup.version_name" disabled class="input bg-gray-50 text-gray-600" />

              <!-- 新建模式 / 手动 -->
              <input v-else-if="form.mode === 'new'" v-model="form.version_name" type="text" class="input" placeholder="如：1.0.0" />

              <!-- 选择已有组 -->
              <select v-else class="input" @change="onGroupSelected" v-model="form.selected_group_key">
                <option value="">请选择已有版本...</option>
                <option v-for="g in existingGroups" :key="gKey(g)" :value="gKey(g)">
                  {{ g.version_name }}  ({{ g.channel === 'stable' ? '正式' : '测试' }}, {{ g.package_type === 'full' ? '完整' : '组件' }}, {{ g.versions.length }} 平台)
                </option>
              </select>

              <!-- 选择已有组时显示已有的平台列表 -->
              <div v-if="(appendToGroup || form.mode === 'existing') && currentGroup" class="mt-2 text-xs text-gray-500">
                已发布平台：
                <span v-for="(v, i) in currentGroup.versions" :key="v.id">
                  <span class="bg-gray-100 px-1.5 py-0.5 rounded mr-1 text-gray-600">{{ v.platform }}/{{ v.architecture }}</span>
                </span>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="label">渠道 <span class="text-red-500">*</span></label>
                <select v-model="form.channel" class="input">
                  <option value="stable">正式版</option>
                  <option value="beta">测试版</option>
                </select>
              </div>
              <div>
                <label class="label">平台 <span class="text-red-500">*</span></label>
                <select v-model="form.platform" class="input">
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
                <select v-model="form.architecture" class="input">
                  <option value="all">全架构</option>
                  <option value="x86_64">x86_64</option>
                  <option value="amd64">amd64</option>
                  <option value="arm64">arm64</option>
                  <option value="x86">x86</option>
                </select>
              </div>
              <div>
                <label class="label">包类型</label>
                <select v-model="form.package_type" class="input">
                  <option value="full">完整包</option>
                  <option value="component">组件包</option>
                </select>
              </div>
            </div>

            <!-- 同平台同架构提前预警 -->
            <div v-if="existingGroupConflict" class="bg-orange-50 border border-orange-200 rounded p-3 text-sm text-orange-800">
              <div class="font-semibold mb-1">⚠ 同版本 + 同平台 + 同架构 已存在</div>
              <div class="text-xs">版本号 {{ existingGroupConflict.version_code }} 已发布过 {{ existingGroupConflict.platform }}/{{ existingGroupConflict.architecture }}</div>
              <div class="text-xs mt-1">后端会返回 409 拒绝，请删除旧版本或改平台/架构</div>
            </div>

            <div class="flex items-center gap-2">
              <input v-model="form.is_force_update" type="checkbox" id="force_update" class="w-4 h-4" />
              <label for="force_update" class="text-sm text-gray-700">强制更新</label>
            </div>

            <!-- 元数据模式 -->
            <div v-if="appStorageType === 'metadata'" class="space-y-3">
              <div>
                <label class="label">下载链接 <span class="text-red-500">*</span></label>
                <div class="flex gap-2">
                  <input v-model="form.download_url" type="text" class="input flex-1" placeholder="https://example.com/files/app-v1.0.0.zip" />
                  <button @click="probeFile" type="button" class="btn btn-secondary whitespace-nowrap" :disabled="!form.download_url || probing">
                    {{ probing ? '探测中...' : '📊 获取大小' }}
                  </button>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="label">文件大小（字节）</label>
                  <input v-model="form.file_size" type="number" class="input" placeholder="可选" />
                </div>
                <div>
                  <label class="label">SHA256 哈希值</label>
                  <input v-model="form.file_hash" type="text" class="input" placeholder="可选" />
                </div>
              </div>
            </div>

            <!-- S3/WebDAV 模式 -->
            <div v-else>
              <label class="label">更新文件 <span class="text-red-500">*</span></label>
              <input ref="fileInput" type="file" class="input" @change="handleFileChange" />
              <p v-if="form.file" class="text-sm text-gray-500 mt-1">
                已选择：{{ form.file.name }} ({{ formatSize(form.file.size) }})
              </p>
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
              <textarea v-model="form.release_notes" class="input" rows="3" placeholder="可选"></textarea>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showUploadModal = false" class="btn btn-secondary">取消</button>
          <button @click="submitUpload" class="btn btn-primary" :disabled="uploading || !!existingGroupConflict">
            {{ uploading ? '上传中...' : (appendToGroup ? '追加发布' : '上传发布') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 版本组详情弹窗（替换折叠块） -->
    <div v-if="detailGroup" class="modal-overlay" @click.self="detailGroup = null">
      <div class="modal-content modal-ultrawide">
        <div class="modal-header">
          <div class="flex items-center gap-3">
            <h3 class="text-lg font-semibold">版本组详情</h3>
            <span class="badge" :class="detailGroup.channel === 'stable' ? 'badge-success' : 'badge-warning'">
              {{ detailGroup.channel === 'stable' ? '正式版' : '测试版' }}
            </span>
            <span class="badge" :class="detailGroup.package_type === 'full' ? 'badge-info' : 'badge-secondary'">
              {{ detailGroup.package_type === 'full' ? '完整包' : '组件包' }}
            </span>
            <span class="text-sm text-gray-400">{{ detailGroup.versions.length }} 个平台 · 最新 code {{ detailGroup.latest_version_code }}</span>
          </div>
          <button @click="detailGroup = null" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body p-0">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 text-gray-600 text-xs">
              <tr>
                <th class="text-left font-medium py-2 px-3">code</th>
                <th class="text-left font-medium py-2 px-3">平台 / 架构</th>
                <th class="text-left font-medium py-2 px-3">大小</th>
                <th class="text-left font-medium py-2 px-3">状态</th>
                <th class="text-left font-medium py-2 px-3">强制</th>
                <th class="text-left font-medium py-2 px-3">发布时间</th>
                <th class="text-right font-medium py-2 px-3">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="v in detailGroup.versions" :key="v.id" class="border-t border-gray-100 hover:bg-gray-50">
                <td class="py-2 px-3 font-mono text-primary-600">{{ v.version_code }}</td>
                <td class="py-2 px-3 text-gray-700">{{ v.platform }} / {{ v.architecture }}</td>
                <td class="py-2 px-3 text-gray-600">{{ formatSize(v.file_size) }}</td>
                <td class="py-2 px-3">
                  <button @click="toggleStatus(v)" class="badge cursor-pointer" :class="v.status === 1 ? 'badge-success' : 'badge-secondary'">
                    {{ v.status === 1 ? '已发布' : '已下架' }}
                  </button>
                </td>
                <td class="py-2 px-3">
                  <span v-if="v.is_force_update" class="badge badge-danger">强制</span>
                  <span v-else class="text-gray-300">-</span>
                </td>
                <td class="py-2 px-3 text-gray-400 text-xs">{{ formatDate(v.created_at) }}</td>
                <td class="py-2 px-3 text-right">
                  <div class="flex gap-2 justify-end">
                    <button v-if="v.download_url" @click="copyUrl(v.download_url)" class="btn btn-sm btn-secondary">复制链接</button>
                    <button @click="deleteVersion(v); if(detailGroup) detailGroup = groups.find(g => gKey(g) === gKey(detailGroup)) || null" class="btn btn-sm btn-danger">删除</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!detailGroup.versions || detailGroup.versions.length === 0">
                <td colspan="7" class="text-center py-8 text-gray-400 text-sm">该版本组已无可用版本</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="modal-footer">
          <button @click="openUploadModal(detailGroup); detailGroup = null" class="btn btn-primary">+ 向此版本组追加平台</button>
          <button @click="detailGroup = null" class="btn btn-secondary">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { success, error, info, confirm as askConfirm } from '../composables/useToast'
import { getVersionGroups, uploadVersionFile, updateVersionStatus, deleteVersion as deleteVersionApi, getApp, probeFileInfo } from '../api'

const route = useRoute()
const router = useRouter()

const appId = route.params.id
const appName = ref('')
const appStorageType = ref('')
const groups = ref([])
const channel = ref('')
const showUploadModal = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const probing = ref(false)
const fileInput = ref(null)
// 弹窗内：appendToGroup 表示从列表行的「追加平台」按钮打开
const appendToGroup = ref(null)
const existingGroups = ref([])   // 已加载的所有组（用来给「选择已有组」下拉框）
const detailGroup = ref(null)    // 当前打开详情弹窗的版本组

const form = ref({
  mode: 'new',            // 'new' | 'existing'
  selected_group_key: '', // 选择已有组时
  version_name: '',       // 新建时
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

// 工具：组 key / 当前选择组 / 冲突检测
function gKey(g) {
  return g.channel + '||' + g.package_type + '||' + g.version_name
}
const currentGroup = computed(() => {
  if (appendToGroup.value) return appendToGroup.value
  const k = form.value.selected_group_key
  if (!k) return null
  return existingGroups.value.find(g => gKey(g) === k)
})
const existingGroupConflict = computed(() => {
  const g = currentGroup.value
  if (!g) return null
  return g.versions.find(v =>
    v.platform === form.value.platform &&
    v.architecture === form.value.architecture &&
    v.package_type === form.value.package_type
  ) || null
})
function onGroupSelected() {
  // 选已有组时，自动同步 channel / package_type（保持一致）
  const g = currentGroup.value
  if (!g) return
  form.value.channel = g.channel
  form.value.package_type = g.package_type
  // version_name 由组锁定，提交时强制用组里的值
}
function effectiveVersionName() {
  if (appendToGroup.value) return appendToGroup.value.version_name
  if (form.value.mode === 'existing') {
    const g = currentGroup.value
    return g ? g.version_name : form.value.version_name
  }
  return form.value.version_name
}

watch([channel], () => loadGroups())

function formatDate(v) { if (!v) return ''; return new Date(v).toLocaleString('zh-CN') }
function formatSize(b) {
  if (!b) return '-'
  if (b < 1024) return b + ' B'
  if (b < 1024*1024) return (b/1024).toFixed(1) + ' KB'
  if (b < 1024*1024*1024) return (b/1024/1024).toFixed(1) + ' MB'
  return (b/1024/1024/1024).toFixed(1) + ' GB'
}
function handleFileChange(e) { form.value.file = e.target.files[0] }
function goBack() { router.push('/apps') }
function openGroupDetail(g) { detailGroup.value = g }

async function loadAppInfo() {
  try {
    const app = await getApp(appId)
    appName.value = app.name
    appStorageType.value = app.storage_profile_id ? app.storage_type : 'metadata'
    // 某些旧记录可能没带 storage_type，用 metadata 兜底
  } catch {}
}

async function loadGroups() {
  try {
    const app = await getApp(appId)
    const res = await getVersionGroups({
      app_id: app.app_id,
      channel: channel.value
    })
    groups.value = res.list || []
    // 同步完整列表给「选择已有组」下拉（不带 channel 过滤）
    const fullRes = await getVersionGroups({ app_id: app.app_id })
    existingGroups.value = fullRes.list || []
    // 如果详情弹窗开着，跟随刷新版本组数据
    if (detailGroup.value) {
      const fresh = groups.value.find(g => gKey(g) === gKey(detailGroup.value))
      detailGroup.value = fresh || null
    }
  } catch (e) {
    error('加载版本组失败：' + (e.response?.data?.error || e.message))
  }
}

function openUploadModal(group) {
  appendToGroup.value = group || null
  showUploadModal.value = true
  if (group) {
    // 追加模式：自动用组的 channel / package_type / version_name
    form.value.mode = 'existing'
    form.value.selected_group_key = gKey(group)
    form.value.version_name = group.version_name
    form.value.channel = group.channel
    form.value.package_type = group.package_type
  } else {
    resetUploadForm()
  }
}

function resetUploadForm() {
  appendToGroup.value = null
  form.value = {
    mode: 'new', selected_group_key: '', version_name: '',
    channel: 'stable', platform: 'windows', architecture: 'all',
    package_type: 'full', is_force_update: false, release_notes: '',
    file: null, download_url: '', file_size: '', file_hash: ''
  }
  if (fileInput.value) fileInput.value.value = ''
}

async function probeFile() {
  if (!form.value.download_url) { info('请先填写下载链接'); return }
  probing.value = true
  try {
    const r = await probeFileInfo(form.value.download_url)
    if (r.file_size) { form.value.file_size = r.file_size.toString(); info('文件大小：' + formatSize(r.file_size)) }
    else if (r.message) { info(r.message) }
  } catch (e) { error('探测失败：' + (e.response?.data?.error || e.message)) }
  finally { probing.value = false }
}

async function submitUpload() {
  // 冲突拦截（前端友好提示，后端兜底返回 409）
  if (existingGroupConflict.value) {
    error(`同版本 + 同平台 + 同架构 已存在 (版本号 ${existingGroupConflict.value.version_code})`)
    return
  }

  const vname = effectiveVersionName()
  if (!vname) { info('请选择或输入版本名称'); return }

  if (appStorageType.value === 'metadata') {
    if (!form.value.download_url) { info('请填写下载链接'); return }
  } else {
    if (!form.value.file) { info('请选择更新文件'); return }
  }

  uploading.value = true
  uploadProgress.value = 0
  const onProgress = (e) => {
    if (e.total > 0) uploadProgress.value = Math.round((e.loaded * 100) / e.total)
  }
  try {
    const app = await getApp(appId)
    const fd = new FormData()
    fd.append('app_id', app.app_id)
    fd.append('version_name', vname)
    fd.append('channel', form.value.channel)
    fd.append('platform', form.value.platform)
    fd.append('architecture', form.value.architecture)
    fd.append('package_type', form.value.package_type)
    fd.append('is_force_update', form.value.is_force_update)
    fd.append('release_notes', form.value.release_notes)

    if (appStorageType.value === 'metadata') {
      fd.append('download_url', form.value.download_url)
      if (form.value.file_size) fd.append('file_size', form.value.file_size)
      if (form.value.file_hash)  fd.append('file_hash', form.value.file_hash)
    } else {
      fd.append('file', form.value.file)
    }

    await uploadVersionFile(fd, onProgress)
    showUploadModal.value = false
    success(appendToGroup.value ? `已向「${vname}」追加 ${form.value.platform} 平台` : `已发布 ${vname}`)
    resetUploadForm()
    await loadGroups()
  } catch (e) {
    const msg = e.response?.data?.error || e.message
    if (e.response?.status === 409) {
      error(`冲突：${msg}`)
    } else {
      error('上传失败：' + msg)
    }
  } finally {
    uploading.value = false
  }
}

async function toggleStatus(ver) {
  const newStatus = ver.status === 1 ? 0 : 1
  try {
    await updateVersionStatus(ver.id, newStatus)
    ver.status = newStatus
  } catch (e) { error('操作失败') }
}

async function deleteVersion(ver) {
  if (!(await askConfirm(`确定删除版本 ${ver.version_name} (${ver.platform}/${ver.architecture}) 吗？`))) return
  try {
    await deleteVersionApi(ver.id)
    await loadGroups()
  } catch (e) { error('删除失败') }
}

function copyUrl(url) {
  let full
  if (url.startsWith('http://') || url.startsWith('https://')) full = url
  else if (url.startsWith('/')) full = window.location.origin + url
  else full = window.location.origin + '/api/download/' + url
  navigator.clipboard?.writeText(full).then(() => success('链接已复制'))
}

onMounted(async () => {
  await loadAppInfo()
  loadGroups()
})
</script>
