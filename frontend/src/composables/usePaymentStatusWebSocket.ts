import { ref, onUnmounted } from 'vue'

const wsBase = import.meta.env.VITE_WS_URL ?? ''

export interface PaymentStatusUpdate {
  type: string
  reservation_code: string
  status: string
  rejection_reason?: string
  billing_sale_id?: number
  ticket_codes?: string[]
}

export function usePaymentStatusWebSocket(
  reservationCode: string,
  onStatusUpdate: (data: PaymentStatusUpdate) => void
) {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let stopped = false

  function connect() {
    if (stopped) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const base = wsBase || `${protocol}//${window.location.host}`
    const url = `${base}/ws/reservations/${reservationCode}/status`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as PaymentStatusUpdate
        if (data.type === 'payment_status_update') {
          onStatusUpdate(data)
        }
      } catch { /* ignore parse errors */ }
    }

    ws.onclose = () => {
      connected.value = false
      if (!stopped) {
        reconnectTimer = setTimeout(connect, 3000)
      }
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function disconnect() {
    stopped = true
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws) {
      ws.close()
      ws = null
    }
    connected.value = false
  }

  connect()

  onUnmounted(() => {
    disconnect()
  })

  return { connected, disconnect }
}
