<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import SeriesChart from '../components/SeriesChart.vue'
import { queryRange, type QueryResp } from '../lib/api'
import { toast } from '../lib/toast'

const panels = [
  { title: 'CPU', q: 'node_cpu_usage' },
  { title: '内存', q: 'node_memory_used_bytes' },
  { title: '延迟', q: 'http_request_duration_ms' },
  { title: 'QPS', q: 'sum by (status) (rate(http_requests_total[1m]))' },
]
const data = ref<Record<string, any[]>>({})
const partial = ref(false)
const degraded = ref<string[]>([])
const paused = ref(false)
const rangeMin = ref(15)
let timer = 0

async function refresh() {
  if (paused.value) return
  try {
    const rs = await Promise.all(panels.map((p) => queryRange(p.q, rangeMin.value, 360)))
    rs.forEach((r: QueryResp, i) => {
      if (r.status !== 'success') {
        toast(r.error || '查询失败', 'err')
        return
      }
      data.value[panels[i].title] = r.data?.result || []
      if (r.partial) {
        partial.value = true
        degraded.value = r.degraded_shards || []
      } else {
        partial.value = false
        degraded.value = []
      }
    })
  } catch (e: any) {
    toast(e.message || '网络失败', 'err')
  }
}

onMounted(() => {
  refresh()
  timer = window.setInterval(refresh, 2000)
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div class="w-full">
    <header class="flex flex-wrap items-end justify-between gap-3 mb-4">
      <div>
        <p class="text-mute font-mono text-xs tracking-[0.3em]">STREAM · GMT+8</p>
        <h1 class="font-display text-4xl font-bold text-amber">多维指标流</h1>
      </div>
      <div class="flex items-center gap-2">
        <label class="text-mute text-xs">窗口</label>
        <select v-model.number="rangeMin" class="bg-panel border border-line px-3 py-1 text-sm">
          <option :value="5">5 分钟</option>
          <option :value="15">15 分钟</option>
          <option :value="60">60 分钟</option>
        </select>
        <button class="px-3 py-1 border border-line" :aria-label="paused ? '继续刷新' : '暂停刷新'" @click="paused = !paused">
          {{ paused ? '继续' : '暂停' }}
        </button>
      </div>
    </header>
    <div v-if="partial" class="mb-4 border border-amber bg-amber/10 px-3 py-2 text-amber text-sm">
      分片降级：部分存储不可达，结果可能残缺。{{ degraded.join(' / ') }}
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 w-full">
      <section v-for="p in panels" :key="p.title" class="border border-line bg-panel/70">
        <div class="px-3 py-2 border-b border-line flex justify-between">
          <h2 class="font-display text-lg">{{ p.title }}</h2>
          <span class="font-mono text-[11px] text-mute">{{ p.q }}</span>
        </div>
        <SeriesChart :series="data[p.title] || []" :height="240" />
      </section>
    </div>
  </div>
</template>
