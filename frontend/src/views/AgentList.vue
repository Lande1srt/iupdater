<template>
  <div class="space-y-6">
    <!-- 头部 -->
    <div class="flex justify-between items-center">
      <div>
        <h2 class="text-2xl font-bold text-gray-800">助手管理</h2>
        <p class="text-sm text-gray-500 mt-1">每个 agent 会主动 checkin 主控，列表实时反映各节点状态</p>
      </div>
      <div class="flex gap-2">
        <label class="flex items-center gap-1 text-xs text-gray-500">
          <input v-model="autoRefresh" type="checkbox" class="mr-1" />
          自动刷新
        </label>
        <button @click="loadData" class="btn btn-secondary">刷新</button>
        <button @click="openRegister" class="btn btn-primary">+ 注册助手</button>
      </div>
    </div>

    <!-- 列表 -->
    <div class="bg-white rounded-lg shadow-sm border border-gray-200">
      <table class="w-full">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称 / Access Key</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">绑定应用</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">实时状态</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">当前版本</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">上次更新</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Host / 平台</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">上次 checkin</th>
            <th class="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="a in agents" :key="a.id" class="hover:bg-gray-50">
            <td class="px-4 py-3">
              <div class="font-medium text-gray-800">{{ a.name }}</div>
              <div class="font-mono text-xs text-gray-500 mt-0.5">
                <span>{{ (a.access_key || '').substring(0, 8) }}...{{ (a.access_key || '').slice(-4) }}</span>
                <button @click="copyKey(a.access_key)" class="ml-1 text-gray-400 hover:text-primary-600" title="复制完整 key">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
                </button>
                <span v-if="a.pending_command" class="ml-1 text-orange-500 text-[10px] inline-flex items-center gap-0.5" title="待下发的控制指令">
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                  {{ a.pending_command }}
                </span>
                <span v-if="a.remote_config" class="ml-1 text-orange-500 text-[10px]" title="待下发的远程配置">
                  <svg class="w-3 h-3 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                </span>
              </div>
            </td>
            <td class="px-4 py-3 text-sm text-gray-600">
              <div>{{ a.app_id }}</div>
              <div v-if="a.channel" class="mt-0.5">
                <span class="text-primary-600 bg-primary-50 px-1.5 py-0.5 rounded text-[10px]">{{ a.channel }}</span>
              </div>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <span :class="stateClass(a)" class="px-2 py-0.5 text-xs rounded font-medium">{{ stateLabel(a) }}</span>
                <span v-if="stateOf(a) === 'downloading' || stateOf(a) === 'updating'" class="text-[10px] text-gray-400">{{ a.status_json?.progress || 0 }}%</span>
              </div>
              <div v-if="stateOf(a) === 'fail' || a.last_message" class="text-xs text-red-500 mt-1 max-w-[180px] truncate" :title="a.status_json?.error || a.last_message">
                {{ a.status_json?.error || a.last_message }}
              </div>
              <div v-else-if="a.last_message" class="text-xs text-gray-400 mt-1 max-w-[180px] truncate" :title="a.last_message">
                {{ a.last_message }}
              </div>
            </td>
            <td class="px-4 py-3 text-sm font-mono">
              <span class="text-gray-700">{{ a.current_version || a.status_json?.current_version || '—' }}</span>
            </td>
            <td class="px-4 py-3 text-xs text-gray-500">
              {{ formatDate(a.status_json?.last_update_at) }}
            </td>
            <td class="px-4 py-3 text-xs text-gray-500">
              <div>{{ a.status_json?.hostname || '—' }}</div>
              <div class="mt-0.5 text-[10px] text-gray-400">
                {{ a.platform || 'all' }} / {{ a.architecture || 'all' }}
              </div>
            </td>
            <td class="px-4 py-3 text-xs">
              <div>{{ formatDate(a.last_checkin_at) }}</div>
              <div v-if="isOffline(a)" class="text-red-500 text-[10px] mt-0.5 inline-flex items-center gap-1">
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
                离线（>5min）
              </div>
            </td>
            <td class="px-4 py-3 text-right space-x-1 whitespace-nowrap">
              <button v-if="hasError(a)" @click="doClearError(a)" class="btn btn-xs bg-yellow-50 text-yellow-700 hover:bg-yellow-100 p-1.5" title="清除错误状态，让界面回到空闲">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
              </button>
              <button @click="doControl(a, 'check_now')" class="btn btn-xs bg-blue-50 text-blue-700 hover:bg-blue-100 p-1.5" title="立即检查更新">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              </button>
              <button @click="doControl(a, 'heartbeat')" class="btn btn-xs bg-pink-50 text-pink-700 hover:bg-pink-100 p-1.5" title="发送探活指令">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
              </button>
              <button @click="openControl(a)" class="btn btn-xs bg-gray-50 text-gray-700 hover:bg-gray-100 p-1.5" title="设置 / 远程控制">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
              </button>
              <button @click="openEdit(a)" class="btn btn-sm btn-secondary">编辑</button>
              <button @click="handleDelete(a)" class="btn btn-sm btn-danger">删除</button>
            </td>
          </tr>
          <tr v-if="agents.length === 0">
            <td colspan="8" class="px-6 py-12 text-center text-gray-400">
              <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-3">
                <svg class="w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>
              </div>
              <div class="text-gray-500 font-medium mb-1">暂无注册的助手</div>
              <div class="text-xs text-gray-400">点击右上角「注册助手」创建一个，生成的 Access Key 填到 agent 的 config.json 里即可开始工作</div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 快速上手提示 -->
    <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 text-sm text-blue-800">
      <div class="font-semibold mb-2 flex items-center gap-2">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" /></svg>
        快速上手
      </div>
      <ol class="list-decimal ml-5 space-y-1 text-xs text-blue-700">
        <li>点击右上角「注册助手」，选择要监控的应用，后端生成 Access Key</li>
        <li>把 Access Key 填到 agent 同目录的 <code class="bg-blue-100 px-1 rounded">config.json</code> 的 <code class="bg-blue-100 px-1 rounded">access_key</code></li>
        <li>运行 <code class="bg-blue-100 px-1 rounded">./agent</code>，它会主动 checkin 主控，列表立刻出现实时状态</li>
      </ol>
    </div>
  </div>

  <!-- 注册弹窗 -->
  <div v-if="showRegister" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showRegister = false">
    <div class="bg-white rounded-lg shadow-xl w-[500px] max-h-[92vh] overflow-hidden flex flex-col">
      <div class="p-5 border-b border-gray-200">
        <h3 class="text-lg font-semibold">注册新助手</h3>
      </div>
      <div v-if="!registeredKey" class="p-5 space-y-4 overflow-y-auto flex-1">
        <div>
          <label class="label">助手名称</label>
          <input v-model="regForm.name" type="text" class="input" placeholder="例如：生产服务器01" />
        </div>
        <div>
          <label class="label">绑定应用 <span class="text-red-500">*</span></label>
          <select v-model="regForm.app_id" class="input">
            <option value="">请选择应用...</option>
            <option v-for="x in apps" :key="x.app_id" :value="x.app_id">{{ x.name }} <span class="text-gray-400 text-xs">({{ x.app_id }})</span></option>
          </select>
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="label">Channel</label>
            <select v-model="regForm.channel" class="input">
              <option value="stable">stable</option>
              <option value="beta">beta</option>
            </select>
          </div>
          <div>
            <label class="label">Platform</label>
            <select v-model="regForm.platform" class="input">
              <option value="">自动</option>
              <option value="windows">windows</option>
              <option value="linux">linux</option>
              <option value="macos">macos</option>
              <option value="all">all</option>
            </select>
          </div>
          <div>
            <label class="label">Architecture</label>
            <select v-model="regForm.architecture" class="input">
              <option value="">自动</option>
              <option value="amd64">amd64</option>
              <option value="arm64">arm64</option>
              <option value="all">all</option>
            </select>
          </div>
        </div>
      </div>
      <div v-else class="p-5 space-y-4 overflow-y-auto flex-1">
        <div class="bg-green-50 border border-green-200 rounded-lg p-4 text-sm">
          <div class="font-semibold text-green-800 mb-2">✅ 注册成功</div>
          <div class="text-green-700 text-xs">以下是这个助手唯一的 Access Key，请立即保存到 agent 的 config.json。</div>
        </div>
        <div>
          <label class="label">Access Key（复制保存）</label>
          <div class="flex gap-2">
            <input ref="keyInput" :value="registeredKey" readonly class="input font-mono text-xs bg-gray-50" />
            <button @click="copyKey(registeredKey); copied = true; setTimeout(() => copied = false, 2000)" class="btn btn-primary">
              <svg v-if="copied" class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
              {{ copied ? '已复制' : '复制' }}
            </button>
          </div>
        </div>
        <div class="bg-gray-50 rounded-lg p-4 text-xs font-mono text-gray-700 border border-gray-200">
{
  "server_url": "http://主控服务器:8080",
  "access_key": "<span class="bg-yellow-200">{{ registeredKey }}</span>",
  "app_id": "{{ regForm.app_id }}",
  "poll_interval_seconds": 60,
  "target_dir": "./app",
  "stop_command": "",
  "start_command": "",
  "api_port": 18080
}
        </div>
      </div>
      <div class="p-4 border-t border-gray-200 flex justify-end gap-2">
        <button v-if="registeredKey" @click="showRegister = false" class="btn btn-primary">完成</button>
        <template v-else>
          <button @click="showRegister = false" class="btn btn-secondary">取消</button>
          <button @click="doRegister" :disabled="registering" class="btn btn-primary">{{ registering ? '注册中...' : '注册' }}</button>
        </template>
      </div>
    </div>
  </div>

  <!-- 编辑弹窗 -->
  <div v-if="showEdit" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showEdit = false">
    <div class="bg-white rounded-lg shadow-xl w-[450px] max-h-[92vh] overflow-hidden flex flex-col">
      <div class="p-5 border-b border-gray-200"><h3 class="text-lg font-semibold">编辑助手</h3></div>
      <div class="p-5 space-y-4 overflow-y-auto flex-1">
        <div>
          <label class="label">名称</label>
          <input v-model="editForm.name" class="input" />
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label class="label">Channel</label>
            <select v-model="editForm.channel" class="input">
              <option value="stable">stable</option>
              <option value="beta">beta</option>
            </select>
          </div>
          <div>
            <label class="label">Platform</label>
            <select v-model="editForm.platform" class="input">
              <option value="">自动</option>
              <option value="windows">windows</option>
              <option value="linux">linux</option>
              <option value="macos">macos</option>
              <option value="all">all</option>
            </select>
          </div>
          <div>
            <label class="label">Architecture</label>
            <select v-model="editForm.architecture" class="input">
              <option value="">自动</option>
              <option value="amd64">amd64</option>
              <option value="arm64">arm64</option>
              <option value="all">all</option>
            </select>
          </div>
        </div>
        <div class="border border-orange-200 bg-orange-50 rounded p-3">
          <label class="flex items-center gap-2 text-sm text-orange-800 cursor-pointer">
            <input type="checkbox" v-model="editForm.reset_access_key" />
            <span>同时重置 Access Key</span>
          </label>
          <div v-if="newKey" class="mt-3 font-mono text-xs bg-white border border-orange-200 p-2 rounded flex justify-between">
            <span>{{ newKey }}</span>
            <button @click="copyKey(newKey)" class="text-orange-600 inline-flex items-center gap-1">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
              复制
            </button>
          </div>
        </div>
      </div>
      <div class="p-4 border-t border-gray-200 flex justify-end gap-2">
        <button @click="showEdit = false" class="btn btn-secondary">取消</button>
        <button @click="doSaveEdit" :disabled="saving" class="btn btn-primary">{{ saving ? '保存中...' : '保存' }}</button>
      </div>
    </div>
  </div>

  <!-- 远程控制弹窗 -->
  <div v-if="showControl" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showControl = false">
    <div class="bg-white rounded-lg shadow-xl w-[680px] max-h-[92vh] overflow-hidden flex flex-col">
      <div class="p-5 border-b border-gray-200 flex items-center justify-between">
        <div>
          <h3 class="text-lg font-semibold">远程控制 — {{ ctrlAgent?.name }}</h3>
          <div v-if="cfgReportedAt" class="text-xs text-gray-400 mt-0.5">配置快照上报于 {{ cfgReportedAt }}</div>
          <div v-else-if="!cfgLoaded" class="text-xs text-amber-500 mt-0.5">暂无配置快照，agent 下次 checkin 后这里会自动填充</div>
          <div v-else class="text-xs text-amber-500 mt-0.5">主控无法感知 access_key / server_url，这两个字段不会显示</div>
        </div>
        <button @click="refreshCfgNow" class="btn btn-sm btn-secondary inline-flex items-center gap-1" :disabled="cfgRefreshing">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
          {{ cfgRefreshing ? '刷新中...' : '强制刷新' }}
        </button>
      </div>
      <div class="p-5 space-y-5 overflow-y-auto flex-1">
        <!-- 快速控制 -->
        <div class="bg-gray-50 rounded p-3">
          <div class="text-xs text-gray-500 mb-2">快速控制（下次 checkin 生效）</div>
          <div class="flex flex-wrap gap-2">
            <button @click="doControl(ctrlAgent, 'force_update', false)" class="btn btn-danger inline-flex items-center gap-1.5">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
              强制推送最新版本
            </button>
            <button @click="doControl(ctrlAgent, 'check_now', false)" class="btn btn-primary inline-flex items-center gap-1.5">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              立即检查更新
            </button>
            <button @click="doControl(ctrlAgent, 'restart', false)" class="btn btn-secondary inline-flex items-center gap-1.5">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
              重启 agent
            </button>
            <button @click="doControl(ctrlAgent, 'heartbeat', false)" class="btn btn-secondary inline-flex items-center gap-1.5">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
              探活
            </button>
          </div>
          <div class="text-xs text-orange-600 mt-2">
            强制推送：忽略 agent 当前版本号，把最新发布版本直接下发执行（覆盖当前安装）
          </div>
        </div>

        <!-- 远程配置可视化表单 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-semibold text-gray-700">远程配置（合并写入 agent 本地 config.json）</label>
            <div class="text-[11px] text-gray-400">留空字段不会下发</div>
          </div>

          <!-- 基础信息（只读） -->
          <div class="grid grid-cols-2 gap-3 bg-gray-50 rounded p-3 mb-3">
            <div>
              <label class="text-xs text-gray-500">App ID <span class="text-red-400">(只读)</span></label>
              <input v-model="cfgForm.app_id" readonly class="input bg-gray-100 text-gray-400" />
            </div>
            <div>
              <label class="text-xs text-gray-500">Agent Hostname <span class="text-red-400">(只读)</span></label>
              <input :value="ctrlAgent?.status_json?.hostname || '—'" readonly class="input bg-gray-100 text-gray-400" />
            </div>
          </div>

          <!-- 标识类 -->
          <div class="grid grid-cols-3 gap-3 mb-3">
            <div>
              <label class="text-xs text-gray-600">发布渠道</label>
              <select v-model="cfgForm.channel" class="input">
                <option value="">— 不修改 —</option>
                <option value="stable">stable 正式</option>
                <option value="beta">beta 测试</option>
                <option value="dev">dev 开发</option>
              </select>
            </div>
            <div>
              <label class="text-xs text-gray-600">平台</label>
              <select v-model="cfgForm.platform" class="input">
                <option value="">— 不修改 —</option>
                <option value="all">all</option>
                <option value="windows">windows</option>
                <option value="linux">linux</option>
                <option value="macos">macos</option>
                <option value="android">android</option>
                <option value="ios">ios</option>
              </select>
            </div>
            <div>
              <label class="text-xs text-gray-600">架构</label>
              <select v-model="cfgForm.architecture" class="input">
                <option value="">— 不修改 —</option>
                <option value="all">all</option>
                <option value="x86_64">x86_64</option>
                <option value="amd64">amd64</option>
                <option value="aarch64">aarch64</option>
                <option value="arm64">arm64</option>
                <option value="i386">i386</option>
              </select>
            </div>
          </div>

          <!-- 路径 -->
          <div class="grid grid-cols-3 gap-3 mb-3">
            <div>
              <label class="text-xs text-gray-600">目标目录 target_dir</label>
              <input v-model="cfgForm.target_dir" class="input" placeholder="留空 = 不修改" />
            </div>
            <div>
              <label class="text-xs text-gray-600">备份目录 backup_dir</label>
              <input v-model="cfgForm.backup_dir" class="input" placeholder="留空 = 不修改" />
            </div>
            <div>
              <label class="text-xs text-gray-600">下载目录 download_dir</label>
              <input v-model="cfgForm.download_dir" class="input" placeholder="留空 = 不修改" />
            </div>
          </div>

          <!-- 数值类 -->
          <div class="grid grid-cols-3 gap-3 mb-3">
            <div>
              <label class="text-xs text-gray-600">轮询间隔 (秒) poll_interval_seconds</label>
              <input v-model.number="cfgForm.poll_interval_seconds" type="number" min="5" class="input" placeholder="留空 = 不修改" />
            </div>
            <div>
              <label class="text-xs text-gray-600">备份保留数 backup_keep (1-200)</label>
              <input v-model.number="cfgForm.backup_keep" type="number" min="1" max="200" class="input" placeholder="留空 = 不修改" />
            </div>
            <div>
              <label class="text-xs text-gray-600">本地面板端口 api_port</label>
              <input v-model.number="cfgForm.api_port" type="number" min="1" max="65535" class="input" placeholder="留空 = 不修改" />
            </div>
          </div>

          <!-- 命令 -->
          <div class="space-y-3 mb-3">
            <div>
              <label class="text-xs text-gray-600">停止命令 stop_command</label>
              <input v-model="cfgForm.stop_command" class="input font-mono text-xs" placeholder="例: ./run.sh stop 或 systemctl stop myapp" />
            </div>
            <div>
              <label class="text-xs text-gray-600">启动命令 start_command</label>
              <input v-model="cfgForm.start_command" class="input font-mono text-xs" placeholder="例: ./run.sh start" />
            </div>
          </div>

          <div class="bg-blue-50 border border-blue-200 rounded p-3 text-xs text-blue-800">
            <div>后端会自动过滤 access_key / server_url / app_id 等敏感字段，防止误覆盖 agent 身份。</div>
          </div>
        </div>
      </div>
      <div class="p-4 border-t border-gray-200 flex justify-end gap-2">
        <button @click="showControl = false" class="btn btn-secondary">关闭</button>
        <button @click="doSaveConfigPatch" :disabled="saving || !configPatchPending" class="btn btn-primary">{{ saving ? '下发中...' : '下发修改' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { getAgents, getAgent, createAgent, updateAgent, deleteAgent, controlAgent, clearAgentError, getApps } from '../api'
import { success, error, confirm as askConfirm } from '../composables/useToast'

const agents = ref([])
const apps = ref([])
const showRegister = ref(false)
const showEdit = ref(false)
const showControl = ref(false)
const registeredKey = ref('')
const copied = ref(false)
const registering = ref(false)
const saving = ref(false)
const autoRefresh = ref(true)
let refreshTimer = null

const regForm = ref({ name: '', app_id: '', channel: 'stable', platform: '', architecture: '' })
const editForm = ref({ name: '', channel: 'stable', platform: '', architecture: '', reset_access_key: false })
const editID = ref(0)
const newKey = ref('')
const ctrlAgent = ref(null)
// cfgForm 与 agent config 字段一一对应，undefined / '' 表示不修改
const cfgForm = ref({
  app_id: '',
  channel: '',
  platform: '',
  architecture: '',
  target_dir: '',
  backup_dir: '',
  download_dir: '',
  backup_keep: undefined,
  poll_interval_seconds: undefined,
  api_port: undefined,
  stop_command: '',
  start_command: '',
})
// cfgSnapshot 弹窗打开时的拷贝，用于计算哪些字段被修改过
const cfgSnapshot = ref({})
const cfgRefreshing = ref(false)

// cfgLoaded = 已成功拿到过 agent 的 cfg 子块（至少一次）
const cfgLoaded = computed(() => !!ctrlAgent.value?.status_json?.cfg)
const cfgReportedAt = computed(() => {
  const t = ctrlAgent.value?.status_json?.cfg?.reported_at
  if (!t) return ''
  try { return new Date(t).toLocaleString() } catch { return '' }
})

// 是否有可下发的修改（与打开时快照对比）
const configPatchPending = computed(() => {
  return Object.keys(buildConfigPatch()).length > 0
})

// buildConfigPatch 从 cfgForm 里挑出「有值且与原始快照不同」的字段，只下发这些
function buildConfigPatch() {
  const out = {}
  const snap = cfgSnapshot.value || {}
  const f = cfgForm.value
  for (const k of Object.keys(f)) {
    const v = f[k]
    // 跳过只读字段
    if (k === 'app_id') continue
    // 空值跳过（不修改）
    if (v === '' || v === undefined || v === null) continue
    // 数字类型 NaN 跳过
    if (typeof v === 'number' && Number.isNaN(v)) continue
    // 与原始快照一致跳过
    if (snap[k] !== undefined && snap[k] === v) continue
    out[k] = v
  }
  return out
}

// --- 实时状态解析 ---
function stateOf(a) {
  try { return a.status_json?.state || '' } catch { return '' }
}
function stateClass(a) {
  const s = stateOf(a)
  if (!s) {
    return a.last_result === 'success' ? 'bg-green-100 text-green-700'
         : a.last_result === 'fail'    ? 'bg-red-100 text-red-700'
         : isOffline(a)                ? 'bg-gray-100 text-gray-500'
         : 'bg-gray-50 text-gray-400'
  }
  switch (s) {
    case 'idle':       return 'bg-gray-100 text-gray-600'
    case 'checking':   return 'bg-blue-100 text-blue-700'
    case 'downloading':return 'bg-yellow-100 text-yellow-700'
    case 'updating':   return 'bg-orange-100 text-orange-700'
    case 'success':    return 'bg-green-100 text-green-700'
    case 'fail':       return 'bg-red-100 text-red-700'
    default:           return 'bg-gray-50 text-gray-500'
  }
}
function stateLabel(a) {
  const m = { idle: '空闲', checking: '检查中', downloading: '下载中', updating: '更新中', success: '成功', fail: '失败' }
  const s = stateOf(a)
  if (m[s]) return m[s]
  if (!s && isOffline(a)) return '离线'
  if (a.last_result === 'success') return '成功'
  if (a.last_result === 'fail')    return '失败'
  return '—'
}

async function loadData() {
  try {
    const res = await getAgents()
    const list = Array.isArray(res) ? res : (res.list || [])
    // 解析 status_json（后端返回可能是字符串，也可能是已解析对象）
    agents.value = list.map(a => ({ ...a, status_json: safeParseStatus(a.status_json) }))
  } catch { /* 静默，手动刷新按钮提示 */ }
}
function safeParseStatus(raw) {
  if (!raw) return null
  if (typeof raw === 'object') return raw
  try { return JSON.parse(raw) } catch { return null }
}

async function loadApps() {
  try {
    const res = await getApps()
    apps.value = Array.isArray(res) ? res : (res.list || [])
  } catch {}
}

// --- 注册 ---
function openRegister() {
  regForm.value = { name: '', app_id: '', channel: 'stable', platform: '', architecture: '' }
  registeredKey.value = ''
  showRegister.value = true
  loadApps()
}
async function doRegister() {
  if (!regForm.value.app_id) { error('请选择要绑定的应用'); return }
  registering.value = true
  try {
    const params = {
      name: regForm.value.name || undefined,
      app_id: regForm.value.app_id,
      channel: regForm.value.channel || undefined,
      platform: regForm.value.platform || undefined,
      architecture: regForm.value.architecture || undefined
    }
    const res = await createAgent(params)
    registeredKey.value = res.access_key
    success('注册成功，请立即保存 Access Key')
    await loadData()
  } catch (e) { error(e.response?.data?.error || '注册失败') }
  finally { registering.value = false }
}

// --- 编辑 ---
function openEdit(a) {
  editID.value = a.id
  editForm.value = {
    name: a.name,
    channel: a.channel || 'stable',
    platform: a.platform,
    architecture: a.architecture,
    reset_access_key: false
  }
  newKey.value = ''
  showEdit.value = true
}
async function doSaveEdit() {
  saving.value = true
  try {
    const params = { ...editForm.value }
    if (params.reset_access_key) params.reset_access_key = true
    else delete params.reset_access_key
    const res = await updateAgent(editID.value, params)
    if (res.access_key) { newKey.value = res.access_key }
    success('已保存')
    showEdit.value = false
    await loadData()
  } catch (e) { error(e.response?.data?.error || '保存失败') }
  finally { saving.value = false }
}

// --- 删除 ---
async function handleDelete(a) {
  const ok = await askConfirm(`确定删除助手「${a.name}」吗？这个节点将从主控列表移除。`)
  if (!ok) return
  try {
    await deleteAgent(a.id)
    success('已删除')
    await loadData()
  } catch { error('删除失败') }
}

// --- 远程控制 ---
async function doControl(a, command, showMsg = true) {
  try {
    const res = await controlAgent(a.id, command, null)
    const hint = command === 'force_update' ? '强制推送已下发 — 下次 checkin 自动执行（会覆盖当前版本）' : `${command} 指令已下发 — ${res.message || '下次 checkin 生效'}`
    if (showMsg) {
      success(hint)
    }
    await loadData()
  } catch (e) { error(e.response?.data?.error || '指令下发失败') }
}

async function doForceUpdate(a) {
  const ok = await askConfirm(`确定要对「${a.name}」强制推送最新版本吗？\n\n这会忽略目标服务器当前版本，直接下载并覆盖最新版本。`)
  if (!ok) return
  await doControl(a, 'force_update')
}

async function openControl(a) {
  ctrlAgent.value = a
  showControl.value = true
  // 拉取最新 agent 详情（含最近 checkin 上报的 cfg 快照）
  await refreshCfgNow(false)
}

// refreshCfgNow 从主控拉取该 agent 最新一次 checkin 上报的 cfg 并预填表单
// silent=false 时会显示 loading + 错误提示；silent=true 用于 openControl 场景静默兜底
async function refreshCfgNow(silent = true) {
  cfgRefreshing.value = true
  try {
    const res = await getAgent(ctrlAgent.value.id)
    const latest = { ...res, status_json: safeParseStatus(res.status_json) }
    // 更新本地引用
    const idx = agents.value.findIndex(x => x.id === ctrlAgent.value.id)
    if (idx >= 0) {
      agents.value[idx] = latest
    }
    ctrlAgent.value = latest
    const cfg = latest.status_json?.cfg || {}
    // 用最新 cfg 预填表单，未上报的字段留默认空
    cfgForm.value = {
      app_id:                cfg.app_id || '',
      channel:               cfg.channel || '',
      platform:              cfg.platform || '',
      architecture:          cfg.architecture || '',
      target_dir:            cfg.target_dir || '',
      backup_dir:            cfg.backup_dir || '',
      download_dir:          cfg.download_dir || '',
      backup_keep:           cfg.backup_keep ?? undefined,
      poll_interval_seconds: cfg.poll_interval_seconds ?? undefined,
      api_port:              cfg.api_port ?? undefined,
      stop_command:          cfg.stop_command || '',
      start_command:         cfg.start_command || '',
    }
    cfgSnapshot.value = { ...cfgForm.value }
    if (!silent) success('已同步 agent 最新配置')
  } catch (e) {
    if (!silent) error(e.response?.data?.error || '拉取 agent 配置失败')
  } finally {
    cfgRefreshing.value = false
  }
}

async function doSaveConfigPatch() {
  const patch = buildConfigPatch()
  if (Object.keys(patch).length === 0) { error('没有需要下发的修改'); return }
  saving.value = true
  try {
    const res = await controlAgent(ctrlAgent.value.id, '', patch)
    success(`远程配置已下发 — ${res.message || '下次 checkin 生效'}`)
    // 下发成功：立即把快照推进，避免同一弹窗重复发
    cfgSnapshot.value = { ...cfgForm.value }
    await loadData()
  } catch (e) {
    error(e.response?.data?.error || '下发配置失败')
  } finally {
    saving.value = false
  }
}

function copyKey(key) {
  navigator.clipboard?.writeText(key)
  success('Access Key 已复制到剪贴板')
}
function formatDate(v) {
  if (!v) return '—'
  try { return new Date(v).toLocaleString() } catch { return v }
}
function isOffline(a) {
  if (!a.last_checkin_at) return true
  return Date.now() - new Date(a.last_checkin_at).getTime() > 5 * 60 * 1000
}

// hasError 判断 agent 记录里是否残留了失败状态（供清除错误按钮显隐用）
function hasError(a) {
  if (a.last_result === 'fail') return true
  if (a.status_json?.state === 'fail') return true
  if (a.status_json?.error) return true
  return false
}

// doClearError 手动清除 agent 残留的错误状态
async function doClearError(a) {
  const ok = await askConfirm(`清除「${a.name}」的错误状态？\n\n这会清掉主控界面显示的失败记录，让 agent 回到空闲状态。agent 自身不会重启或重装。`)
  if (!ok) return
  try {
    await clearAgentError(a.id)
    success('错误状态已清除')
    await loadData()
  } catch (e) { error(e.response?.data?.error || '清除失败') }
}

// --- 自动刷新 ---
function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(() => loadData(), 15000)
}
function stopAutoRefresh() {
  if (refreshTimer) { clearInterval(refreshTimer); refreshTimer = null }
}
onMounted(() => { loadData(); loadApps(); if (autoRefresh.value) startAutoRefresh() })
onUnmounted(stopAutoRefresh)
</script>
