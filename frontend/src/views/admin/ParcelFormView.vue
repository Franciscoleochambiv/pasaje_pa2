<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  createParcel,
  getParcel,
  updateParcel,
  getTripInstances,
  getStops,
  getSegments,
  getBillingPDFUrl,
  lookupDNI,
  lookupRUC,
  retryParcelBilling,
  type TripInstance,
  type Stop,
  type RouteSegment,
  type Parcel,
} from '../../api/client'

const route = useRoute()
const router = useRouter()

// ── Edit vs Create ──
const parcelId = computed(() => {
  const raw = route.params.id
  const id = Array.isArray(raw) ? raw[0] : raw
  if (!id) return null
  const num = Number(id)
  return Number.isFinite(num) && num > 0 ? num : null
})
const isEditing = computed(() => parcelId.value !== null)

// ── Loading / Error ──
const loading = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

// ── Print panel (after create) ──
const showPrintPanel = ref(false)
const printSaleId = ref<number | null>(null)
const printParcelCode = ref('')
const printDocLabel = ref('')
const printWhatsappUrl = ref('')
const printPdfFormat = ref('ticket')
const printBillingError = ref('')
const printRetrying = ref(false)

function openPrintPDF(_format: string) {
  // Imprimir directamente el iframe con el formato seleccionado
  const iframe = document.querySelector('.print-preview') as HTMLIFrameElement
  if (iframe?.contentWindow) {
    iframe.contentWindow.print()
  } else if (printSaleId.value) {
    window.open(getBillingPDFUrl(printSaleId.value, printPdfFormat.value), '_blank')
  }
}

function openPrintWhatsApp() {
  if (printWhatsappUrl.value) {
    window.open(printWhatsappUrl.value, '_blank')
  }
}

function closePrintPanel() {
  showPrintPanel.value = false
  const parcel = createdParcel.value
  if (parcel) {
    router.push({ name: 'admin-encomienda-detalle', params: { id: parcel.id } })
  }
}

async function retryBilling() {
  const parcel = createdParcel.value
  if (!parcel) return
  printRetrying.value = true
  try {
    const result = await retryParcelBilling(parcel.id)
    const updated = result.parcel || parcel
    createdParcel.value = updated
    printSaleId.value = updated.billing_sale_id || null
    printWhatsappUrl.value = result.whatsapp_url || ''
    printBillingError.value = result.billing_error || ''
  } catch (e) {
    printBillingError.value = (e as Error).message
  } finally {
    printRetrying.value = false
  }
}

const createdParcel = ref<Parcel | null>(null)

// ── Viaje ──
const tripInstances = ref<TripInstance[]>([])
const loadingTrips = ref(false)
const selectedTripId = ref<number | null>(null)
const stops = ref<Stop[]>([])
const loadingStops = ref(false)
const segments = ref<RouteSegment[]>([])
const originStopId = ref<number | null>(null)
const destStopId = ref<number | null>(null)

const skipTripWatch = ref(false)

const selectedTrip = computed(() =>
  tripInstances.value.find(t => t.id === selectedTripId.value) ?? null
)

const filteredDestStops = computed(() => {
  if (!originStopId.value) return stops.value
  const originStop = stops.value.find(s => s.id === originStopId.value)
  if (!originStop) return stops.value
  return stops.value.filter(s => s.position > originStop.position)
})

// ── Remitente ──
const senderDocType = ref('DNI')
const senderDocNumber = ref('')
const senderName = ref('')
const senderPhone = ref('')
const lookingUpSender = ref(false)
const senderLookupError = ref('')

// ── Destinatario ──
const receiverDocType = ref('DNI')
const receiverDocNumber = ref('')
const receiverName = ref('')
const receiverPhone = ref('')
const lookingUpReceiver = ref(false)
const receiverLookupError = ref('')

// ── Carga ──
const packageCount = ref(1)
const weightKg = ref<number | null>(null)
const description = ref('')

// ── Facturacion ──
const billingDocType = ref<'boleta' | 'factura' | 'pedido'>('pedido')
const billingEmail = ref('')
const billingRuc = ref('')
const billingRazonSocial = ref('')
const billingAddress = ref('')
const lookingUpBillingRuc = ref(false)
const billingRucError = ref('')

// ── Pago ──
const amountSoles = ref<number>(0)
const paymentMode = ref('origin')
const paymentMethod = ref('efectivo')

// ── Format helpers ──
function formatDeparture(dt: string): string {
  const d = new Date(dt)
  return d.toLocaleDateString('es-PE', {
    weekday: 'short', day: '2-digit', month: 'short',
  }) + ' ' + d.toLocaleTimeString('es-PE', {
    hour: '2-digit', minute: '2-digit', hour12: false,
  })
}

const submitLabel = computed(() => {
  if (isEditing.value) return 'Guardar Cambios'
  if (paymentMode.value === 'origin' && paymentMethod.value) return 'Registrar y Cobrar'
  return 'Registrar'
})

// ── Load trip instances ──
async function loadTrips() {
  loadingTrips.value = true
  try {
    tripInstances.value = await getTripInstances()
  } catch (e) {
    errorMsg.value = 'Error al cargar viajes'
  } finally {
    loadingTrips.value = false
  }
}

// ── When trip changes, load stops and segments ──
watch(selectedTripId, async (tripId) => {
  if (skipTripWatch.value) {
    skipTripWatch.value = false
    return
  }
  stops.value = []
  segments.value = []
  originStopId.value = null
  destStopId.value = null
  if (!tripId) return
  const trip = tripInstances.value.find(t => t.id === tripId)
  if (!trip) return
  loadingStops.value = true
  try {
    const [s, seg] = await Promise.all([
      getStops(trip.route_id),
      getSegments(trip.route_id),
    ])
    stops.value = s
    segments.value = seg
  } catch (e) {
    errorMsg.value = 'Error al cargar paradas'
  } finally {
    loadingStops.value = false
  }
})

// ── When origin + dest selected, auto-fill price from segment ──
watch([originStopId, destStopId], ([origin, dest]) => {
  if (!origin || !dest) return
  const seg = segments.value.find(
    s => s.origin_stop_id === origin && s.dest_stop_id === dest
  )
  if (seg) {
    amountSoles.value = seg.price
  }
})

// ── Sender doc lookup ──
watch(senderDocNumber, async (val) => {
  const num = val.trim()
  senderLookupError.value = ''
  if (senderDocType.value === 'DNI' && num.length === 8) {
    lookingUpSender.value = true
    try {
      const result = await lookupDNI(num)
      if (result.nombre_completo) {
        senderName.value = result.nombre_completo
      } else if (result.nombres) {
        senderName.value = `${result.nombres} ${result.apellido_paterno} ${result.apellido_materno}`.trim()
      }
    } catch {
      senderLookupError.value = 'No se pudo consultar el DNI'
    } finally {
      lookingUpSender.value = false
    }
  } else if (senderDocType.value === 'RUC' && num.length === 11) {
    lookingUpSender.value = true
    try {
      const result = await lookupRUC(num)
      senderName.value = result.nombre_o_razon_social || result.razon_social || ''
    } catch {
      senderLookupError.value = 'No se pudo consultar el RUC'
    } finally {
      lookingUpSender.value = false
    }
  }
})

// ── Receiver doc lookup ──
watch(receiverDocNumber, async (val) => {
  const num = val.trim()
  receiverLookupError.value = ''
  if (receiverDocType.value === 'DNI' && num.length === 8) {
    lookingUpReceiver.value = true
    try {
      const result = await lookupDNI(num)
      if (result.nombre_completo) {
        receiverName.value = result.nombre_completo
      } else if (result.nombres) {
        receiverName.value = `${result.nombres} ${result.apellido_paterno} ${result.apellido_materno}`.trim()
      }
    } catch {
      receiverLookupError.value = 'No se pudo consultar el DNI'
    } finally {
      lookingUpReceiver.value = false
    }
  } else if (receiverDocType.value === 'RUC' && num.length === 11) {
    lookingUpReceiver.value = true
    try {
      const result = await lookupRUC(num)
      receiverName.value = result.nombre_o_razon_social || result.razon_social || ''
    } catch {
      receiverLookupError.value = 'No se pudo consultar el RUC'
    } finally {
      lookingUpReceiver.value = false
    }
  }
})

// ── Billing RUC lookup ──
watch(billingRuc, async (val) => {
  const num = val.trim()
  billingRucError.value = ''
  if (num.length === 11) {
    lookingUpBillingRuc.value = true
    try {
      const result = await lookupRUC(num)
      billingRazonSocial.value = result.nombre_o_razon_social || result.razon_social || ''
      billingAddress.value = result.direccion || ''
    } catch {
      billingRucError.value = 'No se pudo consultar el RUC'
    } finally {
      lookingUpBillingRuc.value = false
    }
  }
})

// ── Load existing parcel for editing ──
async function loadParcel() {
  if (!parcelId.value) return
  loading.value = true
  errorMsg.value = ''
  try {
    const { parcel } = await getParcel(parcelId.value)
    // Cargar stops antes de asignar trip para evitar que el watcher los borre
    const trip = tripInstances.value.find(t => t.id === parcel.trip_instance_id)
    if (trip) {
      try {
        const [s, seg] = await Promise.all([
          getStops(trip.route_id),
          getSegments(trip.route_id),
        ])
        stops.value = s
        segments.value = seg
      } catch { /* se manejan abajo */ }
    }
    skipTripWatch.value = true
    selectedTripId.value = parcel.trip_instance_id
    originStopId.value = parcel.origin_stop_id
    destStopId.value = parcel.dest_stop_id
    senderDocType.value = parcel.sender_doc_type || 'DNI'
    senderDocNumber.value = parcel.sender_doc_number || ''
    senderName.value = parcel.sender_name || ''
    senderPhone.value = parcel.sender_phone || ''
    receiverDocType.value = parcel.receiver_doc_type || 'DNI'
    receiverDocNumber.value = parcel.receiver_doc_number || ''
    receiverName.value = parcel.receiver_name || ''
    receiverPhone.value = parcel.receiver_phone || ''
    packageCount.value = parcel.package_count || 1
    weightKg.value = parcel.weight_kg || null
    description.value = parcel.description || ''
    billingDocType.value = (parcel.billing_doc_type as 'boleta' | 'factura' | 'pedido') || 'pedido'
    billingEmail.value = parcel.billing_email || ''
    billingRuc.value = parcel.billing_ruc || ''
    billingRazonSocial.value = parcel.billing_razon_social || ''
    billingAddress.value = parcel.billing_address || ''
    amountSoles.value = parcel.amount_cents / 100
    paymentMode.value = parcel.payment_mode || 'origin'
    paymentMethod.value = parcel.payment_method || 'efectivo'
  } catch (e) {
    errorMsg.value = 'Error al cargar la encomienda'
  } finally {
    loading.value = false
  }
}

// ── Submit ──
async function submit() {
  errorMsg.value = ''
  successMsg.value = ''

  // Basic validation
  if (!selectedTripId.value) { errorMsg.value = 'Seleccione un viaje'; return }
  if (!originStopId.value) { errorMsg.value = 'Seleccione parada de origen'; return }
  if (!destStopId.value) { errorMsg.value = 'Seleccione parada de destino'; return }
  if (!senderName.value.trim()) { errorMsg.value = 'Ingrese nombre del remitente'; return }
  if (!senderDocNumber.value.trim()) { errorMsg.value = 'Ingrese documento del remitente'; return }
  if (!receiverName.value.trim()) { errorMsg.value = 'Ingrese nombre del destinatario'; return }
  if (!receiverDocNumber.value.trim()) { errorMsg.value = 'Ingrese documento del destinatario'; return }
  if (!receiverPhone.value.trim()) { errorMsg.value = 'Ingrese telefono del destinatario'; return }
  if (!billingEmail.value.trim()) { errorMsg.value = 'Ingrese email de facturacion'; return }
  if (amountSoles.value <= 0) { errorMsg.value = 'El monto debe ser mayor a 0'; return }
  if (billingDocType.value === 'factura' && !billingRuc.value.trim()) { errorMsg.value = 'Ingrese RUC para factura'; return }

  const payload = {
    trip_instance_id: selectedTripId.value,
    origin_stop_id: originStopId.value,
    dest_stop_id: destStopId.value,
    sender_name: senderName.value.trim(),
    sender_doc_type: senderDocType.value,
    sender_doc_number: senderDocNumber.value.trim(),
    sender_phone: senderPhone.value.trim() || undefined,
    receiver_name: receiverName.value.trim(),
    receiver_doc_type: receiverDocType.value,
    receiver_doc_number: receiverDocNumber.value.trim(),
    receiver_phone: receiverPhone.value.trim(),
    package_count: packageCount.value,
    weight_kg: weightKg.value ?? undefined,
    description: description.value.trim() || undefined,
    amount_cents: Math.round(amountSoles.value * 100),
    payment_mode: paymentMode.value,
    payment_method: paymentMode.value === 'origin' ? paymentMethod.value : undefined,
    billing_doc_type: billingDocType.value,
    billing_email: billingEmail.value.trim(),
    billing_ruc: billingDocType.value === 'factura' ? billingRuc.value.trim() : undefined,
    billing_razon_social: billingDocType.value === 'factura' ? billingRazonSocial.value.trim() : undefined,
    billing_address: billingDocType.value === 'factura' ? billingAddress.value.trim() : undefined,
  }

  saving.value = true
  try {
    if (isEditing.value && parcelId.value) {
      await updateParcel(parcelId.value, payload)
      successMsg.value = 'Encomienda actualizada'
      setTimeout(() => {
        router.push({ name: 'admin-encomienda-detalle', params: { id: parcelId.value! } })
      }, 1000)
    } else {
      const result = await createParcel(payload)
      const parcel = result.parcel || (result as any)
      createdParcel.value = parcel

      // Abrir el panel de impresión SIEMPRE (haya billing_sale_id o haya billing_error).
      // Si solo hay error, el panel muestra el mensaje + botón "Reintentar emisión".
      printSaleId.value = parcel.billing_sale_id || null
      printParcelCode.value = parcel.code
      printDocLabel.value = payload.payment_mode === 'destination' ? 'Pedido generado' : 'Comprobante emitido'
      printWhatsappUrl.value = result.whatsapp_url || ''
      printPdfFormat.value = 'ticket'
      printBillingError.value = result.billing_error || ''
      showPrintPanel.value = true
      successMsg.value = 'Encomienda registrada'
    }
  } catch (e) {
    errorMsg.value = (e as Error).message || 'Error al guardar'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadTrips()
  if (isEditing.value) {
    await loadParcel()
  }
})
</script>

<template>
  <div class="parcel-form-page">
    <div class="page-header">
      <button class="btn-back" @click="router.push({ name: 'admin-encomiendas' })">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
        Volver
      </button>
      <div>
        <h1>{{ isEditing ? 'Editar Encomienda' : 'Nueva Encomienda' }}</h1>
        <p>{{ isEditing ? 'Modifique los datos de la encomienda' : 'Complete los datos para registrar una encomienda' }}</p>
      </div>
    </div>

    <!-- Loading overlay -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Cargando datos...</p>
    </div>

    <form v-else @submit.prevent="submit" class="form-sections">

      <!-- ── Viaje ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>
          Viaje
        </h2>
        <div class="field-grid">
          <div class="field full">
            <label>Viaje *</label>
            <select v-model="selectedTripId" :disabled="loadingTrips">
              <option :value="null" disabled>{{ loadingTrips ? 'Cargando viajes...' : 'Seleccione un viaje' }}</option>
              <option v-for="trip in tripInstances" :key="trip.id" :value="trip.id">
                {{ trip.route_name }} - {{ formatDeparture(trip.departure_at) }}
              </option>
            </select>
          </div>
          <div class="field">
            <label>Origen *</label>
            <select v-model="originStopId" :disabled="!selectedTripId || loadingStops">
              <option :value="null" disabled>{{ loadingStops ? 'Cargando...' : 'Seleccione origen' }}</option>
              <option v-for="stop in stops" :key="stop.id" :value="stop.id">
                {{ stop.name }}
              </option>
            </select>
          </div>
          <div class="field">
            <label>Destino *</label>
            <select v-model="destStopId" :disabled="!originStopId || loadingStops">
              <option :value="null" disabled>Seleccione destino</option>
              <option v-for="stop in filteredDestStops" :key="stop.id" :value="stop.id">
                {{ stop.name }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- ── Remitente ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          Remitente
        </h2>
        <div class="field-grid">
          <div class="field">
            <label>Tipo Documento</label>
            <select v-model="senderDocType">
              <option value="DNI">DNI</option>
              <option value="RUC">RUC</option>
            </select>
          </div>
          <div class="field">
            <label>N. Documento *</label>
            <div class="input-with-status">
              <input v-model="senderDocNumber" :placeholder="senderDocType === 'DNI' ? '12345678' : '20123456789'" :maxlength="senderDocType === 'DNI' ? 8 : 11" />
              <span v-if="lookingUpSender" class="lookup-spinner"></span>
            </div>
            <span v-if="senderLookupError" class="field-error">{{ senderLookupError }}</span>
          </div>
          <div class="field">
            <label>Nombre *</label>
            <input v-model="senderName" placeholder="Nombre completo" />
          </div>
          <div class="field">
            <label>Telefono</label>
            <input v-model="senderPhone" placeholder="987654321" />
          </div>
        </div>
      </div>

      <!-- ── Destinatario ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          Destinatario
        </h2>
        <div class="field-grid">
          <div class="field">
            <label>Tipo Documento</label>
            <select v-model="receiverDocType">
              <option value="DNI">DNI</option>
              <option value="RUC">RUC</option>
            </select>
          </div>
          <div class="field">
            <label>N. Documento *</label>
            <div class="input-with-status">
              <input v-model="receiverDocNumber" :placeholder="receiverDocType === 'DNI' ? '12345678' : '20123456789'" :maxlength="receiverDocType === 'DNI' ? 8 : 11" />
              <span v-if="lookingUpReceiver" class="lookup-spinner"></span>
            </div>
            <span v-if="receiverLookupError" class="field-error">{{ receiverLookupError }}</span>
          </div>
          <div class="field">
            <label>Nombre *</label>
            <input v-model="receiverName" placeholder="Nombre completo" />
          </div>
          <div class="field">
            <label>Telefono *</label>
            <input v-model="receiverPhone" placeholder="987654321" />
          </div>
        </div>
      </div>

      <!-- ── Carga ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m7.5 4.27 9 5.15"/><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/></svg>
          Carga
        </h2>
        <div class="field-grid">
          <div class="field">
            <label>Cantidad de Bultos *</label>
            <input v-model.number="packageCount" type="number" min="1" />
          </div>
          <div class="field">
            <label>Peso (kg)</label>
            <input v-model.number="weightKg" type="number" min="0" step="0.1" placeholder="Opcional" />
          </div>
          <div class="field full">
            <label>Descripcion</label>
            <textarea v-model="description" rows="3" placeholder="Descripcion del contenido..."></textarea>
          </div>
        </div>
      </div>

      <!-- ── Facturacion ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" x2="8" y1="13" y2="13"/><line x1="16" x2="8" y1="17" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
          Facturacion
        </h2>
        <div class="field-grid">
          <div class="field full">
            <label>Tipo Comprobante</label>
            <div class="radio-group">
              <label class="radio-label" :class="{ active: billingDocType === 'boleta' }">
                <input type="radio" v-model="billingDocType" value="boleta" />
                Boleta
              </label>
              <label class="radio-label" :class="{ active: billingDocType === 'factura' }">
                <input type="radio" v-model="billingDocType" value="factura" />
                Factura
              </label>
              <label class="radio-label" :class="{ active: billingDocType === 'pedido' }">
                <input type="radio" v-model="billingDocType" value="pedido" />
                Pedido
              </label>
            </div>
          </div>
          <div class="field">
            <label>Email *</label>
            <input v-model="billingEmail" type="email" placeholder="correo@ejemplo.com" />
          </div>
          <template v-if="billingDocType === 'factura'">
            <div class="field">
              <label>RUC *</label>
              <div class="input-with-status">
                <input v-model="billingRuc" placeholder="20123456789" maxlength="11" />
                <span v-if="lookingUpBillingRuc" class="lookup-spinner"></span>
              </div>
              <span v-if="billingRucError" class="field-error">{{ billingRucError }}</span>
            </div>
            <div class="field">
              <label>Razon Social</label>
              <input v-model="billingRazonSocial" placeholder="Auto-completado desde RUC" />
            </div>
            <div class="field">
              <label>Direccion</label>
              <input v-model="billingAddress" placeholder="Auto-completado desde RUC" />
            </div>
          </template>
        </div>
      </div>

      <!-- ── Pago ── -->
      <div class="form-section">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="1" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
          Pago
        </h2>
        <div class="field-grid">
          <div class="field">
            <label>Monto (S/) *</label>
            <input v-model.number="amountSoles" type="number" min="0" step="0.50" />
          </div>
          <div class="field">
            <label>Modalidad de Pago</label>
            <div class="radio-group">
              <label class="radio-label" :class="{ active: paymentMode === 'origin' }">
                <input type="radio" v-model="paymentMode" value="origin" />
                Pago en Origen
              </label>
              <label class="radio-label" :class="{ active: paymentMode === 'destination' }">
                <input type="radio" v-model="paymentMode" value="destination" />
                Pago en Destino
              </label>
            </div>
          </div>
          <div class="field" v-if="paymentMode === 'origin'">
            <label>Metodo de Pago</label>
            <select v-model="paymentMethod">
              <option value="efectivo">Efectivo</option>
              <option value="yape">Yape</option>
              <option value="tarjeta">Tarjeta</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Messages -->
      <div v-if="errorMsg" class="msg msg-error">{{ errorMsg }}</div>
      <div v-if="successMsg" class="msg msg-success">{{ successMsg }}</div>

      <!-- Submit -->
      <div class="form-actions">
        <button type="button" class="btn-cancel" @click="router.push({ name: 'admin-encomiendas' })">
          Cancelar
        </button>
        <button type="submit" class="btn-submit" :disabled="saving">
          <span v-if="saving" class="spinner-sm"></span>
          {{ saving ? 'Guardando...' : submitLabel }}
        </button>
      </div>
    </form>

    <!-- Print Panel Overlay -->
    <Teleport to="body">
      <div v-if="showPrintPanel" class="print-overlay">
        <div class="print-panel">
          <div class="print-header">
            <div>
              <h2 class="print-title">{{ printDocLabel }}</h2>
              <p class="print-code">Encomienda: <strong>{{ printParcelCode }}</strong></p>
            </div>
            <button class="print-close" @click="closePrintPanel">&times;</button>
          </div>

          <div class="print-formats">
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'ticket' }" @click="printPdfFormat = 'ticket'">Ticket 80mm</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4media' }" @click="printPdfFormat = 'a4media'">A4 Media</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4' }" @click="printPdfFormat = 'a4'">A4 Full</button>
          </div>

          <iframe
            v-if="printSaleId"
            :src="getBillingPDFUrl(printSaleId, printPdfFormat)"
            class="print-preview"
            title="Vista previa del comprobante"
          ></iframe>

          <div v-else-if="printBillingError" style="padding:1rem;background:#fef2f2;border:1px solid #fecaca;border-radius:8px;margin:0 1rem 1rem;color:#991b1b;font-size:0.88rem">
            <strong>No se pudo emitir el comprobante:</strong>
            <p style="margin:0.4rem 0 0.75rem;white-space:pre-wrap">{{ printBillingError }}</p>
            <button @click="retryBilling" :disabled="printRetrying"
              style="padding:0.5rem 1rem;background:#1b55f5;color:white;border:none;border-radius:6px;font-weight:600;cursor:pointer">
              {{ printRetrying ? 'Reintentando...' : 'Reintentar emisión' }}
            </button>
          </div>

          <div class="print-actions">
            <button v-if="printSaleId" class="print-action-btn print-btn-print" @click="openPrintPDF(printPdfFormat)">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="6 9 6 2 18 2 18 9"/>
                <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/>
                <rect width="12" height="8" x="6" y="14"/>
              </svg>
              Imprimir
            </button>
            <button v-if="printWhatsappUrl" class="print-action-btn print-btn-wa" @click="openPrintWhatsApp">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
              </svg>
              WhatsApp
            </button>
            <button class="print-action-btn print-btn-detail" @click="closePrintPanel">
              Ir al detalle
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.parcel-form-page {
  padding: 0;
}

.page-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1.75rem;
}
.page-header h1 {
  font-size: 1.4rem;
  font-weight: 800;
  color: var(--slate-900);
}
.page-header p {
  font-size: 0.9rem;
  color: var(--slate-500);
  margin-top: 0.2rem;
}

.btn-back {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.5rem 0.75rem;
  background: white;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--slate-600);
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s;
  white-space: nowrap;
  margin-top: 0.15rem;
}
.btn-back:hover {
  border-color: var(--brand-300);
  color: var(--brand-600);
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 0;
  color: var(--slate-500);
  gap: 1rem;
}

.form-sections {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.form-section {
  background: white;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
}
.form-section h2 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 1rem;
}
.form-section h2 svg {
  color: var(--brand-500);
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 0.85rem;
}
.field.full {
  grid-column: 1 / -1;
}

.field label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.3rem;
}

.field input,
.field select,
.field textarea {
  width: 100%;
  padding: 0.6rem 0.8rem;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}
.field input:focus,
.field select:focus,
.field textarea:focus {
  border-color: var(--brand-500);
  box-shadow: 0 0 0 3px rgba(27, 85, 245, 0.1);
}
.field input:disabled,
.field select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.field textarea {
  resize: vertical;
}

.input-with-status {
  position: relative;
}
.input-with-status input {
  padding-right: 2.5rem;
}

.lookup-spinner {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--brand-500);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

.field-error {
  display: block;
  font-size: 0.75rem;
  color: var(--danger-600, #dc2626);
  margin-top: 0.25rem;
  font-weight: 500;
}

/* Radio group */
.radio-group {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.radio-label {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 1rem;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-600);
  cursor: pointer;
  transition: all 0.2s;
  background: var(--slate-50);
}
.radio-label input[type="radio"] {
  width: auto;
  margin: 0;
  accent-color: var(--brand-500);
}
.radio-label.active {
  border-color: var(--brand-400);
  background: var(--brand-50);
  color: var(--brand-700);
}

/* Messages */
.msg {
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  font-weight: 600;
}
.msg-error {
  background: var(--danger-50, #fef2f2);
  color: var(--danger-700, #b91c1c);
  border: 1px solid var(--danger-200, #fecaca);
}
.msg-success {
  background: var(--success-50, #f0fdf4);
  color: var(--success-700, #15803d);
  border: 1px solid var(--success-200, #bbf7d0);
}

/* Actions */
.form-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 0.5rem;
}

.btn-cancel {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.7rem 1.5rem;
  background: white;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--slate-600);
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s;
}
.btn-cancel:hover {
  border-color: var(--slate-400);
  color: var(--slate-800);
}

.btn-submit {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.7rem 2rem;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.95rem;
  font-weight: 700;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s;
  box-shadow: 0 2px 8px rgba(27, 85, 245, 0.2);
}
.btn-submit:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(27, 85, 245, 0.3);
}
.btn-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Spinners */
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--brand-500);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
.spinner-sm {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 640px) {
  .field-grid {
    grid-template-columns: 1fr;
  }
  .form-actions {
    flex-direction: column-reverse;
  }
  .form-actions .btn-cancel,
  .form-actions .btn-submit {
    width: 100%;
    justify-content: center;
  }
}

/* ── Print Panel ── */
.print-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.print-panel {
  background: white;
  border-radius: var(--radius-xl);
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid var(--slate-200);
  animation: printIn 0.25s ease;
}

@keyframes printIn {
  from { opacity: 0; transform: scale(0.95) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.print-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--slate-200);
}

.print-title {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--slate-900);
}

.print-code {
  font-size: 0.85rem;
  color: var(--slate-500);
  margin-top: 0.15rem;
}

.print-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--slate-400);
  cursor: pointer;
  line-height: 1;
  padding: 0.25rem;
}

.print-close:hover {
  color: var(--slate-900);
}

.print-formats {
  display: flex;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  background: var(--slate-50);
  border-bottom: 1px solid var(--slate-200);
}

.print-fmt-btn {
  flex: 1;
  padding: 0.55rem 0.75rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: white;
  color: var(--slate-600);
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
  text-align: center;
}

.print-fmt-btn:hover {
  border-color: var(--brand-300);
  color: var(--brand-600);
}

.print-fmt-btn.active {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-color: var(--brand-500);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.print-preview {
  width: 100%;
  height: 400px;
  border: none;
  display: block;
}

.print-actions {
  display: flex;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--slate-200);
  background: var(--slate-50);
}

.print-action-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.65rem 1rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
}

.print-btn-print {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.print-btn-print:hover {
  background: linear-gradient(135deg, var(--brand-500), var(--brand-600));
}

.print-btn-wa {
  background: #25D366;
  color: white;
  box-shadow: 0 2px 8px rgba(37, 211, 102, 0.3);
}

.print-btn-wa:hover {
  background: #1fb855;
}

.print-btn-detail {
  background: var(--slate-100);
  color: var(--slate-700);
  border: 2px solid var(--slate-300);
}

.print-btn-detail:hover {
  background: var(--slate-200);
  color: var(--slate-900);
}

@media (max-width: 640px) {
  .print-panel {
    max-height: 95vh;
  }

  .print-preview {
    height: 300px;
  }

  .print-actions {
    flex-direction: column;
  }

  .print-formats {
    flex-direction: column;
  }
}
</style>
