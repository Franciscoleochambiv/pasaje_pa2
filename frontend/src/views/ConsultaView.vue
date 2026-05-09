<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getReservation, type Reservation } from '../api/client'

const route = useRoute()

const code = ref('')
const reservation = ref<Reservation | null>(null)
const loading = ref(false)
const error = ref('')
const searched = ref(false)

function statusLabel(s: string) {
  const map: Record<string, string> = {
    pending: 'Pendiente',
    pending_verification: 'Pago en verificacion',
    confirmed: 'Confirmada',
    cancelled: 'Cancelada',
    expired: 'Expirada',
  }
  return map[s] || s
}

function statusClass(s: string) {
  if (s === 'confirmed') return 'badge-success'
  if (s === 'pending' || s === 'pending_verification') return 'badge-warning'
  if (s === 'cancelled' || s === 'expired') return 'badge-danger'
  return 'badge-muted'
}

async function search() {
  if (!code.value.trim()) return
  loading.value = true
  error.value = ''
  reservation.value = null
  searched.value = true
  try {
    reservation.value = await getReservation(code.value.trim())
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const queryCode = route.query.code as string
  if (queryCode) {
    code.value = queryCode
    search()
  }
})
</script>

<template>
  <div class="consulta-page">
    <!-- Hero Banner -->
    <div class="consulta-hero">
      <div class="hero-bg-pattern"></div>
      <div class="container hero-inner">
        <div class="hero-icon">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
          </svg>
        </div>
        <h1 class="hero-title">Consultar Reserva</h1>
        <p class="hero-subtitle">Ingresa tu codigo de reserva para ver el estado de tu viaje</p>

        <!-- Search Widget -->
        <div class="search-widget">
          <div class="search-input-wrap">
            <svg class="input-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/>
              <path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>
            </svg>
            <input
              v-model="code"
              type="text"
              class="search-input"
              placeholder="Ej: ABC12345"
              @keydown.enter="search"
            />
          </div>
          <button class="search-btn" :disabled="loading || !code.trim()" @click="search">
            <div v-if="loading" class="spinner-sm"></div>
            <template v-else>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              Consultar
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
      <div v-if="searched && !loading && !reservation && !error" class="empty-card">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M16 16s-1.5-2-4-2-4 2-4 2"/><line x1="9" x2="9.01" y1="9" y2="9"/><line x1="15" x2="15.01" y1="9" y2="9"/></svg>
        </div>
        <h3>No encontrada</h3>
        <p>No se encontro una reserva con el codigo ingresado. Verifica e intenta nuevamente.</p>
      </div>

      <!-- Initial state (no search yet) -->
      <div v-if="!searched && !loading" class="info-cards">
        <div class="info-card">
          <div class="info-card-icon ic-1">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/></svg>
          </div>
          <h3>Codigo de reserva</h3>
          <p>Al reservar tu pasaje recibiras un codigo unico de 8 caracteres. Ingresalo arriba.</p>
        </div>
        <div class="info-card">
          <div class="info-card-icon ic-2">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          </div>
          <h3>Estado en tiempo real</h3>
          <p>Consulta si tu reserva esta pendiente, confirmada o expirada.</p>
        </div>
        <div class="info-card">
          <div class="info-card-icon ic-3">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
          </div>
          <h3>Detalle de asientos</h3>
          <p>Verifica los asientos reservados y los datos de tu viaje.</p>
        </div>
      </div>

      <!-- Result -->
      <div v-if="reservation" class="result-card">
        <div class="result-header">
          <div class="result-code-wrap">
            <span class="result-label">Reserva</span>
            <span class="result-code">{{ reservation.code }}</span>
          </div>
          <span class="status-badge" :class="statusClass(reservation.status)">
            {{ statusLabel(reservation.status) }}
          </span>
        </div>

        <!-- Trip info -->
        <div v-if="reservation.route_name || reservation.departure_at" class="trip-banner">
          <div class="trip-route">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6v6"/><path d="M16 6v6"/><path d="M2 14h20"/><path d="M3 18h18"/><path d="M5 22h14"/></svg>
            <span>{{ reservation.route_name || 'Pasaje interprovincial' }}</span>
          </div>
          <div v-if="reservation.origin_stop || reservation.dest_stop" class="trip-stops">
            <strong>{{ reservation.origin_stop || '—' }}</strong>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
            <strong>{{ reservation.dest_stop || '—' }}</strong>
          </div>
          <div v-if="reservation.departure_at" class="trip-departure">
            Salida: {{ new Date(reservation.departure_at).toLocaleString('es-PE', { dateStyle: 'medium', timeStyle: 'short' }) }}
          </div>
        </div>

        <div class="detail-grid">
          <div v-if="reservation.passenger_name" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
            <div>
              <span class="detail-label">Pasajero</span>
              <span class="detail-value">{{ reservation.passenger_name }}</span>
            </div>
          </div>
          <div v-if="reservation.passenger_doc_number" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="12" x="3" y="6" rx="2"/><path d="M7 10h6"/><path d="M7 14h4"/></svg>
            <div>
              <span class="detail-label">Documento</span>
              <span class="detail-value">{{ [reservation.passenger_doc_type, reservation.passenger_doc_number].filter(Boolean).join(' ') }}</span>
            </div>
          </div>
          <div v-if="reservation.passenger_email" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
            <div>
              <span class="detail-label">Email</span>
              <span class="detail-value">{{ reservation.passenger_email }}</span>
            </div>
          </div>
          <div v-if="reservation.passenger_phone" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>
            <div>
              <span class="detail-label">Teléfono</span>
              <span class="detail-value">{{ reservation.passenger_phone }}</span>
            </div>
          </div>
          <div v-if="reservation.total_amount" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            <div>
              <span class="detail-label">Total</span>
              <span class="detail-value">S/ {{ reservation.total_amount.toFixed(2) }}</span>
            </div>
          </div>
          <div v-if="reservation.payment_method || reservation.payment_status" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/></svg>
            <div>
              <span class="detail-label">Pago</span>
              <span class="detail-value">{{ [reservation.payment_method, reservation.payment_status].filter(Boolean).join(' / ') }}</span>
            </div>
          </div>
          <div v-if="reservation.document_type" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            <div>
              <span class="detail-label">Comprobante</span>
              <span class="detail-value">{{ reservation.document_type }}{{ reservation.billing_sale_id ? ' #' + reservation.billing_sale_id : '' }}</span>
            </div>
          </div>
          <div v-if="reservation.expires_at && reservation.status !== 'confirmed'" class="detail-item">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            <div>
              <span class="detail-label">Expira</span>
              <span class="detail-value">{{ new Date(reservation.expires_at).toLocaleString('es-PE') }}</span>
            </div>
          </div>
        </div>

        <div v-if="reservation.items && reservation.items.length" class="seats-section">
          <h3>Asientos reservados</h3>
          <div class="seat-chips">
            <div v-for="item in reservation.items" :key="item.id" class="seat-chip">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>
              <div class="seat-chip-text">
                <span>{{ item.seat_label }}</span>
                <small v-if="item.ticket_code">Boleto {{ item.ticket_code }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Hero claro elegante ── */
.consulta-hero {
  position: relative;
  padding: 3.5rem 0 3rem;
  background:
    radial-gradient(ellipse at 20% 0%, rgba(59,130,246,0.10) 0%, transparent 55%),
    radial-gradient(ellipse at 80% 100%, rgba(16,185,129,0.08) 0%, transparent 55%),
    linear-gradient(180deg, var(--brand-50) 0%, #ffffff 70%);
  border-bottom: 1px solid var(--color-border);
  overflow: hidden;
}

.hero-bg-pattern {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(ellipse at 50% 50%, rgba(59,130,246,0.04) 0%, transparent 60%);
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
  background: var(--brand-50);
  border: 1px solid var(--brand-100);
  color: var(--brand-600);
  margin-bottom: 1rem;
  box-shadow: 0 6px 18px rgba(59,130,246,0.15);
}

.consulta-hero .hero-title {
  font-size: 2rem;
  font-weight: 800;
  color: var(--color-heading);
  letter-spacing: -0.02em;
  margin-bottom: 0.5rem;
}

.consulta-hero .hero-subtitle {
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
  color: var(--brand-400);
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
  border-color: var(--brand-400);
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
  background: linear-gradient(135deg, var(--brand-600), var(--brand-700));
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
  background: linear-gradient(135deg, var(--brand-500), var(--brand-600));
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
  border-radius: var(--radius-lg);
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

.ic-1 { background: var(--brand-50); color: var(--brand-600); }
.ic-2 { background: var(--warning-50); color: var(--warning-600); }
.ic-3 { background: var(--accent-50); color: var(--accent-700); }

@media (prefers-color-scheme: dark) {
  .ic-1 { background: rgba(27, 85, 245, 0.12); }
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
  border-radius: var(--radius-lg);
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

/* ── Result ── */
.result-card {
  background: var(--color-surface);
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
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

.status-badge {
  padding: 0.3rem 0.8rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.badge-success { background: var(--success-50); color: var(--success-700); }
.badge-warning { background: var(--warning-50); color: var(--warning-700); }
.badge-danger { background: var(--danger-50); color: var(--danger-700); }
.badge-muted { background: var(--color-background-mute); color: var(--color-text-muted); }

.detail-grid {
  padding: 1rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.detail-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.detail-item svg {
  flex-shrink: 0;
  color: var(--brand-500);
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

.seats-section {
  padding: 1rem 1.5rem 1.25rem;
  border-top: 1px solid var(--color-border);
}

.seats-section h3 {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--color-heading);
  margin-bottom: 0.65rem;
}

.seat-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.seat-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 0.85rem;
  background: var(--brand-50);
  border: 1px solid var(--brand-200);
  border-radius: var(--radius-md);
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--brand-700);
}

@media (prefers-color-scheme: dark) {
  .seat-chip {
    background: rgba(27, 85, 245, 0.1);
    border-color: rgba(27, 85, 245, 0.2);
  }
}

.seat-chip svg { color: var(--brand-500); }

.seat-chip-text { display: flex; flex-direction: column; line-height: 1.1; }
.seat-chip-text small { font-size: 0.7rem; font-weight: 500; opacity: 0.75; margin-top: 0.15rem; }

.trip-banner {
  padding: 1rem 1.5rem;
  background: var(--brand-50);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
@media (prefers-color-scheme: dark) { .trip-banner { background: rgba(27, 85, 245, 0.08); } }
.trip-route { display: flex; align-items: center; gap: 0.5rem; font-weight: 700; color: var(--brand-700); font-size: 0.95rem; }
.trip-stops { display: flex; align-items: center; gap: 0.5rem; color: var(--color-heading); font-size: 0.92rem; }
.trip-stops svg { color: var(--brand-500); }
.trip-departure { font-size: 0.82rem; color: var(--color-text-muted); }

/* ── Responsive ── */
@media (max-width: 640px) {
  .consulta-hero { padding: 2.5rem 0 2rem; }
  .consulta-hero .hero-title { font-size: 1.5rem; }
  .search-widget { flex-direction: column; }
  .info-cards { grid-template-columns: 1fr; }
}
</style>
