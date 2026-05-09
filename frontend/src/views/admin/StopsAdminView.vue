<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute as useVueRoute } from 'vue-router'
import {
  getRoutes,
  getStops,
  createStop,
  updateStop,
  deleteStop,
  reorderStops,
  getSegments,
  upsertSegment,
  generateSegments,
  deleteSegment,
  type Route,
  type Stop,
  type RouteSegment,
} from '../../api/client'
import Swal from 'sweetalert2'

const vueRoute = useVueRoute()

const routes = ref<Route[]>([])
const selectedRouteId = ref<number>(0)
const stops = ref<Stop[]>([])
const segments = ref<RouteSegment[]>([])
const loading = ref(false)
const loadingStops = ref(false)
const error = ref('')
const activeTab = ref<'stops' | 'segments'>('stops')

// Stop form
const showStopForm = ref(false)
const editingStopId = ref<number | null>(null)
const stopForm = ref({ name: '', code: '', position: 0 })
const stopFormError = ref('')
const stopFormLoading = ref(false)

// Segment form
const showSegmentForm = ref(false)
const segmentForm = ref({ origin_stop_id: 0, dest_stop_id: 0, price: 0 })
const segmentFormError = ref('')
const segmentFormLoading = ref(false)

const selectedRoute = computed(() => routes.value.find(r => r.id === selectedRouteId.value) || null)

const nextPosition = computed(() => {
  if (stops.value.length === 0) return 1
  return Math.max(...stops.value.map(s => s.position)) + 1
})

async function loadRoutes() {
  loading.value = true
  error.value = ''
  try {
    routes.value = await getRoutes()
    // Si viene routeId por URL, seleccionarlo
    const paramId = Number(vueRoute.params.routeId)
    if (paramId && routes.value.some(r => r.id === paramId)) {
      selectedRouteId.value = paramId
    }
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function loadStops() {
  if (!selectedRouteId.value) {
    stops.value = []
    segments.value = []
    return
  }
  loadingStops.value = true
  error.value = ''
  try {
    const [stopsData, segmentsData] = await Promise.all([
      getStops(selectedRouteId.value),
      getSegments(selectedRouteId.value),
    ])
    stops.value = stopsData.sort((a, b) => a.position - b.position)
    segments.value = segmentsData
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loadingStops.value = false
  }
}

watch(selectedRouteId, () => {
  activeTab.value = 'stops'
  loadStops()
})

// ── Stops CRUD ──

function openCreateStop() {
  editingStopId.value = null
  stopForm.value = { name: '', code: '', position: 0 }
  stopFormError.value = ''
  showStopForm.value = true
}

function openEditStop(s: Stop) {
  editingStopId.value = s.id
  stopForm.value = { name: s.name, code: s.code, position: s.position }
  stopFormError.value = ''
  showStopForm.value = true
}

function cancelStopForm() {
  showStopForm.value = false
  editingStopId.value = null
  stopFormError.value = ''
}

async function submitStopForm() {
  const name = stopForm.value.name.trim()
  const code = stopForm.value.code.trim().toUpperCase()
  if (!name || !code) {
    stopFormError.value = 'Nombre y codigo son obligatorios.'
    return
  }
  stopFormLoading.value = true
  stopFormError.value = ''
  try {
    if (editingStopId.value) {
      await updateStop(editingStopId.value, {
        name,
        code,
        position: stopForm.value.position,
        route_id: selectedRouteId.value,
      })
    } else {
      // Al crear, el backend auto-asigna posición
      await createStop(selectedRouteId.value, { name, code, position: 0 })
    }
    showStopForm.value = false
    editingStopId.value = null
    await loadStops()
  } catch (e) {
    stopFormError.value = (e as Error).message
  } finally {
    stopFormLoading.value = false
  }
}

async function handleDeleteStop(s: Stop) {
  const result = await Swal.fire({
    title: 'Eliminar parada?',
    text: `Se eliminara "${s.name}" de esta ruta`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, eliminar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await deleteStop(s.id)
    await loadStops()
    Swal.fire({ title: 'Eliminada', text: 'Parada eliminada correctamente', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function handleReorder() {
  try {
    await reorderStops(selectedRouteId.value)
    await loadStops()
    Swal.fire({ title: 'Listo', text: 'Posiciones corregidas', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    error.value = (e as Error).message
  }
}

// ── Segments CRUD ──

function openCreateSegment() {
  segmentForm.value = { origin_stop_id: 0, dest_stop_id: 0, price: 0 }
  segmentFormError.value = ''
  showSegmentForm.value = true
}

function openEditSegment(seg: RouteSegment) {
  segmentForm.value = {
    origin_stop_id: seg.origin_stop_id,
    dest_stop_id: seg.dest_stop_id,
    price: seg.price,
  }
  segmentFormError.value = ''
  showSegmentForm.value = true
}

function cancelSegmentForm() {
  showSegmentForm.value = false
  segmentFormError.value = ''
}

async function submitSegmentForm() {
  if (!segmentForm.value.origin_stop_id || !segmentForm.value.dest_stop_id) {
    segmentFormError.value = 'Seleccione origen y destino.'
    return
  }
  if (segmentForm.value.origin_stop_id === segmentForm.value.dest_stop_id) {
    segmentFormError.value = 'Origen y destino deben ser diferentes.'
    return
  }
  if (segmentForm.value.price < 0) {
    segmentFormError.value = 'El precio no puede ser negativo.'
    return
  }
  segmentFormLoading.value = true
  segmentFormError.value = ''
  try {
    await upsertSegment(selectedRouteId.value, segmentForm.value)
    showSegmentForm.value = false
    await loadStops()
  } catch (e) {
    segmentFormError.value = (e as Error).message
  } finally {
    segmentFormLoading.value = false
  }
}

async function handleGenerateSegments() {
  try {
    const result = await generateSegments(selectedRouteId.value)
    await loadStops()
    Swal.fire({ title: 'Listo', text: `${result.created} tramos generados. Edita los precios.`, icon: 'success', timer: 2000, showConfirmButton: false })
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function handleDeleteSegment(seg: RouteSegment) {
  const result = await Swal.fire({
    title: 'Eliminar tramo?',
    text: `Se eliminara el tramo ${seg.origin_stop_name} - ${seg.dest_stop_name}`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, eliminar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await deleteSegment(seg.id)
    await loadStops()
    Swal.fire({ title: 'Eliminado', text: 'Tramo eliminado correctamente', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    error.value = (e as Error).message
  }
}

onMounted(loadRoutes)
</script>

<template>
  <div class="admin-page">
    <!-- Header -->
    <div class="page-head">
      <div>
        <h1 class="page-title">Paradas</h1>
        <p class="page-subtitle">Gestiona las paradas y tramos de cada ruta</p>
      </div>
    </div>

    <!-- Selector de ruta -->
    <div class="route-selector">
      <label class="form-label" for="route-select">Selecciona una ruta</label>
      <select id="route-select" v-model.number="selectedRouteId" class="form-input route-select-input">
        <option :value="0" disabled>-- Seleccione una ruta --</option>
        <option v-for="r in routes" :key="r.id" :value="r.id">
          {{ r.name }} ({{ r.code }})
        </option>
      </select>
    </div>

    <!-- Error -->
    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <!-- Loading rutas -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>Cargando rutas...</span>
    </div>

    <!-- No ruta seleccionada -->
    <div v-else-if="!selectedRouteId" class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="empty-icon">
        <circle cx="6" cy="19" r="3"/>
        <path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/>
        <circle cx="18" cy="5" r="3"/>
      </svg>
      <p>Selecciona una ruta para ver y gestionar sus paradas.</p>
    </div>

    <!-- Contenido con ruta seleccionada -->
    <template v-else>
      <!-- Info ruta -->
      <div class="route-info-bar">
        <span class="route-info-name">{{ selectedRoute?.name }}</span>
        <code class="code-badge">{{ selectedRoute?.code }}</code>
        <span class="route-info-price">S/ {{ selectedRoute?.price_per_seat.toFixed(2) }} / asiento</span>
      </div>

      <!-- Tabs -->
      <div class="tabs">
        <button class="tab" :class="{ active: activeTab === 'stops' }" @click="activeTab = 'stops'">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
            <circle cx="12" cy="10" r="3"/>
          </svg>
          Paradas ({{ stops.length }})
        </button>
        <button class="tab" :class="{ active: activeTab === 'segments' }" @click="activeTab = 'segments'">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 2v20"/>
            <path d="m17 5-5-3-5 3"/>
            <path d="m17 19-5 3-5-3"/>
          </svg>
          Tramos / Precios ({{ segments.length }})
        </button>
      </div>

      <!-- Loading stops -->
      <div v-if="loadingStops" class="loading-state">
        <div class="spinner"></div>
        <span>Cargando paradas...</span>
      </div>

      <!-- ═══ STOPS TAB ═══ -->
      <template v-if="!loadingStops && activeTab === 'stops'">
        <div class="section-head">
          <p class="section-hint">Las paradas definen el recorrido de la ruta, en orden de posicion.</p>
          <div class="section-actions">
            <button v-if="stops.length > 0" class="btn btn-ghost" @click="handleReorder" title="Corrige posiciones 1, 2, 3...">
              Reordenar
            </button>
            <button class="btn btn-primary" @click="openCreateStop">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 12h14"/><path d="M12 5v14"/>
            </svg>
            Nueva Parada
          </button>
          </div>
        </div>

        <div v-if="stops.length === 0" class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="empty-icon">
            <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
            <circle cx="12" cy="10" r="3"/>
          </svg>
          <p>No hay paradas en esta ruta.</p>
          <button class="btn btn-primary" @click="openCreateStop">Agregar primera parada</button>
        </div>

        <div v-else class="stops-timeline">
          <div v-for="(s, idx) in stops" :key="s.id" class="stop-card">
            <div class="stop-marker">
              <div class="stop-dot" :class="{ 'dot-first': idx === 0, 'dot-warn': !s.position }">
                {{ s.position || '?' }}
              </div>
              <div v-if="idx < stops.length - 1" class="stop-line"></div>
            </div>
            <div class="stop-body">
              <div class="stop-info">
                <span class="stop-name">{{ s.name }}</span>
                <code class="code-badge">{{ s.code }}</code>
              </div>
              <div class="stop-actions">
                <button class="btn-icon" title="Editar" @click="openEditStop(s)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>
                    <path d="m15 5 4 4"/>
                  </svg>
                </button>
                <button class="btn-icon btn-icon-danger" title="Eliminar" @click="handleDeleteStop(s)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- ═══ SEGMENTS TAB ═══ -->
      <template v-if="!loadingStops && activeTab === 'segments'">
        <div class="section-head">
          <p class="section-hint">Define el precio de cada tramo entre dos paradas.</p>
          <div class="section-actions">
            <button v-if="stops.length >= 2" class="btn btn-ghost" @click="handleGenerateSegments">
              Generar tramos
            </button>
            <button class="btn btn-primary" @click="openCreateSegment" :disabled="stops.length < 2">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M5 12h14"/><path d="M12 5v14"/>
              </svg>
              Nuevo Tramo
            </button>
          </div>
        </div>

        <div v-if="stops.length < 2" class="alert alert-warning">
          Necesitas al menos 2 paradas para crear tramos.
        </div>

        <div v-else-if="segments.length === 0" class="empty-state">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="empty-icon">
            <path d="M12 2v20"/><path d="m17 5-5-3-5 3"/><path d="m17 19-5 3-5-3"/>
          </svg>
          <p>No hay tramos con precio definidos.</p>
          <button class="btn btn-primary" @click="openCreateSegment">Agregar primer tramo</button>
        </div>

        <div v-else class="table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>Origen</th>
                <th>Destino</th>
                <th>Precio</th>
                <th>Acciones</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="seg in segments" :key="seg.id">
                <td class="td-name">{{ seg.origin_stop_name }}</td>
                <td class="td-name">{{ seg.dest_stop_name }}</td>
                <td class="td-price">S/ {{ seg.price.toFixed(2) }}</td>
                <td class="td-actions">
                  <button class="btn-icon" title="Editar precio" @click="openEditSegment(seg)">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/><path d="m15 5 4 4"/>
                    </svg>
                  </button>
                  <button class="btn-icon btn-icon-danger" title="Eliminar" @click="handleDeleteSegment(seg)">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                    </svg>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </template>

    <!-- ═══ STOP MODAL ═══ -->
    <Teleport to="body">
      <div v-if="showStopForm" class="modal-overlay" @click.self="cancelStopForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3 class="modal-title">{{ editingStopId ? 'Editar Parada' : 'Nueva Parada' }}</h3>
            <button class="modal-close" @click="cancelStopForm">&times;</button>
          </div>
          <div v-if="stopFormError" class="alert alert-error">{{ stopFormError }}</div>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="stop-name">Nombre de la parada</label>
              <input id="stop-name" v-model="stopForm.name" type="text" class="form-input" placeholder="Ej: Arequipa" />
            </div>
            <div class="form-group">
              <label class="form-label" for="stop-code">Codigo (3 letras)</label>
              <input id="stop-code" v-model="stopForm.code" type="text" class="form-input" placeholder="Ej: AQP" maxlength="5" style="text-transform: uppercase" />
            </div>
            <div v-if="editingStopId" class="form-group">
              <label class="form-label" for="stop-position">Posicion en la ruta</label>
              <input id="stop-position" v-model.number="stopForm.position" type="number" min="1" class="form-input" />
            </div>
          </div>
          <p v-if="!editingStopId" class="form-hint">La parada se agregara al final del recorrido (posicion {{ nextPosition }}).</p>
          <div class="form-actions">
            <button class="btn btn-ghost" @click="cancelStopForm" :disabled="stopFormLoading">Cancelar</button>
            <button class="btn btn-primary" @click="submitStopForm" :disabled="stopFormLoading">
              <div v-if="stopFormLoading" class="spinner-sm"></div>
              {{ editingStopId ? 'Guardar Cambios' : 'Crear Parada' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- ═══ SEGMENT MODAL ═══ -->
    <Teleport to="body">
      <div v-if="showSegmentForm" class="modal-overlay" @click.self="cancelSegmentForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3 class="modal-title">Tramo / Precio</h3>
            <button class="modal-close" @click="cancelSegmentForm">&times;</button>
          </div>
          <div v-if="segmentFormError" class="alert alert-error">{{ segmentFormError }}</div>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="seg-origin">Parada Origen</label>
              <select id="seg-origin" v-model.number="segmentForm.origin_stop_id" class="form-input">
                <option :value="0" disabled>Seleccione...</option>
                <option v-for="s in stops" :key="s.id" :value="s.id">{{ s.name }} ({{ s.code }})</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label" for="seg-dest">Parada Destino</label>
              <select id="seg-dest" v-model.number="segmentForm.dest_stop_id" class="form-input">
                <option :value="0" disabled>Seleccione...</option>
                <option v-for="s in stops" :key="s.id" :value="s.id">{{ s.name }} ({{ s.code }})</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label" for="seg-price">Precio (S/)</label>
              <input id="seg-price" v-model.number="segmentForm.price" type="number" step="0.01" min="0" class="form-input" placeholder="Ej: 25.00" />
            </div>
          </div>
          <div class="form-actions">
            <button class="btn btn-ghost" @click="cancelSegmentForm" :disabled="segmentFormLoading">Cancelar</button>
            <button class="btn btn-primary" @click="submitSegmentForm" :disabled="segmentFormLoading">
              <div v-if="segmentFormLoading" class="spinner-sm"></div>
              Guardar Tramo
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.admin-page {}

.page-head {
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.page-subtitle {
  font-size: 0.9rem;
  color: var(--slate-500);
  margin-top: 0.25rem;
}

/* ── Route selector ── */
.route-selector {
  margin-bottom: 1.5rem;
}

.route-select-input {
  width: 100%;
  max-width: 400px;
  margin-top: 0.35rem;
}

/* ── Route info bar ── */
.route-info-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: var(--brand-50);
  border: 1px solid var(--brand-200);
  border-radius: var(--radius-md);
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.route-info-name {
  font-weight: 700;
  color: var(--slate-900);
  font-size: 0.95rem;
}

.route-info-price {
  margin-left: auto;
  font-weight: 600;
  color: var(--success-600);
  font-size: 0.88rem;
}

/* ── Tabs ── */
.tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  border-bottom: 2px solid var(--slate-200);
  padding-bottom: 0;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  background: none;
  border: none;
  border-bottom: 3px solid transparent;
  margin-bottom: -2px;
  font-size: 0.88rem;
  font-weight: 600;
  color: var(--slate-500);
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s ease;
}

.tab:hover {
  color: var(--brand-500);
}

.tab.active {
  color: var(--brand-600);
  border-bottom-color: var(--brand-500);
}

/* ── Section head ── */
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.section-hint {
  font-size: 0.85rem;
  color: var(--slate-500);
}

.section-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

/* ── Stops Timeline ── */
.stops-timeline {
  display: flex;
  flex-direction: column;
}

.stop-card {
  display: flex;
  gap: 1rem;
  align-items: stretch;
}

.stop-marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 36px;
  flex-shrink: 0;
}

.stop-dot {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--brand-100);
  color: var(--brand-600);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 700;
  border: 2px solid var(--brand-300);
  flex-shrink: 0;
  z-index: 1;
}

.dot-first {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-color: var(--brand-500);
}

.dot-last {
  background: linear-gradient(135deg, var(--success-400), var(--success-500));
  color: white;
  border-color: var(--success-500);
}

.dot-warn {
  background: #fbbf24;
  color: #78350f;
  border-color: #f59e0b;
  font-weight: 800;
}

.stop-line {
  width: 2px;
  flex: 1;
  background: var(--slate-300);
  min-height: 16px;
}

.stop-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-md);
  padding: 0.75rem 1rem;
  margin-bottom: 0.5rem;
  box-shadow: var(--shadow-sm);
  transition: all 0.15s ease;
}

.stop-body:hover {
  border-color: var(--brand-200);
  box-shadow: var(--shadow-md);
}

.stop-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.stop-name {
  font-weight: 700;
  color: var(--slate-900);
  font-size: 0.92rem;
}

.stop-actions {
  display: flex;
  gap: 0.4rem;
}

/* ── Empty state ── */
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--slate-500);
}

.empty-icon {
  color: var(--slate-300);
  margin-bottom: 1rem;
}

.empty-state p {
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

/* ── Buttons ── */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-weight: 600;
  font-size: 0.88rem;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;
  line-height: 1;
  padding: 0.6rem 1.15rem;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-primary:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--brand-500), var(--brand-600));
  box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
  transform: translateY(-1px);
}

.btn-ghost {
  background: transparent;
  color: var(--slate-500);
  border: 2px solid var(--slate-300);
}

.btn-ghost:hover:not(:disabled) {
  background: var(--slate-100);
  color: var(--slate-900);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--slate-500);
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-icon:hover {
  background: var(--slate-100);
  color: var(--slate-900);
  border-color: var(--slate-400);
}

.btn-icon-danger:hover {
  background: var(--danger-50);
  color: var(--danger-600);
  border-color: var(--danger-100);
}

/* ── Form ── */
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-600);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.form-input {
  padding: 0.65rem 0.85rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  transition: all 0.2s ease;
  outline: none;
}

.form-input:focus {
  border-color: var(--brand-400);
  background: white;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.form-input::placeholder {
  color: var(--slate-400);
}

.form-hint {
  font-size: 0.82rem;
  color: var(--slate-500);
  margin-bottom: 1rem;
}

.form-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
}

/* ── Alert ── */
.alert {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  margin-bottom: 1rem;
}

.alert-error {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100);
}

.alert-warning {
  background: #fffbeb;
  color: #92400e;
  border: 1px solid #fde68a;
}

/* ── Loading ── */
.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 0;
  justify-content: center;
  color: var(--slate-500);
  font-size: 0.9rem;
}

.spinner {
  width: 22px;
  height: 22px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner-sm {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Table ── */
.table-wrapper {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-md);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
}

.data-table thead {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
}

.data-table th {
  padding: 0.75rem 1rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.75rem;
  color: white;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.data-table td {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--slate-100);
  color: var(--slate-700);
}

.data-table tbody tr:last-child td {
  border-bottom: none;
}

.data-table tbody tr:hover {
  background: var(--brand-50);
}

.td-name {
  font-weight: 600;
  color: var(--slate-900);
}

.td-price {
  font-weight: 700;
  color: var(--success-600);
  white-space: nowrap;
}

.td-actions {
  display: flex;
  gap: 0.4rem;
}

.code-badge {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  background: var(--slate-100);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  font-family: 'SF Mono', 'Fira Code', monospace;
}

/* ── Modal ── */
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

/* ── Responsive ── */
@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .section-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .tabs {
    overflow-x: auto;
  }

  .route-info-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .route-info-price {
    margin-left: 0;
  }
}
</style>
