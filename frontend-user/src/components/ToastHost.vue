<script setup lang="ts">
import { dismiss, useToasts } from '../lib/toast'
const toasts = useToasts()
</script>

<template>
  <div class="fixed right-4 bottom-4 z-50 flex flex-col gap-2 w-80">
    <div
      v-for="t in toasts.items"
      :key="t.id"
      class="toast-enter border px-3 py-2 text-sm flex items-start gap-2"
      :class="t.kind === 'err' ? 'border-rose bg-rose/10' : t.kind === 'warn' ? 'border-amber bg-amber/10' : 'border-cyan bg-cyan/10'"
    >
      <p class="flex-1 font-mono text-xs">{{ t.text }}</p>
      <button class="text-mute hover:text-white" aria-label="关闭提示" @click="dismiss(t.id)">×</button>
    </div>
  </div>
</template>
