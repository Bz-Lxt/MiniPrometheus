<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import type { ProfileNode } from '../lib/api'

const props = defineProps<{ root?: ProfileNode | null }>()
const canvas = ref<HTMLCanvasElement | null>(null)
const selected = ref<ProfileNode | null>(null)

type Box = { x: number; y: number; w: number; h: number; n: ProfileNode }

function flatten(n: ProfileNode, depth: number, acc: { n: ProfileNode; depth: number }[]) {
  acc.push({ n, depth })
  for (const c of n.children || []) flatten(c, depth + 1, acc)
}

function heat(ms: number, max: number) {
  const t = max <= 0 ? 0 : Math.min(1, ms / max)
  const r = Math.round(80 + t * 175)
  const g = Math.round(224 - t * 140)
  const b = Math.round(197 - t * 80)
  return `rgb(${r},${g},${b})`
}

function draw() {
  const el = canvas.value
  if (!el) return
  const dpr = window.devicePixelRatio || 1
  const w = el.clientWidth
  const h = 360
  el.width = Math.floor(w * dpr)
  el.height = Math.floor(h * dpr)
  const ctx = el.getContext('2d')
  if (!ctx) return
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, w, h)
  ctx.fillStyle = '#0b1016'
  ctx.fillRect(0, 0, w, h)
  if (!props.root) {
    ctx.fillStyle = '#7d8b99'
    ctx.font = '12px IBM Plex Mono'
    ctx.fillText('执行树将在查询后绘制', 16, 28)
    return
  }
  const rows: { n: ProfileNode; depth: number }[] = []
  flatten(props.root, 0, rows)
  const maxMs = Math.max(...rows.map((r) => r.n.duration_ms || 0), 1)
  const boxH = 44
  const gap = 10
  boxes.length = 0
  rows.forEach((row, i) => {
    const x = 16 + row.depth * 28
    const y = 16 + i * (boxH + gap)
    const bw = Math.min(w - x - 16, 640)
    const b: Box = { x, y, w: bw, h: boxH, n: row.n }
    boxes.push(b)
    ctx.fillStyle = heat(row.n.duration_ms, maxMs)
    ctx.globalAlpha = 0.18
    ctx.fillRect(x, y, bw, boxH)
    ctx.globalAlpha = 1
    ctx.strokeStyle = selected.value?.id === row.n.id ? '#f5c16c' : '#1d2733'
    ctx.strokeRect(x, y, bw, boxH)
    ctx.fillStyle = '#d7e0ea'
    ctx.font = '600 13px IBM Plex Sans Condensed'
    ctx.fillText(row.n.op, x + 10, y + 18)
    ctx.fillStyle = '#7d8b99'
    ctx.font = '11px IBM Plex Mono'
    ctx.fillText(
      `${row.n.duration_ms?.toFixed(2) ?? 0}ms  in=${row.n.in_series} out=${row.n.out_series} scan=${row.n.samples_scanned}${row.n.hit_index ? '  idx' : ''}`,
      x + 10,
      y + 34,
    )
  })
}

const boxes: Box[] = []

function onClick(e: MouseEvent) {
  const el = canvas.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const x = e.clientX - r.left
  const y = e.clientY - r.top
  const hit = [...boxes].reverse().find((b) => x >= b.x && x <= b.x + b.w && y >= b.y && y <= b.y + b.h)
  selected.value = hit?.n ?? null
  draw()
}

onMounted(() => {
  draw()
  canvas.value?.addEventListener('click', onClick)
  window.addEventListener('resize', draw)
})
onUnmounted(() => {
  canvas.value?.removeEventListener('click', onClick)
  window.removeEventListener('resize', draw)
})
watch(() => props.root, draw, { deep: true })
</script>

<template>
  <div>
    <canvas ref="canvas" class="w-full block" style="height: 360px"></canvas>
    <div v-if="selected" class="mt-3 border border-line bg-panel p-3 font-mono text-xs text-mute">
      <div class="text-amber mb-1">{{ selected.op }}</div>
      <pre class="whitespace-pre-wrap">{{ selected.detail }}</pre>
    </div>
  </div>
</template>
