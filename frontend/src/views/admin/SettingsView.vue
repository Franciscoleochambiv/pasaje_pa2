<script setup lang="ts">
import { ref, onMounted } from 'vue'

const settings = ref<Record<string, string>>({})
const saving = ref(false)
const saveMsg = ref('')
const loading = ref(true)

const apiBase = import.meta.env.VITE_API_URL ?? ''

function authHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  if (token) return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
  return { 'Content-Type': 'application/json' }
}

async function loadSettings() {
  loading.value = true
  try {
    const res = await fetch(`${apiBase}/api/admin/settings`, { headers: authHeaders() })
    if (res.ok) {
      const data: { key: string; value: string }[] = await res.json()
      const map: Record<string, string> = {}
      for (const s of data) map[s.key] = s.value
      settings.value = map
    }
  } catch { /* ignore */ }
  loading.value = false
}

async function save() {
  saving.value = true
  saveMsg.value = ''
  try {
    const res = await fetch(`${apiBase}/api/admin/settings`, {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify(settings.value),
    })
    saveMsg.value = res.ok ? 'Configuracion guardada exitosamente' : 'Error al guardar'
    setTimeout(() => { saveMsg.value = '' }, 3000)
  } catch (e) {
    saveMsg.value = (e as Error).message
  }
  saving.value = false
}

onMounted(loadSettings)

const fields = [
  { key: 'company_name', label: 'Nombre de la empresa', icon: 'building', group: 'empresa' },
  { key: 'company_ruc', label: 'RUC', icon: 'hash', group: 'empresa' },
  { key: 'company_address', label: 'Direccion', icon: 'map', group: 'empresa' },
  { key: 'company_phone', label: 'Telefono', icon: 'phone', group: 'empresa' },
  { key: 'company_email', label: 'Email', icon: 'mail', group: 'empresa' },
  { key: 'currency', label: 'Moneda', icon: 'dollar', group: 'precios' },
  { key: 'igv_percent', label: 'IGV (%)', icon: 'percent', group: 'precios' },
  { key: 'yape_business_number', label: 'Numero Yape del negocio', icon: 'phone', group: 'pagos' },
  { key: 'whatsapp_phone', label: 'WhatsApp (con codigo pais, ej: 51999888777)', icon: 'phone', group: 'pagos' },
  { key: 'hold_duration_minutes', label: 'Duracion de reserva temporal (min)', icon: 'clock', group: 'sistema' },
  { key: 'pos_hold_hours', label: 'Horas de retencion POS (punto de venta)', icon: 'clock', group: 'sistema' },
]
</script>

<template>
  <div class="settings-page">
    <div class="settings-header">
      <h1>Configuracion General</h1>
      <p>Configura los datos de la empresa, precios y parametros del sistema</p>
    </div>

    <div v-if="loading" style="text-align:center;padding:3rem;color:var(--color-text-muted)">Cargando...</div>

    <template v-else>
      <!-- Empresa -->
      <div class="settings-section">
        <h2 class="section-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z"/><path d="M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2"/><path d="M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2"/><path d="M10 6h4"/><path d="M10 10h4"/><path d="M10 14h4"/><path d="M10 18h4"/></svg>
          Datos de la Empresa
        </h2>
        <div class="settings-grid">
          <div v-for="f in fields.filter(x => x.group === 'empresa')" :key="f.key" class="setting-field">
            <label>{{ f.label }}</label>
            <input v-model="settings[f.key]" type="text" :placeholder="f.label" />
          </div>
        </div>
      </div>

      <!-- Precios -->
      <div class="settings-section">
        <h2 class="section-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="1" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
          Moneda e Impuestos
        </h2>
        <p style="font-size:0.82rem;color:var(--color-text-muted);margin-bottom:1rem;">El precio por asiento se configura en cada <strong>Ruta</strong> desde el menu de Rutas.</p>
        <div class="settings-grid">
          <div v-for="f in fields.filter(x => x.group === 'precios')" :key="f.key" class="setting-field">
            <label>{{ f.label }}</label>
            <input v-model="settings[f.key]" :type="f.key.includes('price') || f.key.includes('igv') ? 'number' : 'text'" step="0.01" :placeholder="f.label" />
          </div>
        </div>
      </div>

      <!-- Pagos -->
      <div class="settings-section">
        <h2 class="section-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
          Yape y WhatsApp
        </h2>
        <p style="font-size:0.82rem;color:var(--slate-500);margin-bottom:1rem;">Numeros para pagos directos por Yape y notificaciones por WhatsApp.</p>
        <div class="settings-grid">
          <div v-for="f in fields.filter(x => x.group === 'pagos')" :key="f.key" class="setting-field">
            <label>{{ f.label }}</label>
            <input v-model="settings[f.key]" type="text" :placeholder="f.label" />
          </div>
        </div>
      </div>

      <!-- Sistema -->
      <div class="settings-section">
        <h2 class="section-title">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          Parametros del Sistema
        </h2>
        <div class="settings-grid">
          <div v-for="f in fields.filter(x => x.group === 'sistema')" :key="f.key" class="setting-field">
            <label>{{ f.label }}</label>
            <input v-model="settings[f.key]" type="number" :placeholder="f.label" />
          </div>
        </div>
      </div>

      <!-- Save -->
      <div class="settings-actions">
        <button class="btn btn-save" @click="save" :disabled="saving">
          <svg v-if="!saving" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
          {{ saving ? 'Guardando...' : 'Guardar Configuracion' }}
        </button>
        <span v-if="saveMsg" class="save-msg" :class="saveMsg.includes('Error') ? 'msg-err' : 'msg-ok'">{{ saveMsg }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.settings-page { padding: 0; }
.settings-header { margin-bottom: 2rem; }
.settings-header h1 { font-size: 1.4rem; font-weight: 800; color: var(--slate-900); }
.settings-header p { font-size: 0.9rem; color: var(--slate-500); margin-top: 0.25rem; }

.settings-section {
  background: white;
  border: 2px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  margin-bottom: 1.25rem;
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
.section-title svg { color: var(--brand-500); }

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.setting-field label {
  display: block;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.35rem;
}

.setting-field input {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 2px solid var(--slate-200);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.92rem;
  font-weight: 500;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s ease;
}
.setting-field input:focus {
  border-color: var(--brand-500);
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.settings-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 0.5rem;
}

.btn-save {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.75rem;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.92rem;
  font-weight: 700;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(27, 85, 245, 0.2);
}
.btn-save:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(27, 85, 245, 0.3); }
.btn-save:disabled { opacity: 0.6; cursor: not-allowed; }

.save-msg { font-size: 0.88rem; font-weight: 600; }
.msg-ok { color: var(--success-600); }
.msg-err { color: var(--danger-600); }
</style>
