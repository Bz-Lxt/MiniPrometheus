<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import ToastHost from './components/ToastHost.vue'
</script>

<template>
  <div class="min-h-full flex flex-col md:flex-row">
    <aside class="md:w-[72px] w-full border-b md:border-b-0 md:border-r border-line bg-panel/80 backdrop-blur flex md:flex-col items-center gap-2 py-3 px-3 md:px-0">
      <div class="font-display text-amber text-lg tracking-widest md:writing-vertical">MP</div>
      <RouterLink to="/" class="navbtn" aria-label="流式仪表盘">流</RouterLink>
      <RouterLink to="/query" class="navbtn" aria-label="查询剖析器">析</RouterLink>
      <RouterLink to="/cluster" class="navbtn" aria-label="集群与索引">群</RouterLink>
    </aside>
    <main class="flex-1 w-full relative">
      <div class="scanline absolute inset-0"></div>
      <div class="relative z-10 w-full p-4 md:p-6">
        <RouterView />
      </div>
    </main>
    <ToastHost />
  </div>
</template>

<style scoped>
.navbtn {
  @apply w-10 h-10 grid place-items-center border border-line text-mute hover:text-amber hover:border-amber transition;
}
.navbtn.router-link-active {
  @apply text-amber border-amber bg-amber/10;
}
.md\:writing-vertical {
  writing-mode: vertical-rl;
}
</style>
