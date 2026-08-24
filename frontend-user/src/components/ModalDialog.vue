<script setup lang="ts">
defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: []; confirm: [] }>()
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-40 bg-black/60 flex items-center justify-center p-4" @click.self="emit('close')">
    <div class="w-full max-w-md bg-panel border border-line p-5">
      <h3 class="font-display text-xl text-amber mb-3">{{ title }}</h3>
      <slot />
      <div class="mt-5 flex justify-end gap-2">
        <button class="px-3 py-1 border border-line text-mute" @click="emit('close')">取消</button>
        <button class="px-3 py-1 bg-amber text-bg font-semibold" @click="emit('confirm')">确认</button>
      </div>
    </div>
  </div>
</template>
