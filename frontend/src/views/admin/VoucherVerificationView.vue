<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  getPendingVouchers,
  approveVoucher,
  rejectVoucher,
  retryComprobante,
  getVoucherImageUrl,
  type PendingVoucher,
} from '../../api/client'
import Swal from 'sweetalert2'

// ── State ──
const vouchers = ref<PendingVoucher[]>([])
const loading = ref(false)
const error = ref('')
const processing = ref<number | null>(null)
const now = ref(Date.now())
const imageModal = ref<PendingVoucher | null>(null)
const imageLoading = ref(false)

// Track known payment IDs for new-arrival detection
const knownIds = ref<Set<number>>(new Set())
let refreshTimer: ReturnType<typeof setInterval> | null = null
let clockTimer: ReturnType<typeof setInterval> | null = null

// ── Audio notification ──
let audioCtx: AudioContext | null = null

function playBeep() {
  try {
    if (!audioCtx) audioCtx = new AudioContext()
    const osc = audioCtx.createOscillator()
    const gain = audioCtx.createGain()
    osc.connect(gain)
    gain.connect(audioCtx.destination)
    osc.type = 'sine'
    osc.frequency.value = 880
    gain.gain.value = 0.3
    osc.start()
    gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + 0.4)
    osc.stop(audioCtx.currentTime + 0.4)
  } catch {
    // Silently ignore audio errors
  }
}

// ── Browser notifications ──
function requestNotificationPermission() {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
}

function sendBrowserNotification(voucher: PendingVoucher) {
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification('Nuevo voucher Yape pendiente', {
      body: `${voucher.passenger_name} - S/ ${formatAmount(voucher.amount_cents)} - ${voucher.reservation_code}`,
      icon: '/favicon.ico',
    })
  }
}

// ── Data loading ──
async function loadVouchers() {
  if (loading.value) return
  loading.value = vouchers.value.length === 0
  error.value = ''
  try {
    const data = await getPendingVouchers()
    // Sort oldest first
    data.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())

    // Detect new arrivals
    if (knownIds.value.size > 0) {
      for (const v of data) {
        if (!knownIds.value.has(v.payment_id)) {
          playBeep()
          sendBrowserNotification(v)
        }
      }
    }

    // Update known IDs
    knownIds.value = new Set(data.map((v) => v.payment_id))
    vouchers.value = data
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

// ── Formatters ──
function formatAmount(cents: number): string {
  return (cents / 100).toFixed(2)
}

function timeElapsed(isoDate: string): string {
  const diff = now.value - new Date(isoDate).getTime()
  if (diff < 0) return 'ahora'
  const secs = Math.floor(diff / 1000)
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  const remMins = mins % 60
  return `${hours}h ${remMins}m`
}

function countdown(isoDate: string): string {
  const diff = new Date(isoDate).getTime() - now.value
  if (diff <= 0) return 'Expirado'
  const secs = Math.floor(diff / 1000)
  const mins = Math.floor(secs / 60)
  const remSecs = secs % 60
  if (mins >= 60) {
    const hours = Math.floor(mins / 60)
    const remMins = mins % 60
    return `${hours}h ${remMins}m`
  }
  return `${mins}:${String(remSecs).padStart(2, '0')}`
}

function isExpiringSoon(isoDate: string): boolean {
  const diff = new Date(isoDate).getTime() - now.value
  return diff > 0 && diff < 5 * 60 * 1000 // less than 5 min
}

function isExpired(isoDate: string): boolean {
  return new Date(isoDate).getTime() <= now.value
}

const pendingCount = computed(() => vouchers.value.length)

// ── Actions ──
function openImage(voucher: PendingVoucher) {
  imageModal.value = voucher
  imageLoading.value = true
}

function closeModal() {
  imageModal.value = null
  imageLoading.value = false
}

async function handleApprove(voucher: PendingVoucher) {
  // Cerrar el modal de imagen antes del Swal para evitar z-index stack
  // que obliga al operador a cerrarlo manualmente.
  if (imageModal.value?.payment_id === voucher.payment_id) closeModal()
  const result = await Swal.fire({
    title: 'Aprobar voucher',
    html: `
      <div style="text-align:left;font-size:0.9rem;line-height:1.7;color:#475569">
        <p><strong>Reserva:</strong> ${voucher.reservation_code}</p>
        <p><strong>Pasajero:</strong> ${voucher.passenger_name}</p>
        <p><strong>Monto:</strong> S/ ${formatAmount(voucher.amount_cents)}</p>
        <p><strong>Asientos:</strong> ${voucher.seat_labels.join(', ')}</p>
      </div>
      <p style="margin-top:1rem;font-size:0.85rem;color:#64748b">Se confirmara la reserva y se generaran los boletos.</p>
    `,
    icon: 'question',
    showCancelButton: true,
    confirmButtonText: 'Aprobar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#059669',
    customClass: {
      popup: 'swal-custom-popup',
      confirmButton: 'swal-btn-confirm',
      cancelButton: 'swal-btn-cancel',
    },
  })

  if (!result.isConfirmed) return

  processing.value = voucher.payment_id
  try {
    const res = await approveVoucher(voucher.payment_id)
    if (imageModal.value?.payment_id === voucher.payment_id) closeModal()
    await loadVouchers()
    Swal.fire({
      toast: true, position: 'top-end', icon: 'success',
      title: `Aprobado — ${res.reservation_code}. Comprobante en proceso.`,
      showConfirmButton: false, timer: 4000,
    })
  } catch (e) {
    Swal.fire({
      icon: 'error',
      title: 'Error al aprobar',
      text: (e as Error).message,
      confirmButtonColor: '#1b55f5',
      customClass: { popup: 'swal-custom-popup' },
    })
  } finally {
    processing.value = null
  }
}

async function handleReject(voucher: PendingVoucher) {
  if (imageModal.value?.payment_id === voucher.payment_id) closeModal()
  const result = await Swal.fire({
    title: 'Rechazar voucher',
    html: `
      <div style="text-align:left;font-size:0.9rem;line-height:1.7;color:#475569;margin-bottom:0.75rem">
        <p><strong>Reserva:</strong> ${voucher.reservation_code}</p>
        <p><strong>Pasajero:</strong> ${voucher.passenger_name}</p>
        <p><strong>Monto:</strong> S/ ${formatAmount(voucher.amount_cents)}</p>
      </div>
    `,
    input: 'textarea',
    inputLabel: 'Motivo del rechazo',
    inputPlaceholder: 'Ej: Voucher ilegible, monto incorrecto, imagen no corresponde...',
    inputAttributes: {
      'aria-label': 'Motivo del rechazo',
    },
    inputValidator: (value) => {
      if (!value || !value.trim()) return 'Debes ingresar un motivo de rechazo'
      return null
    },
    icon: 'warning',
    showCancelButton: true,
    confirmButtonText: 'Rechazar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#dc2626',
    customClass: {
      popup: 'swal-custom-popup',
      confirmButton: 'swal-btn-confirm',
      cancelButton: 'swal-btn-cancel',
    },
  })

  if (!result.isConfirmed) return

  processing.value = voucher.payment_id
  try {
    await rejectVoucher(voucher.payment_id, result.value)
    if (imageModal.value?.payment_id === voucher.payment_id) closeModal()
    await loadVouchers()
    Swal.fire({
      toast: true,
      position: 'top-end',
      icon: 'info',
      title: 'Voucher rechazado',
      showConfirmButton: false,
      timer: 3000,
    })
  } catch (e) {
    Swal.fire({
      icon: 'error',
      title: 'Error al rechazar',
      text: (e as Error).message,
      confirmButtonColor: '#1b55f5',
      customClass: { popup: 'swal-custom-popup' },
    })
  } finally {
    processing.value = null
  }
}

// ── Lifecycle ──
onMounted(() => {
  requestNotificationPermission()
  loadVouchers()
  refreshTimer = setInterval(loadVouchers, 15000)
  clockTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (clockTimer) clearInterval(clockTimer)
  if (audioCtx) {
    audioCtx.close()
    audioCtx = null
  }
})
</script>

<template>
  <div class="vv-page">
    <div class="vv-header">
      <div class="vv-header-left">
        <h1 class="vv-title">Verificacion de Vouchers</h1>
        <p class="vv-subtitle">Revision y aprobacion de pagos Yape directos</p>
      </div>
      <div class="vv-header-right">
        <span class="vv-badge" :class="{ 'vv-badge-active': pendingCount > 0 }">
          {{ pendingCount }} pendiente{{ pendingCount !== 1 ? 's' : '' }}
        </span>
        <button class="vv-btn vv-btn-refresh" @click="loadVouchers" :disabled="loading">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" :class="{ 'spin-icon': loading }">
            <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/>
            <path d="M21 3v5h-5"/>
          </svg>
          Actualizar
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading && vouchers.length === 0" class="vv-loading">
      <div class="vv-spinner"></div>
      <span>Cargando vouchers pendientes...</span>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="vv-alert vv-alert-error">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" x2="12" y1="8" y2="12"/>
        <line x1="12" x2="12.01" y1="16" y2="16"/>
      </svg>
      <span>{{ error }}</span>
      <button class="vv-btn vv-btn-ghost" @click="loadVouchers">Reintentar</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="vouchers.length === 0" class="vv-empty">
      <div class="vv-empty-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
          <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
      </div>
      <h2 class="vv-empty-title">No hay vouchers pendientes</h2>
      <p class="vv-empty-text">Todos los pagos Yape han sido verificados. Se actualizara automaticamente cada 15 segundos.</p>
    </div>

    <!-- Voucher Table -->
    <div v-else class="vv-table-wrapper">
      <table class="vv-table">
        <thead>
          <tr>
            <th>Reserva</th>
            <th>Pasajero</th>
            <th>Documento</th>
            <th>Ruta</th>
            <th class="vv-th-right">Monto</th>
            <th>Asientos</th>
            <th>Enviado</th>
            <th>Expira en</th>
            <th class="vv-th-center">Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="v in vouchers"
            :key="v.payment_id"
            class="vv-row"
            :class="{
              'vv-row-expired': isExpired(v.hold_expires_at),
              'vv-row-expiring': !isExpired(v.hold_expires_at) && isExpiringSoon(v.hold_expires_at),
              'vv-row-processing': processing === v.payment_id,
            }"
          >
            <td>
              <span class="vv-code">{{ v.reservation_code }}</span>
            </td>
            <td>
              <span class="vv-passenger-name">{{ v.passenger_name }}</span>
            </td>
            <td>
              <span class="vv-doc">{{ v.passenger_doc_type }} {{ v.passenger_doc_number }}</span>
            </td>
            <td>
              <span class="vv-route">{{ v.route_name }}</span>
            </td>
            <td class="vv-td-right">
              <span class="vv-amount">S/ {{ formatAmount(v.amount_cents) }}</span>
            </td>
            <td>
              <div class="vv-seats">
                <span v-for="label in v.seat_labels" :key="label" class="vv-seat-tag">{{ label }}</span>
              </div>
            </td>
            <td>
              <span class="vv-elapsed">{{ timeElapsed(v.created_at) }}</span>
            </td>
            <td>
              <span
                class="vv-countdown"
                :class="{
                  'vv-countdown-danger': isExpired(v.hold_expires_at),
                  'vv-countdown-warning': !isExpired(v.hold_expires_at) && isExpiringSoon(v.hold_expires_at),
                }"
              >
                {{ countdown(v.hold_expires_at) }}
              </span>
            </td>
            <td>
              <div class="vv-actions">
                <button
                  class="vv-btn vv-btn-view"
                  @click="openImage(v)"
                  title="Ver voucher"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                    <circle cx="12" cy="12" r="3"/>
                  </svg>
                </button>
                <button
                  class="vv-btn vv-btn-approve"
                  @click="handleApprove(v)"
                  :disabled="processing === v.payment_id"
                  title="Aprobar"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                </button>
                <button
                  class="vv-btn vv-btn-reject"
                  @click="handleReject(v)"
                  :disabled="processing === v.payment_id"
                  title="Rechazar"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" x2="6" y1="6" y2="18"/>
                    <line x1="6" x2="18" y1="6" y2="18"/>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile Cards (shown on small screens instead of table) -->
    <div v-if="vouchers.length > 0" class="vv-cards-mobile">
      <div
        v-for="v in vouchers"
        :key="'card-' + v.payment_id"
        class="vv-card"
        :class="{
          'vv-card-expired': isExpired(v.hold_expires_at),
          'vv-card-expiring': !isExpired(v.hold_expires_at) && isExpiringSoon(v.hold_expires_at),
        }"
      >
        <div class="vv-card-header">
          <span class="vv-code">{{ v.reservation_code }}</span>
          <span
            class="vv-countdown"
            :class="{
              'vv-countdown-danger': isExpired(v.hold_expires_at),
              'vv-countdown-warning': !isExpired(v.hold_expires_at) && isExpiringSoon(v.hold_expires_at),
            }"
          >
            {{ countdown(v.hold_expires_at) }}
          </span>
        </div>
        <div class="vv-card-body">
          <div class="vv-card-row">
            <span class="vv-card-label">Pasajero</span>
            <span>{{ v.passenger_name }}</span>
          </div>
          <div class="vv-card-row">
            <span class="vv-card-label">Documento</span>
            <span>{{ v.passenger_doc_type }} {{ v.passenger_doc_number }}</span>
          </div>
          <div class="vv-card-row">
            <span class="vv-card-label">Ruta</span>
            <span>{{ v.route_name }}</span>
          </div>
          <div class="vv-card-row">
            <span class="vv-card-label">Monto</span>
            <span class="vv-amount">S/ {{ formatAmount(v.amount_cents) }}</span>
          </div>
          <div class="vv-card-row">
            <span class="vv-card-label">Asientos</span>
            <div class="vv-seats">
              <span v-for="label in v.seat_labels" :key="label" class="vv-seat-tag">{{ label }}</span>
            </div>
          </div>
          <div class="vv-card-row">
            <span class="vv-card-label">Enviado hace</span>
            <span class="vv-elapsed">{{ timeElapsed(v.created_at) }}</span>
          </div>
        </div>
        <div class="vv-card-actions">
          <button class="vv-btn vv-btn-view vv-btn-card" @click="openImage(v)">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            Ver voucher
          </button>
          <button
            class="vv-btn vv-btn-approve vv-btn-card"
            @click="handleApprove(v)"
            :disabled="processing === v.payment_id"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
            Aprobar
          </button>
          <button
            class="vv-btn vv-btn-reject vv-btn-card"
            @click="handleReject(v)"
            :disabled="processing === v.payment_id"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" x2="6" y1="6" y2="18"/>
              <line x1="6" x2="18" y1="6" y2="18"/>
            </svg>
            Rechazar
          </button>
        </div>
      </div>
    </div>

    <!-- Image Modal -->
    <Teleport to="body">
      <Transition name="vv-modal">
        <div v-if="imageModal" class="vv-overlay" @click.self="closeModal">
          <div class="vv-modal">
            <div class="vv-modal-header">
              <div>
                <h2 class="vv-modal-title">Voucher de pago</h2>
                <p class="vv-modal-sub">{{ imageModal.reservation_code }} &mdash; {{ imageModal.passenger_name }}</p>
              </div>
              <button class="vv-btn vv-btn-close" @click="closeModal">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" x2="6" y1="6" y2="18"/>
                  <line x1="6" x2="18" y1="6" y2="18"/>
                </svg>
              </button>
            </div>

            <div class="vv-modal-info">
              <div class="vv-modal-info-item">
                <span class="vv-modal-info-label">Monto</span>
                <span class="vv-amount">S/ {{ formatAmount(imageModal.amount_cents) }}</span>
              </div>
              <div class="vv-modal-info-item">
                <span class="vv-modal-info-label">Documento</span>
                <span>{{ imageModal.passenger_doc_type }} {{ imageModal.passenger_doc_number }}</span>
              </div>
              <div class="vv-modal-info-item">
                <span class="vv-modal-info-label">Ruta</span>
                <span>{{ imageModal.route_name }}</span>
              </div>
              <div class="vv-modal-info-item">
                <span class="vv-modal-info-label">Asientos</span>
                <span>{{ imageModal.seat_labels.join(', ') }}</span>
              </div>
              <div class="vv-modal-info-item">
                <span class="vv-modal-info-label">Expira en</span>
                <span
                  class="vv-countdown"
                  :class="{
                    'vv-countdown-danger': isExpired(imageModal.hold_expires_at),
                    'vv-countdown-warning': !isExpired(imageModal.hold_expires_at) && isExpiringSoon(imageModal.hold_expires_at),
                  }"
                >
                  {{ countdown(imageModal.hold_expires_at) }}
                </span>
              </div>
            </div>

            <div class="vv-modal-image-wrapper">
              <div v-if="imageLoading" class="vv-modal-image-loading">
                <div class="vv-spinner"></div>
                <span>Cargando imagen...</span>
              </div>
              <img
                :src="getVoucherImageUrl(imageModal.payment_id)"
                alt="Voucher de pago Yape"
                class="vv-modal-image"
                @load="imageLoading = false"
                @error="imageLoading = false"
              />
            </div>

            <div class="vv-modal-actions">
              <button
                class="vv-btn vv-btn-approve vv-btn-lg"
                @click="handleApprove(imageModal)"
                :disabled="processing === imageModal.payment_id"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                Aprobar pago
              </button>
              <button
                class="vv-btn vv-btn-reject vv-btn-lg"
                @click="handleReject(imageModal)"
                :disabled="processing === imageModal.payment_id"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" x2="6" y1="6" y2="18"/>
                  <line x1="6" x2="18" y1="6" y2="18"/>
                </svg>
                Rechazar pago
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
/* ── Page Layout ── */
.vv-page {
  max-width: 100%;
}

.vv-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 1.75rem;
  gap: 1rem;
  flex-wrap: wrap;
}

.vv-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.vv-subtitle {
  font-size: 0.88rem;
  color: var(--slate-500);
  margin-top: 0.2rem;
}

.vv-header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

/* ── Badge ── */
.vv-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.35rem 0.85rem;
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  font-weight: 600;
  background: var(--slate-100);
  color: var(--slate-600);
  border: 1px solid var(--slate-200);
}

.vv-badge-active {
  background: var(--warning-50);
  color: var(--warning-700);
  border-color: var(--warning-100);
  animation: vv-pulse 2s ease-in-out infinite;
}

@keyframes vv-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

/* ── Buttons ── */
.vv-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  font-weight: 600;
  font-size: 0.82rem;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s ease;
  line-height: 1;
  white-space: nowrap;
}

.vv-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.vv-btn-refresh {
  padding: 0.5rem 1rem;
  background: white;
  color: var(--slate-700);
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
}

.vv-btn-refresh:hover:not(:disabled) {
  background: var(--brand-50);
  border-color: var(--slate-400);
}

.vv-btn-ghost {
  padding: 0.45rem 0.85rem;
  background: transparent;
  color: var(--slate-500);
  border: 2px solid var(--slate-300);
}

.vv-btn-ghost:hover {
  background: var(--slate-100);
  color: var(--slate-900);
}

.vv-btn-view {
  padding: 0.4rem;
  background: var(--brand-50);
  color: var(--brand-600);
  border-radius: var(--radius-sm);
}

.vv-btn-view:hover {
  background: var(--brand-100);
}

.vv-btn-approve {
  padding: 0.4rem;
  background: var(--success-50);
  color: var(--success-700);
  border-radius: var(--radius-sm);
}

.vv-btn-approve:hover:not(:disabled) {
  background: var(--success-100);
}

.vv-btn-reject {
  padding: 0.4rem;
  background: var(--danger-50);
  color: var(--danger-700);
  border-radius: var(--radius-sm);
}

.vv-btn-reject:hover:not(:disabled) {
  background: var(--danger-100);
}

.vv-btn-lg {
  padding: 0.65rem 1.25rem;
  font-size: 0.88rem;
  border-radius: var(--radius-md);
}

.vv-btn-card {
  padding: 0.5rem 0.85rem;
  flex: 1;
}

.vv-btn-close {
  padding: 0.35rem;
  background: var(--slate-100);
  color: var(--slate-500);
  border-radius: var(--radius-sm);
}

.vv-btn-close:hover {
  background: var(--slate-200);
  color: var(--slate-700);
}

.spin-icon {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Loading ── */
.vv-loading {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 0;
  justify-content: center;
  color: var(--slate-500);
  font-size: 0.9rem;
}

.vv-spinner {
  width: 22px;
  height: 22px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* ── Alert ── */
.vv-alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  margin-bottom: 1.5rem;
}

.vv-alert-error {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
}

/* ── Empty State ── */
.vv-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 4rem 2rem;
  text-align: center;
}

.vv-empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: var(--radius-xl);
  background: var(--success-50);
  color: var(--success-600);
  margin-bottom: 1.25rem;
}

.vv-empty-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 0.4rem;
}

.vv-empty-text {
  font-size: 0.88rem;
  color: var(--slate-500);
  max-width: 360px;
}

/* ── Table ── */
.vv-table-wrapper {
  background: white;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  overflow-x: auto;
}

.vv-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.vv-table thead {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
}

.vv-table th {
  padding: 0.75rem 0.85rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: white;
  white-space: nowrap;
  border-bottom: none;
}

.vv-th-right {
  text-align: right !important;
}

.vv-th-center {
  text-align: center !important;
}

.vv-table tbody tr {
  border-bottom: 1px solid var(--color-border);
  transition: background-color 0.15s ease;
}

.vv-table tbody tr:last-child {
  border-bottom: none;
}

.vv-table tbody tr:hover {
  background: var(--brand-50);
}

.vv-table td {
  padding: 0.7rem 0.85rem;
  vertical-align: middle;
  color: var(--slate-700);
}

.vv-td-right {
  text-align: right;
}

/* Row states */
.vv-row-expired {
  background: var(--danger-50) !important;
}

.vv-row-expiring {
  background: var(--warning-50) !important;
}

.vv-row-processing {
  opacity: 0.6;
  pointer-events: none;
}

/* ── Cell content ── */
.vv-code {
  font-family: 'SF Mono', 'Fira Code', 'Fira Mono', 'Roboto Mono', monospace;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--brand-600);
  background: var(--brand-50);
  padding: 0.2rem 0.5rem;
  border-radius: var(--radius-sm);
}

.vv-passenger-name {
  font-weight: 600;
  color: var(--slate-900);
}

.vv-doc {
  font-size: 0.82rem;
  color: var(--slate-500);
}

.vv-route {
  font-size: 0.82rem;
  color: var(--slate-700);
}

.vv-amount {
  font-weight: 700;
  color: var(--slate-900);
  white-space: nowrap;
}

.vv-seats {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
}

.vv-seat-tag {
  display: inline-block;
  padding: 0.15rem 0.45rem;
  background: var(--slate-100);
  color: var(--slate-700);
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: var(--radius-sm);
}

.vv-elapsed {
  font-size: 0.82rem;
  color: var(--slate-500);
}

.vv-countdown {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--success-600);
}

.vv-countdown-warning {
  color: var(--warning-600);
}

.vv-countdown-danger {
  color: var(--danger-600);
  font-weight: 700;
}

.vv-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
}

/* ── Mobile Cards (hidden on desktop) ── */
.vv-cards-mobile {
  display: none;
}

/* ── Modal ── */
.vv-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.6);
  backdrop-filter: blur(4px);
  padding: 1rem;
}

.vv-modal {
  background: white;
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  width: 100%;
  max-width: 560px;
  max-height: 90vh;
  overflow-y: auto;
  border: 2px solid var(--slate-300);
}

.vv-modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.vv-modal-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--slate-900);
}

.vv-modal-sub {
  font-size: 0.82rem;
  color: var(--slate-500);
  margin-top: 0.15rem;
}

.vv-modal-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.6rem 1.25rem;
  padding: 1rem 1.5rem;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  border-bottom: 1px solid var(--color-border);
}

.vv-modal-info-item {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.vv-modal-info-label {
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--slate-500);
}

.vv-modal-info-item > span:last-child {
  font-size: 0.88rem;
  color: var(--slate-900);
  font-weight: 500;
}

.vv-modal-image-wrapper {
  position: relative;
  padding: 1rem 1.5rem;
  display: flex;
  justify-content: center;
  min-height: 200px;
  background: var(--slate-100);
}

.vv-modal-image-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: var(--slate-500);
  font-size: 0.85rem;
}

.vv-modal-image {
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
  border-radius: var(--radius-md);
  border: 2px solid var(--slate-300);
}

.vv-modal-actions {
  display: flex;
  gap: 0.75rem;
  padding: 1.25rem 1.5rem;
  border-top: 1px solid var(--color-border);
}

.vv-modal-actions .vv-btn {
  flex: 1;
}

/* ── Modal transition ── */
.vv-modal-enter-active,
.vv-modal-leave-active {
  transition: opacity 0.25s ease;
}

.vv-modal-enter-active .vv-modal,
.vv-modal-leave-active .vv-modal {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.vv-modal-enter-from,
.vv-modal-leave-to {
  opacity: 0;
}

.vv-modal-enter-from .vv-modal,
.vv-modal-leave-to .vv-modal {
  transform: translateY(16px) scale(0.97);
  opacity: 0;
}

/* ── Responsive ── */
@media (max-width: 900px) {
  .vv-table-wrapper {
    display: none;
  }

  .vv-cards-mobile {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .vv-card {
    background: white;
    border: 2px solid var(--slate-300);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }

  .vv-card-expired {
    border-color: var(--danger-100);
    background: var(--danger-50);
  }

  .vv-card-expiring {
    border-color: var(--warning-100);
    background: var(--warning-50);
  }

  .vv-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.85rem 1rem;
    background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  }

  .vv-card-header .vv-code {
    background: rgba(255, 255, 255, 0.15);
    color: #ffffff;
  }

  .vv-card-header .vv-countdown {
    color: white;
  }

  .vv-card-header .vv-countdown-warning {
    color: var(--warning-500);
  }

  .vv-card-header .vv-countdown-danger {
    color: var(--danger-500);
  }

  .vv-card-body {
    padding: 0.75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
  }

  .vv-card-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: var(--slate-700);
  }

  .vv-card-label {
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--slate-500);
    text-transform: uppercase;
    letter-spacing: 0.03em;
    flex-shrink: 0;
  }

  .vv-card-actions {
    display: flex;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-top: 1px solid var(--color-border);
  }

  .vv-header {
    flex-direction: column;
  }

  .vv-header-right {
    width: 100%;
    justify-content: space-between;
  }
}

@media (max-width: 480px) {
  .vv-title {
    font-size: 1.25rem;
  }

  .vv-modal {
    max-width: 100%;
    border-radius: var(--radius-lg);
  }

  .vv-modal-info {
    grid-template-columns: 1fr;
  }

  .vv-modal-actions {
    flex-direction: column;
  }

  .vv-card-actions {
    flex-direction: column;
  }

  .vv-btn-card {
    justify-content: center;
  }
}

/* ── Dark mode adjustments ── */
@media (prefers-color-scheme: dark) {
  .vv-seat-tag {
    background: var(--slate-700);
    color: var(--slate-200);
  }

  .vv-code {
    background: rgba(27, 85, 245, 0.15);
    color: var(--brand-300);
  }

  .vv-row-expired {
    background: rgba(239, 68, 68, 0.08) !important;
  }

  .vv-row-expiring {
    background: rgba(245, 158, 11, 0.08) !important;
  }

  .vv-card-expired {
    background: rgba(239, 68, 68, 0.06);
    border-color: var(--slate-700);
  }

  .vv-card-expiring {
    background: rgba(245, 158, 11, 0.06);
    border-color: var(--slate-700);
  }
}
</style>
