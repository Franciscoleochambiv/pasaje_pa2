<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import {
  getRoutes,
  getTripsByRoute,
  getTripSeats,
  getSeatHolder,
  voidReservation,
  type SeatHolder,
  createReservation,
  createAdminReservation,
  getAdminReservations,
  confirmReservation,
  lookupDNI,
  lookupRUC,
  createBillingSale,
  sendBillingEmail,
  getBillingPDFUrl,
  processPayment,
  type Route,
  type TripInstance,
  type SeatInfo,
  type TripSeatResponse,
  type PendingReservationView,
  getAdminSettings,
} from '../../api/client'
import { useSeatsWebSocket } from '../../composables/useSeatsWebSocket'
import { useCulqi } from '../../composables/useCulqi'
import SeatLayoutRenderer from '../../components/SeatLayoutRenderer.vue'
import Swal from 'sweetalert2'

const { initCulqi, openCheckout } = useCulqi()
const culqiReady = ref(false)

// ── State ──
const routes = ref<Route[]>([])
const trips = ref<TripInstance[]>([])
const seatResponse = ref<TripSeatResponse | null>(null)
const seats = computed(() => seatResponse.value?.seats ?? [])

const selectedRouteId = ref<number | null>(null)
const selectedTrip = ref<TripInstance | null>(null)
const selectedSeatIds = ref<Set<number>>(new Set())
const activeFloor = ref(1)
const wsConnected = ref(false)
let wsCleanup: (() => void) | null = null

const loading = ref(false)
const loadingTrips = ref(false)
const loadingSeats = ref(false)
const selling = ref(false)
const errorMsg = ref('')

// ── Reservation (POS hold) ──
const showReserveModal = ref(false)
const reserveDocNumber = ref('')
const reservePhone = ref('')
const reserveName = ref('')
const reserveHours = ref(24)
const reserving = ref(false)
const reserveSuccess = ref(false)
const reserveCode = ref('')
const reserveSeatLabels = ref<string[]>([])

// ── Pending reservations panel ──
const pendingReservations = ref<PendingReservationView[]>([])
const loadingPendingReservations = ref(false)

// ── Convert reservation to sale ──
const showConvertModal = ref(false)
const convertCode = ref('')
const convertSeatLabels = ref<string[]>([])
const convertDocNumber = ref('')
const convertPhone = ref('')
const converting = ref(false)

// Passenger form
const passengerName = ref('')
const docType = ref('DNI')
const docNumber = ref('')
const email = ref('')
const phone = ref('')
const lookingUpDoc = ref(false)
const docLookupError = ref('')
const passengerAddress = ref('')

type BillingDocType = 'boleta' | 'factura' | 'pedido'

// Billing form
const billingDocType = ref<BillingDocType>('pedido')
const pricePerSeat = ref(0)
const billingDocLabel = computed(() => {
  if (billingDocType.value === 'factura') return 'Factura'
  if (billingDocType.value === 'pedido') return 'Pedido'
  return 'Boleta'
})

// Price is loaded from selected route
const pdfFormat = ref('ticket')

// Payment method: ventanilla (cash) or tarjeta (Culqi)
const paymentMethod = ref<'ventanilla' | 'tarjeta'>('ventanilla')

// Sale result
const saleSuccess = ref(false)
const saleTicketCodes = ref<string[]>([])
const saleReservationCode = ref('')
const saleSeatLabels = ref<string[]>([])
const billingSaleId = ref<number | null>(null)
const billingStatus = ref('')
const billingMessage = ref('')
const billingLoading = ref(false)

// Sale modal
const showSaleModal = ref(false)

// ── Computed ──
const selectedRoute = computed(() => routes.value.find(r => r.id === selectedRouteId.value) ?? null)

const selectedSeats = computed(() =>
  seats.value.filter(s => selectedSeatIds.value.has(s.id))
)

const vehicleFloors = computed(() => seatResponse.value?.vehicle_floors ?? 1)
const vehicleLayoutCols = computed(() => seatResponse.value?.vehicle_layout_cols ?? 4)
const vehicleName = computed(() => seatResponse.value?.vehicle_name ?? '')
const layoutElements = computed(() => seatResponse.value?.layout_elements ?? [])

const floorSeats = computed(() => {
  const floorData = seats.value.filter(s => s.floor === activeFloor.value)
  const rows = new Map<number, SeatInfo[]>()
  for (const s of floorData) {
    if (!rows.has(s.row_num)) rows.set(s.row_num, [])
    rows.get(s.row_num)!.push(s)
  }
  for (const [, cols] of rows) {
    cols.sort((a, b) => a.col_num - b.col_num)
  }
  return new Map([...rows].sort((a, b) => a[0] - b[0]))
})

const maxCols = computed(() => vehicleLayoutCols.value)

const seatGrid = computed(() => {
  // Considerar también filas que sólo tienen elementos decorativos.
  const fSeats = seats.value.filter(s => s.floor === activeFloor.value)
  const fElems = layoutElements.value.filter(e => e.floor === activeFloor.value)
  const rowSet = new Set<number>()
  for (const s of fSeats) rowSet.add(s.row_num)
  for (const e of fElems) rowSet.add(e.row_num)
  const rowsSorted = Array.from(rowSet).sort((a, b) => a - b)

  const grid: (SeatInfo | null)[][] = []
  for (const r of rowsSorted) {
    const row: (SeatInfo | null)[] = Array(maxCols.value).fill(null)
    for (const s of fSeats) {
      if (s.row_num !== r) continue
      const colIdx = s.col_num - 1
      if (colIdx >= 0 && colIdx < maxCols.value) row[colIdx] = s
    }
    grid.push(row)
  }
  return grid
})

const gridRowNumbers = computed(() => {
  const fSeats = seats.value.filter(s => s.floor === activeFloor.value)
  const fElems = layoutElements.value.filter(e => e.floor === activeFloor.value)
  const rowSet = new Set<number>()
  for (const s of fSeats) rowSet.add(s.row_num)
  for (const e of fElems) rowSet.add(e.row_num)
  return Array.from(rowSet).sort((a, b) => a - b)
})

function getElementAt(row: number, col: number) {
  return layoutElements.value.find(e =>
    e.floor === activeFloor.value && e.row_num === row && e.col_num === col + 1
  )
}

const seatStats = computed(() => {
  const total = seats.value.length
  const available = seats.value.filter(s => s.status === 'available').length
  const sold = seats.value.filter(s => s.status === 'sold').length
  const held = seats.value.filter(s => s.status === 'held').length
  return { total, available, sold, held }
})

const canSell = computed(() =>
  selectedSeats.value.length > 0 &&
  passengerName.value.trim() !== '' &&
  docNumber.value.trim() !== '' &&
  pricePerSeat.value > 0 &&
  !selling.value
)

const canReserve = computed(() =>
  selectedSeats.value.length > 0 &&
  reserveDocNumber.value.trim() !== '' &&
  reserveName.value.trim() !== '' &&
  !reserving.value
)

const canConvert = computed(() =>
  passengerName.value.trim() !== '' &&
  docNumber.value.trim() !== '' &&
  pricePerSeat.value > 0 &&
  !converting.value
)

// DNI/RUC auto-lookup
watch(docNumber, async (val) => {
  const num = val.trim()
  docLookupError.value = ''
  if (docType.value === 'DNI' && num.length === 8) {
    lookingUpDoc.value = true
    try {
      const result = await lookupDNI(num)
      if (result.nombre_completo) {
        passengerName.value = result.nombre_completo
      } else if (result.nombres) {
        passengerName.value = `${result.nombres} ${result.apellido_paterno} ${result.apellido_materno}`.trim()
      }
    } catch (e) {
      docLookupError.value = 'No se pudo consultar el DNI'
    } finally {
      lookingUpDoc.value = false
    }
  } else if (docType.value === 'RUC' && num.length === 11) {
    lookingUpDoc.value = true
    try {
      const result = await lookupRUC(num)
      passengerName.value = result.nombre_o_razon_social || result.razon_social || ''
      passengerAddress.value = result.direccion || ''
      billingDocType.value = 'factura'
    } catch (e) {
      docLookupError.value = 'No se pudo consultar el RUC'
    } finally {
      lookingUpDoc.value = false
    }
  }
})

// DNI lookup for reservation modal
watch(reserveDocNumber, async (val) => {
  const num = val.trim()
  if (num.length === 8) {
    try {
      const result = await lookupDNI(num)
      if (result.nombre_completo) {
        reserveName.value = result.nombre_completo
      } else if (result.nombres) {
        reserveName.value = `${result.nombres} ${result.apellido_paterno} ${result.apellido_materno}`.trim()
      }
    } catch { /* silent - allow manual entry */ }
  }
})

// If doc type changes to RUC, switch billing to factura
watch(docType, (val) => {
  if (val === 'RUC') {
    billingDocType.value = 'factura'
  } else {
    billingDocType.value = 'pedido'
    passengerAddress.value = ''
  }
})

// Pedido se emite solo en ventanilla. El flujo con tarjeta usa el endpoint
// publico de pago, que solo debe generar boleta/factura.
watch(billingDocType, (val) => {
  if (val === 'pedido' && paymentMethod.value === 'tarjeta') {
    paymentMethod.value = 'ventanilla'
  }
})

function isAisleCol(colIdx: number) {
  const floorData = seats.value.filter(s => s.floor === activeFloor.value)
  const colNum = colIdx + 1
  return !floorData.some(s => s.col_num === colNum)
}

function seatClass(seat: SeatInfo): string {
  if (selectedSeatIds.value.has(seat.id)) return 'seat selected'
  if (seat.status === 'sold') return 'seat sold'
  if (seat.status === 'held') return 'seat held'
  if (seat.status === 'blocked') return 'seat blocked'
  return 'seat available'
}

// ── Loaders ──
async function loadRoutes() {
  loading.value = true
  errorMsg.value = ''
  try {
    routes.value = await getRoutes()
  } catch (e) {
    errorMsg.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function loadTrips() {
  if (!selectedRouteId.value) return
  loadingTrips.value = true
  errorMsg.value = ''
  trips.value = []
  selectedTrip.value = null
  seatResponse.value = null
  selectedSeatIds.value = new Set()
  disconnectWs()
  try {
    const from = new Date().toISOString().slice(0, 10)
    trips.value = await getTripsByRoute(selectedRouteId.value, from)
  } catch (e) {
    errorMsg.value = (e as Error).message
  } finally {
    loadingTrips.value = false
  }
}

async function selectTrip(trip: TripInstance) {
  selectedTrip.value = trip
  loadingSeats.value = true
  errorMsg.value = ''
  selectedSeatIds.value = new Set()
  activeFloor.value = 1
  disconnectWs()
  try {
    seatResponse.value = await getTripSeats(trip.id)
  } catch (e) {
    errorMsg.value = (e as Error).message
    seatResponse.value = null
  } finally {
    loadingSeats.value = false
  }
  // Connect WebSocket
  if (trip.id) {
    const { connected, disconnect } = useSeatsWebSocket(trip.id, (data) => {
      seatResponse.value = data
    })
    wsCleanup = () => { disconnect(); wsConnected.value = false }
    watch(connected, (val) => { wsConnected.value = val }, { immediate: true })
  }
  loadPendingReservations()
}

async function loadPendingReservations() {
  if (!selectedTrip.value) return
  loadingPendingReservations.value = true
  try {
    pendingReservations.value = await getAdminReservations(selectedTrip.value.id)
  } catch {
    pendingReservations.value = []
  } finally {
    loadingPendingReservations.value = false
  }
}

function disconnectWs() {
  if (wsCleanup) { wsCleanup(); wsCleanup = null }
  wsConnected.value = false
}

// Seat holder modal — admin only.
const holder = ref<SeatHolder | null>(null)
const holderLoading = ref(false)
const holderSeatLabel = ref('')

async function showHolder(seat: SeatInfo) {
  if (!selectedTrip.value) return
  holderLoading.value = true
  holder.value = null
  holderSeatLabel.value = seat.label
  try {
    holder.value = await getSeatHolder(selectedTrip.value.id, seat.id)
  } catch {
    holder.value = null
  }
  holderLoading.value = false
}

function closeHolder() {
  holder.value = null
  holderSeatLabel.value = ''
  holderLoading.value = false
}

async function reprintHolderTicket() {
  if (!holder.value?.billing_sale_id) return
  const saleID = holder.value.billing_sale_id
  // Cerrar el modal de titular para evitar que tape el Swal (z-index conflict)
  closeHolder()
  const result = await Swal.fire({
    title: 'Reimprimir comprobante',
    html: `<p style="color:#475569;font-size:0.9rem">Selecciona el formato de impresión para el comprobante <strong>#${saleID}</strong>.</p>`,
    showCancelButton: true,
    showDenyButton: true,
    confirmButtonText: 'Ticket 80mm',
    denyButtonText: 'A4 Media',
    cancelButtonText: 'A4 Full',
    confirmButtonColor: '#1b55f5',
    denyButtonColor: '#0ea5e9',
    cancelButtonColor: '#64748b',
    reverseButtons: true,
    customClass: { popup: 'swal-custom-popup' },
  })
  let format = ''
  if (result.isConfirmed) format = 'ticket'
  else if (result.isDenied) format = 'a4_half'
  else if (result.dismiss === Swal.DismissReason.cancel) format = 'a4_full'
  if (!format) return
  window.open(getBillingPDFUrl(saleID, format), '_blank')
}

const voidingHolder = ref(false)

async function voidHolder() {
  if (!holder.value || !selectedTrip.value) return
  // Capturar valores antes de cerrar el modal y abrir Swal.
  const code = holder.value.reservation_code
  const seatLabel = holder.value.seat_label
  const passengerName = holder.value.passenger_name
  const passengerDoc = [holder.value.passenger_doc_type, holder.value.passenger_doc_number].filter(Boolean).join(' ')
  const tripId = selectedTrip.value.id
  const isSale = !!holder.value.ticket_code
  if (!code) return

  // Cerrar el modal de titular para evitar que tape el Swal (z-index conflict)
  closeHolder()

  const title = isSale
    ? `<span style="font-size:1.15rem;font-weight:700;color:#1e293b">Anular pasaje ${seatLabel}</span>`
    : `<span style="font-size:1.15rem;font-weight:700;color:#1e293b">Anular reserva ${seatLabel}</span>`
  const warningMsg = isSale
    ? '⚠️ Esta acción liberará el asiento y anulará el comprobante de forma permanente.'
    : '⚠️ Esta acción liberará el asiento y cancelará la reserva.'
  const confirmText = isSale ? 'Sí, anular pasaje' : 'Sí, anular reserva'
  const placeholder = isSale
    ? 'Describe brevemente por qué se anula este pasaje...'
    : 'Describe brevemente por qué se anula esta reserva...'

  const { value: reason, isConfirmed } = await Swal.fire({
    title,
    html: `
      <div style="text-align:left;margin-bottom:1rem">
        <div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:12px;padding:1rem;margin-bottom:1rem">
          <div style="display:flex;gap:0.5rem;align-items:center;margin-bottom:0.35rem">
            <span style="font-size:0.75rem;text-transform:uppercase;letter-spacing:0.05em;color:#64748b;font-weight:600">Reserva</span>
            <span style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:0.85rem;color:#0f172a;font-weight:700">${code}</span>
          </div>
          <div style="display:flex;gap:0.5rem;align-items:center;margin-bottom:0.35rem">
            <span style="font-size:0.75rem;text-transform:uppercase;letter-spacing:0.05em;color:#64748b;font-weight:600">Pasajero</span>
            <span style="font-size:0.9rem;color:#0f172a;font-weight:600">${passengerName || '—'}</span>
          </div>
          ${passengerDoc ? `<div style="display:flex;gap:0.5rem;align-items:center"><span style="font-size:0.75rem;text-transform:uppercase;letter-spacing:0.05em;color:#64748b;font-weight:600">Documento</span><span style="font-size:0.9rem;color:#0f172a">${passengerDoc}</span></div>` : ''}
        </div>
        <p style="font-size:0.85rem;color:#dc2626;font-weight:500;margin:0 0 0.75rem 0">
          ${warningMsg}
        </p>
        <label for="swal-void-reason" style="display:block;font-size:0.8rem;font-weight:600;color:#475569;margin-bottom:0.4rem">Motivo de anulación *</label>
        <textarea
          id="swal-void-reason"
          placeholder="${placeholder}"
          style="width:100%;height:80px;border-radius:10px;border:1.5px solid #cbd5e1;padding:0.75rem;font-size:0.9rem;line-height:1.5;resize:none;overflow:hidden;box-sizing:border-box;font-family:inherit;outline:none"
        ></textarea>
      </div>
    `,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonText: confirmText,
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#dc2626',
    cancelButtonColor: '#94a3b8',
    reverseButtons: true,
    focusConfirm: false,
    allowOutsideClick: false,
    customClass: { popup: 'swal-custom-popup', confirmButton: 'swal-btn-confirm', cancelButton: 'swal-btn-cancel' },
    preConfirm: () => {
      const textarea = document.getElementById('swal-void-reason') as HTMLTextAreaElement | null
      const val = textarea?.value?.trim() || ''
      if (!val) {
        Swal.showValidationMessage('Debes ingresar un motivo de anulación')
        return false
      }
      if (val.length < 5) {
        Swal.showValidationMessage('El motivo debe tener al menos 5 caracteres')
        return false
      }
      return val
    },
    didOpen: () => {
      const textarea = document.getElementById('swal-void-reason') as HTMLTextAreaElement | null
      textarea?.focus()
    },
  })
  if (!isConfirmed) return
  voidingHolder.value = true
  try {
    await voidReservation(code, reason as string)
    Swal.fire({
      toast: true, position: 'top-end', icon: 'success',
      title: isSale ? `Asiento ${seatLabel} liberado` : `Reserva ${seatLabel} cancelada`,
      showConfirmButton: false, timer: 3500,
    })
    try { seatResponse.value = await getTripSeats(tripId) } catch { /* WS refresh */ }
    loadPendingReservations()
  } catch (e) {
    Swal.fire({ icon: 'error', title: 'No se pudo anular', text: (e as Error).message, confirmButtonColor: '#1b55f5' })
  } finally {
    voidingHolder.value = false
  }
}

// Wrapper para el evento del SeatLayoutRenderer (entrega RendererSeat).
// Buscamos el SeatInfo completo por id para mantener la firma estricta.
function handleRendererSeatClick(rs: { id?: number }) {
  if (rs.id == null) return
  const full = seats.value.find(s => s.id === rs.id)
  if (full) toggleSeat(full)
}

function toggleSeat(seat: SeatInfo) {
  // Si el asiento está ocupado y no está dentro de la selección actual,
  // mostramos al titular en vez de bloquear silenciosamente.
  if (seat.status !== 'available' && !selectedSeatIds.value.has(seat.id)) {
    showHolder(seat)
    return
  }
  const copy = new Set(selectedSeatIds.value)
  if (copy.has(seat.id)) {
    copy.delete(seat.id)
  } else {
    copy.add(seat.id)
  }
  selectedSeatIds.value = copy
}

function formatDeparture(dt: string): string {
  const d = new Date(dt)
  return d.toLocaleDateString('es-PE', { weekday: 'short', day: '2-digit', month: 'short' }) +
    ' ' + d.toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' })
}

// ── Reserve (POS hold) ──
function openReserveModal() {
  if (selectedSeats.value.length === 0) return
  showReserveModal.value = true
  reserveSuccess.value = false
  reserveCode.value = ''
  reserveSeatLabels.value = selectedSeats.value.map(s => s.label)
}

async function processReserve() {
  if (!canReserve.value || !selectedTrip.value) return
  reserving.value = true
  errorMsg.value = ''
  try {
    const seatIds = selectedSeats.value.map(s => s.id)
    const result = await createAdminReservation({
      trip_instance_id: selectedTrip.value.id,
      seats: seatIds,
      contact_name: reserveName.value.trim(),
      contact_doc_number: reserveDocNumber.value.trim(),
      contact_phone: reservePhone.value.trim(),
      hold_hours: reserveHours.value,
    })
    reserveCode.value = result.code
    reserveSuccess.value = true
    selectedSeatIds.value = new Set()
    loadPendingReservations()
    Swal.fire({
      toast: true, position: 'top-end', icon: 'success',
      title: 'Reserva creada', showConfirmButton: false, timer: 3000,
    })
  } catch (e) {
    const msg = (e as Error).message
    errorMsg.value = msg
    Swal.fire({ icon: 'error', title: 'Error al reservar', text: msg, confirmButtonColor: '#1b55f5' })
  } finally {
    reserving.value = false
  }
}

function resetReserve() {
  showReserveModal.value = false
  reserveSuccess.value = false
  reserveCode.value = ''
  reserveSeatLabels.value = []
  reserveDocNumber.value = ''
  reservePhone.value = ''
  reserveName.value = ''
  reserveHours.value = 24
  errorMsg.value = ''
}

// ── Convert reservation to sale ──
function openConvertModal(res: PendingReservationView) {
  convertCode.value = res.code
  convertSeatLabels.value = res.seat_labels
  convertDocNumber.value = res.contact_doc_number
  convertPhone.value = res.contact_phone
  // Pre-fill sale form with contact data
  docNumber.value = res.contact_doc_number
  phone.value = res.contact_phone
  passengerName.value = res.contact_name || ''
  showConvertModal.value = true
}

async function processConvertSale() {
  if (!convertCode.value || !selectedTrip.value) return
  converting.value = true
  errorMsg.value = ''
  try {
    const selectedBillingDocType = billingDocType.value
    const result = await confirmReservation(convertCode.value, {
      payment_method: paymentMethod.value,
      payment_reference: '',
      passenger_name: passengerName.value.trim(),
      passenger_doc_type: docType.value,
      passenger_doc_number: docNumber.value.trim(),
      passenger_email: email.value.trim(),
      passenger_phone: phone.value.trim(),
      passenger_address: passengerAddress.value.trim(),
      document_type: selectedBillingDocType,
      route_name: selectedRoute.value?.name ?? '',
      seat_labels: convertSeatLabels.value,
      price_per_seat: pricePerSeat.value,
    })
    saleReservationCode.value = convertCode.value
    saleTicketCodes.value = result.ticket_codes
    saleSeatLabels.value = convertSeatLabels.value
    saleSuccess.value = true

    // Billing
    billingLoading.value = true
    try {
      const billingResult = await createBillingSale({
        document_type: selectedBillingDocType,
        trip_instance_id: selectedTrip.value.id,
        reservation_code: convertCode.value,
        passenger_name: passengerName.value.trim(),
        passenger_doc_type: docType.value,
        passenger_doc_number: docNumber.value.trim(),
        passenger_address: passengerAddress.value.trim() || undefined,
        price_per_seat: pricePerSeat.value,
        route_name: selectedRoute.value?.name ?? '',
        seat_labels: convertSeatLabels.value,
      })
      billingSaleId.value = billingResult.billing_sale_id
      billingStatus.value = billingResult.status
      billingMessage.value = billingResult.message || ''
      if (email.value.trim() && billingResult.billing_sale_id > 0) {
        sendBillingEmail({
          email: email.value.trim(),
          passenger_name: passengerName.value.trim(),
          reservation_code: convertCode.value,
          ticket_codes: result.ticket_codes || [],
          total_amount: pricePerSeat.value * convertSeatLabels.value.length,
          seat_labels: convertSeatLabels.value,
          route_name: selectedRoute.value?.name ?? '',
          billing_sale_id: billingResult.billing_sale_id,
          payment_ref: paymentMethod.value,
        }).catch(() => {})
      }
    } catch (e) {
      billingMessage.value = (e as Error).message
      billingStatus.value = 'error'
    } finally {
      billingLoading.value = false
    }

    showConvertModal.value = false
    loadPendingReservations()
    Swal.fire({
      toast: true, position: 'top-end', icon: 'success',
      title: 'Venta completada', showConfirmButton: false, timer: 3000,
    })
  } catch (e) {
    const msg = (e as Error).message
    errorMsg.value = msg
    Swal.fire({ icon: 'error', title: 'Error en la venta', text: msg, confirmButtonColor: '#3B82F6' })
  } finally {
    converting.value = false
  }
}

// ── Sale ──
async function processSale() {
  if (!canSell.value || !selectedTrip.value) return
  selling.value = true
  errorMsg.value = ''
  try {
    const seatIds = selectedSeats.value.map(s => s.id)
    const seatLabels = selectedSeats.value.map(s => s.label)
    const selectedBillingDocType = billingDocType.value

    if (paymentMethod.value === 'tarjeta' && selectedBillingDocType !== 'pedido') {
      // ── Culqi card payment flow ──
      if (!culqiReady.value) {
        await initCulqi()
        culqiReady.value = true
      }

      const totalCents = Math.round(pricePerSeat.value * selectedSeats.value.length * 100)
      const description = `Pasaje ${selectedRoute.value?.name ?? ''} - ${seatLabels.join(', ')}`

      const tokenId = await openCheckout({
        title: 'Pasaje Bus',
        currency: 'PEN',
        amount: totalCents,
        description,
      })

      // Send token to backend — it does everything
      const result = await processPayment({
        token_id: tokenId,
        amount: totalCents,
        email: email.value.trim() || 'ventanilla@pasaje.pe',
        description,
        trip_instance_id: selectedTrip.value.id,
        seats: seatIds,
        passenger_name: passengerName.value.trim(),
        passenger_doc_type: docType.value,
        passenger_doc_number: docNumber.value.trim(),
        passenger_address: passengerAddress.value.trim() || undefined,
        document_type: selectedBillingDocType,
        route_name: selectedRoute.value?.name ?? '',
        seat_labels: seatLabels,
        price_per_seat: pricePerSeat.value,
        passenger_phone: phone.value.trim(),
      })

      saleReservationCode.value = result.reservation_code
      saleTicketCodes.value = result.ticket_codes || []
      saleSeatLabels.value = seatLabels
      billingSaleId.value = result.billing_sale_id || null
      billingStatus.value = result.billing_error ? 'error' : 'ok'
      billingMessage.value = result.billing_error || ''
      saleSuccess.value = true

    } else {
      // ── Cash (ventanilla) flow — existing logic ──
      // Step 1: Create reservation
      const reservation = await createReservation({
        trip_instance_id: selectedTrip.value.id,
        seats: seatIds,
        passenger_name: passengerName.value.trim(),
        passenger_doc_type: docType.value,
        passenger_doc_number: docNumber.value.trim(),
        passenger_email: email.value.trim(),
        passenger_phone: phone.value.trim(),
      })
      // Step 2: Confirm immediately
      const result = await confirmReservation(reservation.code, {
        payment_method: 'ventanilla',
        payment_reference: '',
        passenger_name: passengerName.value.trim(),
        passenger_doc_type: docType.value,
        passenger_doc_number: docNumber.value.trim(),
        passenger_email: email.value.trim(),
        passenger_phone: phone.value.trim(),
        passenger_address: passengerAddress.value.trim(),
        document_type: billingDocType.value,
        route_name: selectedRoute.value?.name ?? '',
        seat_labels: seatLabels,
        price_per_seat: pricePerSeat.value,
      })
      saleReservationCode.value = reservation.code
      saleTicketCodes.value = result.ticket_codes
      saleSeatLabels.value = seatLabels
      saleSuccess.value = true

      // Step 3: Create billing sale
      billingLoading.value = true
      try {
        const billingResult = await createBillingSale({
          document_type: selectedBillingDocType,
          trip_instance_id: selectedTrip.value!.id,
          reservation_code: reservation.code,
          passenger_name: passengerName.value.trim(),
          passenger_doc_type: docType.value,
          passenger_doc_number: docNumber.value.trim(),
          passenger_address: passengerAddress.value.trim() || undefined,
          price_per_seat: pricePerSeat.value,
          route_name: selectedRoute.value?.name ?? '',
          seat_labels: seatLabels,
        })
        billingSaleId.value = billingResult.billing_sale_id
        billingStatus.value = billingResult.status
        billingMessage.value = billingResult.message || ''

        // Auto-send email in background (don't block)
        if (email.value.trim() && billingResult.billing_sale_id > 0) {
          sendBillingEmail({
            email: email.value.trim(),
            passenger_name: passengerName.value.trim(),
            reservation_code: reservation.code,
            ticket_codes: result.ticket_codes || [],
            total_amount: pricePerSeat.value * selectedSeats.value.length,
            seat_labels: seatLabels,
            route_name: selectedRoute.value?.name ?? '',
            billing_sale_id: billingResult.billing_sale_id,
            payment_ref: 'ventanilla',
          }).catch(() => { /* silent - email is best effort */ })
        }
      } catch (e) {
        billingMessage.value = (e as Error).message
        billingStatus.value = 'error'
      } finally {
        billingLoading.value = false
      }
    }

    showSaleModal.value = false
    selectedSeatIds.value = new Set()

    Swal.fire({
      toast: true,
      position: 'top-end',
      icon: 'success',
      title: 'Venta completada',
      showConfirmButton: false,
      timer: 3000,
    })
  } catch (e) {
    const msg = (e as Error).message
    // Ignore Culqi cancellation — not an error
    if (msg === 'Pago cancelado' || msg.includes('cancelado')) {
      errorMsg.value = ''
    } else {
      errorMsg.value = msg
      Swal.fire({
        icon: 'error',
        title: 'Error en la venta',
        text: msg,
        confirmButtonColor: '#3B82F6',
        customClass: {
          popup: 'swal-custom-popup',
          confirmButton: 'swal-btn-confirm',
        },
      })
    }
  } finally {
    selling.value = false
  }
}

function resetSale() {
  // Keep route and trip selected — operator has a queue of passengers
  const keepRoute = selectedRouteId.value
  const keepTrip = selectedTrip.value

  saleSuccess.value = false
  saleTicketCodes.value = []
  saleReservationCode.value = ''
  saleSeatLabels.value = []
  billingSaleId.value = null
  billingStatus.value = ''
  billingMessage.value = ''
  billingLoading.value = false
  selectedSeatIds.value = new Set()
  passengerName.value = ''
  docType.value = 'DNI'
  docNumber.value = ''
  email.value = ''
  phone.value = ''
  passengerAddress.value = ''
  billingDocType.value = 'pedido'
  // Restore price from selected route
  const route = routes.value.find(r => r.id === selectedRouteId.value)
  if (route) pricePerSeat.value = route.price_per_seat || 0
  pdfFormat.value = 'ticket'
  paymentMethod.value = 'ventanilla'
  errorMsg.value = ''
  showSaleModal.value = false
  showConvertModal.value = false
  converting.value = false

  // Restore trip — just reload seats (WebSocket keeps them fresh)
  if (keepTrip && keepRoute) {
    selectedRouteId.value = keepRoute
    selectedTrip.value = keepTrip
    // Seats are already live via WebSocket, just clear selection
  }
}

function openPDF(format: string) {
  if (billingSaleId.value) {
    window.open(getBillingPDFUrl(billingSaleId.value, format), '_blank')
  }
}

function openWhatsApp() {
  if (billingSaleId.value) {
    const pdfUrl = getBillingPDFUrl(billingSaleId.value, 'ticket')
    const fullUrl = window.location.origin + pdfUrl
    const msg = encodeURIComponent(`Tu boleto de viaje: ${fullUrl}`)
    const phoneNum = phone.value.trim().replace(/\D/g, '')
    const waUrl = phoneNum ? `https://wa.me/51${phoneNum}?text=${msg}` : `https://wa.me/?text=${msg}`
    window.open(waUrl, '_blank')
  }
}

// ── Lifecycle ──
watch(selectedRouteId, () => {
  loadTrips()
  // Set price from selected route
  const route = routes.value.find(r => r.id === selectedRouteId.value)
  if (route) pricePerSeat.value = route.price_per_seat || 0
})

async function loadPosSettings() {
  try {
    const settings = await getAdminSettings()
    const hours = parseInt(settings['pos_hold_hours'] || '24', 10)
    if (!isNaN(hours) && hours > 0) {
      reserveHours.value = hours
    }
  } catch { /* fallback to default 24 */ }
}

loadRoutes()
loadPosSettings()

onUnmounted(() => { disconnectWs() })
</script>

<template>
  <div class="pv-page">
    <div class="pv-header">
      <h1 class="pv-title">Punto de Venta</h1>
      <p class="pv-subtitle">Venta de boletos en ventanilla</p>
    </div>

    <!-- Sale success -->
    <div v-if="saleSuccess" class="pv-success-card">
      <div class="pv-success-icon">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
          <polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
      </div>
      <h2 class="pv-success-title">Venta Completada</h2>
      <p class="pv-success-code">Reserva: <strong>{{ saleReservationCode }}</strong></p>

      <!-- Seat number(s) shown big so cashier/passenger can see at a glance -->
      <div v-if="saleSeatLabels.length" class="pv-seat-banner">
        <span class="pv-seat-label">ASIENTO{{ saleSeatLabels.length === 1 ? '' : 'S' }}</span>
        <div class="pv-seat-row">
          <div v-for="lbl in saleSeatLabels" :key="lbl" class="pv-seat-big">{{ lbl }}</div>
        </div>
      </div>

      <div class="pv-tickets">
        <div v-for="code in saleTicketCodes" :key="code" class="pv-ticket">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>
            <path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
          </svg>
          {{ code }}
        </div>
      </div>

      <!-- Billing status -->
      <div v-if="billingLoading" class="pv-billing-status pv-billing-loading">
        <span class="pv-spinner"></span> Generando comprobante...
      </div>
      <div v-else-if="billingSaleId" class="pv-billing-status pv-billing-ok">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" x2="8" y1="13" y2="13"/>
          <line x1="16" x2="8" y1="17" y2="17"/>
          <polyline points="10 9 9 9 8 9"/>
        </svg>
        Comprobante #{{ billingSaleId }} generado ({{ billingDocLabel }})
      </div>
      <div v-else-if="billingStatus === 'error'" class="pv-billing-status pv-billing-err">
        Error al generar comprobante: {{ billingMessage }}
      </div>

      <!-- PDF preview and actions -->
      <div v-if="billingSaleId" class="pv-pdf-section">
        <div class="pv-pdf-formats">
          <button class="pv-pdf-btn" :class="{ active: pdfFormat === 'ticket' }" @click="pdfFormat = 'ticket'">Ticket 80mm</button>
          <button class="pv-pdf-btn" :class="{ active: pdfFormat === 'a4_half' }" @click="pdfFormat = 'a4_half'">A4 Media</button>
          <button class="pv-pdf-btn" :class="{ active: pdfFormat === 'a4_full' }" @click="pdfFormat = 'a4_full'">A4 Full</button>
        </div>
        <iframe
          :src="getBillingPDFUrl(billingSaleId, pdfFormat)"
          class="pv-pdf-preview"
          title="Vista previa del comprobante"
        ></iframe>
        <div class="pv-pdf-actions">
          <button class="pv-btn pv-btn-print" @click="openPDF(pdfFormat)">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="6 9 6 2 18 2 18 9"/>
              <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/>
              <rect width="12" height="8" x="6" y="14"/>
            </svg>
            Imprimir
          </button>
          <button class="pv-btn pv-btn-whatsapp" @click="openWhatsApp">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
            </svg>
            WhatsApp
          </button>
        </div>
      </div>

      <button class="pv-btn pv-btn-new" @click="resetSale">Nueva Venta</button>
    </div>

    <!-- Main single-column layout -->
    <div v-else class="pv-single-col">
      <!-- Step 1: Route + Trip -->
      <div class="pv-card">
        <h3 class="pv-card-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="6" cy="19" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/><circle cx="18" cy="5" r="3"/>
          </svg>
          Seleccionar Viaje
        </h3>

        <div class="pv-field">
          <label class="pv-label">Ruta</label>
          <select v-model="selectedRouteId" class="pv-select" :disabled="loading">
            <option :value="null" disabled>Seleccionar ruta...</option>
            <option v-for="r in routes" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </div>

        <div v-if="loadingTrips" class="pv-loading">Cargando salidas...</div>
        <div v-else-if="selectedRouteId && trips.length === 0 && !loadingTrips" class="pv-empty">No hay salidas disponibles para esta ruta.</div>
        <div v-else-if="trips.length > 0" class="pv-trips">
          <button
            v-for="t in trips"
            :key="t.id"
            class="pv-trip-item"
            :class="{ active: selectedTrip?.id === t.id }"
            @click="selectTrip(t)"
          >
            <div class="pv-trip-time">{{ formatDeparture(t.departure_at) }}</div>
            <div class="pv-trip-route">{{ t.route_name }}</div>
            <span class="pv-trip-badge" :class="t.status">{{ t.status }}</span>
          </button>
        </div>
      </div>

      <!-- Step 2: Seat Map (full width, centered) -->
      <div v-if="selectedTrip && seatResponse" class="pv-card pv-card-seats">
        <div class="pv-card-title-row">
          <h3 class="pv-card-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M19 9V6a2 2 0 0 0-2-2H7a2 2 0 0 0-2 2v3"/>
              <path d="M3 16a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5a2 2 0 0 0-4 0v1.5H7V11a2 2 0 0 0-4 0z"/>
              <path d="M5 18v2"/><path d="M19 18v2"/>
            </svg>
            Asientos — {{ vehicleName }}
          </h3>
          <div v-if="wsConnected" class="pv-ws-badge">
            <span class="pv-ws-dot"></span> EN VIVO
          </div>
        </div>

        <!-- Stats -->
        <div class="pv-seat-stats">
          <span class="pv-stat"><span class="pv-stat-dot available"></span> {{ seatStats.available }} libres</span>
          <span class="pv-stat"><span class="pv-stat-dot sold"></span> {{ seatStats.sold }} vendidos</span>
          <span class="pv-stat"><span class="pv-stat-dot held"></span> {{ seatStats.held }} retenidos</span>
        </div>

        <!-- Seat grid (mismo render que el editor de plantillas) -->
        <div v-if="loadingSeats" class="pv-loading">Cargando asientos...</div>
        <div v-else class="pv-seat-area">
          <SeatLayoutRenderer
            :floors="vehicleFloors"
            :seats="seats"
            :elements="layoutElements"
            :selected-seat-ids="selectedSeatIds"
            :layout-cols="vehicleLayoutCols"
            interactive
            @seat-click="handleRendererSeatClick"
          />
        </div>

        <!-- Pending reservations panel -->
        <div class="pv-pending-reservations">
          <h4 class="pv-pending-title">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            Reservas activas de este viaje
          </h4>
          <div v-if="loadingPendingReservations" class="pv-pending-loading">Cargando...</div>
          <div v-else-if="pendingReservations.length === 0" class="pv-pending-empty">No hay reservas activas para este viaje.</div>
          <div v-else class="pv-pending-list">
            <div v-for="res in pendingReservations" :key="res.id" class="pv-pending-item">
              <div class="pv-pending-main">
                <strong class="pv-pending-code">{{ res.code }}</strong>
                <span class="pv-pending-seats">{{ res.seat_labels.join(', ') }}</span>
                <span class="pv-pending-contact" v-if="res.contact_doc_number">DNI: {{ res.contact_doc_number }} <span v-if="res.contact_phone">| {{ res.contact_phone }}</span></span>
              </div>
              <div class="pv-pending-meta">
                <span class="pv-pending-expiry">Expira: {{ new Date(res.expires_at).toLocaleString('es-PE', { day:'2-digit', month:'short', hour:'2-digit', minute:'2-digit' }) }}</span>
                <button class="pv-pending-convert" @click="openConvertModal(res)">Completar venta</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Floating action bar -->
    <Teleport to="body">
      <transition name="pv-float">
        <div v-if="selectedSeats.length > 0 && !saleSuccess && !showSaleModal" class="pv-float-bar">
          <div class="pv-float-info">
            <div class="pv-float-seats">
              <span v-for="s in selectedSeats" :key="s.id" class="pv-float-seat-tag">{{ s.label }}</span>
            </div>
            <div class="pv-float-summary">
              {{ selectedSeats.length }} asiento{{ selectedSeats.length > 1 ? 's' : '' }}
              &mdash;
              <strong>S/. {{ (pricePerSeat * selectedSeats.length).toFixed(2) }}</strong>
            </div>
          </div>
          <div class="pv-float-actions">
            <button class="pv-float-btn pv-float-btn-reserve" @click="openReserveModal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              Reservar
            </button>
            <button class="pv-float-btn" @click="showSaleModal = true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
              </svg>
              Vender
            </button>
          </div>
        </div>
      </transition>
    </Teleport>

    <!-- Sale modal -->
    <Teleport to="body">
      <div v-if="showSaleModal" class="modal-overlay" @click.self="showSaleModal = false">
        <div class="modal-content pv-sale-modal">
          <!-- Header azul con resumen -->
          <div class="pv-modal-header-blue">
            <button class="pv-modal-close-white" @click="showSaleModal = false">&times;</button>
            <h3 class="pv-modal-title-white">Completar Venta</h3>
            <div class="pv-modal-seats-chips">
              <span v-for="s in selectedSeats" :key="s.id" class="pv-modal-seat-chip">{{ s.label }}</span>
            </div>
            <div class="pv-modal-total-big">
              S/. {{ (pricePerSeat * selectedSeats.length).toFixed(2) }}
              <span class="pv-modal-total-detail">{{ selectedSeats.length }} asiento{{ selectedSeats.length > 1 ? 's' : '' }} x S/. {{ pricePerSeat.toFixed(2) }}</span>
            </div>
          </div>

          <!-- Formulario -->
          <div class="pv-modal-body">
            <div class="pv-field-row">
              <div class="pv-field" style="flex:0 0 120px">
                <label class="pv-label">Documento</label>
                <select v-model="docType" class="pv-select">
                  <option value="DNI">DNI</option>
                  <option value="RUC">RUC</option>
                  <option value="Pasaporte">Pasaporte</option>
                  <option value="CE">CE</option>
                </select>
              </div>
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Numero * <span v-if="lookingUpDoc" class="pv-lookup-badge">Consultando...</span></label>
                <input v-model="docNumber" type="text" class="pv-input" :placeholder="docType === 'RUC' ? '20123456789' : '12345678'" />
                <span v-if="docLookupError" class="pv-doc-error">{{ docLookupError }}</span>
              </div>
            </div>
            <div class="pv-field">
              <label class="pv-label">Nombre completo *</label>
              <input v-model="passengerName" type="text" class="pv-input" placeholder="Juan Perez" />
            </div>
            <div v-if="billingDocType === 'factura'" class="pv-field">
              <label class="pv-label">Direccion (factura) *</label>
              <input v-model="passengerAddress" type="text" class="pv-input" placeholder="Av. Principal 123, Lima" />
            </div>
            <div class="pv-field-row">
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Email <span class="pv-optional">(opcional)</span></label>
                <input v-model="email" type="email" class="pv-input" placeholder="correo@ejemplo.com" />
              </div>
              <div class="pv-field" style="flex:0 0 140px">
                <label class="pv-label">Telefono <span class="pv-optional">(opc)</span></label>
                <input v-model="phone" type="text" class="pv-input" placeholder="987654321" />
              </div>
            </div>

            <!-- Comprobante y precio en fila -->
            <div class="pv-field-row">
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Comprobante</label>
                <div class="pv-radio-group">
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'boleta' }">
                    <input type="radio" v-model="billingDocType" value="boleta" name="billing_doc_modal" />
                    Boleta
                  </label>
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'factura' }">
                    <input type="radio" v-model="billingDocType" value="factura" name="billing_doc_modal" />
                    Factura
                  </label>
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'pedido' }">
                    <input type="radio" v-model="billingDocType" value="pedido" name="billing_doc_modal" />
                    Pedido
                  </label>
                </div>
              </div>
              <div class="pv-field" style="flex:0 0 140px">
                <label class="pv-label">Precio/asiento</label>
                <input v-model.number="pricePerSeat" type="number" step="0.50" min="0" class="pv-input" placeholder="50.00" />
              </div>
            </div>

            <!-- Metodo de pago con cards de colores -->
            <div class="pv-payment-method">
              <label class="pv-label">Metodo de pago</label>
              <div class="pv-payment-options-v2">
                <label class="pv-pay-card pv-pay-cash" :class="{ active: paymentMethod === 'ventanilla' }">
                  <input type="radio" v-model="paymentMethod" value="ventanilla" name="payment_method_modal" />
                  <div class="pv-pay-icon">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
                  </div>
                  <strong>Efectivo</strong>
                  <span>Pago en caja</span>
                </label>
                <label
                  class="pv-pay-card pv-pay-tarjeta"
                  :class="{ active: paymentMethod === 'tarjeta', disabled: billingDocType === 'pedido' }"
                  :title="billingDocType === 'pedido' ? 'Pedido solo se emite con pago en caja' : ''"
                >
                  <input type="radio" v-model="paymentMethod" value="tarjeta" name="payment_method_modal" :disabled="billingDocType === 'pedido'" />
                  <div class="pv-pay-icon">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/></svg>
                  </div>
                  <strong>Tarjeta / Yape</strong>
                  <span>{{ billingDocType === 'pedido' ? 'No disponible para pedido' : 'Culqi checkout' }}</span>
                </label>
              </div>
            </div>

            <div v-if="errorMsg" class="pv-error">{{ errorMsg }}</div>
          </div>

          <!-- Footer con boton grande -->
          <div class="pv-modal-footer">
            <button class="pv-btn pv-btn-ghost" @click="showSaleModal = false" :disabled="selling">Cancelar</button>
            <button
              class="pv-btn pv-modal-sell-btn" :class="paymentMethod === 'tarjeta' ? 'pv-sell-tarjeta' : 'pv-sell-cash'"
              :disabled="!canSell"
              @click="processSale"
            >
              <span v-if="selling" class="pv-spinner"></span>
              <template v-else>
                <svg v-if="paymentMethod === 'ventanilla'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/></svg>
              </template>
              {{ selling ? 'Procesando...' : (paymentMethod === 'tarjeta' ? `Cobrar S/. ${(pricePerSeat * selectedSeats.length).toFixed(2)}` : `Vender S/. ${(pricePerSeat * selectedSeats.length).toFixed(2)}`) }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Reserve modal -->
    <Teleport to="body">
      <div v-if="showReserveModal" class="modal-overlay" @click.self="resetReserve">
        <div class="modal-content pv-sale-modal">
          <div class="pv-modal-header-blue" style="background: linear-gradient(135deg, #F59E0B, #D97706);">
            <button class="pv-modal-close-white" @click="resetReserve">&times;</button>
            <h3 class="pv-modal-title-white">Reservar / Separar asientos</h3>
            <div class="pv-modal-seats-chips">
              <span v-for="lbl in reserveSeatLabels" :key="lbl" class="pv-modal-seat-chip">{{ lbl }}</span>
            </div>
            <div class="pv-modal-total-big" style="font-size: 1.1rem;">
              {{ reserveSeatLabels.length }} asiento{{ reserveSeatLabels.length > 1 ? 's' : '' }}
            </div>
          </div>

          <div v-if="reserveSuccess" class="pv-modal-body" style="text-align:center;padding:2rem">
            <div style="display:inline-flex;align-items:center;justify-content:center;width:64px;height:64px;border-radius:50%;background:#FEF3C7;color:#D97706;margin-bottom:1rem">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            </div>
            <h3 style="color:#1e293b;font-size:1.2rem;font-weight:800;margin-bottom:0.5rem">Reserva creada</h3>
            <p style="color:#64748b;font-size:0.9rem;margin-bottom:1rem">Codigo de reserva</p>
            <div style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:1.5rem;font-weight:800;color:#D97706;background:#FEF3C7;padding:0.75rem 1.25rem;border-radius:12px;display:inline-block">{{ reserveCode }}</div>
            <p style="color:#64748b;font-size:0.82rem;margin-top:1rem">El pasajero debe presentar este codigo antes de la expiracion.</p>
            <button class="pv-btn pv-btn-new" style="margin-top:1.5rem" @click="resetReserve">Aceptar</button>
          </div>

          <div v-else class="pv-modal-body">
            <div class="pv-field-row">
              <div class="pv-field" style="flex:1">
                <label class="pv-label">DNI / Documento *</label>
                <input v-model="reserveDocNumber" type="text" class="pv-input" placeholder="12345678" />
              </div>
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Telefono *</label>
                <input v-model="reservePhone" type="text" class="pv-input" placeholder="987654321" />
              </div>
            </div>
            <div class="pv-field">
              <label class="pv-label">Nombre completo *</label>
              <input v-model="reserveName" type="text" class="pv-input" placeholder="Juan Perez" />
            </div>
            <div class="pv-field" style="flex:0 0 140px">
              <label class="pv-label">Horas de retencion</label>
              <input v-model.number="reserveHours" type="number" min="1" max="168" class="pv-input" />
            </div>
            <div v-if="errorMsg" class="pv-error">{{ errorMsg }}</div>
          </div>

          <div v-if="!reserveSuccess" class="pv-modal-footer">
            <button class="pv-btn pv-btn-ghost" @click="resetReserve" :disabled="reserving">Cancelar</button>
            <button class="pv-btn pv-modal-sell-btn pv-sell-reserve" :disabled="!canReserve" @click="processReserve">
              <span v-if="reserving" class="pv-spinner"></span>
              <template v-else>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              </template>
              {{ reserving ? 'Reservando...' : 'Reservar' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Convert reservation to sale modal -->
    <Teleport to="body">
      <div v-if="showConvertModal" class="modal-overlay" @click.self="showConvertModal = false">
        <div class="modal-content pv-sale-modal">
          <div class="pv-modal-header-blue">
            <button class="pv-modal-close-white" @click="showConvertModal = false">&times;</button>
            <h3 class="pv-modal-title-white">Completar venta de reserva</h3>
            <div class="pv-modal-seats-chips">
              <span v-for="lbl in convertSeatLabels" :key="lbl" class="pv-modal-seat-chip">{{ lbl }}</span>
            </div>
            <div class="pv-modal-total-big">
              S/. {{ (pricePerSeat * convertSeatLabels.length).toFixed(2) }}
              <span class="pv-modal-total-detail">{{ convertSeatLabels.length }} asiento{{ convertSeatLabels.length > 1 ? 's' : '' }} x S/. {{ pricePerSeat.toFixed(2) }}</span>
            </div>
          </div>

          <div class="pv-modal-body">
            <div class="pv-field-row">
              <div class="pv-field" style="flex:0 0 120px">
                <label class="pv-label">Documento</label>
                <select v-model="docType" class="pv-select">
                  <option value="DNI">DNI</option>
                  <option value="RUC">RUC</option>
                  <option value="Pasaporte">Pasaporte</option>
                  <option value="CE">CE</option>
                </select>
              </div>
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Numero * <span v-if="lookingUpDoc" class="pv-lookup-badge">Consultando...</span></label>
                <input v-model="docNumber" type="text" class="pv-input" :placeholder="docType === 'RUC' ? '20123456789' : '12345678'" />
                <span v-if="docLookupError" class="pv-doc-error">{{ docLookupError }}</span>
              </div>
            </div>
            <div class="pv-field">
              <label class="pv-label">Nombre completo *</label>
              <input v-model="passengerName" type="text" class="pv-input" placeholder="Juan Perez" />
            </div>
            <div v-if="billingDocType === 'factura'" class="pv-field">
              <label class="pv-label">Direccion (factura) *</label>
              <input v-model="passengerAddress" type="text" class="pv-input" placeholder="Av. Principal 123, Lima" />
            </div>
            <div class="pv-field-row">
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Email <span class="pv-optional">(opcional)</span></label>
                <input v-model="email" type="email" class="pv-input" placeholder="correo@ejemplo.com" />
              </div>
              <div class="pv-field" style="flex:0 0 140px">
                <label class="pv-label">Telefono <span class="pv-optional">(opc)</span></label>
                <input v-model="phone" type="text" class="pv-input" placeholder="987654321" />
              </div>
            </div>
            <div class="pv-field-row">
              <div class="pv-field" style="flex:1">
                <label class="pv-label">Comprobante</label>
                <div class="pv-radio-group">
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'boleta' }">
                    <input type="radio" v-model="billingDocType" value="boleta" name="billing_doc_convert" />
                    Boleta
                  </label>
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'factura' }">
                    <input type="radio" v-model="billingDocType" value="factura" name="billing_doc_convert" />
                    Factura
                  </label>
                  <label class="pv-radio-card" :class="{ active: billingDocType === 'pedido' }">
                    <input type="radio" v-model="billingDocType" value="pedido" name="billing_doc_convert" />
                    Pedido
                  </label>
                </div>
              </div>
              <div class="pv-field" style="flex:0 0 140px">
                <label class="pv-label">Precio/asiento</label>
                <input v-model.number="pricePerSeat" type="number" step="0.50" min="0" class="pv-input" placeholder="50.00" />
              </div>
            </div>
            <div class="pv-payment-method">
              <label class="pv-label">Metodo de pago</label>
              <div class="pv-payment-options-v2">
                <label class="pv-pay-card pv-pay-cash" :class="{ active: paymentMethod === 'ventanilla' }">
                  <input type="radio" v-model="paymentMethod" value="ventanilla" name="payment_method_convert" />
                  <div class="pv-pay-icon">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
                  </div>
                  <strong>Efectivo</strong>
                  <span>Pago en caja</span>
                </label>
                <label class="pv-pay-card pv-pay-tarjeta" :class="{ active: paymentMethod === 'tarjeta', disabled: billingDocType === 'pedido' }" :title="billingDocType === 'pedido' ? 'Pedido solo se emite con pago en caja' : ''">
                  <input type="radio" v-model="paymentMethod" value="tarjeta" name="payment_method_convert" :disabled="billingDocType === 'pedido'" />
                  <div class="pv-pay-icon">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/></svg>
                  </div>
                  <strong>Tarjeta / Yape</strong>
                  <span>{{ billingDocType === 'pedido' ? 'No disponible para pedido' : 'Culqi checkout' }}</span>
                </label>
              </div>
            </div>
            <div v-if="errorMsg" class="pv-error">{{ errorMsg }}</div>
          </div>

          <div class="pv-modal-footer">
            <button class="pv-btn pv-btn-ghost" @click="showConvertModal = false" :disabled="converting">Cancelar</button>
            <button class="pv-btn pv-modal-sell-btn" :class="paymentMethod === 'tarjeta' ? 'pv-sell-tarjeta' : 'pv-sell-cash'" :disabled="!canConvert" @click="processConvertSale">
              <span v-if="converting" class="pv-spinner"></span>
              <template v-else>
                <svg v-if="paymentMethod === 'ventanilla'" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/></svg>
              </template>
              {{ converting ? 'Procesando...' : (paymentMethod === 'tarjeta' ? `Cobrar S/. ${(pricePerSeat * convertSeatLabels.length).toFixed(2)}` : `Vender S/. ${(pricePerSeat * convertSeatLabels.length).toFixed(2)}`) }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Seat holder modal (admin only) -->
    <Teleport to="body">
      <div v-if="holder || holderLoading" class="pv-holder-overlay" @click.self="closeHolder">
        <div class="pv-holder-modal">
          <div class="pv-holder-head">
            <div>
              <div class="pv-holder-eyebrow">Asiento {{ holderSeatLabel }}</div>
              <h3 class="pv-holder-title">Datos del titular</h3>
            </div>
            <button class="pv-holder-close" @click="closeHolder">&times;</button>
          </div>
          <div v-if="holderLoading" class="pv-holder-loading">Cargando...</div>
          <div v-else-if="holder" class="pv-holder-body">
            <div class="pv-holder-row">
              <span class="pv-holder-label">Reserva</span>
              <strong class="pv-holder-code">{{ holder.reservation_code || '—' }}</strong>
            </div>
            <div v-if="holder.ticket_code" class="pv-holder-row">
              <span class="pv-holder-label">Boleto</span>
              <strong>{{ holder.ticket_code }}</strong>
            </div>
            <div class="pv-holder-row">
              <span class="pv-holder-label">Pasajero</span>
              <strong>{{ holder.passenger_name || '—' }}</strong>
            </div>
            <div class="pv-holder-row">
              <span class="pv-holder-label">Documento</span>
              <strong>{{ [holder.passenger_doc_type, holder.passenger_doc_number].filter(Boolean).join(' ') || '—' }}</strong>
            </div>
            <div class="pv-holder-row">
              <span class="pv-holder-label">Teléfono</span>
              <strong>{{ holder.passenger_phone || '—' }}</strong>
            </div>
            <div class="pv-holder-row">
              <span class="pv-holder-label">Email</span>
              <strong>{{ holder.passenger_email || '—' }}</strong>
            </div>
            <div class="pv-holder-row">
              <span class="pv-holder-label">Pago</span>
              <strong>{{ [holder.payment_method, holder.payment_status].filter(Boolean).join(' / ') || '—' }}</strong>
            </div>
            <div v-if="holder.billing_sale_id" class="pv-holder-row">
              <span class="pv-holder-label">Comprobante</span>
              <strong>#{{ holder.billing_sale_id }}</strong>
            </div>
          </div>
          <div v-else class="pv-holder-empty">Sin información del titular.</div>
          <div class="pv-holder-actions">
            <button v-if="holder?.billing_sale_id" class="pv-btn pv-btn-print" @click="reprintHolderTicket">
              Imprimir comprobante
            </button>
            <button v-if="holder?.reservation_code" class="pv-btn pv-btn-void" :disabled="voidingHolder" @click="voidHolder">
              {{ voidingHolder ? 'Anulando…' : (holder?.ticket_code ? 'Anular pasaje' : 'Anular reserva') }}
            </button>
            <button class="pv-btn pv-btn-ghost" @click="closeHolder">Cerrar</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.pv-page {
  max-width: 1200px;
}

.pv-header {
  margin-bottom: 1.5rem;
}

.pv-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.pv-subtitle {
  font-size: 0.88rem;
  color: var(--slate-500);
  margin-top: 0.2rem;
}

/* ── Single column layout ── */
.pv-single-col {
  max-width: 900px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding-bottom: 5rem; /* space for float bar */
}

/* ── Card ── */
.pv-card {
  background: white;
  border-radius: var(--radius-lg);
  border: 2px solid var(--slate-300);
  padding: 1.25rem;
  box-shadow: var(--shadow-sm);
}

.pv-card-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 1rem;
}

.pv-card-title svg {
  color: var(--brand-500);
  flex-shrink: 0;
}

.pv-card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.pv-card-title-row .pv-card-title {
  margin-bottom: 0;
}

/* ── Form elements ── */
.pv-field {
  margin-bottom: 0.85rem;
}

.pv-field-row {
  display: flex;
  gap: 0.75rem;
}

.pv-label {
  display: block;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-500);
  margin-bottom: 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.pv-optional {
  font-weight: 400;
  text-transform: none;
  letter-spacing: 0;
}

.pv-select,
.pv-input {
  width: 100%;
  padding: 0.7rem 2.5rem 0.7rem 0.85rem;
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.pv-select {
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='18' height='18' viewBox='0 0 24 24' fill='none' stroke='%2364748B' stroke-width='2.5' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.85rem center;
  cursor: pointer;
}

.pv-input {
  padding-right: 0.85rem;
}

.pv-select:focus,
.pv-input:focus {
  outline: none;
  border-color: var(--brand-400);
  background-color: white;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.pv-select:hover:not(:focus),
.pv-input:hover:not(:focus) {
  border-color: var(--slate-400);
}

/* ── Trip list ── */
.pv-trips {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: 260px;
  overflow-y: auto;
}

.pv-trip-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 2px solid var(--slate-200);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  cursor: pointer;
  font-family: inherit;
  font-size: 0.88rem;
  color: var(--slate-700);
  transition: all 0.15s ease;
}

.pv-trip-item:hover {
  border-color: var(--brand-300);
  background: var(--brand-50);
  box-shadow: var(--shadow-sm);
}

.pv-trip-item.active {
  border-color: var(--brand-400);
  background: linear-gradient(135deg, var(--brand-50), var(--accent-50));
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.pv-trip-time {
  font-weight: 700;
  color: var(--slate-900);
  white-space: nowrap;
}

.pv-trip-route {
  flex: 1;
  color: var(--slate-500);
  font-size: 0.82rem;
}

.pv-trip-badge {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: var(--radius-full);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  background: var(--success-100);
  color: var(--success-700);
}

.pv-trip-badge.cancelled {
  background: var(--danger-100);
  color: var(--danger-700);
}

/* ── Seat area ── */
.pv-card-seats {
  background: var(--slate-900);
  border-color: var(--slate-700);
}

.pv-card-seats .pv-card-title {
  color: white;
}

.pv-card-seats .pv-card-title svg {
  color: var(--brand-300);
}

.pv-card-seats .pv-seat-stats {
  color: var(--slate-400);
}

.pv-ws-badge {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--danger-500);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.pv-ws-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--danger-500);
  animation: pvPulse 1.5s infinite;
}

@keyframes pvPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.pv-floor-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.pv-floor-tab {
  padding: 0.4rem 1rem;
  border: 2px solid var(--slate-600);
  border-radius: var(--radius-full);
  background: var(--slate-800);
  font-size: 0.8rem;
  font-weight: 600;
  font-family: inherit;
  color: var(--slate-400);
  cursor: pointer;
  transition: all 0.2s ease;
}

.pv-floor-tab:hover {
  border-color: var(--slate-500);
  color: white;
}

.pv-floor-tab.active {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-color: var(--brand-400);
  box-shadow: 0 2px 8px rgba(96, 165, 250, 0.3);
}

.pv-seat-stats {
  display: flex;
  gap: 1.25rem;
  margin-bottom: 1rem;
  font-size: 0.8rem;
  color: var(--slate-400);
}

.pv-stat {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-weight: 500;
}

.pv-stat-dot {
  width: 12px;
  height: 12px;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}

.pv-stat-dot.available { background: linear-gradient(135deg, #34D399, #10B981); }
.pv-stat-dot.sold { background: linear-gradient(135deg, #FCA5A5, #EF4444); }
.pv-stat-dot.held { background: linear-gradient(135deg, #FCD34D, #F59E0B); }

.pv-seat-area {
  overflow-x: auto;
  padding-bottom: 0.5rem;
  display: flex;
  justify-content: center;
}

.pv-bus-frame {
  display: inline-flex;
  align-items: flex-start;
  gap: 0.75rem;
  background: var(--slate-800);
  border: 2px solid var(--slate-600);
  border-radius: var(--radius-lg);
  padding: 1rem;
  min-width: fit-content;
  box-shadow: inset 0 2px 8px rgba(0,0,0,0.2);
}

.pv-bus-front {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  align-self: center;
  color: var(--slate-500);
  opacity: 0.6;
}

.pv-seat-grid {
  display: flex;
  gap: 5px;
}

.pv-seat-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.seat {
  width: 42px;
  height: 42px;
  border-radius: 8px;
  border: 2px solid transparent;
  font-size: 0.75rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,0.15);
}

.seat.available {
  background: linear-gradient(135deg, #34D399, #10B981);
  color: white;
  border-color: #059669;
}

.seat.available:hover {
  background: linear-gradient(135deg, #6EE7B7, #34D399);
  color: white;
  transform: scale(1.12);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

.seat.selected {
  background: linear-gradient(135deg, #60A5FA, #3B82F6);
  color: white;
  border-color: #2563EB;
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.5);
  transform: scale(1.12);
}

.seat.sold {
  background: linear-gradient(135deg, #FCA5A5, #EF4444);
  color: white;
  border-color: #DC2626;
  cursor: not-allowed;
  opacity: 0.85;
}

.seat.held {
  background: linear-gradient(135deg, #FCD34D, #F59E0B);
  color: #78350F;
  border-color: #D97706;
  cursor: not-allowed;
  opacity: 0.85;
}

.seat.blocked {
  background: var(--slate-700);
  color: var(--slate-500);
  border-color: var(--slate-600);
  cursor: not-allowed;
  opacity: 0.5;
}

.pv-aisle {
  width: 42px;
  height: 16px;
}

.pv-seat-empty {
  width: 42px;
  height: 42px;
}

/* ── Selected seats summary ── */
.pv-hint {
  font-size: 0.85rem;
  color: var(--slate-500);
  text-align: center;
  padding: 1rem 0;
}

.pv-selected-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-bottom: 0.5rem;
}

.pv-seat-tag {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.6rem;
  background: var(--color-primary-light);
  color: var(--brand-700);
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  font-weight: 700;
}

.pv-selected-count {
  font-size: 0.78rem;
  color: var(--slate-500);
}

.pv-form {
  margin-bottom: 0.75rem;
}

.pv-error {
  background: var(--danger-50);
  color: var(--danger-700);
  padding: 0.55rem 0.75rem;
  border-radius: var(--radius-sm);
  font-size: 0.82rem;
  margin-bottom: 0.75rem;
}

/* ── Buttons ── */
.pv-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.7rem 1.25rem;
  font-size: 0.92rem;
  font-weight: 700;
  font-family: inherit;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s ease;
}

.pv-btn-sell {
  background: var(--brand-600);
  color: white;
  box-shadow: 0 4px 12px rgba(27, 85, 245, 0.25);
}

.pv-btn-sell:hover:not(:disabled) {
  background: var(--brand-700);
  box-shadow: 0 6px 20px rgba(27, 85, 245, 0.35);
  transform: translateY(-1px);
}

.pv-btn-sell:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pv-btn-ghost {
  background: transparent;
  color: var(--slate-600);
  border: 2px solid var(--slate-300);
  width: auto;
}

.pv-btn-ghost:hover:not(:disabled) {
  background: var(--slate-50);
  border-color: var(--slate-400);
}

.pv-btn-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pv-btn-new {
  background: var(--success-600);
  color: white;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.25);
  font-size: 1rem;
  padding: 0.85rem 1.5rem;
  margin-top: 1.25rem;
}

.pv-btn-new:hover {
  background: var(--success-700);
  transform: translateY(-1px);
}

.pv-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: pvSpin 0.6s linear infinite;
}

@keyframes pvSpin {
  to { transform: rotate(360deg); }
}

/* ── Success card ── */
.pv-success-card {
  max-width: 500px;
  margin: 2rem auto;
  background: white;
  border-radius: var(--radius-lg);
  border: 2px solid var(--success-500);
  padding: 2rem;
  text-align: center;
  box-shadow: var(--shadow-lg);
}

.pv-success-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: var(--success-100);
  color: var(--success-600);
  margin-bottom: 1rem;
}

.pv-success-title {
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--success-700);
  margin-bottom: 0.5rem;
}

.pv-success-code {
  font-size: 0.92rem;
  color: var(--slate-500);
  margin-bottom: 1rem;
}

.pv-tickets {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.pv-ticket {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.45rem 1rem;
  background: var(--success-50);
  color: var(--success-700);
  border-radius: var(--radius-full);
  font-size: 0.88rem;
  font-weight: 700;
  border: 1px solid var(--success-200);
}

/* ── Loading / empty ── */
.pv-loading,
.pv-empty {
  text-align: center;
  padding: 1.5rem 0;
  font-size: 0.88rem;
  color: var(--slate-500);
}

/* ── DNI/RUC lookup ── */
.pv-lookup-badge {
  font-weight: 400;
  text-transform: none;
  letter-spacing: 0;
  color: var(--brand-500);
  font-size: 0.72rem;
}

.pv-doc-error {
  display: block;
  font-size: 0.75rem;
  color: var(--danger-500);
  margin-top: 0.2rem;
}

/* ── Billing options ── */
.pv-billing-options {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

.pv-billing-options .pv-field {
  flex: 1;
}

.pv-radio-group {
  display: flex;
  gap: 1rem;
}

.pv-radio-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-700);
  cursor: pointer;
}

.pv-radio-label input[type="radio"] {
  accent-color: var(--brand-600);
}

.pv-total-line {
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--slate-900);
  padding: 0.5rem 0;
  border-top: 1px solid var(--color-border);
}

.pv-total-line strong {
  color: var(--brand-700);
  font-size: 1.05rem;
}

.pv-total-detail {
  font-size: 0.78rem;
  font-weight: 400;
  color: var(--slate-500);
  margin-left: 0.35rem;
}

/* ── Billing status ── */
.pv-billing-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.65rem 1rem;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  font-weight: 600;
  margin-top: 1rem;
}

.pv-billing-loading {
  background: var(--brand-50);
  color: var(--brand-700);
}

.pv-billing-ok {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-200);
}

.pv-billing-err {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-200);
}

/* ── PDF section ── */
.pv-pdf-section {
  margin-top: 1rem;
}

.pv-pdf-formats {
  display: flex;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
  justify-content: center;
}

.pv-pdf-btn {
  padding: 0.35rem 0.85rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-full);
  background: white;
  font-size: 0.78rem;
  font-weight: 600;
  font-family: inherit;
  color: var(--slate-500);
  cursor: pointer;
  transition: all 0.15s ease;
}

.pv-pdf-btn.active {
  background: var(--brand-600);
  color: white;
  border-color: var(--brand-600);
}

.pv-pdf-preview {
  width: 100%;
  height: 400px;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: white;
}

.pv-pdf-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  justify-content: center;
}

.pv-btn-print {
  background: var(--brand-600);
  color: white;
  box-shadow: 0 2px 8px rgba(27, 85, 245, 0.2);
}

.pv-btn-print:hover {
  background: var(--brand-700);
}

.pv-btn-whatsapp {
  background: #25D366;
  color: white;
  box-shadow: 0 2px 8px rgba(37, 211, 102, 0.25);
}

.pv-btn-whatsapp:hover {
  background: #1eba59;
}

/* ── Payment method ── */
.pv-payment-method {
  margin-bottom: 0.75rem;
}

.pv-payment-options {
  display: flex;
  gap: 0.5rem;
}

.pv-payment-option {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  padding: 0.5rem 0.75rem;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--slate-50);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--slate-500);
  cursor: pointer;
  transition: all 0.15s ease;
}

.pv-payment-option input[type="radio"] {
  display: none;
}

.pv-payment-option:hover {
  border-color: var(--brand-300);
}

.pv-payment-option.active {
  border-color: var(--brand-500);
  background: var(--brand-50);
  color: var(--brand-700);
}

.pv-payment-option svg {
  flex-shrink: 0;
}

.pv-btn-card {
  background: linear-gradient(135deg, var(--success-600), #047857);
  color: white;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.25);
}

.pv-btn-card:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--success-500), var(--success-600));
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.35);
  transform: translateY(-1px);
}

.pv-btn-card:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Floating action bar ── */
.pv-float-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 900;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.5rem;
  padding: 1.25rem 2rem;
  background: linear-gradient(135deg, #0F172A 0%, #1E293B 100%);
  border-top: 3px solid var(--brand-400);
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.35);
}

.pv-float-info {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
}

.pv-float-seats {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.pv-float-seat-tag {
  display: inline-flex;
  align-items: center;
  padding: 0.3rem 0.7rem;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-radius: var(--radius-full);
  font-size: 0.78rem;
  font-weight: 700;
  box-shadow: 0 2px 6px rgba(96, 165, 250, 0.3);
}

.pv-float-summary {
  font-size: 1rem;
  font-weight: 600;
  color: var(--slate-300);
  white-space: nowrap;
}

.pv-float-summary strong {
  color: white;
  font-size: 1.25rem;
}

.pv-float-btn {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.9rem 2rem;
  background: linear-gradient(135deg, #10B981, #059669);
  color: white;
  border: none;
  border-radius: var(--radius-lg);
  font-size: 1.05rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s ease;
  box-shadow: 0 4px 16px rgba(16, 185, 129, 0.4);
  animation: pvBtnPulse 2s ease-in-out infinite;
}

.pv-float-btn:hover {
  background: linear-gradient(135deg, #34D399, #10B981);
  box-shadow: 0 6px 24px rgba(16, 185, 129, 0.5);
  transform: translateY(-2px);
}

@keyframes pvBtnPulse {
  0%, 100% { box-shadow: 0 4px 16px rgba(16, 185, 129, 0.4); }
  50% { box-shadow: 0 4px 24px rgba(16, 185, 129, 0.6); }
}

/* Float bar enter/leave transition */
.pv-float-enter-active {
  transition: all 0.3s ease;
}

.pv-float-leave-active {
  transition: all 0.2s ease;
}

.pv-float-enter-from,
.pv-float-leave-to {
  transform: translateY(100%);
  opacity: 0;
}

/* ── Modal (same pattern as RoutesAdminView) ── */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background: white;
  border-radius: var(--radius-xl);
  padding: 2rem;
  width: 100%;
  max-width: 520px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid var(--slate-200);
  animation: modalIn 0.2s ease;
  max-height: 90vh;
  overflow-y: auto;
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.95) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--slate-900);
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--slate-500);
  cursor: pointer;
  padding: 0.25rem;
  line-height: 1;
}

.modal-close:hover {
  color: var(--slate-900);
}

/* ── Sale modal premium ── */
.pv-sale-modal {
  padding: 0 !important;
  max-width: 540px !important;
  overflow: hidden !important;
}

.pv-modal-header-blue {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-600));
  padding: 1.5rem 1.75rem 1.25rem;
  position: relative;
  color: white;
}

.pv-modal-close-white {
  position: absolute;
  top: 0.75rem;
  right: 1rem;
  background: none;
  border: none;
  font-size: 1.75rem;
  color: rgba(255,255,255,0.7);
  cursor: pointer;
  line-height: 1;
}

.pv-modal-close-white:hover { color: white; }

.pv-modal-title-white {
  font-size: 1.2rem;
  font-weight: 800;
  margin-bottom: 0.75rem;
}

.pv-modal-seats-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-bottom: 0.65rem;
}

.pv-modal-seat-chip {
  padding: 0.25rem 0.65rem;
  background: rgba(255,255,255,0.2);
  border: 1px solid rgba(255,255,255,0.3);
  border-radius: var(--radius-full);
  font-size: 0.8rem;
  font-weight: 700;
}

.pv-modal-total-big {
  font-size: 1.75rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.pv-modal-total-detail {
  display: block;
  font-size: 0.82rem;
  font-weight: 500;
  opacity: 0.75;
  margin-top: 0.15rem;
}

.pv-modal-body {
  padding: 1.25rem 1.75rem;
  max-height: 50vh;
  overflow-y: auto;
}

/* ── Payment cards v2 ── */
.pv-payment-options-v2 {
  display: flex;
  gap: 0.65rem;
}

.pv-pay-card {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.85rem;
  border: 2px solid var(--slate-200);
  border-radius: var(--radius-md);
  background: white;
  cursor: pointer;
  transition: all 0.2s ease;
}

.pv-pay-card input[type="radio"] { display: none; }
.pv-pay-card.disabled {
  opacity: 0.55;
  cursor: not-allowed;
  filter: grayscale(0.25);
}
.pv-pay-card.disabled:hover {
  transform: none;
  box-shadow: none;
  border-color: var(--slate-200);
}

.pv-pay-card strong {
  font-size: 0.82rem;
  color: var(--slate-700);
}

.pv-pay-card span {
  font-size: 0.68rem;
  color: var(--slate-400);
  font-weight: 500;
}

.pv-pay-icon {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.pv-pay-icon svg {
  width: 18px;
  height: 18px;
}

/* Efectivo - verde */
.pv-pay-cash .pv-pay-icon {
  background: var(--accent-50);
  color: var(--accent-600);
}
.pv-pay-cash:hover {
  border-color: var(--accent-300);
}
.pv-pay-cash.active {
  border-color: var(--accent-500);
  background: var(--accent-50);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.15);
}
.pv-pay-cash.active .pv-pay-icon {
  background: linear-gradient(135deg, var(--accent-400), var(--accent-500));
  color: white;
  box-shadow: 0 3px 10px rgba(16, 185, 129, 0.3);
}
.pv-pay-cash.active strong { color: var(--accent-700); }

/* Tarjeta - naranja */
.pv-pay-tarjeta .pv-pay-icon {
  background: #FFF7ED;
  color: #EA580C;
}
.pv-pay-tarjeta:hover {
  border-color: #FDBA74;
}
.pv-pay-tarjeta.active {
  border-color: #F97316;
  background: #FFF7ED;
  box-shadow: 0 0 0 3px rgba(249, 115, 22, 0.15);
}
.pv-pay-tarjeta.active .pv-pay-icon {
  background: linear-gradient(135deg, #FB923C, #F97316);
  color: white;
  box-shadow: 0 3px 10px rgba(249, 115, 22, 0.3);
}
.pv-pay-tarjeta.active strong { color: #C2410C; }

/* ── Radio cards (boleta/factura) ── */
.pv-radio-card {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 0.75rem;
  border: 2px solid var(--slate-200);
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-500);
  cursor: pointer;
  transition: all 0.2s ease;
  background: white;
}
.pv-radio-card input[type="radio"] { display: none; }
.pv-radio-card:hover { border-color: var(--brand-300); }
.pv-radio-card.active {
  border-color: var(--brand-400);
  background: var(--brand-50);
  color: var(--brand-700);
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.12);
}

/* ── Modal footer ── */
.pv-modal-footer {
  padding: 1rem 1.75rem;
  border-top: 1px solid var(--slate-200);
  display: flex;
  gap: 0.75rem;
  align-items: center;
  background: var(--slate-50);
}

.pv-modal-sell-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.85rem 1.5rem;
  font-size: 1rem;
  font-weight: 700;
  border: none;
  border-radius: var(--radius-md);
  color: white;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.pv-sell-cash {
  background: linear-gradient(135deg, #10B981, #059669);
  box-shadow: 0 4px 14px rgba(16, 185, 129, 0.35);
}
.pv-sell-cash:hover:not(:disabled) {
  background: linear-gradient(135deg, #34D399, #10B981);
  box-shadow: 0 6px 20px rgba(16, 185, 129, 0.45);
  transform: translateY(-1px);
}

.pv-sell-tarjeta {
  background: linear-gradient(135deg, #FB923C, #F97316);
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.35);
}
.pv-sell-tarjeta:hover:not(:disabled) {
  background: linear-gradient(135deg, #FDBA74, #FB923C);
  box-shadow: 0 6px 20px rgba(249, 115, 22, 0.45);
  transform: translateY(-1px);
}

.pv-modal-sell-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none !important;
}

/* ── Responsive ── */
@media (max-width: 640px) {
  .pv-billing-options {
    flex-direction: column;
  }

  .pv-float-bar {
    flex-direction: column;
    gap: 0.65rem;
    padding: 0.75rem 1rem;
  }

  .pv-float-btn {
    width: 100%;
    justify-content: center;
  }

  .modal-content {
    padding: 1.25rem;
    max-width: 100%;
  }
}

/* ── Seat holder modal ── */
.pv-holder-overlay {
  position: fixed; inset: 0; background: rgba(15, 29, 78, 0.55);
  display: flex; align-items: center; justify-content: center; z-index: 9999;
  padding: 1rem;
}
.pv-holder-modal {
  background: white; border-radius: 16px; max-width: 460px; width: 100%;
  box-shadow: 0 24px 48px rgba(15, 29, 78, 0.25);
  overflow: hidden;
}
.pv-holder-head {
  display: flex; justify-content: space-between; align-items: flex-start;
  padding: 1.25rem 1.25rem 0.5rem; gap: 1rem;
}
.pv-holder-eyebrow { font-size: 0.78rem; color: #64748b; font-weight: 600; }
.pv-holder-title { margin: 0.2rem 0 0; font-size: 1.15rem; font-weight: 800; color: #0f1d4e; }
.pv-holder-close {
  background: transparent; border: none; font-size: 1.6rem; line-height: 1;
  color: #94a3b8; cursor: pointer; padding: 0 0.25rem;
}
.pv-holder-close:hover { color: #1e293b; }
.pv-holder-loading, .pv-holder-empty {
  padding: 2rem 1.25rem; text-align: center; color: #64748b; font-size: 0.95rem;
}
.pv-holder-body { padding: 0.5rem 1.25rem 0.25rem; }
.pv-holder-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.55rem 0; border-bottom: 1px solid #e2e8f0; gap: 0.75rem;
}
.pv-holder-row:last-child { border-bottom: none; }
.pv-holder-label { color: #64748b; font-size: 0.85rem; flex-shrink: 0; }
.pv-holder-row strong {
  color: #1e293b; font-weight: 700; font-size: 0.92rem; text-align: right;
  word-break: break-word;
}
.pv-holder-code { color: #1b55f5 !important; font-family: monospace; letter-spacing: 1px; }
.pv-holder-actions {
  display: flex; justify-content: space-between; gap: 0.5rem; padding: 0.75rem 1.25rem 1.25rem;
}
.pv-btn-void {
  background: #dc2626; color: white; border: none; padding: 0.55rem 1rem;
  border-radius: 8px; font-weight: 700; font-size: 0.88rem; cursor: pointer;
}
.pv-btn-void:hover { background: #b91c1c; }
.pv-btn-void:disabled { opacity: 0.6; cursor: not-allowed; }

/* ── Big seat banner on success card ── */
.pv-seat-banner {
  margin: 0.5rem 0 1rem;
  display: flex; flex-direction: column; align-items: center; gap: 0.4rem;
}
.pv-seat-label {
  font-size: 0.78rem; color: #64748b; font-weight: 700; letter-spacing: 1.5px;
}
.pv-seat-row {
  display: flex; flex-wrap: wrap; gap: 0.6rem; justify-content: center;
}
.pv-seat-big {
  min-width: 84px; padding: 0.6rem 1.2rem;
  background: linear-gradient(135deg, #1b55f5, #0f1d4e);
  color: white; border-radius: 12px;
  font-size: 3rem; font-weight: 900; letter-spacing: 2px;
  line-height: 1; text-align: center;
  box-shadow: 0 6px 18px rgba(27, 85, 245, 0.32);
}

/* ── Float bar actions ── */
.pv-float-actions {
  display: flex;
  gap: 0.75rem;
}
.pv-float-btn-reserve {
  background: linear-gradient(135deg, #F59E0B, #D97706) !important;
  box-shadow: 0 4px 16px rgba(245, 158, 11, 0.4) !important;
}
.pv-float-btn-reserve:hover {
  background: linear-gradient(135deg, #FBBF24, #F59E0B) !important;
  box-shadow: 0 6px 24px rgba(245, 158, 11, 0.5) !important;
}

/* ── Pending reservations panel ── */
.pv-pending-reservations {
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--slate-700);
}
.pv-pending-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--slate-300);
  margin-bottom: 0.75rem;
}
.pv-pending-title svg { color: var(--brand-300); }
.pv-pending-loading, .pv-pending-empty {
  font-size: 0.82rem;
  color: var(--slate-500);
  padding: 0.5rem 0;
}
.pv-pending-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.pv-pending-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  background: var(--slate-800);
  border: 1px solid var(--slate-600);
  border-radius: var(--radius-md);
  padding: 0.65rem 0.85rem;
}
.pv-pending-main {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
}
.pv-pending-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85rem;
  color: #FBBF24;
}
.pv-pending-seats {
  font-size: 0.8rem;
  color: var(--slate-300);
  font-weight: 600;
}
.pv-pending-contact {
  font-size: 0.75rem;
  color: var(--slate-400);
}
.pv-pending-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.35rem;
  flex-shrink: 0;
}
.pv-pending-expiry {
  font-size: 0.72rem;
  color: var(--slate-400);
  white-space: nowrap;
}
.pv-pending-convert {
  padding: 0.4rem 0.75rem;
  background: linear-gradient(135deg, #10B981, #059669);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 0.78rem;
  font-weight: 700;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
}
.pv-pending-convert:hover {
  background: linear-gradient(135deg, #34D399, #10B981);
}

/* ── Reserve modal sell button ── */
.pv-sell-reserve {
  background: linear-gradient(135deg, #F59E0B, #D97706) !important;
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.25) !important;
}
.pv-sell-reserve:hover:not(:disabled) {
  background: linear-gradient(135deg, #FBBF24, #F59E0B) !important;
  box-shadow: 0 6px 20px rgba(245, 158, 11, 0.35) !important;
}

@media (max-width: 640px) {
  .pv-float-actions {
    flex-direction: column;
    width: 100%;
  }
  .pv-pending-item {
    flex-direction: column;
    align-items: flex-start;
  }
  .pv-pending-meta {
    align-items: flex-start;
    width: 100%;
    flex-direction: row;
    justify-content: space-between;
  }
}
</style>
