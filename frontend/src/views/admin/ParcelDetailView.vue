<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getParcel,
  updateParcelStatus,
  payParcel,
  retryParcelBilling,
  cancelParcel,
  getBillingPDFUrl,
  type Parcel,
  type ParcelTracking,
} from '../../api/client'
import Swal from 'sweetalert2'

const route = useRoute()
const router = useRouter()

// ── State ──
const parcel = ref<Parcel | null>(null)
const tracking = ref<ParcelTracking[]>([])
const loading = ref(false)
const error = ref('')
const actionLoading = ref(false)

// ── Print panel ──
const showPrintPanel = ref(false)
const printSaleId = ref<number | null>(null)
const printParcelCode = ref('')
const printTitle = ref('')
const printWhatsappUrl = ref('')
const printPdfFormat = ref('ticket')

function openPrintPanel(saleId: number, code: string, title: string, waUrl?: string) {
  printSaleId.value = saleId
  printParcelCode.value = code
  printTitle.value = title
  printWhatsappUrl.value = waUrl || ''
  printPdfFormat.value = 'ticket'
  showPrintPanel.value = true
}

function printFromIframe() {
  const iframe = document.querySelector('.print-preview') as HTMLIFrameElement
  if (iframe?.contentWindow) {
    iframe.contentWindow.print()
  } else if (printSaleId.value) {
    window.open(getBillingPDFUrl(printSaleId.value, printPdfFormat.value), '_blank')
  }
}

function printWhatsApp() {
  if (printWhatsappUrl.value) window.open(printWhatsappUrl.value, '_blank')
}

function closePrintPanel() {
  showPrintPanel.value = false
}

// ── Status maps ──
const statusLabels: Record<string, string> = {
  registered: 'Registrada',
  boarded: 'Embarcada',
  in_transit: 'En tránsito',
  arrived: 'Llegó a destino',
  ready_for_pickup: 'Lista para recoger',
  delivered: 'Entregada',
  cancelled: 'Anulada',
}

const statusColors: Record<string, string> = {
  registered: '#6366F1',
  boarded: '#8B5CF6',
  in_transit: '#F59E0B',
  arrived: '#10B981',
  ready_for_pickup: '#3B82F6',
  delivered: '#059669',
  cancelled: '#EF4444',
}

const nextStatusMap: Record<string, string> = {
  registered: 'boarded',
  boarded: 'in_transit',
  in_transit: 'arrived',
  arrived: 'ready_for_pickup',
  ready_for_pickup: 'delivered',
}

const nextStatusLabels: Record<string, string> = {
  registered: 'Embarcar',
  boarded: 'Iniciar Tránsito',
  in_transit: 'Marcar Llegada',
  arrived: 'Lista para Recoger',
  ready_for_pickup: 'Marcar Entregada',
}

const paymentStatusLabels: Record<string, string> = {
  pending: 'Pendiente',
  paid: 'Pagado',
}

const paymentModeLabels: Record<string, string> = {
  origin: 'Origen',
  destination: 'Destino',
}

// ── Computed ──
const parcelId = computed(() => {
  const raw = route.params.id
  const id = Array.isArray(raw) ? raw[0] : raw
  const num = Number(id)
  return Number.isFinite(num) && num > 0 ? num : 0
})

const statusLabel = computed(() =>
  parcel.value ? (statusLabels[parcel.value.status] ?? parcel.value.status) : ''
)

const statusColor = computed(() =>
  parcel.value ? (statusColors[parcel.value.status] ?? '#94A3B8') : '#94A3B8'
)

const paymentStatusLabel = computed(() =>
  parcel.value ? (paymentStatusLabels[parcel.value.payment_status] ?? parcel.value.payment_status) : ''
)

const canTransition = computed(() =>
  parcel.value ? nextStatusMap[parcel.value.status] !== undefined : false
)

const nextStatusLabel = computed(() =>
  parcel.value ? (nextStatusLabels[parcel.value.status] ?? '') : ''
)

const canCancel = computed(() =>
  parcel.value ? !['delivered', 'cancelled'].includes(parcel.value.status) : false
)

const canPay = computed(() =>
  parcel.value ? parcel.value.payment_status === 'pending' : false
)

const amountFormatted = computed(() => {
  if (!parcel.value) return 'S/ 0.00'
  return `S/ ${(parcel.value.amount_cents / 100).toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
})

// ── Load ──
async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await getParcel(parcelId.value)
    parcel.value = result.parcel
    tracking.value = result.tracking
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

// ── Actions ──
async function handleStatusTransition() {
  if (!parcel.value || !canTransition.value) return

  const nextStatus = nextStatusMap[parcel.value.status]
  if (!nextStatus) return

  const { value: formValues } = await Swal.fire({
    title: nextStatusLabels[parcel.value.status],
    html:
      '<input id="swal-location" class="swal2-input" placeholder="Ubicación (opcional)">' +
      '<textarea id="swal-notes" class="swal2-textarea" placeholder="Notas (opcional)"></textarea>',
    focusConfirm: false,
    showCancelButton: true,
    confirmButtonText: 'Confirmar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#7c3aed',
    preConfirm: () => {
      return {
        location: (document.getElementById('swal-location') as HTMLInputElement)?.value ?? '',
        notes: (document.getElementById('swal-notes') as HTMLTextAreaElement)?.value ?? '',
      }
    },
  })

  if (!formValues) return

  actionLoading.value = true
  try {
    await updateParcelStatus(parcel.value.id, {
      status: nextStatus,
      location: formValues.location || undefined,
      notes: formValues.notes || undefined,
    })
    await Swal.fire({
      icon: 'success',
      title: 'Estado actualizado',
      text: `La encomienda ahora está: ${statusLabels[nextStatus]}`,
      confirmButtonColor: '#7c3aed',
      timer: 2000,
      showConfirmButton: false,
    })
    await load()
  } catch (e) {
    await Swal.fire({
      icon: 'error',
      title: 'Error',
      text: (e as Error).message,
      confirmButtonColor: '#7c3aed',
    })
  } finally {
    actionLoading.value = false
  }
}

async function handlePay() {
  if (!parcel.value || !canPay.value) return

  const { value: paymentMethod } = await Swal.fire({
    title: 'Cobrar encomienda',
    text: amountFormatted.value,
    input: 'select',
    inputOptions: {
      efectivo: 'Efectivo',
      yape: 'Yape',
      tarjeta: 'Tarjeta',
    },
    inputPlaceholder: 'Seleccionar método de pago',
    showCancelButton: true,
    confirmButtonText: 'Cobrar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#7c3aed',
    inputValidator: (value) => {
      if (!value) return 'Debe seleccionar un método de pago'
      return null
    },
  })

  if (!paymentMethod) return

  actionLoading.value = true
  try {
    const result = await payParcel(parcel.value.id, paymentMethod)
    await load()

    const updated = result.parcel
    if (updated?.billing_sale_id) {
      openPrintPanel(updated.billing_sale_id, updated.code || parcel.value!.code, 'Comprobante emitido', result.whatsapp_url)
    } else if (result.billing_error) {
      const retryNow = await Swal.fire({
        icon: 'warning',
        title: 'Pago confirmado sin comprobante',
        text: 'El cobro quedó registrado, pero falló la emisión del comprobante. ¿Reintentar ahora?',
        showCancelButton: true,
        confirmButtonText: 'Reintentar emisión',
        cancelButtonText: 'Más tarde',
        confirmButtonColor: '#7c3aed',
      })

      if (retryNow.isConfirmed) {
        const retry = await retryParcelBilling(parcel.value.id)
        await load()
        if (retry.parcel?.billing_sale_id) {
          openPrintPanel(retry.parcel.billing_sale_id, retry.parcel.code || parcel.value!.code, 'Comprobante emitido', retry.whatsapp_url)
        } else if (retry.billing_error) {
          await Swal.fire({
            icon: 'error',
            title: 'No se pudo emitir',
            text: retry.billing_error,
            confirmButtonColor: '#7c3aed',
          })
        }
      }
    } else {
      Swal.fire({ icon: 'success', title: 'Pago registrado', timer: 1500, showConfirmButton: false })
    }
  } catch (e) {
    await Swal.fire({
      icon: 'error',
      title: 'Error al cobrar',
      text: (e as Error).message,
      confirmButtonColor: '#7c3aed',
    })
  } finally {
    actionLoading.value = false
  }
}

async function handleCancel() {
  if (!parcel.value || !canCancel.value) return

  const { value: reason } = await Swal.fire({
    title: 'Anular encomienda',
    input: 'textarea',
    inputPlaceholder: 'Motivo de la anulación...',
    showCancelButton: true,
    confirmButtonText: 'Anular',
    cancelButtonText: 'Volver',
    confirmButtonColor: '#EF4444',
    inputValidator: (value) => {
      if (!value) return 'Debe ingresar un motivo'
      return null
    },
  })

  if (!reason) return

  actionLoading.value = true
  try {
    await cancelParcel(parcel.value.id, reason)
    await Swal.fire({
      icon: 'success',
      title: 'Encomienda anulada',
      confirmButtonColor: '#7c3aed',
      timer: 2000,
      showConfirmButton: false,
    })
    await load()
  } catch (e) {
    await Swal.fire({
      icon: 'error',
      title: 'Error al anular',
      text: (e as Error).message,
      confirmButtonColor: '#7c3aed',
    })
  } finally {
    actionLoading.value = false
  }
}

// ── Helpers ──
function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleDateString('es-PE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatWeight(kg: number): string {
  return kg != null ? `${kg} kg` : '-'
}

onMounted(load)
</script>

<template>
  <div class="parcel-detail">
    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>Cargando encomienda...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
      <button class="btn btn-sm btn-ghost" @click="load">Reintentar</button>
    </div>

    <!-- Content -->
    <template v-else-if="parcel">
      <!-- Header -->
      <div class="detail-header">
        <div class="header-left">
          <button class="btn-back" @click="router.back()">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5"/><path d="m12 19-7-7 7-7"/></svg>
          </button>
          <div>
            <h1 class="parcel-code">{{ parcel.code }}</h1>
            <p class="parcel-subtitle">Encomienda registrada el {{ formatDate(parcel.created_at) }}</p>
          </div>
        </div>
        <div class="header-badges">
          <span class="badge badge-status" :style="{ background: statusColor }">
            {{ statusLabel }}
          </span>
          <span class="badge" :class="parcel.payment_status === 'paid' ? 'badge-paid' : 'badge-pending'">
            {{ paymentStatusLabel }}
          </span>
        </div>
      </div>

      <!-- Info cards grid -->
      <div class="info-grid">
        <!-- Viaje -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="3" width="15" height="13" rx="2"/><path d="M16 8h4a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
            <span>Viaje</span>
          </div>
          <div class="info-card-body">
            <div class="info-row">
              <span class="info-label">Ruta</span>
              <span class="info-value">{{ parcel.route_name }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Salida</span>
              <span class="info-value">{{ formatDate(parcel.departure_at) }}</span>
            </div>
          </div>
        </div>

        <!-- Tramo -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg>
            <span>Tramo</span>
          </div>
          <div class="info-card-body">
            <div class="segment-route">
              <span class="segment-stop">{{ parcel.origin_stop_name }}</span>
              <svg class="segment-arrow" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
              <span class="segment-stop">{{ parcel.dest_stop_name }}</span>
            </div>
          </div>
        </div>

        <!-- Remitente -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            <span>Remitente</span>
          </div>
          <div class="info-card-body">
            <div class="info-row">
              <span class="info-label">Nombre</span>
              <span class="info-value">{{ parcel.sender_name }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Documento</span>
              <span class="info-value">{{ parcel.sender_doc_type }} {{ parcel.sender_doc_number }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Teléfono</span>
              <span class="info-value">{{ parcel.sender_phone || '-' }}</span>
            </div>
          </div>
        </div>

        <!-- Destinatario -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            <span>Destinatario</span>
          </div>
          <div class="info-card-body">
            <div class="info-row">
              <span class="info-label">Nombre</span>
              <span class="info-value">{{ parcel.receiver_name }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Documento</span>
              <span class="info-value">{{ parcel.receiver_doc_type }} {{ parcel.receiver_doc_number }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Teléfono</span>
              <span class="info-value">{{ parcel.receiver_phone || '-' }}</span>
            </div>
          </div>
        </div>

        <!-- Carga -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><path d="m3.27 6.96 8.73 5.05 8.73-5.05"/><path d="M12 22.08V12.01"/></svg>
            <span>Carga</span>
          </div>
          <div class="info-card-body">
            <div class="info-row">
              <span class="info-label">Bultos</span>
              <span class="info-value">{{ parcel.package_count }} bultos</span>
            </div>
            <div class="info-row">
              <span class="info-label">Peso</span>
              <span class="info-value">{{ formatWeight(parcel.weight_kg) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Descripción</span>
              <span class="info-value">{{ parcel.description || '-' }}</span>
            </div>
          </div>
        </div>

        <!-- Pago -->
        <div class="info-card">
          <div class="info-card-header">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            <span>Pago</span>
          </div>
          <div class="info-card-body">
            <div class="info-row">
              <span class="info-label">Monto</span>
              <span class="info-value info-value-amount">{{ amountFormatted }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Modalidad</span>
              <span class="info-value">{{ paymentModeLabels[parcel.payment_mode] ?? parcel.payment_mode }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Método</span>
              <span class="info-value">{{ parcel.payment_method || '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Comprobante</span>
              <span class="info-value">{{ parcel.billing_doc_type || '-' }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Tracking Timeline -->
      <div class="timeline-section">
        <h2 class="section-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
          Seguimiento
        </h2>

        <div v-if="tracking.length === 0" class="empty-timeline">
          Sin eventos de seguimiento.
        </div>

        <div v-else class="timeline">
          <div
            v-for="(event, idx) in tracking"
            :key="event.id"
            class="timeline-item"
            :class="{ 'timeline-item--first': idx === 0 }"
          >
            <div class="timeline-dot" :style="{ background: statusColors[event.status] ?? '#94A3B8' }"></div>
            <div class="timeline-content">
              <div class="timeline-header">
                <span class="timeline-status" :style="{ color: statusColors[event.status] ?? '#94A3B8' }">
                  {{ statusLabels[event.status] ?? event.status }}
                </span>
                <span class="timeline-time">{{ formatDate(event.created_at) }}</span>
              </div>
              <div v-if="event.location" class="timeline-detail">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
                {{ event.location }}
              </div>
              <div v-if="event.notes" class="timeline-detail">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="M16 13H8"/><path d="M16 17H8"/><path d="M10 9H8"/></svg>
                {{ event.notes }}
              </div>
              <div v-if="event.user_name" class="timeline-detail timeline-user">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                {{ event.user_name }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="actions-bar">
        <!-- Next status transition -->
        <button
          v-if="canTransition"
          class="btn btn-primary"
          :disabled="actionLoading"
          @click="handleStatusTransition"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
          {{ nextStatusLabel }}
        </button>

        <!-- Cobrar -->
        <button
          v-if="canPay"
          class="btn btn-pay"
          :disabled="actionLoading"
          @click="handlePay"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
          Cobrar
        </button>

        <!-- Anular -->
        <button
          v-if="canCancel"
          class="btn btn-danger"
          :disabled="actionLoading"
          @click="handleCancel"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/></svg>
          Anular
        </button>

        <!-- Editar -->
        <router-link
          :to="`/admin/encomiendas/${parcel.id}/editar`"
          class="btn btn-outline"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/><path d="m15 5 4 4"/></svg>
          Editar
        </router-link>

        <!-- Ver Comprobante -->
        <button
          v-if="parcel.billing_sale_id"
          class="btn btn-outline"
          @click="openPrintPanel(parcel.billing_sale_id!, parcel.code, 'Comprobante')"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/><path d="M16 13H8"/><path d="M16 17H8"/><path d="M10 9H8"/></svg>
          Ver Comprobante
        </button>
      </div>
    </template>

    <!-- Print Panel -->
    <Teleport to="body">
      <div v-if="showPrintPanel" class="print-overlay">
        <div class="print-panel">
          <div class="print-header">
            <div>
              <h2 class="print-title">{{ printTitle }}</h2>
              <p class="print-code">Encomienda: <strong>{{ printParcelCode }}</strong></p>
            </div>
            <button class="print-close" @click="closePrintPanel">&times;</button>
          </div>
          <div class="print-formats">
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'ticket' }" @click="printPdfFormat = 'ticket'">Ticket 80mm</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4media' }" @click="printPdfFormat = 'a4media'">A4 Media</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4' }" @click="printPdfFormat = 'a4'">A4 Full</button>
          </div>
          <iframe v-if="printSaleId" :src="getBillingPDFUrl(printSaleId, printPdfFormat)" class="print-preview" title="Vista previa"></iframe>
          <div class="print-actions">
            <button class="print-action-btn print-btn-print" @click="printFromIframe">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 6 2 18 2 18 9"/><path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><rect width="12" height="8" x="6" y="14"/></svg>
              Imprimir
            </button>
            <button v-if="printWhatsappUrl" class="print-action-btn print-btn-wa" @click="printWhatsApp">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/></svg>
              WhatsApp
            </button>
            <button class="print-action-btn print-btn-close" @click="closePrintPanel">Cerrar</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.parcel-detail {
  padding: 0;
}

/* ── Loading & Error ── */
.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 0;
  justify-content: center;
  color: var(--slate-500);
}

.spinner {
  width: 22px;
  height: 22px;
  border: 3px solid var(--slate-200);
  border-top-color: #7c3aed;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.alert {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.85rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
}

.alert-error {
  background: #FEF2F2;
  color: #DC2626;
  border: 1px solid #FECACA;
}

/* ── Header ── */
.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.btn-back {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-md);
  background: white;
  color: var(--slate-500);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.btn-back:hover {
  background: var(--slate-50);
  color: var(--slate-900);
  border-color: var(--slate-300);
}

.parcel-code {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: 0.02em;
  line-height: 1.2;
}

.parcel-subtitle {
  font-size: 0.82rem;
  color: var(--slate-500);
  margin-top: 0.15rem;
}

.header-badges {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.3rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

.badge-status {
  color: white;
}

.badge-paid {
  background: #ECFDF5;
  color: #059669;
  border: 1px solid #A7F3D0;
}

.badge-pending {
  background: #FFF7ED;
  color: #D97706;
  border: 1px solid #FED7AA;
}

/* ── Info Grid ── */
.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.info-card {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-md);
  transition: all 0.2s ease;
}

.info-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-1px);
}

.info-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: linear-gradient(135deg, #f5f3ff, #ede9fe);
  border-bottom: 1px solid #e9e5f5;
  font-size: 0.82rem;
  font-weight: 700;
  color: #7c3aed;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.info-card-header svg {
  color: #7c3aed;
  flex-shrink: 0;
}

.info-card-body {
  padding: 0.75rem 1rem;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 0.35rem 0;
  gap: 0.5rem;
}

.info-row + .info-row {
  border-top: 1px solid var(--slate-100);
}

.info-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-500);
  flex-shrink: 0;
}

.info-value {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-900);
  text-align: right;
  word-break: break-word;
}

.info-value-amount {
  font-size: 1rem;
  font-weight: 800;
  color: #7c3aed;
}

/* ── Segment route display ── */
.segment-route {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  justify-content: center;
  padding: 0.5rem 0;
}

.segment-stop {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--slate-900);
}

.segment-arrow {
  color: #7c3aed;
  flex-shrink: 0;
}

/* ── Timeline ── */
.timeline-section {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  box-shadow: var(--shadow-md);
  margin-bottom: 1.5rem;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 1.25rem;
}

.section-title svg {
  color: #7c3aed;
}

.empty-timeline {
  text-align: center;
  padding: 2rem;
  color: var(--slate-400);
  font-size: 0.88rem;
}

.timeline {
  position: relative;
  padding-left: 1.5rem;
}

.timeline::before {
  content: '';
  position: absolute;
  left: 7px;
  top: 4px;
  bottom: 4px;
  width: 2px;
  background: var(--slate-200);
  border-radius: 1px;
}

.timeline-item {
  position: relative;
  padding-bottom: 1.25rem;
}

.timeline-item:last-child {
  padding-bottom: 0;
}

.timeline-dot {
  position: absolute;
  left: -1.5rem;
  top: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 3px solid white;
  box-shadow: 0 0 0 2px var(--slate-200);
  z-index: 1;
}

.timeline-item--first .timeline-dot {
  width: 18px;
  height: 18px;
  left: calc(-1.5rem - 1px);
  top: 1px;
  box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.2);
}

.timeline-content {
  padding-left: 0.75rem;
}

.timeline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.timeline-status {
  font-size: 0.88rem;
  font-weight: 700;
}

.timeline-time {
  font-size: 0.75rem;
  color: var(--slate-400);
  font-weight: 500;
  white-space: nowrap;
}

.timeline-detail {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  color: var(--slate-500);
  margin-top: 0.2rem;
}

.timeline-detail svg {
  color: var(--slate-400);
  flex-shrink: 0;
}

.timeline-user {
  color: var(--slate-400);
  font-style: italic;
}

/* ── Actions bar ── */
.actions-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
  padding: 1.25rem;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 1.15rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.15s ease;
  text-decoration: none;
  white-space: nowrap;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 0.35rem 0.75rem;
  font-size: 0.82rem;
}

.btn-primary {
  background: linear-gradient(135deg, #8B5CF6, #7c3aed);
  color: white;
  box-shadow: 0 2px 8px rgba(124, 58, 237, 0.3);
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #7c3aed, #6D28D9);
  box-shadow: 0 4px 12px rgba(124, 58, 237, 0.4);
  transform: translateY(-1px);
}

.btn-pay {
  background: linear-gradient(135deg, #34D399, #10B981);
  color: white;
  box-shadow: 0 2px 8px rgba(16, 185, 129, 0.3);
}

.btn-pay:hover:not(:disabled) {
  background: linear-gradient(135deg, #10B981, #059669);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
  transform: translateY(-1px);
}

.btn-danger {
  background: linear-gradient(135deg, #F87171, #EF4444);
  color: white;
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.3);
}

.btn-danger:hover:not(:disabled) {
  background: linear-gradient(135deg, #EF4444, #DC2626);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
  transform: translateY(-1px);
}

.btn-outline {
  background: white;
  color: #7c3aed;
  border: 2px solid #7c3aed;
}

.btn-outline:hover {
  background: #f5f3ff;
  transform: translateY(-1px);
}

.btn-ghost {
  background: transparent;
  color: var(--slate-500);
  border: 2px solid var(--slate-300);
}

.btn-ghost:hover {
  background: var(--slate-100);
  color: var(--slate-900);
}

/* ── Responsive ── */
@media (max-width: 1024px) {
  .info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .info-grid {
    grid-template-columns: 1fr;
  }

  .detail-header {
    flex-direction: column;
  }

  .header-badges {
    margin-left: calc(38px + 0.75rem);
  }

  .actions-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .actions-bar .btn {
    justify-content: center;
  }

  .timeline-header {
    flex-direction: column;
    align-items: flex-start;
  }
}

/* ── Print Panel ── */
.print-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 1rem;
}
.print-panel {
  background: white; border-radius: var(--radius-xl); width: 100%; max-width: 600px;
  max-height: 90vh; overflow-y: auto; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.25);
  border: 1px solid var(--slate-200); animation: printIn 0.25s ease;
}
@keyframes printIn {
  from { opacity: 0; transform: scale(0.95) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}
.print-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding: 1.25rem 1.5rem; border-bottom: 1px solid var(--slate-200);
}
.print-title { font-size: 1.1rem; font-weight: 800; color: var(--slate-900); }
.print-code { font-size: 0.85rem; color: var(--slate-500); margin-top: 0.15rem; }
.print-close { background: none; border: none; font-size: 1.5rem; color: var(--slate-400); cursor: pointer; line-height: 1; }
.print-close:hover { color: var(--slate-900); }
.print-formats {
  display: flex; gap: 0.5rem; padding: 1rem 1.5rem;
  background: var(--slate-50); border-bottom: 1px solid var(--slate-200);
}
.print-fmt-btn {
  flex: 1; padding: 0.55rem 0.75rem; border: 2px solid var(--slate-300); border-radius: var(--radius-md);
  background: white; color: var(--slate-600); font-size: 0.82rem; font-weight: 600;
  cursor: pointer; transition: all 0.15s ease; font-family: inherit; text-align: center;
}
.print-fmt-btn:hover { border-color: var(--brand-300); color: var(--brand-600); }
.print-fmt-btn.active {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white; border-color: var(--brand-500); box-shadow: 0 2px 8px rgba(59,130,246,0.3);
}
.print-preview { width: 100%; height: 400px; border: none; display: block; }
.print-actions {
  display: flex; gap: 0.5rem; padding: 1rem 1.5rem;
  border-top: 1px solid var(--slate-200); background: var(--slate-50);
}
.print-action-btn {
  flex: 1; display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem;
  padding: 0.65rem 1rem; border: none; border-radius: var(--radius-md);
  font-size: 0.88rem; font-weight: 600; cursor: pointer; transition: all 0.15s ease; font-family: inherit;
}
.print-btn-print {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white; box-shadow: 0 2px 8px rgba(59,130,246,0.3);
}
.print-btn-print:hover { background: linear-gradient(135deg, var(--brand-500), var(--brand-600)); }
.print-btn-wa { background: #25D366; color: white; box-shadow: 0 2px 8px rgba(37,211,102,0.3); }
.print-btn-wa:hover { background: #1fb855; }
.print-btn-close { background: var(--slate-100); color: var(--slate-700); border: 2px solid var(--slate-300); }
.print-btn-close:hover { background: var(--slate-200); color: var(--slate-900); }

@media (max-width: 640px) {
  .print-preview { height: 300px; }
  .print-actions, .print-formats { flex-direction: column; }
}
</style>
