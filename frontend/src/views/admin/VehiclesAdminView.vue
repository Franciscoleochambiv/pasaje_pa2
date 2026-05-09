<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getVehicles, createVehicle, updateVehicle, deleteVehicle, getVehicleSeats,
  getBusLayouts,
  type Vehicle, type VehicleSeat, type BusLayout,
} from '../../api/client'
import Swal from 'sweetalert2'

const vehicles = ref<Vehicle[]>([])
const layouts = ref<BusLayout[]>([])
const loading = ref(false)
const error = ref('')
const showForm = ref(false)
const editingId = ref<number | null>(null)

const form = ref<{ plate: string; name: string; capacity: number; seat_count: number; layout_id: number | null }>({
  plate: '', name: '', capacity: 40, seat_count: 40, layout_id: null,
})
const formError = ref('')
const formLoading = ref(false)

const expandedId = ref<number | null>(null)
const expandedSeats = ref<VehicleSeat[]>([])
const seatsLoading = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [v, l] = await Promise.all([getVehicles(), getBusLayouts().catch(() => [])])
    vehicles.value = v
    layouts.value = l
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = { plate: '', name: '', capacity: 40, seat_count: 40, layout_id: null }
  formError.value = ''
  showForm.value = true
}

function openEdit(v: Vehicle) {
  editingId.value = v.id
  form.value = { plate: v.plate, name: v.name, capacity: v.capacity, seat_count: v.capacity, layout_id: v.layout_id ?? null }
  formError.value = ''
  showForm.value = true
}

function cancelForm() {
  showForm.value = false
  editingId.value = null
  formError.value = ''
}

async function submitForm() {
  if (!form.value.plate.trim() || !form.value.name.trim()) {
    formError.value = 'Placa y nombre son obligatorios.'
    return
  }
  if (!form.value.layout_id && form.value.seat_count < 1) {
    formError.value = 'Asignar una plantilla o indicar al menos 1 asiento.'
    return
  }
  formLoading.value = true
  formError.value = ''
  try {
    if (editingId.value) {
      await updateVehicle(editingId.value, {
        plate: form.value.plate, name: form.value.name, capacity: form.value.capacity,
        layout_id: form.value.layout_id ?? undefined,
      })
    } else {
      await createVehicle({
        plate: form.value.plate, name: form.value.name, capacity: form.value.capacity,
        seat_count: form.value.layout_id ? undefined : form.value.seat_count,
        layout_id: form.value.layout_id ?? undefined,
      })
    }
    showForm.value = false
    editingId.value = null
    await load()
  } catch (e) {
    formError.value = (e as Error).message
  } finally {
    formLoading.value = false
  }
}

async function handleDelete(v: Vehicle) {
  const result = await Swal.fire({
    title: 'Desactivar vehiculo?',
    text: `Se desactivara "${v.name || v.plate}"`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, desactivar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' }
  })
  if (result.isConfirmed) {
    try {
      await deleteVehicle(v.id)
      await load()
      Swal.fire({ title: 'Desactivado', text: 'Vehiculo desactivado', icon: 'success', timer: 1500, showConfirmButton: false })
    } catch (e) { error.value = (e as Error).message }
  }
}

async function toggleSeats(v: Vehicle) {
  if (expandedId.value === v.id) {
    expandedId.value = null
    expandedSeats.value = []
    return
  }
  expandedId.value = v.id
  seatsLoading.value = true
  try {
    expandedSeats.value = await getVehicleSeats(v.id)
  } catch (e) {
    expandedSeats.value = []
  } finally {
    seatsLoading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Vehiculos</h1>
        <p class="page-subtitle">Administra la flota de vehiculos</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 12h14"/>
          <path d="M12 5v14"/>
        </svg>
        Nuevo Vehiculo
      </button>
    </div>

    <!-- Modal Form -->
    <Teleport to="body">
      <div v-if="showForm" class="modal-overlay" @click.self="cancelForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3 class="modal-title">{{ editingId ? 'Editar Vehiculo' : 'Nuevo Vehiculo' }}</h3>
            <button class="modal-close" @click="cancelForm">&times;</button>
          </div>
          <div v-if="formError" class="alert alert-error">{{ formError }}</div>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="v-plate">Placa</label>
              <input id="v-plate" v-model="form.plate" type="text" class="form-input" placeholder="Ej: ABC-123" />
            </div>
            <div class="form-group">
              <label class="form-label" for="v-name">Nombre</label>
              <input id="v-name" v-model="form.name" type="text" class="form-input" placeholder="Ej: Bus Ejecutivo 01" />
            </div>
            <div class="form-group">
              <label class="form-label" for="v-capacity">Capacidad</label>
              <input id="v-capacity" v-model.number="form.capacity" type="number" min="1" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label" for="v-layout">Plantilla de bus</label>
              <select id="v-layout" v-model="form.layout_id" class="form-input">
                <option :value="null">— sin plantilla (asientos 1..N) —</option>
                <option v-for="l in layouts" :key="l.id" :value="l.id">{{ l.name }}</option>
              </select>
              <span class="form-hint">
                {{ editingId
                  ? 'Cambiar la plantilla re-clona los asientos (perdés cualquier ajuste manual previo).'
                  : 'Si elegís una plantilla, el bus hereda su layout completo.' }}
              </span>
            </div>
            <div v-if="!editingId && !form.layout_id" class="form-group">
              <label class="form-label" for="v-seats">Numero de Asientos</label>
              <input id="v-seats" v-model.number="form.seat_count" type="number" min="1" class="form-input" />
              <span class="form-hint">Se generan etiquetas 1..N automaticamente</span>
            </div>
          </div>
          <div class="form-actions">
            <button class="btn btn-ghost" @click="cancelForm" :disabled="formLoading">Cancelar</button>
            <button class="btn btn-primary" @click="submitForm" :disabled="formLoading">
              <div v-if="formLoading" class="spinner-sm"></div>
              {{ editingId ? 'Guardar Cambios' : 'Crear Vehiculo' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Error -->
    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>Cargando...</span>
    </div>

    <!-- Table -->
    <div v-else class="table-wrapper">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Placa</th>
            <th>Nombre</th>
            <th>Capacidad</th>
            <th>Estado</th>
            <th>Asientos</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="v in vehicles" :key="v.id">
            <tr>
              <td class="td-id">{{ v.id }}</td>
              <td><code class="code-badge">{{ v.plate }}</code></td>
              <td class="td-name">{{ v.name }}</td>
              <td>{{ v.capacity }}</td>
              <td>
                <span class="status-badge" :class="v.active ? 'badge-success' : 'badge-muted'">
                  {{ v.active ? 'Activo' : 'Inactivo' }}
                </span>
              </td>
              <td>
                <button class="btn-icon" title="Ver asientos" @click="toggleSeats(v)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect width="18" height="18" x="3" y="3" rx="2"/>
                    <path d="M3 9h18"/>
                    <path d="M9 21V9"/>
                  </svg>
                </button>
              </td>
              <td class="td-actions">
                <button class="btn-icon" title="Editar" @click="openEdit(v)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>
                    <path d="m15 5 4 4"/>
                  </svg>
                </button>
                <button class="btn-icon btn-icon-danger" title="Desactivar" @click="handleDelete(v)">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18"/>
                    <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                    <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                  </svg>
                </button>
              </td>
            </tr>
            <!-- Expanded seats row -->
            <tr v-if="expandedId === v.id" class="expanded-row">
              <td colspan="7">
                <div v-if="seatsLoading" class="seats-loading">
                  <div class="spinner-sm"></div>
                  <span>Cargando asientos...</span>
                </div>
                <div v-else-if="expandedSeats.length" class="seats-preview">
                  <span
                    v-for="seat in expandedSeats"
                    :key="seat.id"
                    class="seat-chip"
                  >{{ seat.label }}</span>
                </div>
                <div v-else class="seats-empty">Sin asientos registrados.</div>
              </td>
            </tr>
          </template>
          <tr v-if="vehicles.length === 0">
            <td colspan="7" class="td-empty">No hay vehiculos registrados.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.admin-page {}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
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
  font-size: 0.75rem;
  color: var(--slate-500);
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
  border: 3px solid var(--slate-200);
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
  border-bottom: none;
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

.td-id {
  font-weight: 600;
  color: var(--slate-500);
  font-size: 0.82rem;
}

.td-name {
  font-weight: 600;
  color: var(--slate-900);
}

.td-actions {
  display: flex;
  gap: 0.4rem;
}

.td-empty {
  text-align: center;
  color: var(--slate-500);
  padding: 2rem 1rem !important;
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

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.badge-success {
  background: var(--success-50);
  color: var(--success-700);
}

.badge-muted {
  background: var(--slate-100);
  color: var(--slate-500);
}

/* ── Expanded Seats ── */
.expanded-row td {
  padding: 1rem !important;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
}

.expanded-row:hover {
  background: var(--slate-50) !important;
}

.seats-loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--slate-500);
  font-size: 0.85rem;
}

.seats-loading .spinner-sm {
  border-color: var(--slate-200);
  border-top-color: var(--color-primary);
}

.seats-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.seat-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 32px;
  background: white;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-sm);
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-900);
}

.seats-empty {
  font-size: 0.85rem;
  color: var(--slate-500);
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

  .page-head {
    flex-direction: column;
  }

  .table-wrapper {
    overflow-x: auto;
  }
}
</style>
