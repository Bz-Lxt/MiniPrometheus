import { reactive } from 'vue'

export type Toast = { id: number; text: string; kind: 'ok' | 'err' | 'warn' }

const state = reactive({ items: [] as Toast[] })
let seq = 1

export function useToasts() {
  return state
}

export function toast(text: string, kind: Toast['kind'] = 'ok') {
  const id = seq++
  state.items.push({ id, text, kind })
  setTimeout(() => dismiss(id), 5000)
}

export function dismiss(id: number) {
  state.items = state.items.filter((t) => t.id !== id)
}
