import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '@/api/client'

export function useMonitor(pollMs = 2000) {
  const data = ref<any>(null)
  const wsOk = ref(false)
  let ws: WebSocket | null = null
  let timer: number | null = null
  let stopped = false

  async function poll() {
    try {
      const r = await api.get('/api/v1/monitor/stats')
      data.value = r.data
    } catch {}
  }

  function connectWS() {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = localStorage.getItem('access_token')
    const url = `${proto}//${location.host}/api/v1/monitor/ws` + (token ? `?token=${encodeURIComponent(token)}` : '')
    try {
      ws = new WebSocket(url)
      ws.onopen = () => { wsOk.value = true }
      ws.onmessage = (e) => {
        try { data.value = JSON.parse(e.data) } catch {}
      }
      ws.onclose = () => {
        wsOk.value = false
        if (!stopped) timer = window.setTimeout(poll, pollMs) as any
      }
      ws.onerror = () => { ws?.close() }
    } catch {
      wsOk.value = false
    }
  }

  onMounted(() => {
    stopped = false
    // Try WS first, fallback to poll
    connectWS()
    // Always poll as fallback after 3s if WS not ok
    setTimeout(() => { if (!wsOk.value) { poll(); timer = window.setInterval(poll, pollMs) as any } }, 3000)
  })

  onUnmounted(() => {
    stopped = true
    ws?.close()
    if (timer) { clearInterval(timer); clearTimeout(timer) }
  })

  return { data, wsOk, poll }
}
