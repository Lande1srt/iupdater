<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="confirmState.show" class="fixed inset-0 z-[60] flex items-center justify-center">
        <div class="absolute inset-0 bg-black/40" @click="confirmState.onCancel?.()"></div>
        <div class="relative bg-white rounded-xl shadow-2xl w-[420px] max-w-[90vw] p-6">
          <div class="flex items-center gap-3 mb-4">
            <span class="w-10 h-10 rounded-full bg-yellow-100 text-yellow-600 flex items-center justify-center text-xl">⚠</span>
            <h3 class="text-lg font-semibold text-gray-900">{{ confirmState.title }}</h3>
          </div>
          <p class="text-sm text-gray-600 leading-relaxed mb-6">{{ confirmState.message }}</p>
          <div class="flex justify-end gap-2">
            <button @click="confirmState.onCancel?.()"
              class="px-4 py-2 rounded-lg text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 transition">
              {{ confirmState.cancelText }}
            </button>
            <button @click="confirmState.onConfirm?.()"
              class="px-4 py-2 rounded-lg text-sm font-medium text-white bg-red-600 hover:bg-red-700 transition">
              {{ confirmState.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { confirmState } from '../composables/useToast'
</script>

<style scoped>
.modal-enter-active, .modal-leave-active { transition: opacity .2s; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
.modal-enter-active .relative, .modal-leave-active .relative { transition: transform .2s; }
.modal-enter-from .relative, .modal-leave-to .relative { transform: scale(.95); }
</style>
