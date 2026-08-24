<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getJSON } from '../lib/api'
import { toast } from '../lib/toast'

const status = ref<any>({})
const cluster = ref<any>({})
const labels = ref<string[]>([])
const values = ref<string[]>([])
const picked = ref('')
const explain = ref<any>(null)
const matcher = ref('{job="api"}')

async function load() {
  try {
    status.value = await getJSON('/api/v1/status')
    cluster.value = await getJSON('/api/v1/cluster')
    const lab = await getJSON<any>('/api/v1/labels')
    labels.value = lab.data || []
  } catch (e: any) {
    toast(e.message, 'err')
  }
}

async function pick(name: string) {
  picked.value = name
  const r = await getJSON<any>(`/api/v1/label/${encodeURIComponent(name)}/values`)
  values.value = r.data || []
}

async function runExplain() {
  try {
    explain.value = await getJSON(`/api/v1/index/explain?query=${encodeURIComponent(matcher.value)}`)
  } catch (e: any) {
    toast(e.message, 'err')
  }
}

onMounted(load)
</script>

<template>
  <div class="w-full grid grid-cols-1 lg:grid-cols-2 gap-4">
    <section class="border border-line bg-panel/70 p-4">
      <h1 class="font-display text-3xl text-amber">集群拓扑</h1>
      <p class="text-mute text-sm mt-1">哈希分片 · 单副本 · 失败必须明示</p>
      <div class="mt-4 space-y-2">
        <div v-for="s in cluster.shards || []" :key="s.id" class="border border-line p-3 flex justify-between">
          <div>
            <div class="font-display">分片 {{ s.id }}</div>
            <div class="font-mono text-xs text-mute">{{ s.endpoint }}</div>
          </div>
          <span :class="s.healthy ? 'text-cyan' : 'text-rose'">{{ s.healthy ? 'HEALTHY' : 'DOWN' }}</span>
        </div>
      </div>
    </section>
    <section class="border border-line bg-panel/70 p-4">
      <h2 class="font-display text-2xl">压缩率</h2>
      <div class="mt-3 font-mono text-sm space-y-1">
        <p>series {{ status.head?.series ?? 0 }}</p>
        <p>samples {{ status.head?.samples ?? 0 }}</p>
        <p>raw {{ status.head?.bytes_in ?? 0 }} B</p>
        <p>comp {{ status.head?.bytes_out ?? 0 }} B</p>
        <p class="text-amber text-xl">{{ (status.head?.bytes_per_sample ?? 0).toFixed(3) }} B/sample</p>
      </div>
      <p class="text-mute text-xs mt-3">随机 float64 会膨胀，这是 XOR 的诚实代价，不是 bug。</p>
    </section>
    <section class="lg:col-span-2 border border-line bg-panel/70 p-4">
      <h2 class="font-display text-2xl mb-2">标签浏览器</h2>
      <div class="flex flex-wrap gap-2 mb-3">
        <button v-for="n in labels" :key="n" class="px-2 py-1 border border-line text-xs font-mono" :class="picked===n ? 'border-amber text-amber' : ''" @click="pick(n)">{{ n }}</button>
      </div>
      <div class="flex flex-wrap gap-2 text-cyan font-mono text-xs">
        <span v-for="v in values" :key="v" class="border border-line px-2 py-1">{{ v }}</span>
      </div>
      <div class="mt-4 flex gap-2">
        <input v-model="matcher" class="flex-1 bg-[#0b1016] border border-line px-3 py-2 font-mono text-sm" />
        <button class="px-3 py-2 border border-amber text-amber" @click="runExplain">位图解释</button>
      </div>
      <pre v-if="explain" class="mt-3 text-xs font-mono text-mute overflow-x-auto">{{ JSON.stringify(explain, null, 2) }}</pre>
    </section>
  </div>
</template>
