<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRoutes, getTripsByRoute, getTripSeats, type Route, type TripInstance, type SeatInfo, type TripSeatResponse } from '../api/client'
import { useSeatsWebSocket } from '../composables/useSeatsWebSocket'
import SeatLayoutRenderer from '../components/SeatLayoutRenderer.vue'
import Swal from 'sweetalert2'

const route = useRoute()
const router = useRouter()

const routes = ref<Route[]>([])
const trips = ref<TripInstance[]>([])
const seatResponse = ref<TripSeatResponse | null>(null)
const seats = computed(() => seatResponse.value?.seats ?? [])
const loading = ref(false)
const error = ref('')

const selectedSeatIds = ref<Set<number>>(new Set())
const activeFloor = ref(1)
const wsConnected = ref(false)
let wsCleanup: (() => void) | null = null

const routeId = computed(() => {
  const id = route.params.routeId
  return typeof id === 'string' && id ? parseInt(id, 10) : null
})
const tripId = computed(() => {
  const id = route.params.tripId
  return typeof id === 'string' && id ? parseInt(id, 10) : null
})

const currentRoute = computed(() => {
  if (!routeId.value) return null
  return routes.value.find(r => r.id === routeId.value) || null
})

const currentTrip = computed(() => {
  if (!tripId.value) return null
  return trips.value.find(t => t.id === tripId.value) || null
})

const seatStats = computed(() => {
  const total = seats.value.length
  const available = seats.value.filter(s => s.status === 'available').length
  const sold = seats.value.filter(s => s.status === 'sold').length
  const held = seats.value.filter(s => s.status === 'held').length
  return { total, available, sold, held }
})

const selectedSeats = computed(() => {
  return seats.value.filter(s => selectedSeatIds.value.has(s.id))
})

const vehicleFloors = computed(() => seatResponse.value?.vehicle_floors ?? 1)
const vehicleLayoutCols = computed(() => seatResponse.value?.vehicle_layout_cols ?? 4)
const vehicleName = computed(() => seatResponse.value?.vehicle_name ?? '')
const vehicleSeatType = computed(() => seatResponse.value?.vehicle_seat_type ?? 'regular')
const layoutElements = computed(() => seatResponse.value?.layout_elements ?? [])

// Asientos del piso activo, organizados por fila
const floorSeats = computed(() => {
  const floorData = seats.value.filter(s => s.floor === activeFloor.value)
  const rows = new Map<number, SeatInfo[]>()
  for (const s of floorData) {
    if (!rows.has(s.row_num)) rows.set(s.row_num, [])
    rows.get(s.row_num)!.push(s)
  }
  // Sort each row by col_num
  for (const [, cols] of rows) {
    cols.sort((a, b) => a.col_num - b.col_num)
  }
  return new Map([...rows].sort((a, b) => a[0] - b[0]))
})

const maxCols = computed(() => vehicleLayoutCols.value)

// Build a grid: for each row, create an array of maxCols slots (seat or null for aisle)
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

function seatTypeLabel(t: string) {
  const m: Record<string, string> = { regular: 'Regular', semi_cama: 'Semi-Cama', cama: 'Cama', suite: 'Suite' }
  return m[t] || t
}

// Determine if a column index is an aisle (no seats in any row for this floor)
function isAisleCol(colIdx: number) {
  const floorData = seats.value.filter(s => s.floor === activeFloor.value)
  const colNum = colIdx + 1
  if (floorData.some(s => s.col_num === colNum)) return false
  const fElems = layoutElements.value.filter(e => e.floor === activeFloor.value)
  if (fElems.some(e => e.col_num === colNum)) return false
  return true
}

async function loadRoutes() {
  loading.value = true
  error.value = ''
  try {
    routes.value = await getRoutes()
  } catch (e) {
    error.value = (e as Error).message
    routes.value = []
  } finally {
    loading.value = false
  }
}

async function loadTrips() {
  if (!routeId.value) return
  loading.value = true
  error.value = ''
  try {
    const from = new Date().toISOString().slice(0, 10)
    trips.value = await getTripsByRoute(routeId.value, from)
  } catch (e) {
    error.value = (e as Error).message
    trips.value = []
  } finally {
    loading.value = false
  }
}

async function loadSeats() {
  if (!tripId.value) return
  loading.value = true
  error.value = ''
  selectedSeatIds.value = new Set()
  activeFloor.value = 1
  try {
    seatResponse.value = await getTripSeats(tripId.value)
  } catch (e) {
    error.value = (e as Error).message
    seatResponse.value = null
  } finally {
    loading.value = false
  }

  // Connect WebSocket for real-time updates
  if (wsCleanup) { wsCleanup(); wsCleanup = null }
  wsConnected.value = false
  if (tripId.value) {
    const { connected, disconnect } = useSeatsWebSocket(tripId.value, (data) => {
      seatResponse.value = data
    })
    wsCleanup = () => { disconnect(); wsConnected.value = false }
    // Sync connected state
    watch(connected, (val) => { wsConnected.value = val }, { immediate: true })
  }
}

function goTrips(r: Route) {
  router.push(`/viajes/ruta/${r.id}`)
}

function goSeats(t: TripInstance) {
  router.push(`/viajes/salida/${t.id}`)
}

// Wrapper para el evento del SeatLayoutRenderer (que entrega RendererSeat,
// un subtipo de SeatInfo). Re-encontramos el SeatInfo completo por id para
// que TypeScript valide correctamente.
function handleRendererSeatClick(rs: { id?: number }) {
  if (rs.id == null) return
  const full = seats.value.find(s => s.id === rs.id)
  if (full) toggleSeat(full)
}

function toggleSeat(s: SeatInfo) {
  if (s.status === 'sold' || s.status === 'blocked') {
    const isSold = s.status === 'sold'
    const headerBg = isSold ? 'linear-gradient(135deg,#EF4444,#DC2626)' : 'linear-gradient(135deg,#94A3B8,#64748B)'
    const label = isSold ? 'Vendido' : 'Bloqueado'
    Swal.fire({
      icon: undefined,
      title: '',
      html: `<div style="margin:-1.5rem -1.5rem 0;padding:1.25rem 1.5rem;background:${headerBg};border-radius:16px 16px 0 0;text-align:center;margin-bottom:1.25rem">
          <div style="display:inline-flex;align-items:center;justify-content:center;width:48px;height:48px;border-radius:50%;background:rgba(255,255,255,0.2);border:2px solid rgba(255,255,255,0.3);margin-bottom:0.4rem">
            <span style="font-size:1.2rem;font-weight:800;color:white">${s.label}</span>
          </div>
          <div style="font-size:1rem;font-weight:700;color:white">${label}</div>
        </div>
        <p style="font-size:0.9rem;color:#64748b;text-align:center;line-height:1.5">Este asiento no esta disponible.<br>Selecciona otro asiento libre.</p>`,
      confirmButtonText: 'Entendido',
      confirmButtonColor: '#3B82F6',
      customClass: { popup: 'swal-custom-popup', confirmButton: 'swal-btn-confirm' },
    })
    return
  }

  if (s.status === 'held') {
    Swal.fire({
      icon: undefined,
      title: '',
      html: `<div style="margin:-1.5rem -1.5rem 0;padding:1.25rem 1.5rem;background:linear-gradient(135deg,#F59E0B,#D97706);border-radius:16px 16px 0 0;text-align:center;margin-bottom:1.25rem">
          <div style="display:inline-flex;align-items:center;justify-content:center;width:48px;height:48px;border-radius:50%;background:rgba(255,255,255,0.2);border:2px solid rgba(255,255,255,0.3);margin-bottom:0.4rem">
            <span style="font-size:1.2rem;font-weight:800;color:white">${s.label}</span>
          </div>
          <div style="font-size:1rem;font-weight:700;color:white">Reservado temporalmente</div>
        </div>
        <p style="font-size:0.9rem;color:#64748b;text-align:center;line-height:1.5">Otro pasajero esta completando su compra.<br>Estara disponible pronto si no confirma.</p>
      </div>`,
      confirmButtonText: 'Entendido',
      confirmButtonColor: '#3B82F6',
      showClass: { popup: 'swal2-show animate__animated animate__fadeInUp animate__faster' },
      customClass: { icon: 'swal-no-border', popup: 'swal-custom-popup' },
    })
    return
  }

  const isSelected = selectedSeatIds.value.has(s.id)

  if (isSelected) {
    const set = new Set(selectedSeatIds.value)
    set.delete(s.id)
    selectedSeatIds.value = set

    const Toast = Swal.mixin({ toast: true, position: 'bottom-end', showConfirmButton: false, timer: 1500, timerProgressBar: true })
    Toast.fire({ icon: 'info', title: `Asiento ${s.label} removido` })
    return
  }

  // Available seat → show selection modal
  const floorLabel = vehicleFloors.value > 1 ? `Piso ${s.floor}` : ''
  const tags = [seatTypeLabel(s.seat_type), `Fila ${s.row_num}`, floorLabel].filter(Boolean)

  Swal.fire({
    title: '',
    html: `<div style="margin:-1.5rem -1.5rem 0;padding:1.25rem 1.5rem;background:linear-gradient(135deg,#3B82F6,#2563EB);border-radius:16px 16px 0 0;text-align:center;margin-bottom:1.25rem">
        <div style="display:inline-flex;align-items:center;justify-content:center;width:52px;height:52px;border-radius:50%;background:rgba(255,255,255,0.2);border:2px solid rgba(255,255,255,0.3);margin-bottom:0.5rem">
          <span style="font-size:1.35rem;font-weight:800;color:white">${s.label}</span>
        </div>
        <div style="font-size:1.1rem;font-weight:800;color:white">Seleccion de Pasaje</div>
        <div style="font-size:0.82rem;color:rgba(255,255,255,0.75);margin-top:0.15rem">Asiento ${s.label}</div>
      </div>
      <div style="display:flex;justify-content:center;gap:0.4rem;margin-bottom:0.85rem;flex-wrap:wrap">
        ${tags.map(t => `<span style="padding:0.25rem 0.7rem;background:#F1F5F9;border-radius:20px;font-size:0.78rem;font-weight:600;color:#475569">${t}</span>`).join('')}
        <span style="padding:0.25rem 0.7rem;background:#ECFDF5;border:1px solid #A7F3D0;border-radius:20px;font-size:0.78rem;font-weight:700;color:#059669">Disponible</span>
      </div>
      <p style="font-size:0.9rem;color:#64748b;text-align:center">Deseas reservar este asiento para tu viaje?</p>`,
    showCancelButton: true,
    confirmButtonText: 'Si, seleccionar',
    cancelButtonText: 'Cancelar',
    confirmButtonColor: '#3B82F6',
    cancelButtonColor: '#e2e8f0',
    customClass: {
      popup: 'swal-custom-popup',
      confirmButton: 'swal-btn-confirm',
      cancelButton: 'swal-btn-cancel',
    },
  }).then((result) => {
    if (result.isConfirmed) {
      const set = new Set(selectedSeatIds.value)
      set.add(s.id)
      selectedSeatIds.value = set

      const Toast = Swal.mixin({ toast: true, position: 'bottom-end', showConfirmButton: false, timer: 2000, timerProgressBar: true })
      Toast.fire({ icon: 'success', title: `Asiento ${s.label} seleccionado` })
    }
  })
}

function isSeatSelected(s: SeatInfo) {
  return selectedSeatIds.value.has(s.id)
}

function goReserva() {
  if (!tripId.value || selectedSeatIds.value.size === 0) return
  const seatIdsStr = Array.from(selectedSeatIds.value).join(',')
  router.push(`/viajes/reservar/${tripId.value}?seats=${seatIdsStr}`)
}

function formatDate(iso: string) {
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' })
}

function formatTime(iso: string) {
  const d = new Date(iso)
  return d.toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' })
}

function seatStatusLabel(s: string) {
  const map: Record<string, string> = { available: 'Disponible', held: 'Reservado', sold: 'Vendido', blocked: 'Bloqueado' }
  return map[s] || s
}

function seatStatusClass(s: SeatInfo) {
  if (isSeatSelected(s)) return 'selected'
  return s.status === 'available' ? 'available' : s.status === 'sold' ? 'sold' : s.status === 'held' ? 'held' : 'blocked'
}

onMounted(() => {
  loadRoutes()
  if (tripId.value) loadSeats()
  else if (routeId.value) loadTrips()
})

watch([routeId, tripId], () => {
  if (wsCleanup) { wsCleanup(); wsCleanup = null }
  if (tripId.value) loadSeats()
  else if (routeId.value) loadTrips()
})
</script>

<template>
  <div class="viajes-page">
    <!-- Hero Banner for Routes -->
    <div v-if="!loading && !error && !routeId && !tripId" class="viajes-hero">
      <div class="viajes-hero-bg"></div>
      <div class="container viajes-hero-inner">
        <div class="viajes-hero-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/>
            <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
            <circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>
          </svg>
        </div>
        <h1 class="viajes-hero-title">Buscar Viajes</h1>
        <p class="viajes-hero-subtitle">Elige tu ruta y encuentra los mejores horarios para viajar</p>
        <div class="hero-features">
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>
            Compra segura
          </span>
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
            Elige tu asiento
          </span>
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/></svg>
            Boleto digital
          </span>
        </div>
      </div>
    </div>

    <!-- Hero Banner for Trips -->
    <div v-if="!loading && !error && routeId && !tripId" class="viajes-hero trips-hero">
      <div class="viajes-hero-bg"></div>
      <div class="container viajes-hero-inner">
        <div class="viajes-hero-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/>
          </svg>
        </div>
        <h1 class="viajes-hero-title" v-if="currentRoute">{{ currentRoute.name }}</h1>
        <h1 class="viajes-hero-title" v-else>Salidas Disponibles</h1>
        <p class="viajes-hero-subtitle">Elige fecha y horario para tu viaje</p>
        <div class="hero-features">
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            Multiples horarios
          </span>
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>
            Bus Semi-Cama y Cama
          </span>
          <span class="hero-feat">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>
            Reserva inmediata
          </span>
        </div>
      </div>
    </div>

    <!-- Hero Banner for Seats -->
    <div v-if="!loading && !error && tripId" class="viajes-hero seats-hero">
      <div class="viajes-hero-bg seats-hero-bg"></div>
      <div class="container viajes-hero-inner">
        <div class="viajes-hero-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
          </svg>
        </div>
        <h1 class="viajes-hero-title">Selecciona tus Asientos</h1>
        <p class="viajes-hero-subtitle" v-if="seatResponse">{{ vehicleName }} · {{ seatStats.available }} asientos disponibles</p>
        <div class="hero-features">
          <span class="hero-feat feat-green">
            <span class="feat-dot green"></span> Disponible
          </span>
          <span class="hero-feat feat-blue">
            <span class="feat-dot blue"></span> Seleccionado
          </span>
          <span class="hero-feat feat-red">
            <span class="feat-dot red"></span> Vendido
          </span>
          <span class="hero-feat feat-yellow">
            <span class="feat-dot yellow"></span> Reservado
          </span>
        </div>
      </div>
    </div>

    <div class="container viajes-content">
      <!-- Breadcrumb (solo cuando hay seleccion) -->
      <nav v-if="routeId || tripId" class="breadcrumb" aria-label="Navegacion">
        <router-link to="/viajes" class="crumb">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 18l6-6-6-6"/><path d="M3 12h13"/><path d="M21 5v14"/>
          </svg>
          Rutas
        </router-link>
        <template v-if="routeId">
          <svg class="crumb-sep" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
          <router-link v-if="tripId && currentRoute" :to="`/viajes/ruta/${routeId}`" class="crumb">
            {{ currentRoute.name }}
          </router-link>
          <span v-else class="crumb active">Salidas</span>
        </template>
        <template v-if="tripId">
          <svg class="crumb-sep" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
          <span class="crumb active">Asientos</span>
        </template>
      </nav>

      <!-- Error State -->
      <div v-if="error" class="alert alert-error">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"/><line x1="12" x2="12" y1="8" y2="12"/><line x1="12" x2="12.01" y1="16" y2="16"/>
        </svg>
        <span>{{ error }}</span>
      </div>

      <!-- Loading State -->
      <div v-else-if="loading" class="loading-state">
        <div class="spinner-lg"></div>
        <span>Cargando...</span>
      </div>

      <!-- ROUTES LIST -->
      <section v-else-if="!routeId && !tripId" class="section-routes">
        <h2 class="section-heading">Rutas disponibles</h2>
        <div class="routes-grid-v2">
          <button
            v-for="r in routes"
            :key="r.id"
            type="button"
            class="route-card-v2"
            @click="goTrips(r)"
          >
            <div class="rc-top">
              <div class="rc-icon">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/>
                  <path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/>
                  <circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>
                </svg>
              </div>
              <span class="rc-code">{{ r.code }}</span>
            </div>
            <div class="rc-body">
              <div class="rc-cities">
                <span class="rc-city">{{ r.name.split(' - ')[0] || r.name }}</span>
                <div class="rc-arrow-line">
                  <div class="rc-dot"></div>
                  <div class="rc-line"></div>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m9 18 6-6-6-6"/></svg>
                  <div class="rc-line"></div>
                  <div class="rc-dot dest"></div>
                </div>
                <span class="rc-city">{{ r.name.split(' - ')[1] || '' }}</span>
              </div>
            </div>
            <div class="rc-footer">
              <span>Ver salidas disponibles</span>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
            </div>
          </button>
        </div>

        <!-- Info section below routes -->
        <div class="routes-info">
          <div class="ri-card">
            <div class="ri-icon ri-1">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            </div>
            <div>
              <h4>Horarios frecuentes</h4>
              <p>Salidas diarias en la manana y noche</p>
            </div>
          </div>
          <div class="ri-card">
            <div class="ri-icon ri-2">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>
            </div>
            <div>
              <h4>Buses 1 y 2 pisos</h4>
              <p>Servicio Semi-Cama y Cama disponible</p>
            </div>
          </div>
          <div class="ri-card">
            <div class="ri-icon ri-3">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>
            </div>
            <div>
              <h4>Reserva garantizada</h4>
              <p>Tu asiento queda asegurado al instante</p>
            </div>
          </div>
        </div>

        <div v-if="routes.length === 0" class="empty-state">
          <div class="empty-icon-wrapper">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M16 16s-1.5-2-4-2-4 2-4 2"/><line x1="9" x2="9.01" y1="9" y2="9"/><line x1="15" x2="15.01" y1="9" y2="9"/></svg>
          </div>
          <h3>Sin rutas disponibles</h3>
          <p>No hay rutas activas en este momento.</p>
        </div>
      </section>

      <!-- TRIPS LIST -->
      <section v-else-if="routeId && !tripId" class="section-trips">
        <h2 class="section-heading">{{ trips.length }} salida{{ trips.length !== 1 ? 's' : '' }} disponible{{ trips.length !== 1 ? 's' : '' }}</h2>

        <!-- Trip Cards -->
        <div class="trips-grid">
          <button
            v-for="t in trips"
            :key="t.id"
            type="button"
            class="trip-card-v2"
            @click="goSeats(t)"
          >
            <div class="trip-card-top">
              <div class="trip-time-block">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                </svg>
                <span class="trip-time-lg">{{ formatTime(t.departure_at) }}</span>
              </div>
              <span class="trip-status-v2" :class="t.status">
                <span class="status-dot-sm"></span>
                {{ t.status === 'scheduled' ? 'Programado' : t.status }}
              </span>
            </div>
            <div class="trip-card-body">
              <div class="trip-route-line">
                <div class="route-point origin">
                  <div class="point-dot"></div>
                  <span>{{ t.route_name.split(' - ')[0] || t.route_name }}</span>
                </div>
                <div class="route-line-connector">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14"/><path d="m19 12-7 7-7-7"/></svg>
                </div>
                <div class="route-point destination">
                  <div class="point-dot dest"></div>
                  <span>{{ t.route_name.split(' - ')[1] || '' }}</span>
                </div>
              </div>
              <div class="trip-meta">
                <span class="trip-meta-item">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/></svg>
                  {{ formatDate(t.departure_at) }}
                </span>
              </div>
            </div>
            <div class="trip-card-action">
              <span class="trip-select-text">Seleccionar asientos</span>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
            </div>
          </button>
        </div>
        <div v-if="trips.length === 0" class="empty-state">
          <div class="empty-icon-wrapper">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect width="18" height="18" x="3" y="4" rx="2" ry="2"/>
              <line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/>
              <line x1="3" x2="21" y1="10" y2="10"/>
            </svg>
          </div>
          <h3>Sin salidas disponibles</h3>
          <p>No hay salidas programadas para esta ruta en los proximos dias.</p>
          <router-link to="/viajes" class="empty-link">Ver otras rutas</router-link>
        </div>
      </section>

      <!-- SEATS MAP -->
      <section v-else class="section-seats">
        <!-- Stats Bar Compact -->
        <div v-if="seats.length" class="seats-stats-h">
          <div class="stat-h">
            <span class="stat-h-dot green"></span>
            <span class="stat-h-val">{{ seatStats.available }}</span>
            <span class="stat-h-lbl">disponibles</span>
          </div>
          <div class="stat-h">
            <span class="stat-h-dot red"></span>
            <span class="stat-h-val">{{ seatStats.sold }}</span>
            <span class="stat-h-lbl">vendidos</span>
          </div>
          <div class="stat-h">
            <span class="stat-h-dot yellow"></span>
            <span class="stat-h-val">{{ seatStats.held }}</span>
            <span class="stat-h-lbl">reservados</span>
          </div>
          <div class="stat-h" v-if="vehicleFloors > 1">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>
            <span class="stat-h-lbl">{{ vehicleFloors }} pisos</span>
          </div>
          <div class="stat-h live-indicator" v-if="wsConnected">
            <span class="live-dot"></span>
            <span class="stat-h-lbl">EN VIVO</span>
          </div>
        </div>

        <!-- Bus Diagram (mismo render que el editor de plantillas) -->
        <div class="bus-h-container">
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

        <!-- Legend is now shown in the hero banner above -->

        <!-- Selected Seats Summary -->
        <div v-if="selectedSeats.length > 0" class="selection-summary">
          <div class="selection-info">
            <h3 class="selection-title">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
              </svg>
              Asientos seleccionados ({{ selectedSeats.length }})
            </h3>
            <div class="selection-chips">
              <span v-for="s in selectedSeats" :key="s.id" class="selection-chip">
                <template v-if="vehicleFloors === 2">P{{ s.floor }}-</template>{{ s.label }}
                <button class="chip-remove" @click.stop="toggleSeat(s)" title="Quitar">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
                  </svg>
                </button>
              </span>
            </div>
          </div>
          <button class="btn btn-primary btn-lg" @click="goReserva">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>
            </svg>
            Continuar
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.viajes-page {
  min-height: 80vh;
}

/* ── Hero Banner ── */
.viajes-hero {
  position: relative;
  padding: 3.5rem 0 2.5rem;
  overflow: hidden;
}

.viajes-hero-bg {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at 20% 0%, rgba(59,130,246,0.10) 0%, transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(16,185,129,0.08) 0%, transparent 55%),
    linear-gradient(180deg, var(--brand-50) 0%, #ffffff 70%);
  border-bottom: 1px solid var(--color-border);
}

.viajes-hero-bg::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(ellipse at 50% 50%, rgba(59,130,246,0.04) 0%, transparent 60%);
}

.viajes-hero-inner {
  position: relative;
  text-align: center;
}

.viajes-hero-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 68px;
  height: 68px;
  border-radius: 50%;
  background: var(--brand-50);
  border: 1px solid var(--brand-100);
  color: var(--brand-600);
  margin-bottom: 1rem;
  box-shadow: 0 6px 18px rgba(59,130,246,0.15);
}

.viajes-hero-title {
  font-size: 2.25rem;
  font-weight: 800;
  color: var(--color-heading);
  margin-bottom: 0.5rem;
  letter-spacing: -0.02em;
}

.viajes-hero-subtitle {
  font-size: 1.05rem;
  color: var(--color-text-muted);
  margin-bottom: 1.5rem;
  max-width: 450px;
  margin-left: auto;
  margin-right: auto;
}

.hero-features {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.hero-feat {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.85rem;
  background: #ffffff;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  color: var(--color-text);
  font-size: 0.75rem;
  font-weight: 600;
  box-shadow: var(--shadow-xs);
}

/* ── Content ── */
.viajes-content {
  padding-top: 2rem;
  padding-bottom: 4rem;
}

/* ── Hero Variants ── */
.seats-hero-bg {
  background:
    radial-gradient(ellipse at 20% 0%, rgba(16,185,129,0.10) 0%, transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(59,130,246,0.08) 0%, transparent 55%),
    linear-gradient(180deg, var(--accent-50) 0%, #ffffff 70%) !important;
}

.feat-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  display: inline-block;
}
.feat-dot.green { background: var(--accent-500); }
.feat-dot.blue { background: var(--brand-500); }
.feat-dot.red { background: var(--danger-500); }
.feat-dot.yellow { background: var(--warning-500); }

/* ── Compact Stats Bar ── */
.seats-stats-h {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  padding: 0.75rem 1.25rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.stat-h {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.82rem;
  color: var(--color-text-muted);
}

.stat-h-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.stat-h-dot.green { background: var(--success-500); }
.stat-h-dot.red { background: var(--danger-500); }
.stat-h-dot.yellow { background: var(--warning-500); }

.stat-h-val {
  font-weight: 800;
  color: var(--color-heading);
  font-size: 0.92rem;
}

.stat-h-lbl {
  font-weight: 500;
}

.stat-h svg { color: var(--brand-500); }

.live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
  animation: livePulse 2s infinite;
  flex-shrink: 0;
}

@keyframes livePulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
  50% { box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.1); }
}

.live-indicator {
  color: #059669 !important;
  font-weight: 700 !important;
}

.section-heading {
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--color-heading);
  margin-bottom: 1.25rem;
  letter-spacing: -0.01em;
}

/* ── Breadcrumb ── */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  margin-bottom: 1.75rem;
  padding: 0.65rem 1rem;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-xs);
}

.crumb {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--color-text-muted);
  text-decoration: none;
  transition: color 0.2s ease;
}

a.crumb:hover {
  color: var(--color-primary);
}

.crumb.active {
  color: var(--color-heading);
  font-weight: 600;
}

.crumb-sep {
  color: var(--color-text-muted);
  opacity: 0.4;
  flex-shrink: 0;
}

/* ── Alert ── */
.alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  margin-bottom: 1.5rem;
}

.alert-error {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
}

/* ── Loading ── */
.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 0;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 0.9rem;
}

.spinner-lg {
  width: 24px;
  height: 24px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Empty State ── */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 3rem 1rem;
  color: var(--color-text-muted);
  text-align: center;
}

.empty-state svg {
  opacity: 0.4;
}

.empty-state p {
  font-size: 0.95rem;
}

/* ── ROUTES SECTION ── */
.routes-grid-v2 {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.route-card-v2 {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition: all 0.25s ease;
  text-align: left;
  width: 100%;
  font-family: inherit;
  color: inherit;
  overflow: hidden;
}

.route-card-v2:hover {
  border-color: var(--brand-300);
  box-shadow: var(--shadow-xl);
  transform: translateY(-3px);
}

.rc-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  background: linear-gradient(135deg, var(--brand-200) 0%, var(--brand-400) 100%);
  border-bottom: 1px solid var(--brand-300);
}

.rc-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: #ffffff;
  color: var(--brand-600);
  box-shadow: 0 1px 3px rgba(15,23,42,0.08);
}

.rc-code {
  padding: 0.2rem 0.6rem;
  background: #ffffff;
  border-radius: var(--radius-full);
  color: var(--brand-700);
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  box-shadow: 0 1px 2px rgba(15,23,42,0.06);
}

.rc-body {
  padding: 1.25rem;
  flex: 1;
}

.rc-cities {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.rc-city {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-heading);
  white-space: nowrap;
}

.rc-arrow-line {
  display: flex;
  align-items: center;
  gap: 0;
  flex: 1;
  min-width: 40px;
  color: var(--brand-400);
}

.rc-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  border: 2px solid var(--brand-500);
  background: white;
  flex-shrink: 0;
}

.rc-dot.dest {
  background: var(--brand-500);
}

.rc-line {
  flex: 1;
  height: 2px;
  background: var(--brand-200);
}

@media (prefers-color-scheme: dark) {
  .rc-line { background: rgba(27, 85, 245, 0.3); }
  .rc-dot { background: var(--color-background); }
}

.rc-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  padding: 0.85rem 1.25rem;
  background: var(--brand-50);
  border-top: 1px solid var(--brand-100);
  color: var(--brand-600);
  font-size: 0.85rem;
  font-weight: 700;
  transition: background 0.2s ease;
}

@media (prefers-color-scheme: dark) {
  .rc-footer {
    background: rgba(27, 85, 245, 0.08);
    border-top-color: rgba(27, 85, 245, 0.15);
  }
}

.route-card-v2:hover .rc-footer {
  background: var(--brand-100);
}

@media (prefers-color-scheme: dark) {
  .route-card-v2:hover .rc-footer { background: rgba(27, 85, 245, 0.15); }
}

/* Routes Info Section */
.routes-info {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  padding: 1.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.ri-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.ri-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.ri-1 { background: var(--warning-50); color: var(--warning-600); }
.ri-2 { background: var(--accent-50); color: var(--accent-700); }
.ri-3 { background: var(--success-50); color: var(--success-600); }

@media (prefers-color-scheme: dark) {
  .ri-1 { background: rgba(245, 158, 11, 0.1); }
  .ri-2 { background: rgba(6, 201, 170, 0.1); }
  .ri-3 { background: rgba(16, 185, 129, 0.1); }
}

.ri-card h4 {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.15rem;
}

.ri-card p {
  font-size: 0.78rem;
  color: var(--color-text-muted);
  line-height: 1.4;
}

/* ── TRIPS SECTION ── */
.trips-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 1rem;
}

.trip-card-v2 {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 2px solid var(--slate-600);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  width: 100%;
  font-family: inherit;
  color: inherit;
  overflow: hidden;
}

.trip-card-v2:hover {
  border-color: var(--brand-300);
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.trip-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.85rem 1.15rem;
  background: var(--brand-50);
  border-bottom: 1px solid var(--brand-100);
}

.trip-time-block {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--brand-600);
}

.trip-time-lg {
  font-size: 1.35rem;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.trip-status-v2 {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.65rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.trip-status-v2.scheduled {
  background: var(--success-50);
  color: var(--success-700);
}

.status-dot-sm {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.trip-card-body {
  padding: 1rem 1.15rem;
  flex: 1;
}

.trip-route-line {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  margin-bottom: 0.75rem;
}

.route-point {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--color-heading);
}

.point-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2.5px solid var(--brand-500);
  background: white;
  flex-shrink: 0;
}

.point-dot.dest {
  background: var(--brand-500);
}

.route-line-connector {
  margin-left: 3px;
  padding: 0 0 0 0;
  color: var(--color-border);
  height: 18px;
  display: flex;
  align-items: center;
}

.trip-meta {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.trip-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.78rem;
  color: var(--color-text-muted);
  font-weight: 500;
}

.trip-card-action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  padding: 0.75rem 1.15rem;
  background: var(--brand-50);
  border-top: 1px solid var(--brand-100);
  color: var(--brand-600);
  font-size: 0.85rem;
  font-weight: 700;
  transition: background 0.2s ease;
}

@media (prefers-color-scheme: dark) {
  .trip-card-action {
    background: rgba(27, 85, 245, 0.08);
    border-top-color: rgba(27, 85, 245, 0.15);
  }
}

.trip-card-v2:hover .trip-card-action {
  background: var(--brand-100);
}

@media (prefers-color-scheme: dark) {
  .trip-card-v2:hover .trip-card-action {
    background: rgba(27, 85, 245, 0.15);
  }
}

.trip-select-text {
  font-size: 0.82rem;
}

.empty-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--color-background-mute);
  color: var(--color-text-muted);
  margin-bottom: 0.5rem;
}

.empty-state h3 {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.25rem;
}

.empty-link {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  margin-top: 0.75rem;
  padding: 0.5rem 1rem;
  background: var(--brand-50);
  color: var(--brand-600);
  border-radius: var(--radius-full);
  font-size: 0.85rem;
  font-weight: 600;
  text-decoration: none;
  transition: background 0.2s ease;
}

.empty-link:hover {
  background: var(--brand-100);
}

/* ── SEATS SECTION ── */
.seats-stats {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.stat {
  flex: 1;
  min-width: 100px;
  padding: 0.85rem 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--color-heading);
  line-height: 1;
}

.stat-label {
  display: block;
  font-size: 0.72rem;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-weight: 500;
  margin-top: 0.35rem;
}

/* (old stat cards removed, using compact .seats-stats-h now) */

/* (vehicle-info moved to hero banner) */

/* ── Floor Tabs ── */
.floor-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}

.floor-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: white;
  color: var(--slate-500);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.floor-tab:hover {
  border-color: var(--brand-300);
  color: var(--brand-500);
}

.floor-tab.active {
  border-color: var(--brand-400);
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

@media (prefers-color-scheme: dark) {
  .floor-tab.active {
    background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
    color: white;
  }
}

.floor-type {
  font-size: 0.68rem;
  opacity: 0.7;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

/* ══════════════════════════════
   HORIZONTAL BUS LAYOUT
   ══════════════════════════════ */
.bus-h-container {
  margin-bottom: 1.5rem;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.bus-h-frame {
  display: flex;
  align-items: stretch;
  background: var(--slate-900);
  border: 2px solid var(--slate-600);
  border-radius: var(--radius-lg) 20px 20px var(--radius-lg);
  overflow: hidden;
  min-width: fit-content;
  box-shadow: inset 0 2px 8px rgba(0,0,0,0.2);
}

/* ── Driver (left) ── */
.bus-h-front {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1rem 0.85rem;
  background: var(--slate-800);
  border-right: 2px solid var(--slate-600);
  min-width: 70px;
}

.driver-h {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--slate-700);
  color: var(--slate-400);
}

@media (prefers-color-scheme: dark) {
  .driver-h { background: var(--slate-700); color: var(--slate-400); }
}

.driver-h-label {
  font-size: 0.6rem;
  font-weight: 700;
  color: var(--slate-400);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  writing-mode: horizontal-tb;
}

.door-h {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.15rem;
  color: var(--slate-500);
  opacity: 0.7;
  margin-top: 0.25rem;
}

.door-h span {
  font-size: 0.55rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ── Seats Area (center, horizontal) ── */
.seats-h-area {
  flex: 1;
  padding: 0.6rem;
  overflow-x: auto;
}

.seats-h-grid {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

/* Each row of the grid = one column of the physical bus (top/bottom) */
.seats-h-row {
  display: flex;
  gap: 3px;
}

.aisle-h-row {
  display: flex;
  gap: 3px;
  height: 18px;
  align-items: center;
}

.aisle-h-cell {
  width: 44px;
  flex-shrink: 0;
}

.aisle-h-label {
  font-size: 0.55rem;
  font-weight: 700;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: 0.8;
  white-space: nowrap;
}

.seat-h-cell {
  width: 44px;
  height: 44px;
  flex-shrink: 0;
}

.seat-h-empty {
  width: 44px;
  height: 44px;
}

.seat-h {
  width: 44px;
  height: 44px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  font-weight: 700;
  font-size: 0.75rem;
  cursor: default;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  border: 2px solid transparent;
  user-select: none;
  gap: 1px;
}

.sh-icon { opacity: 0.6; }
.sh-label { line-height: 1; }

/* Seat states (dark theme with gradients - matching admin PuntoVenta) */

.seat-h.available {
  background: linear-gradient(135deg, #34D399, #10B981);
  color: white;
  border-color: #059669;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0,0,0,0.15);
}

.seat-h.available:hover {
  transform: scale(1.12);
  background: linear-gradient(135deg, #6EE7B7, #34D399);
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
  z-index: 2;
}

.seat-h.selected {
  background: linear-gradient(135deg, #60A5FA, #3B82F6);
  color: white;
  border-color: #2563EB;
  cursor: pointer;
  transform: scale(1.12);
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.5);
  z-index: 1;
}

.seat-h.selected:hover {
  transform: scale(1.15);
  box-shadow: 0 6px 18px rgba(59, 130, 246, 0.6);
  z-index: 2;
}

.seat-h.selected .sh-icon { opacity: 1; color: white; }

.seat-h.sold {
  background: linear-gradient(135deg, #FCA5A5, #EF4444);
  color: white;
  border-color: #DC2626;
  opacity: 0.85;
  cursor: pointer;
}

.seat-h.held {
  background: linear-gradient(135deg, #FCD34D, #F59E0B);
  color: #78350F;
  border-color: #D97706;
  opacity: 0.85;
  cursor: pointer;
}

.seat-h.blocked {
  background: var(--slate-700);
  color: var(--slate-500);
  border-color: var(--slate-600);
  opacity: 0.5;
  cursor: pointer;
}

/* ── Rear (right) ── */
.bus-h-rear {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 0.65rem;
  background: var(--slate-800);
  border-left: 2px solid var(--slate-600);
  writing-mode: vertical-rl;
  text-orientation: mixed;
  font-size: 0.6rem;
  font-weight: 700;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

/* ── Legend ── */
.legend {
  display: flex;
  gap: 1.5rem;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 1.5rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.82rem;
  color: var(--color-text-muted);
}

.legend-dot {
  width: 14px;
  height: 14px;
  border-radius: 3px;
  border: 2px solid transparent;
}

.legend-dot.available {
  background: linear-gradient(135deg, #34D399, #10B981);
  border-color: #059669;
}

.legend-dot.selected-dot {
  background: linear-gradient(135deg, #60A5FA, #3B82F6);
  border-color: #2563EB;
}

.legend-dot.sold {
  background: linear-gradient(135deg, #FCA5A5, #EF4444);
  border-color: #DC2626;
}

.legend-dot.held {
  background: linear-gradient(135deg, #FCD34D, #F59E0B);
  border-color: #D97706;
}

.legend-dot.blocked {
  background: var(--slate-700);
  border-color: var(--slate-500);
}

/* ── Selection Summary ── */
.selection-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  padding: 1.5rem 1.75rem;
  background: linear-gradient(135deg, #0F172A, #1E3A8A);
  border: 2px solid rgba(96, 165, 250, 0.3);
  border-radius: var(--radius-xl);
  box-shadow: 0 8px 32px rgba(15, 23, 42, 0.25);
  animation: slideUp 0.3s ease;
  color: white;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.selection-info {
  flex: 1;
  min-width: 0;
}

.selection-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: white;
  margin-bottom: 0.65rem;
}

.selection-title svg {
  color: #60A5FA;
}

.selection-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.selection-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.3rem 0.7rem;
  background: linear-gradient(135deg, #60A5FA, #3B82F6);
  color: white;
  border: none;
  border-radius: var(--radius-full);
  box-shadow: 0 2px 6px rgba(59, 130, 246, 0.3);
  font-size: 0.8rem;
  font-weight: 600;
}

.chip-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: none;
  background: transparent;
  color: var(--brand-400);
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.15s ease;
  padding: 0;
}

.chip-remove:hover {
  background: var(--brand-200);
  color: var(--brand-700);
}

/* ── Buttons ── */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-weight: 600;
  font-size: 0.9rem;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;
  line-height: 1;
}

.btn-primary {
  background: linear-gradient(135deg, var(--brand-600), var(--brand-700));
  color: white;
  box-shadow: var(--shadow-sm), 0 1px 2px rgba(27, 85, 245, 0.2);
}

.btn-primary:hover {
  background: linear-gradient(135deg, var(--brand-500), var(--brand-600));
  box-shadow: var(--shadow-md), 0 2px 8px rgba(27, 85, 245, 0.25);
  color: white;
  transform: translateY(-1px);
}

.btn-lg {
  padding: 0.85rem 1.75rem;
  font-size: 0.95rem;
  border-radius: var(--radius-md);
  white-space: nowrap;
  flex-shrink: 0;
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .viajes-page {
    padding: 1.5rem 0 3rem;
  }

  .page-title {
    font-size: 1.35rem;
  }

  .routes-grid-v2 {
    grid-template-columns: 1fr;
  }

  .trips-grid {
    grid-template-columns: 1fr;
  }


  .routes-info {
    grid-template-columns: 1fr;
  }

  .viajes-hero-title {
    font-size: 1.6rem;
  }

  .hero-features {
    flex-direction: column;
    align-items: center;
  }

  .seats-stats {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
  }

  .bus-h-container {
    margin-left: -1.5rem;
    margin-right: -1.5rem;
    padding: 0 0.5rem;
  }

  .selection-summary {
    flex-direction: column;
    align-items: stretch;
  }

  .btn-lg {
    width: 100%;
    justify-content: center;
  }
}
</style>
