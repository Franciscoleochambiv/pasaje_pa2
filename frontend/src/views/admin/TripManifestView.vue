<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getTripManifest, type TripManifest, type SeatInfo, type Parcel } from '../../api/client'

const route = useRoute()

const manifest = ref<TripManifest | null>(null)
const loading = ref(false)
const error = ref('')

const rawTripId = route.params.tripId
const tripId = Number(Array.isArray(rawTripId) ? rawTripId[0] : rawTripId) || 0

const statusLabels: Record<string, string> = {
  registered: 'Registrada',
  boarded: 'Embarcada',
  in_transit: 'En transito',
  arrived: 'Llego a destino',
  ready_for_pickup: 'Lista para recoger',
  delivered: 'Entregada',
  cancelled: 'Anulada',
}

const statusBadgeClass: Record<string, string> = {
  registered: 'badge-blue',
  boarded: 'badge-indigo',
  in_transit: 'badge-yellow',
  arrived: 'badge-purple',
  ready_for_pickup: 'badge-orange',
  delivered: 'badge-green',
  cancelled: 'badge-red',
}

const soldSeats = computed<SeatInfo[]>(() => {
  if (!manifest.value) return []
  return manifest.value.seats.filter(s => s.status === 'sold')
})

const activeParcels = computed<Parcel[]>(() => {
  if (!manifest.value) return []
  return manifest.value.parcels.filter(p => p.status !== 'cancelled')
})

const totalRevenue = computed(() => {
  if (!manifest.value) return 0
  return manifest.value.totals.revenue_passengers + manifest.value.totals.revenue_parcels
})

function formatAmount(cents: number): string {
  return 'S/ ' + (cents / 100).toFixed(2)
}

function printPage() {
  window.print()
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    manifest.value = await getTripManifest(tripId)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <!-- Header -->
    <div class="page-head">
      <div>
        <h1 class="page-title">Hoja de Ruta</h1>
        <p class="page-subtitle" v-if="manifest">
          {{ manifest.trip.route_name }} &mdash; {{ manifest.trip.vehicle_name }}
        </p>
      </div>
      <button class="btn btn-primary no-print" @click="printPage">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="6 9 6 2 18 2 18 9"/>
          <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/>
          <rect x="6" y="14" width="12" height="8"/>
        </svg>
        Imprimir
      </button>
    </div>

    <!-- Error -->
    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <span>Cargando manifiesto...</span>
    </div>

    <template v-if="manifest && !loading">
      <!-- Summary Cards -->
      <div class="summary-row">
        <div class="summary-card">
          <span class="summary-label">Total Pasajeros</span>
          <span class="summary-value">{{ manifest.totals.passengers }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">Total Encomiendas</span>
          <span class="summary-value">{{ manifest.totals.parcels }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">Ingresos Pasajes</span>
          <span class="summary-value summary-money">{{ formatAmount(manifest.totals.revenue_passengers) }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">Ingresos Encomiendas</span>
          <span class="summary-value summary-money">{{ formatAmount(manifest.totals.revenue_parcels) }}</span>
        </div>
        <div class="summary-card summary-card-total">
          <span class="summary-label">Ingreso Total</span>
          <span class="summary-value summary-money">{{ formatAmount(totalRevenue) }}</span>
        </div>
      </div>

      <!-- Passengers Table -->
      <div class="section">
        <h2 class="section-title">
          Pasajeros
          <span class="section-count">{{ soldSeats.length }}</span>
        </h2>
        <div class="table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>Asiento</th>
                <th>Piso</th>
                <th>Estado</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="seat in soldSeats" :key="seat.id">
                <td class="td-seat-label">{{ seat.label }}</td>
                <td class="td-center">{{ seat.floor }}</td>
                <td>
                  <span class="status-badge badge-green">Vendido</span>
                </td>
              </tr>
              <tr v-if="soldSeats.length === 0">
                <td colspan="3" class="td-empty">No hay pasajeros</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Parcels Table -->
      <div class="section">
        <h2 class="section-title">
          Encomiendas
          <span class="section-count">{{ activeParcels.length }}</span>
        </h2>
        <div class="table-wrapper">
          <table class="data-table">
            <thead>
              <tr>
                <th>Codigo</th>
                <th>Origen / Destino</th>
                <th>Remitente</th>
                <th>Destinatario</th>
                <th>Bultos</th>
                <th>Peso</th>
                <th>Monto</th>
                <th>Estado</th>
                <th>Pago</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="p in activeParcels"
                :key="p.id"
                class="parcel-row"
                @click="$router.push(`/admin/encomiendas/${p.id}`)"
              >
                <td class="td-code">
                  <code class="code-badge">{{ p.code }}</code>
                </td>
                <td class="td-route">
                  <span class="route-text">{{ p.origin_stop_name }} &rarr; {{ p.dest_stop_name }}</span>
                </td>
                <td class="td-name">{{ p.sender_name }}</td>
                <td class="td-name">{{ p.receiver_name }}</td>
                <td class="td-center">{{ p.package_count }}</td>
                <td class="td-center">{{ p.weight_kg }} kg</td>
                <td class="td-price">{{ formatAmount(p.amount_cents) }}</td>
                <td>
                  <span class="status-badge" :class="statusBadgeClass[p.status] || 'badge-muted'">
                    {{ statusLabels[p.status] || p.status }}
                  </span>
                </td>
                <td>
                  <span class="status-badge" :class="p.payment_status === 'paid' ? 'badge-green' : 'badge-yellow'">
                    {{ p.payment_status === 'paid' ? 'Pagado' : 'Pendiente' }}
                  </span>
                </td>
              </tr>
              <tr v-if="activeParcels.length === 0">
                <td colspan="9" class="td-empty">No hay encomiendas</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
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
  font-size: 0.95rem;
  color: var(--slate-500);
  margin-top: 0.25rem;
  font-weight: 500;
}

/* -- Summary Cards -- */
.summary-row {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.summary-card {
  flex: 1;
  min-width: 150px;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  box-shadow: var(--shadow-sm);
}

.summary-card-total {
  background: linear-gradient(135deg, var(--brand-50), var(--brand-100));
  border-color: var(--brand-200);
}

.summary-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--slate-500);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.summary-value {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--slate-900);
  letter-spacing: -0.02em;
}

.summary-money {
  color: var(--success-600);
}

.summary-card-total .summary-money {
  color: var(--brand-700);
}

/* -- Sections -- */
.section {
  margin-bottom: 2rem;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--slate-800);
  margin-bottom: 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.section-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.6rem;
  height: 1.6rem;
  padding: 0 0.4rem;
  background: var(--brand-100);
  color: var(--brand-700);
  border-radius: var(--radius-full);
  font-size: 0.78rem;
  font-weight: 700;
}

/* -- Buttons -- */
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

/* -- Alert -- */
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

/* -- Loading -- */
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

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* -- Table -- */
.table-wrapper {
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow-md);
  overflow-x: auto;
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
  white-space: nowrap;
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

.parcel-row {
  cursor: pointer;
}

.td-code {
  white-space: nowrap;
}

.td-seat-label {
  font-weight: 700;
  color: var(--slate-900);
  font-size: 0.95rem;
}

.td-route {
  max-width: 200px;
}

.route-text {
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--slate-700);
}

.td-name {
  font-weight: 600;
  color: var(--slate-900);
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.td-center {
  text-align: center;
  font-weight: 600;
}

.td-price {
  font-weight: 700;
  color: var(--success-600);
  white-space: nowrap;
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

/* -- Status badges -- */
.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  white-space: nowrap;
}

.badge-blue {
  background: #EFF6FF;
  color: #2563EB;
}

.badge-indigo {
  background: #EEF2FF;
  color: #4F46E5;
}

.badge-yellow {
  background: var(--warning-50);
  color: var(--warning-700);
}

.badge-purple {
  background: #F5F3FF;
  color: #7C3AED;
}

.badge-orange {
  background: #FFF7ED;
  color: #C2410C;
}

.badge-green {
  background: var(--success-50);
  color: var(--success-700);
}

.badge-red {
  background: var(--danger-50);
  color: var(--danger-700);
}

.badge-muted {
  background: var(--slate-100);
  color: var(--slate-500);
}

/* -- Responsive -- */
@media (max-width: 768px) {
  .page-head {
    flex-direction: column;
  }

  .summary-row {
    flex-direction: column;
  }

  .summary-card {
    min-width: unset;
  }
}

/* -- Print Styles -- */
@media print {
  .no-print {
    display: none !important;
  }

  .admin-page {
    padding: 0;
  }

  .page-title {
    font-size: 1.2rem;
    color: #000;
  }

  .page-subtitle {
    color: #333;
  }

  .summary-row {
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .summary-card {
    border: 1px solid #ccc;
    box-shadow: none;
    padding: 0.5rem 0.75rem;
    background: white !important;
  }

  .summary-card-total {
    background: #f0f0f0 !important;
    border-color: #999;
  }

  .summary-label {
    font-size: 0.65rem;
    color: #555;
  }

  .summary-value {
    font-size: 1.1rem;
    color: #000;
  }

  .summary-money {
    color: #000 !important;
  }

  .table-wrapper {
    box-shadow: none;
    border: 1px solid #ccc;
  }

  .data-table thead {
    background: #333 !important;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .data-table th {
    color: white !important;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .data-table td {
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #ddd;
    font-size: 0.8rem;
  }

  .data-table tbody tr:hover {
    background: transparent;
  }

  .parcel-row {
    cursor: default;
  }

  .status-badge {
    border: 1px solid #ccc;
    background: transparent !important;
    color: #000 !important;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }

  .section {
    margin-bottom: 1rem;
    page-break-inside: avoid;
  }

  .section-title {
    font-size: 0.95rem;
  }

  .section-count {
    background: transparent;
    color: #000;
    border: 1px solid #999;
  }
}
</style>
