import { ref } from 'vue'

// 全局 Toast 状态，右上角弹窗
export const toasts = ref([])
let toastId = 0

export function toast(message, type = 'info', duration = 3000) {
  const id = ++toastId
  toasts.value.push({ id, message, type })
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration)
  }
  return id
}

export function removeToast(id) {
  const idx = toasts.value.findIndex(t => t.id === id)
  if (idx !== -1) toasts.value.splice(idx, 1)
}

export const success = (msg, duration = 3000) => toast(msg, 'success', duration)
export const error = (msg, duration = 5000) => toast(msg, 'error', duration)
export const info = (msg, duration = 3000) => toast(msg, 'info', duration)
export const warning = (msg, duration = 4000) => toast(msg, 'warning', duration)

// 居中确认弹窗（需要用户点击按钮才关闭）
export const confirmState = ref({
  show: false,
  title: '确认操作',
  message: '',
  confirmText: '确定',
  cancelText: '取消',
  onConfirm: null,
  onCancel: null
})

export function showConfirm(options) {
  return new Promise((resolve) => {
    confirmState.value = {
      show: true,
      title: options.title || '确认操作',
      message: options.message || '',
      confirmText: options.confirmText || '确定',
      cancelText: options.cancelText || '取消',
      onConfirm: null,
      onCancel: null
    }
    confirmState.value.onConfirm = () => {
      confirmState.value.show = false
      resolve(true)
    }
    confirmState.value.onCancel = () => {
      confirmState.value.show = false
      resolve(false)
    }
  })
}

export function confirm(message, title = '确认操作') {
  return showConfirm({ title, message })
}
