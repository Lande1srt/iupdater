<template>
  <div class="fixed top-4 right-4 z-50 space-y-2 w-80 pointer-events-none">
    <TransitionGroup name="toast">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg border backdrop-blur-sm"
        :class="toastClass(t.type)"
      >
        <span class="text-lg leading-none mt-0.5">{{ toastIcon(t.type) }}</span>
        <p class="flex-1 text-sm leading-snug break-all">{{ t.message }}</p>
        <button @click="removeToast(t.id)" class="text-current opacity-50 hover:opacity-100 leading-none text-lg">&times;</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { toasts, removeToast } from '../composables/useToast'

function toastClass(type) {
  switch (type) {
    case 'success': return 'bg-green-50/95 border-green-200 text-green-800'
    case 'error':   return 'bg-red-50/95 border-red-200 text-red-800'
    case 'warning': return 'bg-yellow-50/95 border-yellow-200 text-yellow-800'
    default:        return 'bg-blue-50/95 border-blue-200 text-blue-800'
  }
}

function toastIcon(type) {
  switch (type) {
    case 'success': return '✓'
    case 'error':   return '✕'
    case 'warning': return '⚠'
    default:        return 'ℹ'
  }
}
</script>

<style scoped>
.toast-enter-active { transition: all .25s ease-out; }
.toast-leave-active { transition: all .2s ease-in; }
.toast-enter-from { opacity: 0; transform: translateX(40px); }
.toast-leave-to   { opacity: 0; transform: translateX(40px); }
.toast-move       { transition: transform .25s; }
</style>
