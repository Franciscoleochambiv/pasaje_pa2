<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getTripSeats, lookupDNI, lookupRUC, processPayment,
  getBillingPDFUrl, submitYapeDirectPayment, getYapeConfig,
  type SeatInfo, type YapeDirectResponse,
} from '../api/client'
import { useCulqi } from '../composables/useCulqi'
import { usePaymentStatusWebSocket, type PaymentStatusUpdate } from '../composables/usePaymentStatusWebSocket'

const route = useRoute()
const router = useRouter()
const { initCulqi, openCheckout } = useCulqi()

const tripId = computed(() => parseInt(route.params.tripId as string, 10))
const seatIds = computed(() => {
  const s = route.query.seats as string
  return s ? s.split(',').map(Number) : []
})

const seats = ref<SeatInfo[]>([])
const selectedSeats = computed(() => seats.value.filter(s => seatIds.value.includes(s.id)))

const loading = ref(false)
const error = ref('')

// Customer authentication
const customerUser = ref<{ name: string; email: string; role: string } | null>(null)
const isCustomerAuthenticated = computed(() => {
  return customerUser.value !== null && customerUser.value.role === 'customer'
})

function loadCustomerAuth() {
  const token = localStorage.getItem('customer_token')
  const userStr = localStorage.getItem('customer_user')
  if (token && userStr) {
    try {
      customerUser.value = JSON.parse(userStr)
    } catch {
      customerUser.value = null
    }
  }
}

function getLoginRedirectUrl(): string {
  const seatsQuery = seatIds.value.join(',')
  const redirectPath = `/viajes/reservar/${tripId.value}?seats=${seatsQuery}`
  return `/login/cliente?redirect=${encodeURIComponent(redirectPath)}`
}

// Passenger form
const passenger = ref({
  name: '',
  doc_type: 'DNI',
  doc_number: '',
  email: '',
  phone: '',
  address: '',
})
const documentType = ref<'boleta' | 'factura'>('boleta')
const lookingUpDoc = ref(false)
const docLookupError = ref('')

// Price (loaded from trip's route via seats endpoint)
const pricePerSeat = ref(0)
const routeName = ref('')
const totalAmount = computed(() => pricePerSeat.value * seatIds.value.length)
const totalAmountCents = computed(() => Math.round(totalAmount.value * 100))

// Payment method selection
const paymentMethod = ref<'culqi' | 'yape_direct'>('culqi')

// Payment result (Culqi)
const chargeId = ref('')
const reservationCode = ref('')
const ticketCodes = ref<string[]>([])
const billingSaleId = ref<number | null>(null)
const billingError = ref('')
const confirmed = ref(false)

// Yape direct state
const yapeConfig = ref<{ yape_number: string; whatsapp_phone: string } | null>(null)
const yapeVoucherFile = ref<File | null>(null)
const yapeVoucherPreview = ref('')
const yapeDragging = ref(false)
const yapeUploading = ref(false)
const yapeUploadError = ref('')

// Yape verification state
const yapeVerifying = ref(false)
const yapeHoldExpiresAt = ref('')
const yapeWhatsappPhone = ref('')
const yapeCountdown = ref('')
const yapeRejectionReason = ref('')
const yapeExpired = ref(false)
const yapeRejected = ref(false)
let countdownInterval: ReturnType<typeof setInterval> | null = null
let wsDisconnect: (() => void) | null = null

const dniFound = ref(false)

// DNI/RUC auto-lookup
watch(() => passenger.value.doc_number, async (val) => {
  const num = val.trim()
  docLookupError.value = ''
  dniFound.value = false

  if (documentType.value === 'boleta' && num.length === 8) {
    lookingUpDoc.value = true
    try {
      const result = await lookupDNI(num)
      if (result.nombre_completo) {
        passenger.value.name = result.nombre_completo
        dniFound.value = true
      } else if (result.nombres) {
        passenger.value.name = `${result.nombres} ${result.apellido_paterno} ${result.apellido_materno}`.trim()
        dniFound.value = true
      } else {
        docLookupError.value = 'DNI no encontrado. Ingresa los datos manualmente.'
      }
    } catch {
      docLookupError.value = 'DNI no encontrado. Ingresa los datos manualmente.'
    } finally {
      lookingUpDoc.value = false
    }
  }

  if (documentType.value === 'factura' && num.length === 11) {
    lookingUpDoc.value = true
    try {
      const result = await lookupRUC(num)
      if (result.nombre_o_razon_social || result.razon_social) {
        passenger.value.name = result.nombre_o_razon_social || result.razon_social
        dniFound.value = true
      }
      if (result.direccion) {
        passenger.value.address = result.direccion
      }
    } catch {
      docLookupError.value = 'RUC no encontrado. Ingresa los datos manualmente.'
    } finally {
      lookingUpDoc.value = false
    }
  }
})

watch(documentType, (val) => {
  if (val === 'factura') {
    passenger.value.doc_type = 'RUC'
  } else {
    passenger.value.doc_type = 'DNI'
  }
  passenger.value.doc_number = ''
  passenger.value.name = ''
  passenger.value.address = ''
  dniFound.value = false
  docLookupError.value = ''
})

function prefillFromCustomer() {
  if (customerUser.value) {
    if (customerUser.value.name && !passenger.value.name) {
      passenger.value.name = customerUser.value.name
    }
    if (customerUser.value.email && !passenger.value.email) {
      passenger.value.email = customerUser.value.email
    }
  }
}

const formError = ref('')
const submitting = ref(false)

// Culqi init state
const culqiReady = ref(false)

async function loadSeats() {
  loading.value = true
  error.value = ''
  try {
    const resp = await getTripSeats(tripId.value)
    seats.value = resp.seats ?? []
    // Load price and route name from the trip response
    if (resp.price_per_seat) pricePerSeat.value = resp.price_per_seat
    if (resp.route_name) routeName.value = resp.route_name
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

// ---- Form validation ----
function validateForm(): boolean {
  formError.value = ''
  if (!passenger.value.name.trim()) {
    formError.value = 'El nombre completo es obligatorio.'
    return false
  }
  if (!passenger.value.doc_number.trim()) {
    formError.value = 'El numero de documento es obligatorio.'
    return false
  }
  if (!passenger.value.email.trim()) {
    formError.value = 'El email es obligatorio.'
    return false
  }
  if (!passenger.value.phone.trim()) {
    formError.value = 'El telefono es obligatorio.'
    return false
  }
  if (seatIds.value.length === 0) {
    formError.value = 'No hay asientos seleccionados.'
    return false
  }
  return true
}

// ---- Culqi payment flow ----
async function handlePay() {
  if (!validateForm()) return

  submitting.value = true
  try {
    if (!culqiReady.value) {
      await initCulqi()
      culqiReady.value = true
    }

    const seatLabels = selectedSeats.value.map(s => s.label)
    const description = `Pasaje - ${seatLabels.length} asiento${seatLabels.length > 1 ? 's' : ''}: ${seatLabels.join(', ')}`

    const tokenId = await openCheckout({
      title: 'Pasaje Bus',
      currency: 'PEN',
      amount: totalAmountCents.value,
      description,
    })

    const result = await processPayment({
      token_id: tokenId,
      amount: totalAmountCents.value,
      email: passenger.value.email.trim(),
      description,
      trip_instance_id: tripId.value,
      seats: seatIds.value,
      passenger_name: passenger.value.name.trim(),
      passenger_doc_type: passenger.value.doc_type,
      passenger_doc_number: passenger.value.doc_number.trim(),
      passenger_address: passenger.value.address.trim() || undefined,
      document_type: documentType.value,
      route_name: '',
      seat_labels: seatLabels.length > 0 ? seatLabels : seatIds.value.map(String),
      price_per_seat: pricePerSeat.value,
      passenger_phone: passenger.value.phone.trim(),
    })

    chargeId.value = result.charge_id
    reservationCode.value = result.reservation_code
    ticketCodes.value = result.ticket_codes || []
    billingSaleId.value = result.billing_sale_id || null
    billingError.value = result.billing_error || ''
    confirmed.value = true
  } catch (e) {
    const msg = (e as Error).message
    if (msg === 'Pago cancelado' || msg.includes('cancelado')) {
      formError.value = ''
    } else {
      formError.value = msg
    }
  } finally {
    submitting.value = false
  }
}

// ---- Yape Direct flow ----
async function loadYapeConfig() {
  try {
    yapeConfig.value = await getYapeConfig()
  } catch { /* ignore */ }
}

function handleYapeFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files[0]) {
    processVoucherFile(input.files[0])
  }
}

function handleYapeDrop(e: DragEvent) {
  yapeDragging.value = false
  if (e.dataTransfer?.files && e.dataTransfer.files[0]) {
    processVoucherFile(e.dataTransfer.files[0])
  }
}

function processVoucherFile(file: File) {
  yapeUploadError.value = ''

  if (!file.type.startsWith('image/')) {
    yapeUploadError.value = 'Solo se permiten imagenes (JPG, PNG, etc.)'
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    yapeUploadError.value = 'La imagen no debe superar los 10 MB.'
    return
  }

  yapeVoucherFile.value = file
  const reader = new FileReader()
  reader.onload = (ev) => {
    yapeVoucherPreview.value = ev.target?.result as string
  }
  reader.readAsDataURL(file)
}

function removeVoucher() {
  yapeVoucherFile.value = null
  yapeVoucherPreview.value = ''
}

const voucherInput = ref<HTMLInputElement | null>(null)

async function handleYapeSubmit() {
  if (!validateForm()) return
  if (!yapeVoucherFile.value) {
    yapeUploadError.value = 'Debes subir el comprobante de pago.'
    return
  }

  yapeUploading.value = true
  yapeUploadError.value = ''

  try {
    const seatLabels = selectedSeats.value.map(s => s.label)
    const fd = new FormData()
    fd.append('trip_instance_id', String(tripId.value))
    fd.append('amount', String(totalAmountCents.value))
    fd.append('seats', seatIds.value.join(','))
    fd.append('passenger_name', passenger.value.name.trim())
    fd.append('passenger_doc_type', passenger.value.doc_type)
    fd.append('passenger_doc_number', passenger.value.doc_number.trim())
    fd.append('passenger_email', passenger.value.email.trim())
    fd.append('passenger_phone', passenger.value.phone.trim())
    fd.append('passenger_address', passenger.value.address.trim())
    fd.append('document_type', documentType.value)
    fd.append('route_name', '')
    fd.append('seat_labels', (seatLabels.length > 0 ? seatLabels : seatIds.value.map(String)).join(','))
    fd.append('price_per_seat', String(pricePerSeat.value))
    fd.append('voucher', yapeVoucherFile.value)

    const result: YapeDirectResponse = await submitYapeDirectPayment(fd)

    reservationCode.value = result.reservation_code
    yapeHoldExpiresAt.value = result.hold_expires_at
    yapeWhatsappPhone.value = result.whatsapp_phone || yapeConfig.value?.whatsapp_phone || ''
    yapeVerifying.value = true

    startCountdown()
    startPaymentWs()
  } catch (e) {
    yapeUploadError.value = (e as Error).message
  } finally {
    yapeUploading.value = false
  }
}

function startCountdown() {
  updateCountdown()
  countdownInterval = setInterval(updateCountdown, 1000)
}

function updateCountdown() {
  if (!yapeHoldExpiresAt.value) return
  const now = Date.now()
  const expires = new Date(yapeHoldExpiresAt.value).getTime()
  const diff = expires - now

  if (diff <= 0) {
    yapeCountdown.value = '00:00'
    yapeExpired.value = true
    if (countdownInterval) {
      clearInterval(countdownInterval)
      countdownInterval = null
    }
    return
  }

  const mins = Math.floor(diff / 60000)
  const secs = Math.floor((diff % 60000) / 1000)
  yapeCountdown.value = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
}

function startPaymentWs() {
  const { disconnect } = usePaymentStatusWebSocket(reservationCode.value, (data: PaymentStatusUpdate) => {
    if (data.status === 'approved') {
      confirmed.value = true
      yapeVerifying.value = false
      if (data.billing_sale_id) billingSaleId.value = data.billing_sale_id
      if (data.ticket_codes) ticketCodes.value = data.ticket_codes
      if (countdownInterval) {
        clearInterval(countdownInterval)
        countdownInterval = null
      }
    } else if (data.status === 'rejected') {
      yapeRejected.value = true
      yapeRejectionReason.value = data.rejection_reason || 'El comprobante fue rechazado.'
      if (countdownInterval) {
        clearInterval(countdownInterval)
        countdownInterval = null
      }
    }
  })
  wsDisconnect = disconnect
}

function retryYapeUpload() {
  yapeRejected.value = false
  yapeRejectionReason.value = ''
  yapeVerifying.value = false
  yapeVoucherFile.value = null
  yapeVoucherPreview.value = ''
  reservationCode.value = ''
}

const whatsappUrl = computed(() => {
  const phone = yapeWhatsappPhone.value || yapeConfig.value?.whatsapp_phone || ''
  const text = `Hola, he subido mi voucher de Yape para la reserva ${reservationCode.value} por S/ ${totalAmount.value.toFixed(2)}`
  return `https://wa.me/${phone}?text=${encodeURIComponent(text)}`
})

// Watch payment method to load Yape config
watch(paymentMethod, (val) => {
  if (val === 'yape_direct' && !yapeConfig.value) {
    loadYapeConfig()
  }
})

onMounted(() => {
  loadCustomerAuth()
  prefillFromCustomer()
  loadSeats()
})

onUnmounted(() => {
  if (countdownInterval) clearInterval(countdownInterval)
  if (wsDisconnect) wsDisconnect()
})
</script>

<template>
  <div class="reserva-page">
    <!-- Hero -->
    <div class="reserva-hero" v-if="!confirmed && !yapeVerifying">
      <div class="reserva-hero-bg"></div>
      <div class="container reserva-hero-inner">
        <div class="reserva-hero-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
          </svg>
        </div>
        <h1 class="reserva-hero-title">Comprar Pasaje</h1>
        <p class="reserva-hero-sub">Completa tus datos y paga de forma segura</p>
        <div v-if="selectedSeats.length || seatIds.length" class="hero-seats-preview">
          <span class="hsp-label">Asientos:</span>
          <span v-for="s in selectedSeats" :key="s.id" class="hsp-chip">{{ s.label }}</span>
          <template v-if="selectedSeats.length === 0">
            <span v-for="id in seatIds" :key="id" class="hsp-chip">{{ id }}</span>
          </template>
        </div>
      </div>
    </div>

    <div class="container reserva-content">
      <!-- AUTH REQUIRED -->
      <div v-if="!isCustomerAuthenticated && !confirmed && !yapeVerifying" class="auth-required-card">
        <div class="auth-required-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
        </div>
        <h2 class="auth-required-title">Para continuar con la compra, inicia sesion</h2>
        <p class="auth-required-subtitle">Necesitas una cuenta para comprar y gestionar tus pasajes</p>
        <router-link :to="getLoginRedirectUrl()" class="btn btn-primary btn-lg">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>
            <polyline points="10 17 15 12 10 7"/>
            <line x1="15" x2="3" y1="12" y2="12"/>
          </svg>
          Iniciar sesion con Google
        </router-link>
      </div>

      <!-- AUTHENTICATED USER BANNER -->
      <div v-if="isCustomerAuthenticated && !confirmed && !yapeVerifying" class="auth-banner">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
          <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
        <span>Comprando como: <strong>{{ customerUser?.name }}</strong> ({{ customerUser?.email }})</span>
      </div>

      <!-- SUCCESS STATE -->
      <div v-if="confirmed" class="success-card">
        <div class="success-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <h1 class="success-title">Compra Exitosa</h1>
        <p class="success-subtitle" v-if="chargeId">Tu pago fue procesado exitosamente con Culqi</p>
        <p class="success-subtitle" v-else>Tu pago con Yape fue aprobado</p>

        <div class="detail-card">
          <div class="detail-row">
            <span class="detail-label">Codigo de Reserva</span>
            <span class="detail-value code-value">{{ reservationCode }}</span>
          </div>
          <div v-if="chargeId" class="detail-row">
            <span class="detail-label">Cargo Culqi</span>
            <span class="detail-value code-value" style="font-size:0.85rem;">{{ chargeId }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Pasajero</span>
            <span class="detail-value">{{ passenger.name }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Asientos</span>
            <span class="detail-value">{{ selectedSeats.map(s => s.label).join(', ') || seatIds.join(', ') }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Total Pagado</span>
            <span class="detail-value" style="color:var(--success-600);font-weight:800;">S/ {{ totalAmount.toFixed(2) }}</span>
          </div>
        </div>

        <div v-if="ticketCodes.length" class="tickets-section">
          <h3 class="tickets-title">Codigos de Boleto</h3>
          <div class="tickets-grid">
            <div v-for="code in ticketCodes" :key="code" class="ticket-chip">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>
                <path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
              </svg>
              {{ code }}
            </div>
          </div>
        </div>

        <div v-if="billingSaleId" class="billing-pdf-section">
          <a :href="getBillingPDFUrl(billingSaleId, 'ticket')" target="_blank" class="btn btn-success btn-lg">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/>
            </svg>
            Descargar Boleta
          </a>
        </div>
        <div v-else-if="billingError" class="alert alert-error" style="margin-top:1rem;">
          No se pudo generar la boleta: {{ billingError }}
        </div>

        <div class="success-actions">
          <router-link to="/viajes" class="btn btn-primary btn-lg">Buscar otro viaje</router-link>
          <router-link :to="`/consulta?code=${reservationCode}`" class="btn btn-ghost btn-lg">Consultar reserva</router-link>
        </div>
      </div>

      <!-- YAPE VERIFICATION WAITING STATE -->
      <div v-else-if="yapeVerifying && !yapeExpired && !yapeRejected" class="verification-card">
        <div class="verification-pulse-ring">
          <div class="verification-icon">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
            </svg>
          </div>
        </div>
        <h1 class="verification-title">Pago en verificacion</h1>
        <p class="verification-subtitle">Estamos verificando tu comprobante de Yape. Te notificaremos cuando sea aprobado.</p>

        <div class="verification-code-card">
          <span class="verification-code-label">Tu codigo de reserva</span>
          <span class="verification-code-value">{{ reservationCode }}</span>
        </div>

        <div class="verification-countdown">
          <div class="countdown-ring">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
            </svg>
            <span>Tiempo restante: <strong>{{ yapeCountdown }}</strong></span>
          </div>
        </div>

        <div class="verification-steps">
          <div class="v-step">
            <div class="v-step-num done">1</div>
            <div class="v-step-text">Comprobante enviado</div>
          </div>
          <div class="v-step-line"></div>
          <div class="v-step">
            <div class="v-step-num active">2</div>
            <div class="v-step-text">En revision</div>
          </div>
          <div class="v-step-line"></div>
          <div class="v-step">
            <div class="v-step-num">3</div>
            <div class="v-step-text">Aprobado</div>
          </div>
        </div>

        <a :href="whatsappUrl" target="_blank" class="btn btn-whatsapp btn-lg">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
          </svg>
          Contactar por WhatsApp
        </a>
        <p class="verification-hint">Si necesitas ayuda, contactanos por WhatsApp adjuntando tu codigo de reserva.</p>
      </div>

      <!-- YAPE REJECTED STATE -->
      <div v-else-if="yapeVerifying && yapeRejected" class="rejected-card">
        <div class="rejected-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
        </div>
        <h1 class="rejected-title">Comprobante Rechazado</h1>
        <p class="rejected-reason">{{ yapeRejectionReason }}</p>
        <p class="rejected-subtitle">Puedes intentar nuevamente subiendo un comprobante valido.</p>
        <div class="rejected-actions">
          <button class="btn btn-primary btn-lg" @click="retryYapeUpload">Intentar nuevamente</button>
          <router-link to="/viajes" class="btn btn-ghost btn-lg">Buscar otro viaje</router-link>
        </div>
      </div>

      <!-- YAPE EXPIRED STATE -->
      <div v-else-if="yapeVerifying && yapeExpired" class="expired-card">
        <div class="expired-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
          </svg>
        </div>
        <h1 class="expired-title">Reserva Expirada</h1>
        <p class="expired-subtitle">El tiempo de verificacion ha expirado. Los asientos han sido liberados.</p>
        <p class="expired-hint">Si realizaste el pago, contacta a soporte con tu codigo de reserva: <strong>{{ reservationCode }}</strong></p>
        <div class="expired-actions">
          <router-link to="/viajes" class="btn btn-primary btn-lg">Buscar otro viaje</router-link>
          <a :href="whatsappUrl" target="_blank" class="btn btn-whatsapp btn-lg">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
            </svg>
            Contactar soporte
          </a>
        </div>
      </div>

      <!-- PURCHASE FORM -->
      <div v-else-if="isCustomerAuthenticated">
        <div v-if="loading" class="loading-state">
          <div class="spinner"></div>
          <span>Cargando informacion del viaje...</span>
        </div>

        <div v-else-if="error" class="alert alert-error">{{ error }}</div>

        <template v-else>
          <!-- Section: Asientos seleccionados -->
          <div class="section-card">
            <div class="section-header">
              <div class="section-number">1</div>
              <div>
                <h2 class="section-title">Asientos seleccionados</h2>
                <p class="section-subtitle">Confirma tus asientos para este viaje</p>
              </div>
            </div>
            <div class="seats-summary">
              <div class="summary-chips">
                <span v-for="s in selectedSeats" :key="s.id" class="summary-chip">{{ s.label }}</span>
                <span v-if="selectedSeats.length === 0 && seatIds.length > 0" class="summary-fallback">IDs: {{ seatIds.join(', ') }}</span>
              </div>
            </div>
          </div>

          <!-- Section: Comprobante -->
          <div class="section-card">
            <div class="section-header">
              <div class="section-number">2</div>
              <div>
                <h2 class="section-title">Tipo de comprobante</h2>
                <p class="section-subtitle">Selecciona el documento tributario que necesitas</p>
              </div>
            </div>
            <div class="comprobante-options">
              <label class="comprobante-option" :class="{ active: documentType === 'boleta' }">
                <input type="radio" v-model="documentType" value="boleta" />
                <div class="co-icon">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/></svg>
                </div>
                <div>
                  <strong>Boleta</strong>
                  <span>Para personas con DNI</span>
                </div>
              </label>
              <label class="comprobante-option" :class="{ active: documentType === 'factura' }">
                <input type="radio" v-model="documentType" value="factura" />
                <div class="co-icon">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" x2="8" y1="13" y2="13"/><line x1="16" x2="8" y1="17" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
                </div>
                <div>
                  <strong>Factura</strong>
                  <span>Para empresas con RUC</span>
                </div>
              </label>
            </div>
          </div>

          <!-- Section: Datos del pasajero -->
          <div class="section-card">
            <div class="section-header">
              <div class="section-number">3</div>
              <div>
                <h2 class="section-title">Datos del pasajero</h2>
                <p class="section-subtitle">Completa la informacion del viajero</p>
              </div>
            </div>

            <div v-if="formError" class="alert alert-error">{{ formError }}</div>

            <div class="form-grid">
              <div class="form-group">
                <label class="form-label">Tipo de documento</label>
                <div class="doc-type-fixed">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
                  </svg>
                  {{ documentType === 'factura' ? 'RUC' : 'DNI' }}
                </div>
              </div>

              <div class="form-group">
                <label class="form-label" for="p-doc-num">
                  {{ documentType === 'factura' ? 'Numero de RUC' : 'Numero de DNI' }}
                  <span v-if="lookingUpDoc" class="doc-lookup-badge"><span class="lookup-spinner"></span> Consultando...</span>
                  <span v-if="dniFound" class="doc-found-badge">Encontrado</span>
                </label>
                <input id="p-doc-num" v-model="passenger.doc_number" type="text" class="form-input"
                  :placeholder="documentType === 'factura' ? 'Ej: 20123456789' : 'Ej: 70123456'"
                  :maxlength="documentType === 'factura' ? 11 : 8" />
                <span v-if="docLookupError" class="doc-lookup-warn">{{ docLookupError }}</span>
              </div>

              <div class="form-group form-full">
                <label class="form-label" for="p-name">{{ documentType === 'factura' ? 'Razon social' : 'Nombre completo' }}</label>
                <input id="p-name" v-model="passenger.name" type="text" class="form-input"
                  :placeholder="documentType === 'factura' ? 'Ej: TRANSPORTES ANCALLA SAC' : 'Se autocompleta con el DNI'"
                  :readonly="dniFound" />
                <span v-if="dniFound" class="doc-auto-label">Autocompletado desde {{ documentType === 'factura' ? 'SUNAT' : 'RENIEC' }}</span>
              </div>

              <div v-if="documentType === 'factura'" class="form-group form-full">
                <label class="form-label" for="p-address">Direccion fiscal</label>
                <input id="p-address" v-model="passenger.address" type="text" class="form-input" placeholder="Ej: Av. Principal 123, Arequipa" />
              </div>

              <div class="form-group">
                <label class="form-label" for="p-email">Email</label>
                <input id="p-email" v-model="passenger.email" type="email" class="form-input"
                  :readonly="isCustomerAuthenticated && !!passenger.email" placeholder="Ej: juan@correo.com" />
                <span v-if="isCustomerAuthenticated && !!passenger.email" class="doc-auto-label">Desde tu cuenta Google</span>
              </div>

              <div class="form-group">
                <label class="form-label" for="p-phone">Telefono</label>
                <input id="p-phone" v-model="passenger.phone" type="tel" class="form-input" placeholder="Ej: 987654321" />
              </div>
            </div>
          </div>

          <!-- Price Summary -->
          <div class="price-summary-card">
            <div class="price-summary-header">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
              </svg>
              <span>Resumen de precio</span>
            </div>
            <div class="price-summary-body">
              <div class="price-row">
                <span>{{ seatIds.length }} asiento{{ seatIds.length > 1 ? 's' : '' }} x S/ {{ pricePerSeat.toFixed(2) }}</span>
                <span>S/ {{ totalAmount.toFixed(2) }}</span>
              </div>
              <div class="price-total-row">
                <span>Total a pagar</span>
                <span class="price-total-value">S/ {{ totalAmount.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <!-- Section: Metodo de pago -->
          <div class="section-card">
            <div class="section-header">
              <div class="section-number">4</div>
              <div>
                <h2 class="section-title">Metodo de pago</h2>
                <p class="section-subtitle">Elige como deseas pagar</p>
              </div>
            </div>

            <div class="payment-methods">
              <label class="payment-method-card" :class="{ active: paymentMethod === 'culqi' }">
                <input type="radio" v-model="paymentMethod" value="culqi" />
                <div class="pm-radio-dot"><div class="pm-radio-inner"></div></div>
                <div class="pm-icon pm-icon-card">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/>
                  </svg>
                </div>
                <div class="pm-content">
                  <strong>Tarjeta / Yape</strong>
                  <span>Visa, Mastercard, Yape via Culqi</span>
                </div>
                <div class="pm-brands">
                  <span class="pm-brand-badge">Visa</span>
                  <span class="pm-brand-badge">MC</span>
                  <span class="pm-brand-badge pm-brand-yape">Yape</span>
                </div>
              </label>

              <label class="payment-method-card" :class="{ active: paymentMethod === 'yape_direct' }">
                <input type="radio" v-model="paymentMethod" value="yape_direct" />
                <div class="pm-radio-dot"><div class="pm-radio-inner"></div></div>
                <div class="pm-icon pm-icon-yape">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"/><path d="M9 12l2 2 4-4"/>
                  </svg>
                </div>
                <div class="pm-content">
                  <strong>Yape Directo</strong>
                  <span>Transferencia directa + sube tu voucher</span>
                </div>
              </label>
            </div>

            <!-- Culqi flow action -->
            <div v-if="paymentMethod === 'culqi'" class="payment-action-area">
              <button class="btn btn-pay btn-lg btn-full" :disabled="submitting" @click="handlePay">
                <div v-if="submitting" class="spinner-sm"></div>
                <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/>
                </svg>
                Pagar S/ {{ totalAmount.toFixed(2) }}
              </button>
            </div>

            <!-- Yape Direct flow -->
            <div v-if="paymentMethod === 'yape_direct'" class="yape-direct-flow">
              <div class="yape-number-card">
                <div class="yape-number-header">
                  <div class="yape-number-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
                    </svg>
                  </div>
                  <span class="yape-number-label">Yapea a este numero</span>
                </div>
                <div class="yape-number-value">{{ yapeConfig?.yape_number || 'Cargando...' }}</div>
                <div class="yape-number-amount">Monto exacto: <strong>S/ {{ totalAmount.toFixed(2) }}</strong></div>
              </div>

              <div class="yape-steps">
                <div class="yape-step">
                  <div class="yape-step-num">1</div>
                  <div class="yape-step-text">Abre tu app de <strong>Yape</strong> y realiza el pago al numero indicado</div>
                </div>
                <div class="yape-step">
                  <div class="yape-step-num">2</div>
                  <div class="yape-step-text">Toma una <strong>captura de pantalla</strong> del comprobante de pago</div>
                </div>
                <div class="yape-step">
                  <div class="yape-step-num">3</div>
                  <div class="yape-step-text"><strong>Sube la imagen</strong> del comprobante aqui abajo</div>
                </div>
              </div>

              <div class="voucher-upload-area">
                <div v-if="!yapeVoucherPreview" class="voucher-dropzone" :class="{ dragging: yapeDragging }"
                  @dragover.prevent="yapeDragging = true" @dragleave.prevent="yapeDragging = false"
                  @drop.prevent="handleYapeDrop" @click="voucherInput?.click()">
                  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="17 8 12 3 7 8"/><line x1="12" x2="12" y1="3" y2="15"/>
                  </svg>
                  <p class="dropzone-text">Arrastra tu comprobante aqui o haz clic para seleccionar</p>
                  <p class="dropzone-hint">JPG, PNG - maximo 10 MB</p>
                  <span class="btn-outline-sm dropzone-btn">Seleccionar imagen</span>
                </div>

                <input ref="voucherInput" type="file" accept="image/*" capture="environment" class="voucher-file-input" @change="handleYapeFileSelect" />

                <div v-if="yapeVoucherPreview" class="voucher-preview">
                  <img :src="yapeVoucherPreview" alt="Comprobante de pago" class="voucher-preview-img" />
                  <div class="voucher-preview-info">
                    <span class="voucher-preview-name">{{ yapeVoucherFile?.name }}</span>
                    <span class="voucher-preview-size">{{ yapeVoucherFile ? (yapeVoucherFile.size / 1024).toFixed(0) + ' KB' : '' }}</span>
                  </div>
                  <button class="voucher-remove-btn" @click="removeVoucher" title="Quitar imagen">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                  </button>
                </div>

                <div v-if="yapeUploadError" class="alert alert-error" style="margin-top:0.75rem;">{{ yapeUploadError }}</div>
              </div>

              <button class="btn btn-yape btn-lg btn-full" :disabled="yapeUploading || !yapeVoucherFile" @click="handleYapeSubmit">
                <div v-if="yapeUploading" class="spinner-sm"></div>
                <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M22 2L11 13"/><path d="M22 2l-7 20-4-9-9-4z"/>
                </svg>
                Enviar comprobante
              </button>
            </div>
          </div>

          <!-- Back button -->
          <div class="bottom-actions">
            <button class="btn btn-ghost" @click="router.back()">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/>
              </svg>
              Volver
            </button>
          </div>

          <!-- Security badges -->
          <div class="security-badges">
            <div class="security-badge">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/></svg>
              Pago seguro
            </div>
            <div class="security-badge">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              Datos encriptados
            </div>
            <div class="security-badge">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
              Visa, Mastercard, Yape
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.reserva-page { min-height: 80vh; }

/* Hero */
.reserva-hero { position: relative; padding: 2.5rem 0 2rem; overflow: hidden; }
.reserva-hero-bg { position: absolute; inset: 0; background: radial-gradient(ellipse at 20% 0%, rgba(59,130,246,0.10) 0%, transparent 55%), radial-gradient(ellipse at 80% 100%, rgba(16,185,129,0.08) 0%, transparent 55%), linear-gradient(180deg, var(--brand-50) 0%, #ffffff 70%); border-bottom: 1px solid var(--color-border); }
.reserva-hero-bg::before { content: ''; position: absolute; inset: 0; background-image: radial-gradient(ellipse at 50% 50%, rgba(59,130,246,0.04) 0%, transparent 60%); }
.reserva-hero-inner { position: relative; text-align: center; max-width: 600px; }
.reserva-hero-icon { display: inline-flex; align-items: center; justify-content: center; width: 56px; height: 56px; border-radius: 50%; background: var(--brand-50); border: 1px solid var(--brand-100); color: var(--brand-600); margin-bottom: 0.75rem; box-shadow: 0 6px 18px rgba(59,130,246,0.15); }
.reserva-hero-title { font-size: 1.75rem; font-weight: 800; color: var(--color-heading); margin-bottom: 0.3rem; letter-spacing: -0.02em; }
.reserva-hero-sub { font-size: 0.95rem; color: var(--color-text-muted); margin-bottom: 1rem; }
.hero-seats-preview { display: flex; align-items: center; justify-content: center; gap: 0.4rem; flex-wrap: wrap; }
.hsp-label { font-size: 0.78rem; font-weight: 600; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.04em; }
.hsp-chip { padding: 0.2rem 0.6rem; background: #ffffff; border: 1px solid var(--brand-200); border-radius: var(--radius-full); color: var(--brand-700); font-size: 0.8rem; font-weight: 700; box-shadow: var(--shadow-xs); }

/* Content */
.reserva-content { max-width: 720px; padding-top: 1.75rem; padding-bottom: 4rem; }

/* Section Cards */
.section-card { background: var(--color-surface); border: 2px solid var(--color-border); border-radius: var(--radius-lg); padding: 1.5rem 1.75rem; box-shadow: var(--shadow-sm); margin-bottom: 1.25rem; }
.section-header { display: flex; align-items: flex-start; gap: 1rem; margin-bottom: 1.25rem; }
.section-number { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; min-width: 32px; border-radius: 50%; background: linear-gradient(135deg, var(--brand-600), var(--brand-700)); color: white; font-size: 0.82rem; font-weight: 800; line-height: 1; }
.section-title { font-size: 1.05rem; font-weight: 800; color: var(--color-heading); letter-spacing: -0.02em; line-height: 1.2; }
.section-subtitle { font-size: 0.82rem; color: var(--color-text-muted); margin-top: 0.15rem; }

/* Seats Summary */
.seats-summary { padding: 0.75rem 1rem; background: var(--brand-50); border: 1px solid var(--brand-200); border-radius: var(--radius-md); }
.summary-chips { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.summary-chip { display: inline-flex; align-items: center; justify-content: center; padding: 0.3rem 0.65rem; background: white; border: 1px solid var(--brand-200); border-radius: var(--radius-full); font-size: 0.85rem; font-weight: 700; color: var(--brand-700); }
.summary-fallback { font-size: 0.85rem; color: var(--brand-600); }

/* Form */
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
.form-full { grid-column: 1 / -1; }
.form-group { display: flex; flex-direction: column; gap: 0.35rem; }
.form-label { font-size: 0.82rem; font-weight: 600; color: var(--color-heading); }
.form-input { padding: 0.7rem 0.95rem; border: 2px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-background); color: var(--color-heading); font-size: 0.92rem; font-family: inherit; font-weight: 500; transition: border-color 0.2s ease, box-shadow 0.2s ease; outline: none; }
.form-input:focus { border-color: var(--brand-500); box-shadow: 0 0 0 3px rgba(27, 85, 245, 0.1); background: white; }
.form-input::placeholder { color: var(--color-text-muted); opacity: 0.6; }

/* Price Summary */
.price-summary-card { background: linear-gradient(135deg, var(--brand-50), var(--accent-50)); border: 2px solid var(--brand-200); border-radius: var(--radius-lg); overflow: hidden; margin-bottom: 1.25rem; box-shadow: var(--shadow-md); }
.price-summary-header { display: flex; align-items: center; gap: 0.6rem; padding: 0.85rem 1.25rem; background: rgba(27, 85, 245, 0.06); border-bottom: 1px solid var(--brand-200); font-size: 0.85rem; font-weight: 700; color: var(--brand-700); text-transform: uppercase; letter-spacing: 0.03em; }
.price-summary-header svg { color: var(--brand-500); }
.price-summary-body { padding: 1rem 1.25rem; }
.price-row { display: flex; justify-content: space-between; align-items: center; font-size: 0.88rem; color: var(--color-text-muted); margin-bottom: 0.65rem; }
.price-total-row { display: flex; justify-content: space-between; align-items: center; padding-top: 0.75rem; border-top: 2px solid var(--brand-200); font-size: 1rem; font-weight: 700; color: var(--color-heading); }
.price-total-value { font-size: 1.35rem; font-weight: 800; color: var(--brand-700); }

/* Payment Methods */
.payment-methods { display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 1.5rem; }
.payment-method-card { display: flex; align-items: center; gap: 0.85rem; padding: 1.15rem 1.25rem; border: 2px solid var(--color-border); border-radius: var(--radius-lg); cursor: pointer; transition: all 0.2s ease; background: var(--color-surface); }
.payment-method-card input[type="radio"] { display: none; }
.payment-method-card:hover { border-color: var(--brand-300); background: var(--color-surface-hover); }
.payment-method-card.active { border-color: var(--brand-500); background: var(--brand-50); box-shadow: 0 0 0 3px rgba(27, 85, 245, 0.1); }
@media (prefers-color-scheme: dark) { .payment-method-card.active { background: rgba(27, 85, 245, 0.08); } }
.pm-radio-dot { width: 20px; height: 20px; min-width: 20px; border: 2px solid var(--color-border); border-radius: 50%; display: flex; align-items: center; justify-content: center; transition: all 0.2s ease; }
.payment-method-card.active .pm-radio-dot { border-color: var(--brand-600); }
.pm-radio-inner { width: 10px; height: 10px; border-radius: 50%; background: transparent; transition: all 0.2s ease; }
.payment-method-card.active .pm-radio-inner { background: var(--brand-600); }
.pm-icon { display: flex; align-items: center; justify-content: center; width: 48px; height: 48px; border-radius: var(--radius-md); flex-shrink: 0; transition: all 0.2s ease; }
.pm-icon-card { background: var(--color-background-mute); color: var(--color-text-muted); }
.payment-method-card.active .pm-icon-card { background: var(--brand-100); color: var(--brand-600); }
.pm-icon-yape { background: #f0e6ff; color: #7c3aed; }
.payment-method-card.active .pm-icon-yape { background: #e9d5ff; color: #6d28d9; }
.pm-content { flex: 1; min-width: 0; }
.pm-content strong { display: block; font-size: 0.95rem; font-weight: 700; color: var(--color-heading); }
.pm-content span { display: block; font-size: 0.78rem; color: var(--color-text-muted); margin-top: 0.1rem; }
.pm-brands { display: flex; gap: 0.3rem; flex-shrink: 0; }
.pm-brand-badge { padding: 0.2rem 0.45rem; border-radius: var(--radius-sm); font-size: 0.65rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; background: var(--color-background-mute); color: var(--color-text-muted); border: 1px solid var(--color-border); }
.pm-brand-yape { background: #f0e6ff; color: #7c3aed; border-color: #e0cfff; }

/* Payment Action Area */
.payment-action-area { margin-top: 0.5rem; }

/* Yape Direct Flow */
.yape-direct-flow { margin-top: 0.5rem; }
.yape-number-card { background: linear-gradient(135deg, #f5f0ff, #ede5ff); border: 2px solid #d8b4fe; border-radius: var(--radius-lg); padding: 1.5rem; text-align: center; margin-bottom: 1.25rem; }
.yape-number-header { display: flex; align-items: center; justify-content: center; gap: 0.5rem; margin-bottom: 0.75rem; }
.yape-number-icon { display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 50%; background: #7c3aed; color: white; }
.yape-number-label { font-size: 0.85rem; font-weight: 700; color: #6d28d9; text-transform: uppercase; letter-spacing: 0.04em; }
.yape-number-value { font-size: 2rem; font-weight: 800; color: #5b21b6; letter-spacing: 0.08em; font-family: 'SF Mono', 'Fira Code', monospace; margin-bottom: 0.5rem; }
.yape-number-amount { font-size: 0.88rem; color: #7c3aed; }
.yape-number-amount strong { font-weight: 800; font-size: 1.05rem; }

/* Yape Steps */
.yape-steps { display: flex; flex-direction: column; gap: 0; margin-bottom: 1.5rem; padding: 0 0.5rem; }
.yape-step { display: flex; align-items: flex-start; gap: 0.85rem; padding: 0.65rem 0; }
.yape-step-num { display: flex; align-items: center; justify-content: center; width: 28px; height: 28px; min-width: 28px; border-radius: 50%; background: var(--brand-100); color: var(--brand-700); font-size: 0.78rem; font-weight: 800; }
.yape-step-text { font-size: 0.88rem; color: var(--color-text); line-height: 1.5; padding-top: 0.15rem; }
.yape-step-text strong { font-weight: 700; color: var(--color-heading); }

/* Voucher Upload */
.voucher-upload-area { margin-bottom: 1.25rem; }
.voucher-dropzone { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.65rem; padding: 2rem 1.5rem; border: 2px dashed var(--color-border); border-radius: var(--radius-lg); cursor: pointer; transition: all 0.2s ease; background: var(--color-background); text-align: center; }
.voucher-dropzone:hover { border-color: var(--brand-400); background: var(--brand-50); }
.voucher-dropzone.dragging { border-color: var(--brand-500); background: var(--brand-50); box-shadow: 0 0 0 4px rgba(27, 85, 245, 0.1); }
.voucher-dropzone svg { color: var(--color-text-muted); }
.dropzone-text { font-size: 0.9rem; font-weight: 600; color: var(--color-heading); }
.dropzone-hint { font-size: 0.78rem; color: var(--color-text-muted); }
.dropzone-btn { display: inline-flex; align-items: center; justify-content: center; padding: 0.5rem 1rem; border: 1px solid var(--brand-300); border-radius: var(--radius-md); background: white; color: var(--brand-600); font-size: 0.82rem; font-weight: 600; cursor: pointer; transition: all 0.2s ease; margin-top: 0.25rem; }
.dropzone-btn:hover { background: var(--brand-50); border-color: var(--brand-500); }
.voucher-file-input { position: absolute; width: 0; height: 0; overflow: hidden; opacity: 0; }
.voucher-preview { display: flex; align-items: center; gap: 1rem; padding: 0.85rem 1rem; border: 2px solid var(--success-100); border-radius: var(--radius-lg); background: var(--success-50); }
.voucher-preview-img { width: 64px; height: 64px; object-fit: cover; border-radius: var(--radius-md); border: 1px solid var(--color-border); flex-shrink: 0; }
.voucher-preview-info { flex: 1; min-width: 0; }
.voucher-preview-name { display: block; font-size: 0.85rem; font-weight: 600; color: var(--color-heading); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.voucher-preview-size { display: block; font-size: 0.75rem; color: var(--color-text-muted); margin-top: 0.1rem; }
.voucher-remove-btn { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 50%; border: none; background: var(--danger-50); color: var(--danger-600); cursor: pointer; transition: all 0.2s ease; flex-shrink: 0; }
.voucher-remove-btn:hover { background: var(--danger-100); color: var(--danger-700); }

/* Verification State */
.verification-card { text-align: center; padding: 3rem 2rem; background: var(--color-surface); border: 2px solid var(--warning-100); border-radius: var(--radius-xl); box-shadow: var(--shadow-lg); }
.verification-pulse-ring { display: inline-flex; align-items: center; justify-content: center; width: 96px; height: 96px; border-radius: 50%; background: var(--warning-50); margin-bottom: 1.25rem; animation: pulse-ring 2s ease-in-out infinite; position: relative; }
.verification-pulse-ring::before { content: ''; position: absolute; inset: -8px; border-radius: 50%; border: 2px solid var(--warning-500); opacity: 0.3; animation: pulse-outer 2s ease-in-out infinite; }
@keyframes pulse-ring { 0%, 100% { box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.2); } 50% { box-shadow: 0 0 0 16px rgba(245, 158, 11, 0); } }
@keyframes pulse-outer { 0%, 100% { transform: scale(1); opacity: 0.3; } 50% { transform: scale(1.15); opacity: 0; } }
.verification-icon { color: var(--warning-600); }
.verification-title { font-size: 1.75rem; font-weight: 800; color: var(--color-heading); letter-spacing: -0.02em; margin-bottom: 0.35rem; }
.verification-subtitle { font-size: 0.92rem; color: var(--color-text-muted); margin-bottom: 1.5rem; max-width: 420px; margin-left: auto; margin-right: auto; line-height: 1.5; }
.verification-code-card { display: inline-flex; flex-direction: column; align-items: center; gap: 0.35rem; padding: 1rem 2rem; background: var(--color-background-soft); border: 1px solid var(--color-border); border-radius: var(--radius-lg); margin-bottom: 1.25rem; }
.verification-code-label { font-size: 0.75rem; font-weight: 600; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
.verification-code-value { font-size: 1.5rem; font-weight: 800; color: var(--brand-600); font-family: 'SF Mono', 'Fira Code', monospace; letter-spacing: 0.06em; }
.verification-countdown { margin-bottom: 1.75rem; }
.countdown-ring { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.6rem 1.25rem; background: var(--warning-50); border: 1px solid var(--warning-100); border-radius: var(--radius-full); font-size: 0.88rem; color: var(--warning-700); }
.countdown-ring svg { color: var(--warning-500); }
.countdown-ring strong { font-weight: 800; font-size: 1rem; font-family: 'SF Mono', 'Fira Code', monospace; }
.verification-steps { display: flex; align-items: center; justify-content: center; gap: 0; margin-bottom: 2rem; }
.v-step { display: flex; flex-direction: column; align-items: center; gap: 0.4rem; }
.v-step-num { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; border-radius: 50%; font-size: 0.8rem; font-weight: 800; background: var(--color-background-mute); color: var(--color-text-muted); border: 2px solid var(--color-border); transition: all 0.3s ease; }
.v-step-num.done { background: var(--success-600); color: white; border-color: var(--success-600); }
.v-step-num.active { background: var(--warning-500); color: white; border-color: var(--warning-500); animation: step-pulse 1.5s ease-in-out infinite; }
@keyframes step-pulse { 0%, 100% { box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.3); } 50% { box-shadow: 0 0 0 8px rgba(245, 158, 11, 0); } }
.v-step-text { font-size: 0.72rem; font-weight: 600; color: var(--color-text-muted); max-width: 80px; text-align: center; line-height: 1.3; }
.v-step-line { width: 40px; height: 2px; background: var(--color-border); margin-bottom: 1.25rem; }
.verification-hint { font-size: 0.78rem; color: var(--color-text-muted); margin-top: 0.75rem; }

/* Rejected & Expired */
.rejected-card, .expired-card { text-align: center; padding: 3rem 2rem; background: var(--color-surface); border-radius: var(--radius-xl); box-shadow: var(--shadow-lg); }
.rejected-card { border: 2px solid var(--danger-100); }
.expired-card { border: 2px solid var(--warning-100); }
.rejected-icon { display: inline-flex; align-items: center; justify-content: center; width: 88px; height: 88px; border-radius: 50%; background: var(--danger-50); color: var(--danger-600); margin-bottom: 1.25rem; }
.expired-icon { display: inline-flex; align-items: center; justify-content: center; width: 88px; height: 88px; border-radius: 50%; background: var(--warning-50); color: var(--warning-600); margin-bottom: 1.25rem; }
.rejected-title { font-size: 1.5rem; font-weight: 800; color: var(--danger-700); margin-bottom: 0.5rem; }
.expired-title { font-size: 1.5rem; font-weight: 800; color: var(--warning-700); margin-bottom: 0.5rem; }
.rejected-reason { font-size: 0.95rem; color: var(--danger-600); background: var(--danger-50); border: 1px solid var(--danger-100); border-radius: var(--radius-md); padding: 0.75rem 1rem; display: inline-block; margin-bottom: 0.75rem; max-width: 400px; font-weight: 500; }
.rejected-subtitle, .expired-subtitle { font-size: 0.92rem; color: var(--color-text-muted); margin-bottom: 1rem; max-width: 400px; margin-left: auto; margin-right: auto; line-height: 1.5; }
.expired-hint { font-size: 0.85rem; color: var(--color-text-muted); margin-bottom: 1.5rem; }
.expired-hint strong { font-weight: 700; color: var(--brand-600); font-family: 'SF Mono', 'Fira Code', monospace; }
.rejected-actions, .expired-actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; }

/* Alerts */
.alert { display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1rem; border-radius: var(--radius-md); font-size: 0.88rem; margin-bottom: 1rem; }
.alert-error { background: var(--danger-50); color: var(--danger-700); border: 1px solid var(--danger-100); }

/* Loading & Spinners */
.loading-state { display: flex; align-items: center; gap: 0.75rem; padding: 3rem 0; justify-content: center; color: var(--color-text-muted); font-size: 0.9rem; }
.spinner { width: 22px; height: 22px; border: 3px solid var(--color-border); border-top-color: var(--color-primary); border-radius: 50%; animation: spin 0.8s linear infinite; }
.spinner-sm { width: 18px; height: 18px; border: 2px solid rgba(255, 255, 255, 0.3); border-top-color: white; border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Buttons */
.btn { display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; font-weight: 600; font-size: 0.9rem; border: none; border-radius: var(--radius-sm); cursor: pointer; transition: all 0.2s ease; text-decoration: none; line-height: 1; }
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary { background: linear-gradient(135deg, var(--brand-600), var(--brand-700)); color: white; box-shadow: var(--shadow-sm), 0 1px 2px rgba(27, 85, 245, 0.2); }
.btn-primary:hover:not(:disabled) { background: linear-gradient(135deg, var(--brand-500), var(--brand-600)); box-shadow: var(--shadow-md), 0 2px 8px rgba(27, 85, 245, 0.25); color: white; transform: translateY(-1px); }
.btn-pay { background: linear-gradient(135deg, var(--success-600), #047857); color: white; box-shadow: var(--shadow-sm), 0 2px 8px rgba(16, 185, 129, 0.3); font-weight: 700; font-size: 1rem; letter-spacing: -0.01em; }
.btn-pay:hover:not(:disabled) { background: linear-gradient(135deg, var(--success-500), var(--success-600)); box-shadow: var(--shadow-md), 0 4px 16px rgba(16, 185, 129, 0.35); color: white; transform: translateY(-1px); }
.btn-yape { background: linear-gradient(135deg, #7c3aed, #6d28d9); color: white; box-shadow: var(--shadow-sm), 0 2px 8px rgba(124, 58, 237, 0.3); font-weight: 700; font-size: 1rem; letter-spacing: -0.01em; }
.btn-yape:hover:not(:disabled) { background: linear-gradient(135deg, #8b5cf6, #7c3aed); box-shadow: var(--shadow-md), 0 4px 16px rgba(124, 58, 237, 0.35); color: white; transform: translateY(-1px); }
.btn-whatsapp { background: linear-gradient(135deg, #25d366, #128c7e); color: white; box-shadow: var(--shadow-sm), 0 2px 8px rgba(37, 211, 102, 0.3); }
.btn-whatsapp:hover:not(:disabled) { background: linear-gradient(135deg, #2ee670, #25d366); box-shadow: var(--shadow-md), 0 4px 16px rgba(37, 211, 102, 0.35); color: white; transform: translateY(-1px); }
.btn-ghost { background: transparent; color: var(--color-text-muted); border: 1px solid var(--color-border); padding: 0.6rem 1.15rem; }
.btn-ghost:hover:not(:disabled) { background: var(--color-background-mute); color: var(--color-heading); }
.btn-lg { padding: 0.85rem 1.75rem; font-size: 0.95rem; border-radius: var(--radius-md); }
.btn-full { width: 100%; justify-content: center; }
.btn-outline-sm { display: inline-flex; align-items: center; justify-content: center; padding: 0.4rem 0.85rem; border: 1px solid var(--brand-300); border-radius: var(--radius-sm); background: white; color: var(--brand-600); font-size: 0.8rem; font-weight: 600; cursor: pointer; }

/* Bottom Actions */
.bottom-actions { display: flex; justify-content: flex-start; margin-bottom: 1rem; }

/* Success State */
.success-card { text-align: center; padding: 3rem 2rem; background: var(--color-surface); border: 2px solid var(--success-200); border-radius: var(--radius-xl); box-shadow: var(--shadow-lg); }
.success-icon { display: inline-flex; align-items: center; justify-content: center; width: 88px; height: 88px; border-radius: 50%; background: linear-gradient(135deg, var(--success-50), var(--success-100)); color: var(--success-600); margin-bottom: 1.25rem; box-shadow: 0 8px 24px rgba(16, 185, 129, 0.2); }
.success-title { font-size: 1.75rem; font-weight: 800; color: var(--color-heading); letter-spacing: -0.02em; margin-bottom: 0.35rem; }
.success-subtitle { font-size: 0.95rem; color: var(--color-text-muted); margin-bottom: 1.75rem; }
.success-card .detail-card { text-align: left; }
.detail-card { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); overflow: hidden; margin-bottom: 1.25rem; }
.detail-row { display: flex; justify-content: space-between; align-items: center; padding: 0.85rem 1.25rem; border-bottom: 1px solid var(--color-border); }
.detail-row:last-child { border-bottom: none; }
.detail-label { font-size: 0.85rem; font-weight: 500; color: var(--color-text-muted); }
.detail-value { font-size: 0.9rem; font-weight: 600; color: var(--color-heading); }
.code-value { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 1.1rem; letter-spacing: 0.05em; color: var(--brand-600); }

/* Tickets */
.tickets-section { margin-bottom: 1.75rem; }
.tickets-title { font-size: 0.92rem; font-weight: 700; color: var(--color-heading); margin-bottom: 0.75rem; }
.tickets-grid { display: flex; flex-wrap: wrap; gap: 0.5rem; justify-content: center; }
.ticket-chip { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.6rem 1rem; background: var(--success-50); border: 1px solid var(--success-100); border-radius: var(--radius-md); font-size: 0.9rem; font-weight: 700; color: var(--success-700); font-family: 'SF Mono', 'Fira Code', monospace; letter-spacing: 0.03em; }
.ticket-chip svg { color: var(--success-500); }
.success-actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; }

/* Auth */
.auth-required-card { text-align: center; padding: 3rem 2rem; background: var(--color-surface); border: 2px solid var(--brand-100); border-radius: var(--radius-xl); box-shadow: var(--shadow-lg); }
.auth-required-icon { display: inline-flex; align-items: center; justify-content: center; width: 72px; height: 72px; border-radius: 50%; background: linear-gradient(135deg, var(--brand-50), var(--brand-100)); color: var(--brand-600); margin-bottom: 1.25rem; }
.auth-required-title { font-size: 1.25rem; font-weight: 700; color: var(--color-heading); margin-bottom: 0.5rem; }
.auth-required-subtitle { font-size: 0.9rem; color: var(--color-text-muted); margin-bottom: 1.5rem; }
.auth-banner { display: flex; align-items: center; gap: 0.65rem; padding: 0.75rem 1rem; background: var(--success-50); border: 1px solid var(--success-100); border-radius: var(--radius-md); margin-bottom: 1.25rem; font-size: 0.88rem; color: var(--success-700); }
.auth-banner strong { font-weight: 700; }

/* Comprobante Selector */
.comprobante-options { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
.comprobante-option { display: flex; align-items: center; gap: 0.75rem; padding: 1rem 1.15rem; border: 2px solid var(--color-border); border-radius: var(--radius-lg); cursor: pointer; transition: all 0.2s ease; background: var(--color-surface); }
.comprobante-option input[type="radio"] { display: none; }
.comprobante-option:hover { border-color: var(--brand-300); }
.comprobante-option.active { border-color: var(--brand-500); background: var(--brand-50); box-shadow: 0 0 0 3px rgba(27, 85, 245, 0.1); }
@media (prefers-color-scheme: dark) { .comprobante-option.active { background: rgba(27, 85, 245, 0.08); } }
.co-icon { display: flex; align-items: center; justify-content: center; width: 44px; height: 44px; border-radius: var(--radius-md); background: var(--color-background-mute); color: var(--color-text-muted); flex-shrink: 0; transition: all 0.2s ease; }
.comprobante-option.active .co-icon { background: var(--brand-100); color: var(--brand-600); }
.comprobante-option strong { display: block; font-size: 0.95rem; font-weight: 700; color: var(--color-heading); }
.comprobante-option span { display: block; font-size: 0.78rem; color: var(--color-text-muted); margin-top: 0.1rem; }

/* Document Type Fixed */
.doc-type-fixed { display: flex; align-items: center; gap: 0.5rem; padding: 0.7rem 0.95rem; border: 2px solid var(--brand-200); border-radius: var(--radius-md); background: var(--brand-50); color: var(--brand-700); font-size: 0.92rem; font-weight: 700; }
@media (prefers-color-scheme: dark) { .doc-type-fixed { background: rgba(27, 85, 245, 0.08); border-color: rgba(27, 85, 245, 0.2); } }
.doc-type-fixed svg { color: var(--brand-500); }

/* DNI Lookup */
.doc-lookup-badge { font-weight: 500; color: var(--brand-500); font-size: 0.72rem; margin-left: 0.35rem; }
.lookup-spinner { display: inline-block; width: 10px; height: 10px; border: 2px solid var(--brand-200); border-top-color: var(--brand-500); border-radius: 50%; animation: spin 0.6s linear infinite; vertical-align: middle; margin-right: 0.2rem; }
.doc-found-badge { font-weight: 600; color: var(--success-600); font-size: 0.72rem; margin-left: 0.35rem; }
.doc-found-badge::before { content: 'v '; }
.doc-lookup-warn { display: block; font-size: 0.75rem; color: var(--warning-600); margin-top: 0.25rem; font-weight: 500; }
.doc-auto-label { display: block; font-size: 0.68rem; color: var(--brand-500); margin-top: 0.2rem; font-weight: 500; }
input[readonly] { background: var(--color-background-mute) !important; cursor: default; }

/* Billing PDF */
.billing-pdf-section { margin-bottom: 1.5rem; text-align: center; }
.btn-success { background: linear-gradient(135deg, var(--success-600), var(--success-700)); color: white; box-shadow: var(--shadow-sm), 0 1px 2px rgba(16, 185, 129, 0.2); }
.btn-success:hover { background: linear-gradient(135deg, var(--success-500), var(--success-600)); box-shadow: var(--shadow-md), 0 2px 8px rgba(16, 185, 129, 0.25); color: white; transform: translateY(-1px); }

/* Security Badges */
.security-badges { display: flex; justify-content: center; gap: 1.5rem; flex-wrap: wrap; margin-top: 1.25rem; padding: 1rem 0; border-top: 1px solid var(--color-border); }
.security-badge { display: flex; align-items: center; gap: 0.4rem; font-size: 0.78rem; font-weight: 500; color: var(--color-text-muted); }
.security-badge svg { color: var(--success-500); flex-shrink: 0; }

/* Responsive */
@media (max-width: 640px) {
  .form-grid { grid-template-columns: 1fr; }
  .btn-lg { width: 100%; justify-content: center; }
  .success-actions, .rejected-actions, .expired-actions { flex-direction: column; }
  .comprobante-options { grid-template-columns: 1fr; }
  .security-badges { flex-direction: column; align-items: center; gap: 0.75rem; }
  .payment-method-card { flex-wrap: wrap; }
  .pm-brands { width: 100%; margin-left: calc(20px + 48px + 0.85rem * 2); margin-top: 0.25rem; }
  .yape-number-value { font-size: 1.5rem; }
  .verification-steps { gap: 0; }
  .v-step-line { width: 24px; }
  .v-step-text { font-size: 0.65rem; max-width: 65px; }
  .section-card { padding: 1.25rem; }
  .reserva-content { padding-top: 1.25rem; }
}
</style>
