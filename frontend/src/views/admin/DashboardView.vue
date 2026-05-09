<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getAdminStats, type AdminStats } from '../../api/client'

const stats = ref<AdminStats | null>(null)
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    stats.value = await getAdminStats()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

const totalSeats = computed(() => {
  if (!stats.value) return 1
  return (stats.value.seats_available + stats.value.seats_sold + stats.value.seats_held + stats.value.seats_blocked) || 1
})

const seatPcts = computed(() => {
  if (!stats.value) return { available: 0, sold: 0, held: 0, blocked: 0 }
  const t = totalSeats.value
  return {
    available: (stats.value.seats_available / t) * 100,
    sold: (stats.value.seats_sold / t) * 100,
    held: (stats.value.seats_held / t) * 100,
    blocked: (stats.value.seats_blocked / t) * 100,
  }
})

const donutStyle = computed(() => {
  const p = seatPcts.value
  const a = p.available
  const b = a + p.sold
  const c = b + p.held
  return {
    background: `conic-gradient(
      #10B981 0deg ${a * 3.6}deg,
      #EF4444 ${a * 3.6}deg ${b * 3.6}deg,
      #F59E0B ${b * 3.6}deg ${c * 3.6}deg,
      #94A3B8 ${c * 3.6}deg 360deg
    )`
  }
})

const maxSale = computed(() => {
  if (!stats.value) return 1
  return Math.max(stats.value.sales_today, stats.value.sales_week, stats.value.sales_month, 1)
})

function fmt(n: number) {
  return n.toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

onMounted(load)
</script>

<template>
  <div class="dashboard">
    <div class="page-head">
      <h1 class="page-title">Dashboard</h1>
      <p class="page-subtitle">Resumen general del sistema</p>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>Cargando estadisticas...</span>
    </div>

    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
      <button class="btn btn-sm btn-ghost" @click="load">Reintentar</button>
    </div>

    <template v-else-if="stats">
      <!-- Row 1: KPI Cards -->
      <div class="kpi-grid">
        <div class="kpi-card">
          <div class="kpi-icon kpi-blue">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="6" cy="19" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/><circle cx="18" cy="5" r="3"/></svg>
          </div>
          <div class="kpi-body">
            <span class="kpi-value">{{ stats.total_routes }}</span>
            <span class="kpi-label">Rutas</span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon kpi-emerald">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="3" width="15" height="13" rx="2"/><path d="M16 8h4a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><circle cx="5.5" cy="18.5" r="2.5"/><circle cx="18.5" cy="18.5" r="2.5"/></svg>
          </div>
          <div class="kpi-body">
            <span class="kpi-value">{{ stats.total_vehicles }}</span>
            <span class="kpi-label">Vehiculos</span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon kpi-amber">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
          </div>
          <div class="kpi-body">
            <span class="kpi-value">{{ stats.upcoming_trips }}</span>
            <span class="kpi-label">Viajes Proximos</span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon kpi-green">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/></svg>
          </div>
          <div class="kpi-body">
            <span class="kpi-value">{{ stats.reservations_today }}</span>
            <span class="kpi-label">Reservas Hoy</span>
          </div>
        </div>
      </div>

      <!-- Row 2: Sales + Donut -->
      <div class="charts-row">
        <!-- Sales Card -->
        <div class="chart-card">
          <h3 class="chart-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            Ventas
          </h3>
          <div class="sales-big">
            <span class="sales-amount">S/ {{ fmt(stats.sales_today) }}</span>
            <span class="sales-label">Hoy</span>
          </div>
          <div class="sales-bars">
            <div class="sales-bar-row">
              <span class="bar-label">Hoy</span>
              <div class="bar-track">
                <div class="bar-fill bar-blue" :style="{ width: (stats.sales_today / maxSale * 100) + '%' }"></div>
              </div>
              <span class="bar-value">S/ {{ fmt(stats.sales_today) }}</span>
            </div>
            <div class="sales-bar-row">
              <span class="bar-label">Semana</span>
              <div class="bar-track">
                <div class="bar-fill bar-emerald" :style="{ width: (stats.sales_week / maxSale * 100) + '%' }"></div>
              </div>
              <span class="bar-value">S/ {{ fmt(stats.sales_week) }}</span>
            </div>
            <div class="sales-bar-row">
              <span class="bar-label">Mes</span>
              <div class="bar-track">
                <div class="bar-fill bar-gradient" :style="{ width: (stats.sales_month / maxSale * 100) + '%' }"></div>
              </div>
              <span class="bar-value">S/ {{ fmt(stats.sales_month) }}</span>
            </div>
          </div>
          <div class="sales-total">
            Total acumulado: <strong>S/ {{ fmt(stats.total_sales_amount) }}</strong>
          </div>
        </div>

        <!-- Donut Card -->
        <div class="chart-card">
          <h3 class="chart-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="6" width="20" height="12" rx="2"/><path d="M2 10h20"/><path d="M6 14h.01"/><path d="M10 14h.01"/></svg>
            Ocupacion de Asientos
          </h3>
          <div class="donut-area">
            <div class="donut" :style="donutStyle">
              <div class="donut-hole">
                <span class="donut-total">{{ totalSeats }}</span>
                <span class="donut-total-label">asientos</span>
              </div>
            </div>
            <div class="donut-legend">
              <div class="legend-item">
                <span class="legend-dot" style="background:#10B981"></span>
                <span class="legend-text">Disponibles</span>
                <strong>{{ stats.seats_available }}</strong>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background:#EF4444"></span>
                <span class="legend-text">Vendidos</span>
                <strong>{{ stats.seats_sold }}</strong>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background:#F59E0B"></span>
                <span class="legend-text">Retenidos</span>
                <strong>{{ stats.seats_held }}</strong>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background:#94A3B8"></span>
                <span class="legend-text">Bloqueados</span>
                <strong>{{ stats.seats_blocked }}</strong>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Row 3: Reservas Hoy -->
      <div class="reservations-card">
        <h3 class="chart-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          Reservas de Hoy
        </h3>
        <div class="res-pills">
          <div class="res-pill res-confirmed">
            <span class="res-pill-count">{{ stats.confirmed_today }}</span>
            <span class="res-pill-label">Confirmadas</span>
          </div>
          <div class="res-pill res-pending">
            <span class="res-pill-count">{{ stats.pending_today }}</span>
            <span class="res-pill-label">Pendientes</span>
          </div>
          <div class="res-pill res-expired">
            <span class="res-pill-count">{{ stats.expired_today }}</span>
            <span class="res-pill-label">Expiradas</span>
          </div>
          <div class="res-pill res-cancelled">
            <span class="res-pill-count">{{ stats.cancelled_today }}</span>
            <span class="res-pill-label">Canceladas</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.dashboard { padding: 0; }

.page-head { margin-bottom: 1.75rem; }
.page-title { font-size: 1.5rem; font-weight: 800; color: var(--slate-900); }
.page-subtitle { font-size: 0.9rem; color: var(--slate-500); margin-top: 0.25rem; }

.loading-state { display: flex; align-items: center; gap: 0.75rem; padding: 3rem 0; justify-content: center; color: var(--slate-500); }
.spinner { width: 22px; height: 22px; border: 3px solid var(--slate-200); border-top-color: var(--brand-400); border-radius: 50%; animation: spin 0.6s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ── KPI Grid ── */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.kpi-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  transition: all 0.3s ease;
}

.kpi-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.kpi-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: white;
}

.kpi-blue { background: linear-gradient(135deg, #60A5FA, #3B82F6); box-shadow: 0 3px 10px rgba(59,130,246,0.3); }
.kpi-emerald { background: linear-gradient(135deg, #34D399, #10B981); box-shadow: 0 3px 10px rgba(16,185,129,0.3); }
.kpi-amber { background: linear-gradient(135deg, #FBBF24, #F59E0B); box-shadow: 0 3px 10px rgba(245,158,11,0.3); }
.kpi-green { background: linear-gradient(135deg, #4ADE80, #22C55E); box-shadow: 0 3px 10px rgba(34,197,94,0.3); }

.kpi-body { display: flex; flex-direction: column; }
.kpi-value { font-size: 1.75rem; font-weight: 800; color: var(--slate-900); line-height: 1; }
.kpi-label { font-size: 0.78rem; font-weight: 600; color: var(--slate-500); text-transform: uppercase; letter-spacing: 0.04em; margin-top: 0.2rem; }

/* ── Charts Row ── */
.charts-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.chart-card {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  box-shadow: var(--shadow-md);
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--slate-900);
  margin-bottom: 1.25rem;
}

.chart-title svg { color: var(--brand-400); }

/* ── Sales Card ── */
.sales-big {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}

.sales-amount {
  font-size: 2rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.sales-label {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--slate-400);
  text-transform: uppercase;
}

.sales-bars {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.sales-bar-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.bar-label {
  width: 55px;
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-500);
  flex-shrink: 0;
}

.bar-track {
  flex: 1;
  height: 10px;
  background: var(--slate-100);
  border-radius: 5px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: 5px;
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
  min-width: 4px;
}

.bar-blue { background: linear-gradient(90deg, #60A5FA, #3B82F6); }
.bar-emerald { background: linear-gradient(90deg, #34D399, #10B981); }
.bar-gradient { background: linear-gradient(90deg, #60A5FA, #10B981); }

.bar-value {
  width: 85px;
  text-align: right;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--slate-700);
  flex-shrink: 0;
}

.sales-total {
  padding-top: 0.75rem;
  border-top: 1px solid var(--slate-100);
  font-size: 0.85rem;
  color: var(--slate-500);
}

.sales-total strong {
  color: var(--slate-900);
  font-size: 1rem;
}

/* ── Donut Chart ── */
.donut-area {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.donut {
  width: 150px;
  height: 150px;
  border-radius: 50%;
  position: relative;
  flex-shrink: 0;
  box-shadow: var(--shadow-md);
}

.donut-hole {
  position: absolute;
  inset: 25px;
  background: white;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.donut-total {
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--slate-900);
  line-height: 1;
}

.donut-total-label {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--slate-400);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.donut-legend {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  flex: 1;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.82rem;
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  flex-shrink: 0;
}

.legend-text {
  flex: 1;
  color: var(--slate-500);
  font-weight: 500;
}

.legend-item strong {
  color: var(--slate-900);
  font-weight: 700;
}

/* ── Reservations Card ── */
.reservations-card {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  box-shadow: var(--shadow-md);
}

.res-pills {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
}

.res-pill {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1rem 0.5rem;
  border-radius: var(--radius-md);
  text-align: center;
  transition: transform 0.2s ease;
}

.res-pill:hover { transform: translateY(-2px); }

.res-pill-count {
  font-size: 1.75rem;
  font-weight: 800;
  line-height: 1;
  margin-bottom: 0.25rem;
}

.res-pill-label {
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.res-confirmed { background: #ECFDF5; }
.res-confirmed .res-pill-count { color: #059669; }
.res-confirmed .res-pill-label { color: #047857; }

.res-pending { background: #FFF7ED; }
.res-pending .res-pill-count { color: #D97706; }
.res-pending .res-pill-label { color: #B45309; }

.res-expired { background: var(--slate-100); }
.res-expired .res-pill-count { color: var(--slate-500); }
.res-expired .res-pill-label { color: var(--slate-400); }

.res-cancelled { background: #FEF2F2; }
.res-cancelled .res-pill-count { color: #DC2626; }
.res-cancelled .res-pill-label { color: #B91C1C; }

/* ── Responsive ── */
@media (max-width: 1024px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .charts-row { grid-template-columns: 1fr; }
}

@media (max-width: 640px) {
  .kpi-grid { grid-template-columns: 1fr; }
  .res-pills { grid-template-columns: repeat(2, 1fr); }
  .donut-area { flex-direction: column; }
}

/* ── Alert ── */
.alert { display: flex; align-items: center; gap: 0.5rem; padding: 0.85rem 1rem; border-radius: var(--radius-md); font-size: 0.88rem; }
.alert-error { background: #FEF2F2; color: #DC2626; border: 1px solid #FECACA; }
.btn { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.5rem 1rem; border: none; border-radius: var(--radius-md); font-size: 0.85rem; font-weight: 600; cursor: pointer; font-family: inherit; transition: all 0.15s ease; }
.btn-sm { padding: 0.35rem 0.75rem; font-size: 0.82rem; }
.btn-ghost { background: transparent; color: var(--slate-500); border: 2px solid var(--slate-300); }
.btn-ghost:hover { background: var(--slate-100); color: var(--slate-900); }
</style>
