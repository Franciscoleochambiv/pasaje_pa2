<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getRoutes, createRoute, updateRoute, deleteRoute, type Route } from '../../api/client'
import Swal from 'sweetalert2'

const router = useRouter()

const routes = ref<Route[]>([])
const loading = ref(false)
const error = ref('')
const showForm = ref(false)
const editingId = ref<number | null>(null)

const form = ref({ name: '', code: '', price_per_seat: 0 })
const formError = ref('')
const formLoading = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    routes.value = await getRoutes()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.value = { name: '', code: '', price_per_seat: 0 }
  formError.value = ''
  showForm.value = true
}

function openEdit(r: Route) {
  editingId.value = r.id
  form.value = { name: r.name, code: r.code, price_per_seat: r.price_per_seat }
  formError.value = ''
  showForm.value = true
}

function cancelForm() {
  showForm.value = false
  editingId.value = null
  formError.value = ''
}

async function submitForm() {
  if (!form.value.name.trim() || !form.value.code.trim()) {
    formError.value = 'Todos los campos son obligatorios.'
    return
  }
  formLoading.value = true
  formError.value = ''
  try {
    if (editingId.value) {
      await updateRoute(editingId.value, form.value)
    } else {
      await createRoute(form.value)
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

async function handleDelete(r: Route) {
  const result = await Swal.fire({
    title: 'Desactivar ruta?',
    text: `Se desactivara "${r.name}"`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, desactivar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await deleteRoute(r.id)
    await load()
    Swal.fire({ title: 'Desactivada', text: 'Ruta desactivada correctamente', icon: 'success', timer: 1500, showConfirmButton: false })
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
        <h1 class="page-title">Rutas</h1>
        <p class="page-subtitle">Administra las rutas de transporte</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 12h14"/>
          <path d="M12 5v14"/>
        </svg>
        Nueva Ruta
      </button>
    </div>

    <!-- Modal Form -->
    <Teleport to="body">
      <div v-if="showForm" class="modal-overlay" @click.self="cancelForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3 class="modal-title">{{ editingId ? 'Editar Ruta' : 'Nueva Ruta' }}</h3>
            <button class="modal-close" @click="cancelForm">&times;</button>
          </div>
          <div v-if="formError" class="alert alert-error">{{ formError }}</div>
          <div class="form-grid">
            <div class="form-group">
              <label class="form-label" for="route-name">Nombre</label>
              <input id="route-name" v-model="form.name" type="text" class="form-input" placeholder="Ej: Arequipa - Cusco" />
            </div>
            <div class="form-group">
              <label class="form-label" for="route-code">Codigo</label>
              <input id="route-code" v-model="form.code" type="text" class="form-input" placeholder="Ej: AQP-CUS" />
            </div>
            <div class="form-group">
              <label class="form-label" for="route-price">Precio por asiento (S/)</label>
              <input id="route-price" v-model.number="form.price_per_seat" type="number" step="0.01" min="0" class="form-input" placeholder="Ej: 85.00" />
            </div>
          </div>
          <div class="form-actions">
            <button class="btn btn-ghost" @click="cancelForm" :disabled="formLoading">Cancelar</button>
            <button class="btn btn-primary" @click="submitForm" :disabled="formLoading">
              <div v-if="formLoading" class="spinner-sm"></div>
              {{ editingId ? 'Guardar Cambios' : 'Crear Ruta' }}
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
            <th>Nombre</th>
            <th>Codigo</th>
            <th>Precio</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in routes" :key="r.id">
            <td class="td-id">{{ r.id }}</td>
            <td class="td-name">{{ r.name }}</td>
            <td><code class="code-badge">{{ r.code }}</code></td>
            <td class="td-price">S/ {{ r.price_per_seat.toFixed(2) }}</td>
            <td>
              <span class="status-badge" :class="r.active ? 'badge-success' : 'badge-muted'">
                {{ r.active ? 'Activa' : 'Inactiva' }}
              </span>
            </td>
            <td class="td-actions">
              <button class="btn-icon btn-icon-stops" title="Paradas" @click="router.push({ name: 'admin-stops', params: { routeId: r.id } })">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
                  <circle cx="12" cy="10" r="3"/>
                </svg>
              </button>
              <button class="btn-icon" title="Editar" @click="openEdit(r)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/>
                  <path d="m15 5 4 4"/>
                </svg>
              </button>
              <button class="btn-icon btn-icon-danger" title="Desactivar" @click="handleDelete(r)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M3 6h18"/>
                  <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                  <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="routes.length === 0">
            <td colspan="6" class="td-empty">No hay rutas registradas.</td>
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

.btn-icon-stops {
  border-color: var(--brand-200);
  color: var(--brand-500);
}

.btn-icon-stops:hover {
  background: var(--brand-50);
  color: var(--brand-600);
  border-color: var(--brand-300);
}

/* ── Form ── */
.form-card {
  background: white;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.form-title {
  font-size: 1rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 1rem;
}

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

/* ── Price ── */
.td-price {
  font-weight: 700;
  color: var(--success-600);
  white-space: nowrap;
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
