<script setup lang="ts">
import { ref } from 'vue'
import SeriesChart from '../components/SeriesChart.vue'
import ExecTree from '../components/ExecTree.vue'
import { queryInstant, queryRange, type ProfileNode, type QueryResp } from '../lib/api'
import { toast } from '../lib/toast'

const expr = ref('sum by (job) (rate(http_requests_total[1m]))')
const err = ref('')
const profile = ref<ProfileNode | null>(null)
const series = ref<any[]>([])
const instant = ref<any[]>([])
const running = ref(false)
const partial = ref(false)

const hints = [
  'node_cpu_usage',
  'http_requests_total{status="500"}',
  'rate(http_requests_total[1m])',
  'sum by (instance) (node_cpu_usage)',
  'avg_over_time(http_request_duration_ms[2m])',
  'topk(3, node_cpu_usage)',
]

function highlight(s: string) {
  return s
    .replace(/(&)/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/(rate|irate|increase|delta|sum|avg|min|max|count|topk|quantile|avg_over_time|max_over_time|min_over_time|sum_over_time|count_over_time|histogram_quantile|by|without)\b/g, '<span class="text-amber">$1</span>')
    .replace(/("[^"]*")/g, '<span class="text-cyan">$1</span>')
    .replace(/(\[[0-9smhd]+\])/g, '<span class="text-rose">$1</span>')
}

async function run() {
  running.value = true
  err.value = ''
  try {
    const [r, i]: QueryResp[] = await Promise.all([queryRange(expr.value, 15, 480), queryInstant(expr.value)])
    if (r.status !== 'success') {
      err.value = r.error || '查询失败'
      toast(err.value, 'err')
      return
    }
    series.value = r.data?.result || []
    instant.value = i.data?.result || []
    profile.value = r.profile || i.profile || null
    partial.value = !!r.partial
  } catch (e: any) {
    err.value = e.message
    toast(err.value, 'err')
  } finally {
    running.value = false
  }
}
</script>

<template>
  <div class="w-full grid grid-cols-1 xl:grid-cols-2 gap-4">
    <section class="border border-line bg-panel/70 p-4">
      <p class="text-mute font-mono text-xs tracking-[0.3em]">MINIQL</p>
      <h1 class="font-display text-3xl text-amber mb-3">查询剖析器</h1>
      <textarea v-model="expr" rows="4" class="w-full bg-[#0b1016] border border-line p-3 font-mono text-sm" spellcheck="false" />
      <div class="mt-2 text-xs font-mono" v-html="highlight(expr)"></div>
      <div class="flex flex-wrap gap-2 mt-3">
        <button v-for="h in hints" :key="h" class="text-[11px] font-mono border border-line px-2 py-1 text-mute hover:text-amber" @click="expr = h">{{ h }}</button>
      </div>
      <button class="mt-4 px-4 py-2 bg-amber text-bg font-display font-bold" :disabled="running" @click="run">
        {{ running ? '执行中…' : '执行并绘制计划树' }}
      </button>
      <p v-if="err" class="mt-3 text-rose text-sm">{{ err }}</p>
      <p v-if="partial" class="mt-3 text-amber text-sm">分片降级，结果可能残缺。</p>
    </section>
    <section class="border border-line bg-panel/70 p-4">
      <h2 class="font-display text-xl mb-2">执行树</h2>
      <ExecTree :root="profile" />
    </section>
    <section class="xl:col-span-2 border border-line bg-panel/70">
      <div class="px-3 py-2 border-b border-line font-display">范围结果</div>
      <SeriesChart :series="series" :height="260" />
    </section>
    <section class="xl:col-span-2 border border-line bg-panel/70 p-3 overflow-x-auto">
      <h2 class="font-display mb-2">即时向量</h2>
      <table class="w-full text-left text-xs font-mono">
        <thead class="text-mute"><tr><th class="py-1">metric</th><th>value</th></tr></thead>
        <tbody>
          <tr v-for="(row, i) in instant" :key="i" class="border-t border-line">
            <td class="py-1 pr-3">{{ JSON.stringify(row.metric) }}</td>
            <td>{{ Array.isArray(row.value) ? row.value[1] : row.value }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
