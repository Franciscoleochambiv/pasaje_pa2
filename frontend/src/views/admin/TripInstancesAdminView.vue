<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getTripInstances, createTripInstance, updateTripInstance, deleteTripInstance,
  getTripTemplates,
  type TripInstance, type TripTemplate
} from '../../api/client'
import Swal from 'sweetalert2'

const instances = ref<TripInstance[]>([])
const templates = ref<TripTemplate[]>([])
const loading = ref(false)
const error = ref('')
const showForm = ref(false)
const editingId = ref<number | null>(null)

const form = ref({ trip_template_id: 0, departure_at: '' })
const formError = ref('')
const formLoading = ref(false)

function statusLabel(s: string) {
  const map: Record<string, string> = {
    scheduled: 'Programado',
    boarding: 'Abordando',
    departed: 'En ruta',
    arrived: 'Llegado',
    cancelled: 'Cancelado',
  }
  return map[s] || s
}

function statusClass(s: string) {
  if (s === 'scheduled') return 'badge-success'
  if (s === 'departed' || s === 'boarding') return 'badge-warning'
  if (s === 'arrived') return 'badge-info'
  if (s === 'cancelled') return 'badge-danger'
  return 'badge-muted'
}

function formatDatetime(iso: string) {
  const d = new Date(iso)
  return d.toLocaleString('es-PE', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [i, t] = await Promise.all([getTripInstances(), getTripTemplates()])
    instances.value = i
    templates.value = t
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = { trip_template_id: 0, departure_at: '' }
  formError.value = ''
  showForm.value = true
}

function cancelForm() {
  showForm.value = false
  editingId.value = null
  formError.value = ''
}

async function submitForm() {
  if (!form.value.trip_template_id) {
    formError.value = 'Selecciona una plantilla.'
    return
  }
  if (!form.value.departure_at) {
    formError.value = 'Ingresa la fecha y hora de salida.'
    return
  }
  formLoading.value = true
  formError.value = ''
  try {
    const departureIso = new Date(form.value.departure_at).toISOString()
    await createTripInstance({ trip_template_id: form.value.trip_template_id, departure_at: departureIso })
    showForm.value = false
    editingId.value = null
    await load()
  } catch (e) {
    formError.value = (e as Error).message
  } finally {
    formLoading.value = false
  }
}

async function handleChangeStatus(inst: TripInstance) {
  const statusOptions: Record<string, string> = {
    scheduled: 'Programado',
    boarding: 'Abordando',
    departed: 'En ruta',
    arrived: 'Llegado',
    cancelled: 'Cancelado',
  }
  const inputOptions: Record<string, string> = {}
  for (const [key, label] of Object.entries(statusOptions)) {
    if (key !== inst.status) {
      inputOptions[key] = label
    }
  }
  const result = await Swal.fire({
    title: 'Cambiar estado',
    html: `<p style="color:#64748b;font-size:0.9rem;margin-bottom:0.5rem;">Viaje: <strong>${inst.route_name}</strong></p>
           <p style="color:#64748b;font-size:0.85rem;">Estado actual: <span style="font-weight:700;color:#0f172a;">${statusLabel(inst.status)}</span></p>`,
    input: 'select',
    inputOptions,
    inputPlaceholder: 'Seleccionar nuevo estado',
    showCancelButton: true,
    confirmButtonColor: '#3B82F6',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Cambiar estado',
    cancelButtonText: 'Cancelar',
    customClass: {
      popup: 'swal-custom-popup',
      input: 'swal-select-premium',
      confirmButton: 'swal-btn-confirm',
      cancelButton: 'swal-btn-cancel',
    },
    inputValidator: (value: string) => {
      if (!value) return 'Debes seleccionar un estado'
      return null
    }
  })
  if (result.isConfirmed && result.value) {
    try {
      await updateTripInstance(inst.id, { status: result.value as string })
      await load()
      Swal.fire({ title: 'Estado actualizado', icon: 'success', timer: 1500, showConfirmButton: false })
    } catch (e) {
      error.value = (e as Error).message
    }
  }
}

async function handleEditDeparture(inst: TripInstance) {
  // Convert to local datetime-local format
  const d = new Date(inst.departure_at)
  const localStr = d.getFullYear() + '-' +
    String(d.getMonth() + 1).padStart(2, '0') + '-' +
    String(d.getDate()).padStart(2, '0') + 'T' +
    String(d.getHours()).padStart(2, '0') + ':' +
    String(d.getMinutes()).padStart(2, '0')

  const result = await Swal.fire({
    title: 'Cambiar hora de salida',
    html: `<p style="color:#64748b;font-size:0.9rem;margin-bottom:0.5rem;">Viaje: <strong>${inst.route_name}</strong></p>
           <p style="color:#64748b;font-size:0.85rem;margin-bottom:1rem;">Hora actual: <strong>${formatDatetime(inst.departure_at)}</strong></p>
           <input type="datetime-local" id="swal-departure" value="${localStr}" style="display:block;width:100%;box-sizing:border-box;padding:0.75rem 1rem;border:2px solid #CBD5E1;border-radius:12px;font-size:0.95rem;font-weight:500;font-family:inherit;background:#F8FAFC;color:#0F172A;outline:none;margin-top:0.5rem;">`,
    showCancelButton: true,
    confirmButtonColor: '#3B82F6',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Guardar cambio',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup', confirmButton: 'swal-btn-confirm', cancelButton: 'swal-btn-cancel' },
    preConfirm: () => {
      const input = document.getElementById('swal-departure') as HTMLInputElement
      if (!input?.value) {
        Swal.showValidationMessage('Ingresa una fecha y hora')
        return false
      }
      return input.value
    }
  })
  if (result.isConfirmed && result.value) {
    try {
      const iso = new Date(result.value as string).toISOString()
      await updateTripInstance(inst.id, { departure_at: iso })
      await load()
      Swal.fire({ title: 'Hora actualizada', icon: 'success', timer: 1500, showConfirmButton: false })
    } catch (e) {
      error.value = (e as Error).message
    }
  }
}

async function handleDelete(t: TripInstance) {
  const result = await Swal.fire({
    title: 'Cancelar viaje?',
    text: `Se cancelara el viaje ${t.route_name} del ${formatDatetime(t.departure_at)}`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, cancelar viaje',
    cancelButtonText: 'No',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await deleteTripInstance(t.id)
    await load()
    Swal.fire({ title: 'Cancelado', text: 'Viaje cancelado correctamente', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    error.value = (e as Error).message
  }
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Viajes (Salidas)</h1>
        <p class="page-subtitle">Genera y administra las instancias de viaje</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 12h14"/>
          <path d="M12 5v14"/>
        </svg>
        Generar Salida
      </button>
    </div>

    <!-- Modal Form -->
    <Teleport to="body">
      <div v-if="showForm" class="modal-overlay" @click.self="cancelForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3 class="modal-title">{{ editingId ? 'Editar Salida' : 'Generar Salida' }}</h3>
            <button class="modal-close" @click="cancelForm">&times;</button>
          </div>
          <div v-if="formError" class="alert alert-error">{{ formError }}</div>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="inst-tpl">Plantilla</label>
              <div class="custom-select-wrapper">
                <select id="inst-tpl" v-model.number="form.trip_template_id" class="custom-select">
                  <option :value="0" disabled>Seleccionar plantilla...</option>
                  <option v-for="t in templates" :key="t.id" :value="t.id">
                    {{ t.name }} ({{ t.route_name || `Ruta #${t.route_id}` }})
                  </option>
                </select>
                <div class="custom-select-icon">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
                </div>
              </div>
            </div>
            <div class="form-group">
              <label class="form-label" for="inst-dt">Fecha y Hora de Salida</label>
              <input id="inst-dt" v-model="form.departure_at" type="datetime-local" class="form-input" />
            </div>
          </div>
          <div class="form-actions">
            <button class="btn btn-ghost" @click="cancelForm" :disabled="formLoading">Cancelar</button>
            <button class="btn btn-primary" @click="submitForm" :disabled="formLoading">
              <div v-if="formLoading" class="spinner-sm"></div>
              {{ editingId ? 'Guardar Cambios' : 'Generar Salida' }}
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
            <th>Ruta</th>
            <th>Fecha/Hora Salida</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="inst in instances" :key="inst.id">
            <td class="td-id">{{ inst.id }}</td>
            <td class="td-name">{{ inst.route_name || `Ruta #${inst.route_id}` }}</td>
            <td>{{ formatDatetime(inst.departure_at) }}</td>
            <td>
              <span class="status-badge" :class="statusClass(inst.status)">
                {{ statusLabel(inst.status) }}
              </span>
            </td>
            <td class="td-actions">
              <button class="btn-icon" title="Cambiar hora de salida" @click="handleEditDeparture(inst)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                </svg>
              </button>
              <button class="btn-icon" title="Cambiar estado" @click="handleChangeStatus(inst)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>
                  <path d="m15 5 4 4"/>
                </svg>
              </button>
              <button class="btn-icon btn-icon-danger" title="Cancelar viaje" @click="handleDelete(inst)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M3 6h18"/>
                  <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                  <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="instances.length === 0">
            <td colspan="5" class="td-empty">No hay salidas registradas.</td>
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
  width: 100%;
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
  box-sizing: border-box;
}

.form-input:focus {
  border-color: var(--brand-400);
  background: white;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.form-input::placeholder {
  color: var(--slate-400);
}

/* ── Custom Select (premium look) ── */
.custom-select-wrapper {
  position: relative;
  width: 100%;
}

.custom-select {
  width: 100%;
  padding: 0.65rem 2.5rem 0.65rem 0.85rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  transition: all 0.2s ease;
  outline: none;
  box-sizing: border-box;
  appearance: none;
  -webkit-appearance: none;
  cursor: pointer;
}

.custom-select:focus {
  border-color: var(--brand-400);
  background: white;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.custom-select:hover:not(:focus) {
  border-color: var(--slate-400);
}

.custom-select-icon {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--slate-400);
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  background: var(--slate-100);
  border-radius: 4px;
  transition: all 0.2s ease;
}

.custom-select:focus + .custom-select-icon {
  color: var(--brand-500);
  background: var(--brand-50);
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

.badge-warning {
  background: var(--warning-50);
  color: var(--warning-700);
}

.badge-info {
  background: var(--info-50);
  color: var(--info-600);
}

.badge-danger {
  background: var(--danger-50);
  color: var(--danger-700);
}

.badge-muted {
  background: var(--slate-100);
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
  max-width: 540px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid var(--slate-200);
  animation: modalIn 0.2s ease;
  overflow: hidden;
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
