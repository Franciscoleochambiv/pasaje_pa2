<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Swal from 'sweetalert2'
import {
  getBusLayouts, createBusLayout, deleteBusLayout,
  getBusLayoutFull, saveBusLayoutFull,
  type BusLayout, type BusLayoutFull,
} from '../../api/client'
import SeatLayoutEditor from '../../components/SeatLayoutEditor.vue'

const layouts = ref<BusLayout[]>([])
const loading = ref(false)
const error = ref('')

const showCreate = ref(false)
const createForm = ref({
  name: '', brand: '', model: '', description: '',
  floors: 2, layout_cols: 13, seat_type: 'regular',
  preview_image_url: '',
})
const createError = ref('')
const createLoading = ref(false)

const editing = ref<BusLayoutFull | null>(null)
const editingLoading = ref(false)
const savingLayout = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    layouts.value = await getBusLayouts()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.value = { name: '', brand: '', model: '', description: '', floors: 1, layout_cols: 4, seat_type: 'regular', preview_image_url: '' }
  createError.value = ''
  showCreate.value = true
}

async function submitCreate() {
  if (!createForm.value.name.trim()) {
    createError.value = 'El nombre es obligatorio.'
    return
  }
  createLoading.value = true
  createError.value = ''
  try {
    const { id } = await createBusLayout(createForm.value)
    showCreate.value = false
    await load()
    await openEditor(id)
  } catch (e) {
    createError.value = (e as Error).message
  } finally {
    createLoading.value = false
  }
}

async function openEditor(id: number) {
  editingLoading.value = true
  try {
    editing.value = await getBusLayoutFull(id)
  } catch (e) {
    Swal.fire({ icon: 'error', title: 'No se pudo cargar', text: (e as Error).message })
  } finally {
    editingLoading.value = false
  }
}

async function handleSave(payload: BusLayoutFull) {
  if (!editing.value) return
  savingLayout.value = true
  try {
    await saveBusLayoutFull(editing.value.layout.id, payload)
    Swal.fire({ icon: 'success', title: 'Plantilla guardada', timer: 1500, showConfirmButton: false })
    editing.value = null
    await load()
  } catch (e) {
    Swal.fire({ icon: 'error', title: 'Error al guardar', text: (e as Error).message })
  } finally {
    savingLayout.value = false
  }
}

async function handleDelete(l: BusLayout) {
  const result = await Swal.fire({
    title: 'Eliminar plantilla?',
    text: `Se borrará "${l.name}". Los buses generados desde esta plantilla NO se ven afectados.`,
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    confirmButtonText: 'Si, eliminar',
    cancelButtonText: 'Cancelar',
  })
  if (!result.isConfirmed) return
  try {
    await deleteBusLayout(l.id)
    await load()
  } catch (e) {
    Swal.fire({ icon: 'error', title: 'No se pudo eliminar', text: (e as Error).message })
  }
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Plantillas de Bus</h1>
        <p class="page-subtitle">Define el modelo de cada bus una sola vez y reutilízalo al crear vehículos.</p>
      </div>
      <button class="btn btn-primary" @click="openCreate">+ Nueva Plantilla</button>
    </div>

    <div v-if="error" class="alert alert-error">{{ error }}</div>

    <div v-if="loading" class="loading-state"><div class="spinner"></div><span>Cargando...</span></div>

    <div v-else-if="layouts.length === 0" class="empty">
      No hay plantillas todavía. Creá la primera para empezar.
    </div>

    <div v-else class="grid-cards">
      <div v-for="l in layouts" :key="l.id" class="card">
        <div class="card-thumb">
          <img v-if="l.preview_image_url" :src="l.preview_image_url" alt="Referencia" />
          <div v-else class="card-thumb-placeholder">🚌</div>
        </div>
        <div class="card-body">
          <h3 class="card-title">{{ l.name }}</h3>
          <p v-if="l.brand || l.model" class="card-subtitle">{{ l.brand }} {{ l.model }}</p>
          <div class="card-meta">
            <span>{{ l.floors }} {{ l.floors === 1 ? 'piso' : 'pisos' }}</span>
            <span>·</span>
            <span>{{ l.layout_cols }} cols</span>
            <span>·</span>
            <span>{{ l.seat_type }}</span>
          </div>
          <p v-if="l.description" class="card-desc">{{ l.description }}</p>
        </div>
        <div class="card-actions">
          <button class="btn-ghost" @click="openEditor(l.id)">Editar</button>
          <button class="btn-icon-danger" title="Eliminar" @click="handleDelete(l)">✕</button>
        </div>
      </div>
    </div>

    <!-- Modal: nueva plantilla -->
    <Teleport to="body">
      <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
        <div class="modal-content">
          <h3 class="modal-title">Nueva plantilla</h3>
          <div v-if="createError" class="alert alert-error">{{ createError }}</div>
          <div class="form-group">
            <label class="form-label">Nombre</label>
            <input v-model="createForm.name" type="text" class="form-input" placeholder="Ej. Scania K410 - 58 asientos doble piso" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Marca</label>
              <input v-model="createForm.brand" type="text" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label">Modelo</label>
              <input v-model="createForm.model" type="text" class="form-input" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Pisos</label>
              <select v-model.number="createForm.floors" class="form-input">
                <option :value="1">1 piso</option>
                <option :value="2">2 pisos</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">Columnas</label>
              <input v-model.number="createForm.layout_cols" type="number" min="1" max="20" class="form-input" />
            </div>
            <div class="form-group">
              <label class="form-label">Tipo</label>
              <select v-model="createForm.seat_type" class="form-input">
                <option value="regular">Regular</option>
                <option value="semi_cama">Semi cama</option>
                <option value="cama">Cama</option>
                <option value="suite">Suite</option>
              </select>
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Descripción (opcional)</label>
            <input v-model="createForm.description" type="text" class="form-input" />
          </div>
          <div class="form-group">
            <label class="form-label">Imagen referencia URL (opcional)</label>
            <input v-model="createForm.preview_image_url" type="text" class="form-input" placeholder="https://..." />
          </div>
          <div class="modal-actions">
            <button class="btn-ghost" @click="showCreate = false" :disabled="createLoading">Cancelar</button>
            <button class="btn-primary" @click="submitCreate" :disabled="createLoading">
              {{ createLoading ? 'Creando...' : 'Crear y editar layout' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Modal grande: editor visual -->
    <Teleport to="body">
      <div v-if="editing" class="editor-overlay" @click.self="editing = null">
        <div class="editor-shell">
          <div class="editor-shell-head">
            <h3>Editor de layout — {{ editing.layout.name }}</h3>
            <button class="modal-close" @click="editing = null">&times;</button>
          </div>
          <div class="editor-shell-body">
            <SeatLayoutEditor
              :layout="editing.layout"
              :seats="editing.seats"
              :elements="editing.elements"
              @save="(p) => handleSave({ layout: p.layout, seats: p.seats, elements: p.elements })"
              @cancel="editing = null"
            />
          </div>
        </div>
      </div>
      <div v-if="savingLayout" class="loading-overlay"><div class="spinner"></div><span style="margin-left: 0.7rem; color: white">Guardando...</span></div>
      <div v-if="editingLoading && !editing" class="loading-overlay"><div class="spinner"></div><span style="margin-left: 0.7rem; color: white">Cargando layout...</span></div>
    </Teleport>
  </div>
</template>

<style scoped>
.admin-page {}
.page-head {
  display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem;
  margin-bottom: 1.5rem; flex-wrap: wrap;
}
.page-title { font-size: 1.5rem; font-weight: 800; color: var(--slate-900); letter-spacing: -0.02em; }
.page-subtitle { font-size: 0.9rem; color: var(--slate-500); margin-top: 0.25rem; }

.btn { display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; font-weight: 700; font-size: 0.88rem; border: none; border-radius: 8px; cursor: pointer; padding: 0.6rem 1.15rem; }
.btn-primary { background: linear-gradient(135deg, var(--brand-400), var(--brand-500)); color: white; box-shadow: 0 4px 12px rgba(59,130,246,0.3); }
.btn-primary:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 6px 18px rgba(59,130,246,0.4); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-ghost { background: transparent; color: var(--slate-700); border: 2px solid var(--slate-300); padding: 0.55rem 1rem; border-radius: 8px; cursor: pointer; font-weight: 600; }
.btn-ghost:hover { background: var(--slate-50); }
.btn-icon-danger { width: 32px; height: 32px; border: 1px solid var(--slate-300); background: white; border-radius: 6px; cursor: pointer; color: var(--danger-600); }
.btn-icon-danger:hover { background: var(--danger-50); border-color: var(--danger-300); }

.alert { padding: 0.7rem 1rem; border-radius: 8px; margin-bottom: 1rem; font-size: 0.85rem; }
.alert-error { background: var(--danger-50); color: var(--danger-700); border: 1px solid var(--danger-100); }

.loading-state { display: flex; align-items: center; gap: 0.7rem; padding: 3rem 0; justify-content: center; color: var(--slate-500); font-size: 0.9rem; }
.spinner { width: 22px; height: 22px; border: 3px solid var(--slate-200); border-top-color: var(--brand-400); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.empty { text-align: center; padding: 3rem 1rem; color: var(--slate-500); background: white; border: 2px dashed var(--slate-200); border-radius: 12px; }

.grid-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }
.card {
  display: flex; flex-direction: column;
  background: white; border: 1px solid var(--slate-200); border-radius: 14px;
  overflow: hidden; box-shadow: var(--shadow-sm);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.card-thumb { aspect-ratio: 16/7; background: var(--slate-100); display: flex; align-items: center; justify-content: center; }
.card-thumb img { width: 100%; height: 100%; object-fit: cover; }
.card-thumb-placeholder { font-size: 2.4rem; opacity: 0.45; }
.card-body { padding: 0.85rem 1rem; flex: 1; }
.card-title { font-size: 0.95rem; font-weight: 700; color: var(--slate-900); }
.card-subtitle { font-size: 0.8rem; color: var(--slate-600); margin-top: 0.15rem; }
.card-meta { display: flex; gap: 0.4rem; flex-wrap: wrap; font-size: 0.72rem; color: var(--slate-500); margin-top: 0.5rem; }
.card-desc { font-size: 0.78rem; color: var(--slate-500); margin-top: 0.5rem; line-height: 1.35; }
.card-actions { display: flex; gap: 0.4rem; padding: 0.6rem 1rem; border-top: 1px solid var(--slate-100); }
.card-actions .btn-ghost { flex: 1; padding: 0.4rem; font-size: 0.82rem; }

/* Form */
.form-group { display: flex; flex-direction: column; gap: 0.3rem; margin-bottom: 0.85rem; }
.form-row { display: flex; gap: 0.75rem; }
.form-row .form-group { flex: 1; }
.form-label { font-size: 0.7rem; font-weight: 700; color: var(--slate-600); text-transform: uppercase; letter-spacing: 0.04em; }
.form-input { padding: 0.55rem 0.75rem; border: 2px solid var(--slate-300); border-radius: 8px; font-size: 0.88rem; font-family: inherit; }
.form-input:focus { outline: none; border-color: var(--brand-400); box-shadow: 0 0 0 3px rgba(96,165,250,0.15); }

/* Modal pequeño */
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 1rem;
}
.modal-content { background: white; border-radius: 16px; padding: 1.75rem; width: 100%; max-width: 540px; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.25); }
.modal-title { font-size: 1.15rem; font-weight: 800; margin-bottom: 1rem; color: var(--slate-900); }
.modal-actions { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1rem; }
.modal-close { background: none; border: none; font-size: 1.6rem; cursor: pointer; color: var(--slate-500); }
.modal-close:hover { color: var(--slate-900); }

/* Modal grande para editor */
.editor-overlay {
  position: fixed; inset: 0; background: rgba(15, 23, 42, 0.7);
  display: flex; align-items: center; justify-content: center; z-index: 1050; padding: 1rem;
}
.editor-shell {
  background: white; border-radius: 18px;
  width: 100%; max-width: 1280px; max-height: 92vh;
  display: flex; flex-direction: column;
  box-shadow: 0 30px 60px rgba(0,0,0,0.3);
  overflow: hidden;
}
.editor-shell-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 1rem 1.25rem; border-bottom: 1px solid var(--slate-200);
}
.editor-shell-head h3 { font-size: 1.05rem; font-weight: 800; color: var(--slate-900); }
.editor-shell-body { flex: 1; overflow-y: auto; padding: 1rem 1.25rem; }

.loading-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.4);
  display: flex; align-items: center; justify-content: center; z-index: 1200;
}
.loading-overlay .spinner { border-color: rgba(255,255,255,0.3); border-top-color: white; }
</style>
