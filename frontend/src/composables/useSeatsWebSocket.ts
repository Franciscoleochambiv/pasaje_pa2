import { ref, onUnmounted } from 'vue'
import type { TripSeatResponse } from '../api/client'

export function useSeatsWebSocket(tripId: number, onUpdate: (data: TripSeatResponse) => void) {
  const connected = ref(false)
  let socket: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws/trips/${tripId}/seats`

    socket = new WebSocket(url)

    socket.onopen = () => {
      connected.value = true
      console.log(`[WS] Connected to trip ${tripId}`)
    }

    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'seat_update' && msg.data) {
          onUpdate(msg.data as TripSeatResponse)
        }
      } catch (e) {
        console.warn('[WS] Failed to parse message', e)
      }
    }

    socket.onclose = () => {
      connected.value = false
      console.log(`[WS] Disconnected from trip ${tripId}, reconnecting...`)
      reconnectTimer = setTimeout(connect, 3000)
    }

    socket.onerror = () => {
      socket?.close()
    }
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = null
    if (socket) {
      socket.onclose = null // prevent reconnect
      socket.close()
      socket = null
    }
    connected.value = false
  }

  connect()

  onUnmounted(() => {
    disconnect()
  })

  return { connected, disconnect }
}
