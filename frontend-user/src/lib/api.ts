export const API = ''

export type ProfileNode = {
  id: number
  op: string
  detail: string
  duration_ms: number
  in_series: number
  out_series: number
  samples_scanned: number
  hit_index: boolean
  children?: ProfileNode[]
}

export type QueryResp = {
  status: string
  error?: string
  errorType?: string
  partial?: boolean
  degraded_shards?: string[]
  profile?: ProfileNode
  data?: {
    resultType: string
    result: any
  }
}

export async function queryRange(q: string, minutes = 15, maxPoints = 400): Promise<QueryResp> {
  const end = Date.now() / 1000
  const start = end - minutes * 60
  const u = `/api/v1/query_range?query=${encodeURIComponent(q)}&start=${start}&end=${end}&step=5&max_points=${maxPoints}`
  const r = await fetch(u)
  return r.json()
}

export async function queryInstant(q: string): Promise<QueryResp> {
  const r = await fetch(`/api/v1/query?query=${encodeURIComponent(q)}`)
  return r.json()
}

export async function getJSON<T>(path: string): Promise<T> {
  const r = await fetch(path)
  if (!r.ok) throw new Error(`${path} ${r.status}`)
  return r.json()
}

export function formatTs(ms: number): string {
  const d = new Date(ms)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
