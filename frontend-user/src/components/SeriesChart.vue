<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { formatTs } from '../lib/api'

type Series = { metric: Record<string, string>; values: [number, string][] }

const props = defineProps<{ series: Series[]; height?: number }>()
const canvas = ref<HTMLCanvasElement | null>(null)
let raf = 0

function color(i: number) {
  const hues = [166, 42, 200, 12, 280, 90]
  return `hsl(${hues[i % hues.length]} 70% 62%)`
}

function draw() {
  const el = canvas.value
  if (!el) return
  const dpr = window.devicePixelRatio || 1
  const w = el.clientWidth
  const h = props.height ?? 220
  el.width = Math.floor(w * dpr)
  el.height = Math.floor(h * dpr)
  const ctx = el.getContext('2d')
  if (!ctx) return
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, w, h)
  ctx.fillStyle = '#0b1016'
  ctx.fillRect(0, 0, w, h)
  const series = props.series.slice(0, 80)
  let minT = Infinity, maxT = -Infinity, minV = Infinity, maxV = -Infinity
  for (const s of series) {
    for (const [t, vs] of s.values || []) {
      const v = Number(vs)
      if (!Number.isFinite(v)) continue
      if (t < minT) minT = t
      if (t > maxT) maxT = t
      if (v < minV) minV = v
      if (v > maxV) maxV = v
    }
  }
  if (!Number.isFinite(minT) || maxT <= minT) {
    ctx.fillStyle = '#7d8b99'
    ctx.font = '12px IBM Plex Mono'
    ctx.fillText('等待样本…', 16, 28)
    return
  }
  if (maxV === minV) {
    maxV += 1
    minV -= 1
  }
  const pad = { l: 48, r: 12, t: 12, b: 28 }
  ctx.strokeStyle = '#1d2733'
  ctx.lineWidth = 1
  for (let i = 0; i <= 4; i++) {
    const y = pad.t + ((h - pad.t - pad.b) * i) / 4
    ctx.beginPath()
    ctx.moveTo(pad.l, y)
    ctx.lineTo(w - pad.r, y)
    ctx.stroke()
    const val = maxV - ((maxV - minV) * i) / 4
    ctx.fillStyle = '#7d8b99'
    ctx.font = '10px IBM Plex Mono'
    ctx.fillText(val.toFixed(2), 4, y + 3)
  }
  ctx.strokeStyle = '#5ee0c5'
  series.forEach((s, i) => {
    ctx.beginPath()
    ctx.strokeStyle = color(i)
    ctx.lineWidth = 1.2
    let started = false
    for (const [t, vs] of s.values || []) {
      const v = Number(vs)
      if (!Number.isFinite(v)) continue
      const x = pad.l + ((t - minT) / (maxT - minT)) * (w - pad.l - pad.r)
      const y = pad.t + (1 - (v - minV) / (maxV - minV)) * (h - pad.t - pad.b)
      if (!started) {
        ctx.moveTo(x, y)
        started = true
      } else ctx.lineTo(x, y)
    }
    ctx.stroke()
  })
  ctx.fillStyle = '#7d8b99'
  ctx.font = '10px IBM Plex Mono'
  ctx.fillText(formatTs(minT * 1000), pad.l, h - 8)
  ctx.textAlign = 'right'
  ctx.fillText(formatTs(maxT * 1000), w - pad.r, h - 8)
  ctx.textAlign = 'left'
}

function loop() {
  draw()
  raf = requestAnimationFrame(loop)
}

onMounted(() => {
  loop()
  window.addEventListener('resize', draw)
})
onUnmounted(() => {
  cancelAnimationFrame(raf)
  window.removeEventListener('resize', draw)
})
watch(() => props.series, draw, { deep: true })
</script>

<template>
  <canvas ref="canvas" class="w-full block" :style="{ height: (height ?? 220) + 'px' }"></canvas>
</template>
