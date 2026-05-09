<script setup lang="ts">
import { ref, onMounted } from 'vue'

const apiBase = import.meta.env.VITE_API_URL ?? ''
function authHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  if (token) return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
  return { 'Content-Type': 'application/json' }
}

// Config
const config = ref({ go_service_url: '', tenant_slug: '', apiperu_url: '', apiperu_token: '', tenant_email: '', tenant_password: '' })
const settings = ref<Record<string, string>>({})
const tenants = ref<any[]>([])
const loadingTenants = ref(false)
const testing = ref(false)
const testResult = ref<any>(null)
const saving = ref(false)
const saveMsg = ref('')
const syncing = ref(false)
const syncMsg = ref('')

interface SubscriptionInfo {
  slug: string
  name: string
  is_active: boolean
  trial_ends_at: string | null
  subscribed_until: string | null
  days_remaining: number | null
  status: 'active' | 'expiring_soon' | 'expired' | 'trial' | 'unknown'
}
const subscription = ref<SubscriptionInfo | null>(null)
const subError = ref('')
const loadingSub = ref(false)
async function loadSubscription() {
  loadingSub.value = true
  subError.value = ''
  try {
    const res = await fetch(`${apiBase}/api/admin/billing-config/subscription`, { headers: authHeaders() })
    if (res.ok) subscription.value = await res.json()
    else subError.value = (await res.json())?.error || `HTTP ${res.status}`
  } catch (e) { subError.value = (e as Error).message }
  loadingSub.value = false
}
function formatDate(iso: string | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { year: 'numeric', month: 'long', day: 'numeric' })
}

async function loadConfig() {
  try {
    const res = await fetch(`${apiBase}/api/admin/billing-config`, { headers: authHeaders() })
    if (res.ok) config.value = { ...config.value, ...(await res.json()) }
  } catch {}
}

async function loadSettings() {
  try {
    const res = await fetch(`${apiBase}/api/admin/settings`, { headers: authHeaders() })
    if (res.ok) {
      const data: { key: string; value: string }[] = await res.json()
      const map: Record<string, string> = {}
      for (const s of data) map[s.key] = s.value
      settings.value = map
    }
  } catch {}
}

async function loadTenants() {
  loadingTenants.value = true
  try {
    const res = await fetch(`${apiBase}/api/admin/billing-config/tenants`, { headers: authHeaders() })
    if (res.ok) tenants.value = await res.json()
  } catch {}
  loadingTenants.value = false
}

function selectTenant(t: any) {
  config.value.tenant_slug = t.slug
  if (t.empresa) {
    settings.value.company_name = t.empresa.razon_emisor || ''
    settings.value.company_ruc = t.empresa.ruc_emisor || ''
    settings.value.company_phone = t.empresa.telefonos || ''
    const dir = [t.empresa.direccion, t.empresa.distrito, t.empresa.provincia].filter(Boolean).join(', ')
    settings.value.company_address = dir + (t.empresa.ciudad ? ' - ' + t.empresa.ciudad : '')
  }
}

async function saveAll() {
  saving.value = true
  saveMsg.value = ''
  try {
    await fetch(`${apiBase}/api/admin/billing-config`, { method: 'PUT', headers: authHeaders(), body: JSON.stringify(config.value) })
    await fetch(`${apiBase}/api/admin/settings`, { method: 'PUT', headers: authHeaders(), body: JSON.stringify(settings.value) })
    saveMsg.value = 'Configuracion guardada'
    setTimeout(() => { saveMsg.value = '' }, 3000)
  } catch (e) { saveMsg.value = (e as Error).message }
  saving.value = false
}

async function testConnection() {
  testing.value = true
  testResult.value = null
  try {
    const res = await fetch(`${apiBase}/api/admin/billing-config/test`, { headers: authHeaders() })
    testResult.value = await res.json()
  } catch (e) { testResult.value = { status: 'error', message: (e as Error).message } }
  testing.value = false
}

async function syncEmpresa() {
  syncing.value = true
  syncMsg.value = ''
  try {
    const res = await fetch(`${apiBase}/api/admin/billing-config/sync-empresa`, { method: 'POST', headers: authHeaders() })
    const data = await res.json()
    if (res.ok) {
      syncMsg.value = data.message || 'Sincronizado'
      loadSettings()
    } else {
      syncMsg.value = data.error || 'Error'
    }
    setTimeout(() => { syncMsg.value = '' }, 4000)
  } catch (e) { syncMsg.value = (e as Error).message }
  syncing.value = false
}

onMounted(() => { loadConfig(); loadSettings(); loadSubscription() })
</script>

<template>
  <div class="billing-page">
    <div class="billing-header">
      <h1>Facturacion y Configuracion</h1>
      <p>Conecta con el sistema de venta, configura datos de empresa y precios</p>
    </div>

    <!-- 0. Subscription -->
    <div class="config-section">
      <div class="section-head">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          Vigencia API Facturación
        </h2>
        <button class="btn-sm btn-outline" @click="loadSubscription" :disabled="loadingSub">
          {{ loadingSub ? 'Consultando...' : 'Refrescar' }}
        </button>
      </div>
      <div v-if="subError" class="test-badge err">{{ subError }}</div>
      <div v-else-if="!subscription && loadingSub" class="sync-msg">Consultando landlord...</div>
      <div v-else-if="subscription" class="sub-card" :class="subscription.status">
        <div class="sub-row">
          <span class="sub-label">Tenant</span>
          <strong>{{ subscription.name || subscription.slug }}</strong>
        </div>
        <div class="sub-row">
          <span class="sub-label">Vigente hasta</span>
          <strong>{{ formatDate(subscription.subscribed_until || subscription.trial_ends_at) }}</strong>
        </div>
        <div class="sub-row">
          <span class="sub-label">Días restantes</span>
          <strong v-if="subscription.days_remaining !== null">
            {{ subscription.days_remaining }}
            <span v-if="subscription.days_remaining < 0"> (vencido)</span>
          </strong>
          <strong v-else>—</strong>
        </div>
        <div class="sub-row">
          <span class="sub-label">Estado</span>
          <span class="sub-pill" :class="subscription.status">
            {{ subscription.status === 'active' ? 'Activa'
              : subscription.status === 'expiring_soon' ? 'Por vencer'
              : subscription.status === 'expired' ? 'Vencida'
              : subscription.status === 'trial' ? 'Prueba'
              : 'Desconocido' }}
          </span>
        </div>
      </div>
    </div>

    <!-- 1. Tenant Selection -->
    <div class="config-section">
      <div class="section-head">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z"/><path d="M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2"/><path d="M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2"/></svg>
          Seleccionar Tenant
        </h2>
        <button class="btn-sm btn-outline" @click="loadTenants" :disabled="loadingTenants">
          {{ loadingTenants ? 'Cargando...' : 'Cargar Tenants' }}
        </button>
      </div>
      <div v-if="config.tenant_slug" class="current-tenant">
        Tenant actual: <strong>{{ config.tenant_slug }}</strong>
      </div>
      <div v-if="tenants.length" class="tenant-grid">
        <button v-for="t in tenants" :key="t.id" class="tenant-card" :class="{ active: config.tenant_slug === t.slug }" @click="selectTenant(t)">
          <div class="tc-name">{{ t.name }}</div>
          <div class="tc-slug">{{ t.slug }}</div>
          <div v-if="t.empresa" class="tc-ruc">RUC: {{ t.empresa.ruc_emisor }}</div>
          <div class="tc-check" v-if="config.tenant_slug === t.slug">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
          </div>
        </button>
      </div>
    </div>

    <!-- 2. Connection Config -->
    <div class="config-section">
      <h2>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" x2="3" y1="12" y2="12"/></svg>
        Conexion
      </h2>
      <div class="field-grid">
        <div class="field">
          <label>URL Go Service</label>
          <input v-model="config.go_service_url" placeholder="https://goventa.facturame.online" />
        </div>
        <div class="field">
          <label>Email Tenant</label>
          <input v-model="config.tenant_email" placeholder="admin@gmail.com" />
        </div>
        <div class="field">
          <label>Password Tenant</label>
          <input v-model="config.tenant_password" type="password" placeholder="••••••••" />
        </div>
        <div class="field">
          <label>Token API Peru</label>
          <input v-model="config.apiperu_token" placeholder="apiperu_..." />
        </div>
      </div>
      <div class="section-actions">
        <button class="btn-sm btn-outline" @click="testConnection" :disabled="testing">
          {{ testing ? 'Probando...' : 'Probar Conexion' }}
        </button>
      </div>
      <div v-if="testResult" class="test-badge" :class="testResult.status === 'ok' ? 'ok' : 'err'">
        {{ testResult.message }}
      </div>
    </div>

    <!-- 3. Datos de Empresa (from venta) -->
    <div class="config-section">
      <div class="section-head">
        <h2>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z"/><path d="M10 6h4"/><path d="M10 10h4"/><path d="M10 14h4"/></svg>
          Datos de la Empresa
        </h2>
        <button class="btn-sm btn-accent" @click="syncEmpresa" :disabled="syncing">
          <svg v-if="!syncing" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
          {{ syncing ? 'Sincronizando...' : 'Sincronizar desde Venta' }}
        </button>
      </div>
      <div v-if="syncMsg" class="sync-msg">{{ syncMsg }}</div>
      <div class="field-grid">
        <div class="field full">
          <label>Razon Social</label>
          <input v-model="settings.company_name" placeholder="Nombre de la empresa" />
        </div>
        <div class="field">
          <label>RUC</label>
          <input v-model="settings.company_ruc" placeholder="20XXXXXXXXX" />
        </div>
        <div class="field">
          <label>Telefono</label>
          <input v-model="settings.company_phone" placeholder="956921947" />
        </div>
        <div class="field full">
          <label>Direccion</label>
          <input v-model="settings.company_address" placeholder="Calle, Distrito, Provincia" />
        </div>
        <div class="field">
          <label>Email</label>
          <input v-model="settings.company_email" placeholder="empresa@email.com" />
        </div>
      </div>
    </div>

    <!-- 4. Precios -->
    <div class="config-section">
      <h2>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="1" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
        Precios y Parametros
      </h2>
      <div class="field-grid">
        <div class="field">
          <label>Precio por Asiento (S/)</label>
          <input v-model="settings.price_per_seat" type="number" step="0.50" min="0" placeholder="45.00" />
        </div>
        <div class="field">
          <label>IGV (%)</label>
          <input v-model="settings.igv_percent" type="number" placeholder="18" />
        </div>
        <div class="field">
          <label>Duracion Reserva (min)</label>
          <input v-model="settings.hold_duration_minutes" type="number" placeholder="10" />
        </div>
      </div>
    </div>

    <!-- Save -->
    <div class="save-bar">
      <button class="btn-save" @click="saveAll" :disabled="saving">
        <svg v-if="!saving" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
        {{ saving ? 'Guardando...' : 'Guardar Todo' }}
      </button>
      <span v-if="saveMsg" class="save-ok">{{ saveMsg }}</span>
    </div>
  </div>
</template>

<style scoped>
.billing-page { padding: 0; }
.billing-header { margin-bottom: 1.75rem; }
.billing-header h1 { font-size: 1.4rem; font-weight: 800; color: var(--slate-900); }
.billing-header p { font-size: 0.9rem; color: var(--slate-500); margin-top: 0.2rem; }

.config-section {
  background: white;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  margin-bottom: 1.25rem;
}
.config-section h2 {
  display: flex; align-items: center; gap: 0.5rem;
  font-size: 1rem; font-weight: 700; color: var(--slate-900); margin-bottom: 1rem;
}
.config-section h2 svg { color: var(--brand-500); }

.section-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem; }
.section-head h2 { margin-bottom: 0; }

.field-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 0.85rem; }
.field.full { grid-column: 1 / -1; }
.field label { display: block; font-size: 0.75rem; font-weight: 600; color: var(--slate-500); text-transform: uppercase; letter-spacing: 0.03em; margin-bottom: 0.3rem; }
.field input { width: 100%; padding: 0.6rem 0.8rem; border: 2px solid var(--color-border); border-radius: var(--radius-md); background: var(--slate-50); color: var(--slate-900); font-size: 0.9rem; font-weight: 500; font-family: inherit; outline: none; transition: border-color 0.2s; }
.field input:focus { border-color: var(--brand-500); box-shadow: 0 0 0 3px rgba(27,85,245,0.1); }

.section-actions { margin-top: 1rem; }

/* Buttons */
.btn-sm { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.5rem 1rem; font-size: 0.82rem; font-weight: 600; border-radius: var(--radius-md); cursor: pointer; font-family: inherit; border: none; transition: all 0.2s; }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-outline { background: white; border: 2px solid var(--brand-200); color: var(--brand-600); }
.btn-outline:hover:not(:disabled) { border-color: var(--brand-400); background: var(--brand-50); }
.btn-accent { background: var(--accent-50); border: 2px solid var(--accent-200); color: var(--accent-700); }
.btn-accent:hover:not(:disabled) { border-color: var(--accent-400); background: var(--accent-100); }

.test-badge { margin-top: 0.75rem; padding: 0.5rem 1rem; border-radius: var(--radius-md); font-size: 0.82rem; font-weight: 600; }
.test-badge.ok { background: var(--success-50); color: var(--success-700); border: 1px solid var(--success-200); }
.test-badge.err { background: var(--danger-50); color: var(--danger-700); border: 1px solid var(--danger-200); }

.sync-msg { padding: 0.4rem 0.8rem; background: var(--success-50); color: var(--success-700); border-radius: var(--radius-sm); font-size: 0.82rem; font-weight: 600; margin-bottom: 0.75rem; }

.current-tenant { font-size: 0.85rem; color: var(--slate-500); margin-bottom: 0.75rem; }
.current-tenant strong { color: var(--brand-600); }

/* Tenant Grid */
.tenant-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 0.75rem; }
.tenant-card {
  position: relative; text-align: left; padding: 1rem 1.15rem;
  background: var(--slate-50); border: 2px solid var(--color-border); border-radius: var(--radius-md);
  cursor: pointer; font-family: inherit; color: inherit; transition: all 0.2s; width: 100%;
}
.tenant-card:hover { border-color: var(--brand-300); box-shadow: var(--shadow-sm); }
.tenant-card.active { border-color: var(--brand-500); background: var(--brand-50); }
@media (prefers-color-scheme: dark) { .tenant-card.active { background: rgba(27,85,245,0.08); } }
.tc-name { font-size: 0.85rem; font-weight: 700; color: var(--slate-900); line-height: 1.3; margin-bottom: 0.25rem; }
.tc-slug { font-size: 0.75rem; color: var(--brand-500); font-weight: 600; }
.tc-ruc { font-size: 0.72rem; color: var(--slate-500); margin-top: 0.15rem; }
.tc-check { position: absolute; top: 0.75rem; right: 0.75rem; color: var(--brand-500); }

/* Save Bar */
.save-bar { display: flex; align-items: center; gap: 1rem; margin-top: 0.5rem; }
.btn-save {
  display: inline-flex; align-items: center; gap: 0.5rem;
  padding: 0.75rem 2rem; background: linear-gradient(135deg, var(--brand-400), var(--brand-500)); color: white;
  border: none; border-radius: var(--radius-md); font-size: 0.95rem; font-weight: 700; cursor: pointer;
  font-family: inherit; transition: all 0.2s; box-shadow: 0 2px 8px rgba(27,85,245,0.2);
}
.btn-save:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(27,85,245,0.3); }
.btn-save:disabled { opacity: 0.6; cursor: not-allowed; }
.save-ok { color: var(--success-600); font-weight: 600; font-size: 0.88rem; }
.sub-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 10px; padding: 1rem 1.25rem; }
.sub-card.expiring_soon { border-color: #fbbf24; background: #fffbeb; }
.sub-card.expired { border-color: #ef4444; background: #fef2f2; }
.sub-row { display: flex; justify-content: space-between; align-items: center; padding: 0.45rem 0; border-bottom: 1px solid #e2e8f0; }
.sub-row:last-child { border-bottom: none; }
.sub-label { color: #64748b; font-size: 0.88rem; }
.sub-pill { padding: 0.2rem 0.7rem; border-radius: 999px; font-size: 0.78rem; font-weight: 700; }
.sub-pill.active { background: #d1fae5; color: #065f46; }
.sub-pill.expiring_soon { background: #fef3c7; color: #92400e; }
.sub-pill.expired { background: #fee2e2; color: #991b1b; }
.sub-pill.trial { background: #dbeafe; color: #1e40af; }
.sub-pill.unknown { background: #e2e8f0; color: #475569; }
</style>
