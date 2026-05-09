<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { trackParcel, type ParcelPublicInfo } from '../api/client'

const route = useRoute()

const code = ref('')
const parcel = ref<ParcelPublicInfo | null>(null)
const loading = ref(false)
const error = ref('')
const searched = ref(false)

const statusMap: Record<string, string> = {
  registered: 'Registrada',
  boarded: 'Embarcada',
  in_transit: 'En transito',
  arrived: 'Llego a destino',
  ready_for_pickup: 'Lista para recoger',
  delivered: 'Entregada',
  cancelled: 'Anulada',
}

const paymentStatusMap: Record<string, string> = {
  pending: 'Pendiente',
  paid: 'Pagado',
  partial: 'Parcial',
}

const paymentModeMap: Record<string, string> = {
  origin: 'Pago en origen',
  destination: 'Pago en destino',
}

function statusLabel(s: string) {
  return statusMap[s] || s
}

function paymentStatusLabel(s: string) {
  return paymentStatusMap[s] || s
}

function paymentModeLabel(s: string) {
  return paymentModeMap[s] || s
}

function statusClass(s: string) {
  if (s === 'delivered') return 'badge-success'
  if (s === 'ready_for_pickup' || s === 'arrived') return 'badge-accent'
  if (s === 'in_transit' || s === 'boarded') return 'badge-warning'
  if (s === 'cancelled') return 'badge-danger'
  if (s === 'registered') return 'badge-purple'
  return 'badge-muted'
}

function paymentBadgeClass(s: string) {
  if (s === 'paid') return 'badge-success'
  if (s === 'pending') return 'badge-warning'
  return 'badge-muted'
}

function formatDate(iso: string) {
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { day: '2-digit', month: 'short', year: 'numeric' })
}

function formatDateTime(iso: string) {
  const d = new Date(iso)
  return d.toLocaleString('es-PE', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function search() {
  if (!code.value.trim()) return
  loading.value = true
  error.value = ''
  parcel.value = null
  searched.value = true
  try {
    parcel.value = await trackParcel(code.value.trim())
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const raw = route.params.code
  const paramCode = Array.isArray(raw) ? raw[0] : raw
  if (paramCode && typeof paramCode === 'string') {
    code.value = paramCode
    search()
  }
})
</script>

<template>
  <div class="tracking-page">
    <!-- Hero Banner -->
    <div class="tracking-hero">
      <div class="hero-bg-pattern"></div>
      <div class="container hero-inner">
        <div class="hero-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
            <line x1="12" y1="22.08" x2="12" y2="12"/>
          </svg>
        </div>
        <h1 class="hero-title">Rastrear Encomienda</h1>
        <p class="hero-subtitle">Ingresa el codigo de tu encomienda para conocer su estado y ubicacion</p>

        <!-- Search Widget -->
        <div class="search-widget">
          <div class="search-input-wrap">
            <svg class="input-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
            <input
              v-model="code"
              type="text"
              class="search-input"
              placeholder="Ej: ENC-00042"
              @keydown.enter="search"
            />
          </div>
          <button class="search-btn" :disabled="loading || !code.trim()" @click="search">
            <div v-if="loading" class="spinner-sm"></div>
            <template v-else>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              Rastrear
            </template>
          </button>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="container content-area">
      <!-- Error -->
      <div v-if="error" class="alert alert-error">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" x2="12" y1="8" y2="12"/><line x1="12" x2="12.01" y1="16" y2="16"/></svg>
        <span>{{ error }}</span>
      </div>

      <!-- No result -->
      <div v-if="searched && !loading && !parcel && !error" class="empty-card">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            <line x1="9" x2="15" y1="9" y2="15"/>
            <line x1="15" x2="9" y1="9" y2="15"/>
          </svg>
        </div>
        <h3>No encontrada</h3>
        <p>No se encontro una encomienda con el codigo ingresado. Verifica e intenta nuevamente.</p>
      </div>

      <!-- Initial state (no search yet) -->
      <div v-if="!searched && !loading" class="info-cards">
        <div class="info-card">
          <div class="info-card-icon ic-1">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
          </div>
          <h3>Codigo de encomienda</h3>
          <p>Al registrar tu encomienda recibiras un codigo unico. Ingresalo arriba para rastrear.</p>
        </div>
        <div class="info-card">
          <div class="info-card-icon ic-2">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <h3>Seguimiento en vivo</h3>
          <p>Consulta cada etapa del envio: registro, embarque, transito y entrega.</p>
        </div>
        <div class="info-card">
          <div class="info-card-icon ic-3">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
              <circle cx="12" cy="10" r="3"/>
            </svg>
          </div>
          <h3>Ubicacion actual</h3>
          <p>Conoce donde se encuentra tu encomienda y cuando estara lista para recoger.</p>
        </div>
      </div>

      <!-- Result -->
      <div v-if="parcel" class="result-section">
        <!-- Status Header Card -->
        <div class="result-card">
          <div class="result-header">
            <div class="result-code-wrap">
              <span class="result-label">Encomienda</span>
              <span class="result-code">{{ parcel.code }}</span>
            </div>
            <span class="status-badge" :class="statusClass(parcel.status)">
              {{ statusLabel(parcel.status) }}
            </span>
          </div>

          <!-- Route Info -->
          <div class="route-strip">
            <div class="route-endpoint">
              <div class="route-dot origin"></div>
              <div>
                <span class="route-label">Origen</span>
                <span class="route-name">{{ parcel.origin_stop }}</span>
              </div>
            </div>
            <div class="route-arrow">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>
              </svg>
            </div>
            <div class="route-endpoint">
              <div class="route-dot dest"></div>
              <div>
                <span class="route-label">Destino</span>
                <span class="route-name">{{ parcel.dest_stop }}</span>
              </div>
            </div>
          </div>

          <!-- Details Grid -->
          <div class="detail-grid">
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
              </svg>
              <div>
                <span class="detail-label">Destinatario</span>
                <span class="detail-value">{{ parcel.receiver_name }}</span>
              </div>
            </div>
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              </svg>
              <div>
                <span class="detail-label">Bultos</span>
                <span class="detail-value">{{ parcel.package_count }} {{ parcel.package_count === 1 ? 'bulto' : 'bultos' }}</span>
              </div>
            </div>
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><path d="M8 12h8"/><path d="M12 8v8"/>
              </svg>
              <div>
                <span class="detail-label">Peso</span>
                <span class="detail-value">{{ parcel.weight_kg }} kg</span>
              </div>
            </div>
            <div v-if="parcel.description" class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                <polyline points="14 2 14 8 20 8"/>
              </svg>
              <div>
                <span class="detail-label">Descripcion</span>
                <span class="detail-value">{{ parcel.description }}</span>
              </div>
            </div>
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/>
              </svg>
              <div>
                <span class="detail-label">Modo de pago</span>
                <span class="detail-value">{{ paymentModeLabel(parcel.payment_mode) }}</span>
              </div>
            </div>
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
              </svg>
              <div>
                <span class="detail-label">Estado de pago</span>
                <span class="status-badge sm" :class="paymentBadgeClass(parcel.payment_status)">
                  {{ paymentStatusLabel(parcel.payment_status) }}
                </span>
              </div>
            </div>
            <div class="detail-item">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/>
              </svg>
              <div>
                <span class="detail-label">Fecha de registro</span>
                <span class="detail-value">{{ formatDate(parcel.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Timeline Card -->
        <div v-if="parcel.tracking && parcel.tracking.length" class="timeline-card">
          <h3 class="timeline-title">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
            Historial de seguimiento
          </h3>
          <div class="timeline">
            <div
              v-for="(event, idx) in parcel.tracking"
              :key="event.id"
              class="timeline-item"
              :class="{ 'is-first': idx === 0, 'is-last': idx === parcel.tracking.length - 1 }"
            >
              <div class="timeline-rail">
                <div class="timeline-dot" :class="idx === 0 ? 'dot-active' : 'dot-past'"></div>
                <div v-if="idx < parcel.tracking.length - 1" class="timeline-line"></div>
              </div>
              <div class="timeline-content">
                <div class="timeline-header">
                  <span class="timeline-status" :class="statusClass(event.status)">
                    {{ statusLabel(event.status) }}
                  </span>
                  <span class="timeline-date">{{ formatDateTime(event.created_at) }}</span>
                </div>
                <div v-if="event.location" class="timeline-location">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/>
                  </svg>
                  {{ event.location }}
                </div>
                <div v-if="event.notes" class="timeline-notes">{{ event.notes }}</div>
                <div v-if="event.user_name" class="timeline-user">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
                  </svg>
                  {{ event.user_name }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Hero claro elegante (acento violeta para encomiendas) ── */
.tracking-hero {
  position: relative;
  padding: 3.5rem 0 3rem;
  background:
    radial-gradient(ellipse at 20% 0%, rgba(124,58,237,0.08) 0%, transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(59,130,246,0.06) 0%, transparent 55%),
    linear-gradient(180deg, #f5f3ff 0%, #ffffff 70%);
  border-bottom: 1px solid var(--color-border);
  overflow: hidden;
}

.hero-bg-pattern {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(ellipse at 50% 50%, rgba(124,58,237,0.04) 0%, transparent 60%);
}

.hero-inner {
  position: relative;
  text-align: center;
  max-width: 600px;
}

.hero-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: #f3e8ff;
  border: 1px solid #e9d5ff;
  color: #7c3aed;
  margin-bottom: 1rem;
  box-shadow: 0 6px 18px rgba(124,58,237,0.15);
}

.tracking-hero .hero-title {
  font-size: 2rem;
  font-weight: 800;
  color: var(--color-heading);
  letter-spacing: -0.02em;
  margin-bottom: 0.5rem;
}

.tracking-hero .hero-subtitle {
  font-size: 1rem;
  color: var(--color-text-muted);
  margin-bottom: 2rem;
}

.search-widget {
  display: flex;
  gap: 0.65rem;
  max-width: 500px;
  margin: 0 auto;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 0.5rem;
  box-shadow: 0 10px 40px rgba(15,23,42,0.08), 0 4px 12px rgba(15,23,42,0.04);
}

.search-input-wrap {
  flex: 1;
  position: relative;
}

.input-icon {
  position: absolute;
  left: 0.85rem;
  top: 50%;
  transform: translateY(-50%);
  color: #7c3aed;
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.75rem 1rem 0.75rem 2.85rem;
  border: 2px solid transparent;
  border-radius: var(--radius-md);
  background: var(--color-background);
  color: var(--color-heading);
  font-size: 1rem;
  font-weight: 600;
  font-family: inherit;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  outline: none;
  transition: border-color 0.2s ease;
}

.search-input:focus {
  border-color: #a78bfa;
}

.search-input::placeholder {
  color: var(--color-text-muted);
  opacity: 0.5;
  text-transform: none;
  font-weight: 400;
  letter-spacing: 0;
}

.search-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.75rem 1.25rem;
  background: linear-gradient(135deg, #7c3aed, #6d28d9);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
  font-family: inherit;
  transition: all 0.2s ease;
}

.search-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
}

.search-btn:disabled { opacity: 0.6; cursor: not-allowed; }

.spinner-sm {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* ── Content ── */
.content-area {
  max-width: 720px;
  padding-top: 2rem;
  padding-bottom: 4rem;
}

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

/* ── Info Cards (initial state) ── */
.info-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-top: 1rem;
}

.info-card {
  padding: 1.5rem 1.25rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  text-align: center;
  transition: box-shadow 0.2s ease;
}

.info-card:hover { box-shadow: var(--shadow-md); }

.info-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  margin-bottom: 0.85rem;
}

.ic-1 { background: rgba(124, 58, 237, 0.1); color: #7c3aed; }
.ic-2 { background: var(--warning-50); color: var(--warning-600); }
.ic-3 { background: var(--accent-50); color: var(--accent-700); }

@media (prefers-color-scheme: dark) {
  .ic-1 { background: rgba(124, 58, 237, 0.15); }
  .ic-2 { background: rgba(245, 158, 11, 0.12); }
  .ic-3 { background: rgba(6, 201, 170, 0.12); }
}

.info-card h3 {
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.35rem;
}

.info-card p {
  font-size: 0.82rem;
  color: var(--color-text-muted);
  line-height: 1.5;
}

/* ── Empty State ── */
.empty-card {
  text-align: center;
  padding: 3rem 2rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--color-background-mute);
  color: var(--color-text-muted);
  margin-bottom: 1rem;
  opacity: 0.6;
}

.empty-card h3 {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.3rem;
}

.empty-card p {
  font-size: 0.9rem;
  color: var(--color-text-muted);
}

/* ── Result Section ── */
.result-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.result-card {
  background: var(--color-surface);
  border: 2px solid var(--color-border);
  border-radius: 14px;
  overflow: hidden;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  background: var(--color-background-soft);
  border-bottom: 1px solid var(--color-border);
}

.result-label {
  display: block;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.result-code {
  font-size: 1.4rem;
  font-weight: 800;
  color: var(--color-heading);
  font-family: 'SF Mono', 'Fira Code', monospace;
  letter-spacing: 0.04em;
}

/* ── Status Badges ── */
.status-badge {
  padding: 0.3rem 0.8rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.status-badge.sm {
  padding: 0.2rem 0.6rem;
  font-size: 0.7rem;
}

.badge-success { background: var(--success-50); color: var(--success-700); }
.badge-warning { background: var(--warning-50); color: var(--warning-700); }
.badge-danger { background: var(--danger-50); color: var(--danger-700); }
.badge-muted { background: var(--color-background-mute); color: var(--color-text-muted); }
.badge-purple { background: rgba(124, 58, 237, 0.1); color: #7c3aed; }
.badge-accent { background: var(--accent-50); color: var(--accent-700); }

@media (prefers-color-scheme: dark) {
  .badge-purple { background: rgba(124, 58, 237, 0.2); color: #a78bfa; }
  .badge-accent { background: rgba(5, 150, 105, 0.15); color: #34d399; }
}

/* ── Route Strip ── */
.route-strip {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.25rem;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-background-soft);
}

.route-endpoint {
  display: flex;
  align-items: center;
  gap: 0.65rem;
}

.route-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.route-dot.origin { background: #7c3aed; }
.route-dot.dest { background: #059669; }

.route-label {
  display: block;
  font-size: 0.68rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.route-name {
  display: block;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--color-heading);
}

.route-arrow {
  color: var(--color-text-muted);
  opacity: 0.4;
  flex-shrink: 0;
}

/* ── Detail Grid ── */
.detail-grid {
  padding: 1.25rem 1.5rem;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.detail-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.detail-item svg {
  flex-shrink: 0;
  color: #7c3aed;
  margin-top: 0.1rem;
}

.detail-label {
  display: block;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.detail-value {
  font-size: 0.92rem;
  font-weight: 600;
  color: var(--color-heading);
}

/* ── Timeline Card ── */
.timeline-card {
  background: var(--color-surface);
  border: 2px solid var(--color-border);
  border-radius: 14px;
  padding: 1.5rem;
  animation: fadeIn 0.3s ease 0.1s both;
}

.timeline-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 1.5rem;
}

.timeline-title svg {
  color: #7c3aed;
}

/* ── Timeline ── */
.timeline {
  display: flex;
  flex-direction: column;
}

.timeline-item {
  display: flex;
  gap: 1rem;
  position: relative;
}

.timeline-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  width: 20px;
}

.timeline-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  flex-shrink: 0;
  z-index: 1;
  border: 3px solid var(--color-surface);
  box-sizing: content-box;
}

.dot-active {
  background: #7c3aed;
  border-color: rgba(124, 58, 237, 0.2);
  box-shadow: 0 0 0 4px rgba(124, 58, 237, 0.15);
}

.dot-past {
  background: var(--color-border);
  border-color: var(--color-surface);
}

.timeline-line {
  width: 2px;
  flex: 1;
  min-height: 24px;
  background: var(--color-border);
}

.timeline-content {
  flex: 1;
  padding-bottom: 1.5rem;
}

.timeline-item.is-last .timeline-content {
  padding-bottom: 0;
}

.timeline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.35rem;
}

.timeline-status {
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.timeline-date {
  font-size: 0.78rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.timeline-location {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.85rem;
  color: var(--color-heading);
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.timeline-location svg {
  color: #7c3aed;
  flex-shrink: 0;
}

.timeline-notes {
  font-size: 0.82rem;
  color: var(--color-text-muted);
  line-height: 1.5;
  margin-bottom: 0.15rem;
}

.timeline-user {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.75rem;
  color: var(--color-text-muted);
  opacity: 0.7;
}

.timeline-user svg {
  flex-shrink: 0;
}

/* ── Responsive ── */
@media (max-width: 640px) {
  .tracking-hero { padding: 2.5rem 0 2rem; }
  .tracking-hero .hero-title { font-size: 1.5rem; }
  .search-widget { flex-direction: column; }
  .info-cards { grid-template-columns: 1fr; }
  .detail-grid { grid-template-columns: 1fr; }
  .route-strip { flex-direction: column; gap: 0.75rem; }
  .route-arrow { transform: rotate(90deg); }
  .timeline-header { flex-direction: column; align-items: flex-start; gap: 0.25rem; }
}
</style>
