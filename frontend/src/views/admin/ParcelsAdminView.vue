<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getParcels, updateParcelStatus, payParcel, retryParcelBilling, cancelParcel, getBillingPDFUrl, type Parcel } from '../../api/client'
import Swal from 'sweetalert2'

// ── Print panel (after pay) ──
const showPrintPanel = ref(false)
const printSaleId = ref<number | null>(null)
const printParcelCode = ref('')
const printWhatsappUrl = ref('')
const printPdfFormat = ref('ticket')

function openPrintPDF(_format: string) {
  const iframe = document.querySelector('.print-preview') as HTMLIFrameElement
  if (iframe?.contentWindow) {
    iframe.contentWindow.print()
  } else if (printSaleId.value) {
    window.open(getBillingPDFUrl(printSaleId.value, printPdfFormat.value), '_blank')
  }
}
function openPrintWhatsApp() {
  if (printWhatsappUrl.value) window.open(printWhatsappUrl.value, '_blank')
}
function closePrintPanel() {
  showPrintPanel.value = false
}

const router = useRouter()

const parcels = ref<Parcel[]>([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const filterStatus = ref('')
const filterDateFrom = ref('')
const filterDateTo = ref('')
const page = ref(0)
const pageSize = 50

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

const nextStatusMap: Record<string, string> = {
  registered: 'boarded',
  boarded: 'in_transit',
  in_transit: 'arrived',
  arrived: 'ready_for_pickup',
  ready_for_pickup: 'delivered',
}

const nextStatusLabel: Record<string, string> = {
  registered: 'Embarcar',
  boarded: 'En transito',
  in_transit: 'Llego',
  arrived: 'Lista p/ recoger',
  ready_for_pickup: 'Entregar',
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const hasNext = computed(() => (page.value + 1) < totalPages.value)
const hasPrev = computed(() => page.value > 0)

function formatAmount(cents: number): string {
  return 'S/ ' + (cents / 100).toFixed(2)
}

function formatDate(iso: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function todayStr(): string {
  return new Date().toISOString().slice(0, 10)
}

function setToday() {
  filterDateFrom.value = todayStr()
  filterDateTo.value = todayStr()
  onFilterChange()
}

function clearDates() {
  filterDateFrom.value = ''
  filterDateTo.value = ''
  onFilterChange()
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params: { status?: string; date_from?: string; date_to?: string; limit: number; offset: number } = {
      limit: pageSize,
      offset: page.value * pageSize,
    }
    if (filterStatus.value) params.status = filterStatus.value
    if (filterDateFrom.value) params.date_from = filterDateFrom.value
    if (filterDateTo.value) params.date_to = filterDateTo.value
    const result = await getParcels(params)
    parcels.value = result.data
    total.value = result.total
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

function onFilterChange() {
  page.value = 0
  load()
}

function nextPage() {
  if (hasNext.value) { page.value++; load() }
}

function prevPage() {
  if (hasPrev.value) { page.value--; load() }
}

async function handleAdvanceStatus(p: Parcel) {
  const next = nextStatusMap[p.status]
  if (!next) return
  const result = await Swal.fire({
    title: 'Cambiar estado',
    html: `<p style="color:#64748b;font-size:0.9rem;">Encomienda: <strong>${p.code}</strong></p>
           <p style="color:#64748b;font-size:0.85rem;margin-top:0.25rem;">${statusLabels[p.status]} &rarr; <strong>${statusLabels[next]}</strong></p>`,
    icon: 'question',
    showCancelButton: true,
    confirmButtonColor: '#3B82F6',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Confirmar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await updateParcelStatus(p.id, { status: next })
    await load()
    Swal.fire({ title: 'Estado actualizado', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    Swal.fire({ title: 'Error', text: (e as Error).message, icon: 'error' })
  }
}

async function handlePay(p: Parcel) {
  const { value: method } = await Swal.fire({
    title: 'Cobrar encomienda',
    html: `<p style="color:#64748b;font-size:0.9rem;">Encomienda: <strong>${p.code}</strong></p>
           <p style="color:#64748b;font-size:0.85rem;margin-top:0.25rem;">Monto: <strong>${formatAmount(p.amount_cents)}</strong></p>`,
    input: 'select',
    inputOptions: { cash: 'Efectivo', yape: 'Yape', transfer: 'Transferencia' },
    inputPlaceholder: 'Metodo de pago',
    showCancelButton: true,
    confirmButtonColor: '#16A34A',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Cobrar',
    cancelButtonText: 'Cancelar',
    customClass: { popup: 'swal-custom-popup' },
    inputValidator: (v: string) => { if (!v) return 'Selecciona un metodo de pago'; return null },
  })
  if (!method) return
  try {
    const result = await payParcel(p.id, method as string)
    await load()

    const updatedParcel = result.parcel
    if (updatedParcel?.billing_sale_id) {
      printSaleId.value = updatedParcel.billing_sale_id
      printParcelCode.value = updatedParcel.code || p.code
      printWhatsappUrl.value = result.whatsapp_url || ''
      printPdfFormat.value = 'ticket'
      showPrintPanel.value = true
    } else if (result.billing_error) {
      const retryNow = await Swal.fire({
        icon: 'warning',
        title: 'Pago confirmado sin comprobante',
        text: 'El cobro quedó registrado, pero falló la emisión del comprobante. ¿Reintentar ahora?',
        showCancelButton: true,
        confirmButtonColor: '#16A34A',
        cancelButtonColor: '#94A3B8',
        confirmButtonText: 'Reintentar emisión',
        cancelButtonText: 'Más tarde',
      })

      if (retryNow.isConfirmed) {
        const retry = await retryParcelBilling(p.id)
        if (retry.parcel?.billing_sale_id) {
          printSaleId.value = retry.parcel.billing_sale_id
          printParcelCode.value = retry.parcel.code || p.code
          printWhatsappUrl.value = retry.whatsapp_url || ''
          printPdfFormat.value = 'ticket'
          showPrintPanel.value = true
        } else if (retry.billing_error) {
          await Swal.fire({ title: 'No se pudo emitir', text: retry.billing_error, icon: 'error' })
        }
      }
    } else {
      Swal.fire({ title: 'Pago registrado', icon: 'success', timer: 1500, showConfirmButton: false })
    }
  } catch (e) {
    Swal.fire({ title: 'Error', text: (e as Error).message, icon: 'error' })
  }
}

async function handleCancel(p: Parcel) {
  const result = await Swal.fire({
    title: 'Anular encomienda?',
    html: `<p style="color:#64748b;font-size:0.9rem;">Se anulara la encomienda <strong>${p.code}</strong></p>`,
    icon: 'warning',
    input: 'textarea',
    inputPlaceholder: 'Motivo de anulacion (opcional)',
    showCancelButton: true,
    confirmButtonColor: '#EF4444',
    cancelButtonColor: '#94A3B8',
    confirmButtonText: 'Si, anular',
    cancelButtonText: 'No',
    customClass: { popup: 'swal-custom-popup' },
  })
  if (!result.isConfirmed) return
  try {
    await cancelParcel(p.id, result.value || undefined)
    await load()
    Swal.fire({ title: 'Anulada', text: 'Encomienda anulada correctamente', icon: 'success', timer: 1500, showConfirmButton: false })
  } catch (e) {
    Swal.fire({ title: 'Error', text: (e as Error).message, icon: 'error' })
  }
}

function goToDetail(p: Parcel) {
  router.push({ name: 'admin-encomienda-detalle', params: { id: p.id } })
}

function goToNew() {
  router.push({ name: 'admin-encomienda-nueva' })
}

// ── Export helpers ──

function buildExportTitle(): string {
  let title = 'Reporte de Encomiendas'
  if (filterDateFrom.value || filterDateTo.value) {
    title += ` — ${filterDateFrom.value || '...'} al ${filterDateTo.value || '...'}`
  }
  if (filterStatus.value) {
    title += ` (${statusLabels[filterStatus.value] || filterStatus.value})`
  }
  return title
}

function buildDateRange(): string {
  if (filterDateFrom.value && filterDateTo.value) {
    if (filterDateFrom.value === filterDateTo.value) {
      return new Date(filterDateFrom.value + 'T12:00:00').toLocaleDateString('es-PE', { weekday: 'long', day: '2-digit', month: 'long', year: 'numeric' })
    }
    return `${new Date(filterDateFrom.value + 'T12:00:00').toLocaleDateString('es-PE', { day: '2-digit', month: 'short', year: 'numeric' })} — ${new Date(filterDateTo.value + 'T12:00:00').toLocaleDateString('es-PE', { day: '2-digit', month: 'short', year: 'numeric' })}`
  }
  return 'Todas las fechas'
}

function computeStats(rows: Parcel[]) {
  const totalCount = rows.length
  const paid = rows.filter(r => r.payment_status === 'paid')
  const pending = rows.filter(r => r.payment_status === 'pending')
  const cancelled = rows.filter(r => r.status === 'cancelled')
  const delivered = rows.filter(r => r.status === 'delivered')
  const totalAmount = rows.reduce((s, r) => s + r.amount_cents, 0)
  const paidAmount = paid.reduce((s, r) => s + r.amount_cents, 0)
  const pendingAmount = pending.reduce((s, r) => s + r.amount_cents, 0)
  const totalBultos = rows.reduce((s, r) => s + r.package_count, 0)
  return { totalCount, paid: paid.length, pending: pending.length, cancelled: cancelled.length, delivered: delivered.length, totalAmount, paidAmount, pendingAmount, totalBultos }
}

const statusColorMap: Record<string, { bg: string; color: string }> = {
  registered:       { bg: '#DBEAFE', color: '#1E40AF' },
  boarded:          { bg: '#E0E7FF', color: '#3730A3' },
  in_transit:       { bg: '#FEF3C7', color: '#92400E' },
  arrived:          { bg: '#EDE9FE', color: '#5B21B6' },
  ready_for_pickup: { bg: '#FFEDD5', color: '#9A3412' },
  delivered:        { bg: '#DCFCE7', color: '#166534' },
  cancelled:        { bg: '#FEE2E2', color: '#991B1B' },
}

function exportPDF() {
  const title = buildExportTitle()
  const dateRange = buildDateRange()
  const rows = parcels.value
  const stats = computeStats(rows)
  const printWin = window.open('', '_blank')
  if (!printWin) return

  const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>${title}</title>
<style>
  @page { size: landscape; margin: 12mm; }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: 'Segoe UI', -apple-system, Arial, sans-serif; color: #1e293b; font-size: 10px; line-height: 1.4; }

  /* Header */
  .report-header { display: flex; justify-content: space-between; align-items: flex-start; padding-bottom: 16px; margin-bottom: 16px; border-bottom: 3px solid #1e40af; }
  .brand { display: flex; align-items: center; gap: 12px; }
  .brand-icon { width: 44px; height: 44px; background: linear-gradient(135deg, #3b82f6, #1e40af); border-radius: 10px; display: flex; align-items: center; justify-content: center; color: white; font-size: 20px; font-weight: 900; }
  .brand-name { font-size: 22px; font-weight: 900; color: #1e293b; letter-spacing: -0.02em; }
  .brand-sub { font-size: 10px; color: #64748b; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; }
  .report-meta { text-align: right; }
  .report-meta .title { font-size: 13px; font-weight: 800; color: #1e40af; margin-bottom: 2px; }
  .report-meta .date-range { font-size: 11px; font-weight: 600; color: #334155; }
  .report-meta .generated { font-size: 9px; color: #94a3b8; margin-top: 4px; }

  /* Dashboard */
  .dashboard { display: flex; gap: 10px; margin-bottom: 18px; }
  .kpi { flex: 1; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 10px 14px; text-align: center; position: relative; overflow: hidden; }
  .kpi::before { content: ''; position: absolute; top: 0; left: 0; right: 0; height: 3px; }
  .kpi-blue::before { background: #3b82f6; }
  .kpi-green::before { background: #16a34a; }
  .kpi-amber::before { background: #f59e0b; }
  .kpi-purple::before { background: #8b5cf6; }
  .kpi-red::before { background: #ef4444; }
  .kpi .kpi-label { font-size: 8px; color: #64748b; text-transform: uppercase; font-weight: 700; letter-spacing: 0.08em; }
  .kpi .kpi-value { font-size: 20px; font-weight: 900; color: #1e293b; margin: 2px 0; }
  .kpi .kpi-sub { font-size: 8px; color: #94a3b8; font-weight: 500; }

  /* Table */
  table { width: 100%; border-collapse: separate; border-spacing: 0; border: 1px solid #cbd5e1; border-radius: 8px; overflow: hidden; }
  thead th { background: linear-gradient(180deg, #1e3a5f 0%, #1e293b 100%); color: #e2e8f0; padding: 8px 10px; text-align: left; font-size: 8px; text-transform: uppercase; letter-spacing: 0.06em; font-weight: 700; border-bottom: 2px solid #3b82f6; }
  thead th.r { text-align: right; }
  thead th.c { text-align: center; }
  tbody td { padding: 6px 10px; font-size: 9.5px; border-bottom: 1px solid #f1f5f9; vertical-align: middle; }
  tbody tr:nth-child(even) td { background: #f8fafc; }
  tbody tr:last-child td { border-bottom: none; }
  .cell-num { text-align: center; font-weight: 600; color: #475569; }
  .cell-code { font-family: 'SF Mono', 'Consolas', monospace; font-weight: 700; font-size: 9px; color: #1e293b; background: #f1f5f9; padding: 2px 6px; border-radius: 4px; }
  .cell-route { font-size: 9px; color: #475569; }
  .cell-route b { color: #1e293b; font-weight: 700; }
  .cell-name { font-weight: 600; color: #334155; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .cell-price { text-align: right; font-weight: 800; color: #16a34a; font-family: 'SF Mono', 'Consolas', monospace; font-size: 9.5px; }
  .cell-badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 7.5px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; }
  .cell-date { font-size: 8.5px; color: #64748b; white-space: nowrap; }

  /* Totals */
  .totals-row td { background: #f1f5f9 !important; font-weight: 800; border-top: 2px solid #cbd5e1; font-size: 10px; }

  /* Footer */
  .report-footer { display: flex; justify-content: space-between; align-items: center; margin-top: 14px; padding-top: 10px; border-top: 1px solid #e2e8f0; }
  .footer-left { font-size: 8px; color: #94a3b8; }
  .footer-right { font-size: 8px; color: #94a3b8; }
  .footer-right b { color: #64748b; }

  @media print {
    body { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    .kpi, thead th, .cell-badge, .totals-row td { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  }
</style></head><body>

<div class="report-header">
  <div class="brand">
    <div class="brand-icon">P</div>
    <div>
      <div class="brand-name">Pasaje</div>
      <div class="brand-sub">Sistema de Transporte y Encomiendas</div>
    </div>
  </div>
  <div class="report-meta">
    <div class="title">Reporte de Encomiendas</div>
    <div class="date-range">${dateRange}</div>
    <div class="generated">Generado: ${new Date().toLocaleString('es-PE')}${filterStatus.value ? ' | Filtro: ' + statusLabels[filterStatus.value] : ''}</div>
  </div>
</div>

<div class="dashboard">
  <div class="kpi kpi-blue">
    <div class="kpi-label">Total Encomiendas</div>
    <div class="kpi-value">${stats.totalCount}</div>
    <div class="kpi-sub">${stats.totalBultos} bultos</div>
  </div>
  <div class="kpi kpi-green">
    <div class="kpi-label">Cobrado</div>
    <div class="kpi-value">S/ ${(stats.paidAmount / 100).toFixed(2)}</div>
    <div class="kpi-sub">${stats.paid} pagadas</div>
  </div>
  <div class="kpi kpi-amber">
    <div class="kpi-label">Por Cobrar</div>
    <div class="kpi-value">S/ ${(stats.pendingAmount / 100).toFixed(2)}</div>
    <div class="kpi-sub">${stats.pending} pendientes</div>
  </div>
  <div class="kpi kpi-purple">
    <div class="kpi-label">Entregadas</div>
    <div class="kpi-value">${stats.delivered}</div>
    <div class="kpi-sub">${((stats.delivered / Math.max(stats.totalCount, 1)) * 100).toFixed(0)}% del total</div>
  </div>
  <div class="kpi kpi-red">
    <div class="kpi-label">Anuladas</div>
    <div class="kpi-value">${stats.cancelled}</div>
    <div class="kpi-sub">${((stats.cancelled / Math.max(stats.totalCount, 1)) * 100).toFixed(0)}% del total</div>
  </div>
</div>

<table>
<thead><tr>
  <th class="c">#</th><th>Codigo</th><th>Origen</th><th>Destino</th><th>Remitente</th><th>Destinatario</th>
  <th class="c">Bultos</th><th class="r">Monto</th><th class="c">Estado</th><th class="c">Pago</th><th>Fecha</th>
</tr></thead>
<tbody>
${rows.map((p, i) => {
  const sc = statusColorMap[p.status] || { bg: '#f1f5f9', color: '#475569' }
  const pc = p.payment_status === 'paid' ? { bg: '#DCFCE7', color: '#166534' } : { bg: '#FEF3C7', color: '#92400E' }
  return `<tr>
  <td class="cell-num">${i + 1}</td>
  <td><span class="cell-code">${p.code}</span></td>
  <td class="cell-route"><b>${p.origin_stop_name}</b></td>
  <td class="cell-route"><b>${p.dest_stop_name}</b></td>
  <td class="cell-name">${p.sender_name}</td>
  <td class="cell-name">${p.receiver_name}</td>
  <td class="cell-num">${p.package_count}</td>
  <td class="cell-price">${formatAmount(p.amount_cents)}</td>
  <td style="text-align:center"><span class="cell-badge" style="background:${sc.bg};color:${sc.color}">${statusLabels[p.status] || p.status}</span></td>
  <td style="text-align:center"><span class="cell-badge" style="background:${pc.bg};color:${pc.color}">${p.payment_status === 'paid' ? 'Pagado' : 'Pendiente'}</span></td>
  <td class="cell-date">${formatDate(p.created_at)}</td>
</tr>`}).join('')}
<tr class="totals-row">
  <td colspan="6" style="text-align:right; color:#475569;">TOTALES</td>
  <td class="cell-num">${stats.totalBultos}</td>
  <td class="cell-price" style="font-size:11px;">S/ ${(stats.totalAmount / 100).toFixed(2)}</td>
  <td colspan="3"></td>
</tr>
</tbody>
</table>

<div class="report-footer">
  <div class="footer-left">Pasaje — Sistema de Transporte y Encomiendas</div>
  <div class="footer-right">Pagina 1 de 1 &nbsp;|&nbsp; Total: <b>${stats.totalCount}</b> registros</div>
</div>

</body></html>`

  printWin.document.write(html)
  printWin.document.close()
  printWin.onload = () => { printWin.print() }
}

function exportExcel() {
  const title = buildExportTitle()
  const dateRange = buildDateRange()
  const rows = parcels.value
  const stats = computeStats(rows)

  const S = {
    font: 'font-family:Calibri,Arial;',
    hBrand: 'font-size:20pt;font-weight:bold;color:#1e40af;font-family:Calibri,Arial;',
    hSub: 'font-size:10pt;color:#64748b;font-family:Calibri,Arial;',
    hTitle: 'font-size:12pt;font-weight:bold;color:#334155;font-family:Calibri,Arial;',
    hGen: 'font-size:8pt;color:#94a3b8;font-family:Calibri,Arial;',
    kpiLabel: 'font-size:8pt;color:#64748b;font-weight:bold;font-family:Calibri,Arial;text-align:center;',
    kpiVal: 'font-size:14pt;font-weight:bold;color:#1e293b;font-family:Calibri,Arial;text-align:center;',
    kpiSub: 'font-size:8pt;color:#94a3b8;font-family:Calibri,Arial;text-align:center;',
    th: 'background:#1e293b;color:#ffffff;font-weight:bold;font-size:9pt;font-family:Calibri,Arial;text-align:center;padding:6px 8px;border:1px solid #334155;',
    td: 'font-size:10pt;font-family:Calibri,Arial;border:1px solid #e2e8f0;padding:4px 6px;',
    tdEven: 'background:#f8fafc;',
    tdCode: 'font-weight:bold;background:#eef2ff;',
    tdName: 'font-weight:bold;color:#334155;',
    tdNum: 'text-align:center;font-weight:bold;',
    tdPrice: 'text-align:right;font-weight:bold;color:#16a34a;',
    totals: 'background:#dbeafe;font-weight:bold;font-size:11pt;border:1px solid #93c5fd;font-family:Calibri,Arial;padding:6px 8px;',
    footer: 'font-size:8pt;color:#94a3b8;font-family:Calibri,Arial;',
  }

  const thStyle = S.th
  const kpiBorder = (color: string) => `border-top:4px solid ${color};background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;`

  const html = `<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel" xmlns="http://www.w3.org/TR/REC-html40">
<head><meta charset="utf-8">
<!--[if gte mso 9]><xml><x:ExcelWorkbook><x:ExcelWorksheets><x:ExcelWorksheet>
<x:Name>Encomiendas</x:Name><x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions>
</x:ExcelWorksheet></x:ExcelWorksheets></x:ExcelWorkbook></xml><![endif]-->
</head><body>
<table>
  <tr><td colspan="14" style="${S.hBrand}">PASAJE</td></tr>
  <tr><td colspan="14" style="${S.hSub}">Sistema de Transporte y Encomiendas</td></tr>
  <tr><td colspan="14"></td></tr>
  <tr><td colspan="14" style="${S.hTitle}">${title}</td></tr>
  <tr><td colspan="14" style="${S.hSub}">${dateRange}</td></tr>
  <tr><td colspan="14" style="${S.hGen}">Generado: ${new Date().toLocaleString('es-PE')}</td></tr>
  <tr><td colspan="14"></td></tr>

  <!-- KPI labels -->
  <tr>
    <td colspan="2" style="${S.kpiLabel}${kpiBorder('#3b82f6')}">TOTAL ENCOMIENDAS</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiLabel}${kpiBorder('#16a34a')}">COBRADO</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiLabel}${kpiBorder('#f59e0b')}">POR COBRAR</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiLabel}${kpiBorder('#8b5cf6')}">ENTREGADAS</td>
    <td colspan="4" style="${S.font}"></td>
  </tr>
  <!-- KPI values -->
  <tr>
    <td colspan="2" style="${S.kpiVal}background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">${stats.totalCount}</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiVal}color:#16a34a;background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">S/ ${(stats.paidAmount / 100).toFixed(2)}</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiVal}color:#d97706;background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">S/ ${(stats.pendingAmount / 100).toFixed(2)}</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiVal}color:#7c3aed;background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;">${stats.delivered}</td>
    <td colspan="4" style="${S.font}"></td>
  </tr>
  <!-- KPI subs -->
  <tr>
    <td colspan="2" style="${S.kpiSub}background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;">${stats.totalBultos} bultos</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiSub}background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;">${stats.paid} pagadas</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiSub}background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;">${stats.pending} pendientes</td>
    <td style="${S.font}"></td>
    <td colspan="2" style="${S.kpiSub}background:#f8fafc;border-left:1px solid #e2e8f0;border-right:1px solid #e2e8f0;border-bottom:1px solid #e2e8f0;">${((stats.delivered / Math.max(stats.totalCount, 1)) * 100).toFixed(0)}% del total</td>
    <td colspan="4" style="${S.font}"></td>
  </tr>
  <tr><td colspan="14"></td></tr>

  <!-- Column headers -->
  <tr>
    <th style="${thStyle}">#</th>
    <th style="${thStyle}">CODIGO</th>
    <th style="${thStyle}">ORIGEN</th>
    <th style="${thStyle}">DESTINO</th>
    <th style="${thStyle}">REMITENTE</th>
    <th style="${thStyle}">DOC REMIT.</th>
    <th style="${thStyle}">TEL REMIT.</th>
    <th style="${thStyle}">DESTINATARIO</th>
    <th style="${thStyle}">DOC DEST.</th>
    <th style="${thStyle}">TEL DEST.</th>
    <th style="${thStyle}">BULTOS</th>
    <th style="${thStyle}">MONTO</th>
    <th style="${thStyle}">ESTADO</th>
    <th style="${thStyle}">PAGO</th>
  </tr>

  ${rows.map((p, i) => {
    const sc = statusColorMap[p.status] || { bg: '#f1f5f9', color: '#475569' }
    const pc = p.payment_status === 'paid' ? { bg: '#DCFCE7', color: '#166534' } : { bg: '#FEF3C7', color: '#92400E' }
    const bg = i % 2 === 1 ? S.tdEven : ''
    const base = S.td + bg
    return `<tr>
    <td style="${base}${S.tdNum}">${i + 1}</td>
    <td style="${base}${S.tdCode}">${p.code}</td>
    <td style="${base}${S.tdName}">${p.origin_stop_name}</td>
    <td style="${base}${S.tdName}">${p.dest_stop_name}</td>
    <td style="${base}${S.tdName}">${p.sender_name}</td>
    <td style="${base}">${p.sender_doc_type} ${p.sender_doc_number}</td>
    <td style="${base}">${p.sender_phone || ''}</td>
    <td style="${base}${S.tdName}">${p.receiver_name}</td>
    <td style="${base}">${p.receiver_doc_type} ${p.receiver_doc_number}</td>
    <td style="${base}">${p.receiver_phone}</td>
    <td style="${base}${S.tdNum}">${p.package_count}</td>
    <td style="${base}${S.tdPrice}">${(p.amount_cents / 100).toFixed(2)}</td>
    <td style="${base}background:${sc.bg};color:${sc.color};font-weight:bold;text-align:center;">${statusLabels[p.status] || p.status}</td>
    <td style="${base}background:${pc.bg};color:${pc.color};font-weight:bold;text-align:center;">${p.payment_status === 'paid' ? 'Pagado' : 'Pendiente'}</td>
  </tr>`}).join('')}

  <tr>
    <td colspan="10" style="${S.totals}text-align:right;">TOTALES</td>
    <td style="${S.totals}text-align:center;">${stats.totalBultos}</td>
    <td style="${S.totals}text-align:right;color:#16a34a;">S/ ${(stats.totalAmount / 100).toFixed(2)}</td>
    <td style="${S.totals}"></td>
    <td style="${S.totals}"></td>
  </tr>

  <tr><td colspan="14"></td></tr>
  <tr><td colspan="14" style="${S.footer}">Pasaje — Sistema de Transporte y Encomiendas | ${stats.totalCount} registros</td></tr>
</table>
</body></html>`

  const blob = new Blob(['\uFEFF' + html], { type: 'application/vnd.ms-excel;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `Encomiendas_${filterDateFrom.value || 'todos'}_${filterDateTo.value || ''}.xls`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  // Iniciar con la fecha de hoy
  filterDateFrom.value = todayStr()
  filterDateTo.value = todayStr()
  load()
})
</script>

<template>
  <div class="admin-page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Encomiendas</h1>
        <p class="page-subtitle">Administra los envios y encomiendas</p>
      </div>
      <button class="btn btn-primary" @click="goToNew">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 12h14"/>
          <path d="M12 5v14"/>
        </svg>
        Nueva Encomienda
      </button>
    </div>

    <!-- Filters -->
    <div class="filters-bar">
      <div class="filters-row">
        <div class="filter-group">
          <label class="filter-label">Desde</label>
          <input type="date" v-model="filterDateFrom" @change="onFilterChange" class="filter-input" />
        </div>
        <div class="filter-group">
          <label class="filter-label">Hasta</label>
          <input type="date" v-model="filterDateTo" @change="onFilterChange" class="filter-input" />
        </div>
        <div class="filter-group">
          <label class="filter-label">Estado</label>
          <div class="custom-select-wrapper">
            <select v-model="filterStatus" @change="onFilterChange" class="custom-select">
              <option value="">Todos</option>
              <option value="registered">Registrada</option>
              <option value="boarded">Embarcada</option>
              <option value="in_transit">En transito</option>
              <option value="arrived">Llego a destino</option>
              <option value="ready_for_pickup">Lista para recoger</option>
              <option value="delivered">Entregada</option>
              <option value="cancelled">Anulada</option>
            </select>
            <div class="custom-select-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>
            </div>
          </div>
        </div>
        <div class="filter-group filter-buttons">
          <label class="filter-label">&nbsp;</label>
          <div class="filter-btn-row">
            <button class="btn btn-ghost btn-sm" @click="setToday">Hoy</button>
            <button class="btn btn-ghost btn-sm" @click="clearDates">Todas</button>
          </div>
        </div>
      </div>
      <div class="filters-right">
        <div class="filter-count">
          {{ total }} encomienda{{ total !== 1 ? 's' : '' }}
        </div>
        <div class="export-buttons">
          <button class="btn btn-export" @click="exportPDF" :disabled="parcels.length === 0" title="Exportar PDF">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
            PDF
          </button>
          <button class="btn btn-export" @click="exportExcel" :disabled="parcels.length === 0" title="Exportar Excel/CSV">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
              <line x1="8" y1="13" x2="16" y2="13"/>
              <line x1="8" y1="17" x2="16" y2="17"/>
            </svg>
            Excel
          </button>
        </div>
      </div>
    </div>

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
            <th>Codigo</th>
            <th>Ruta</th>
            <th>Remitente</th>
            <th>Destinatario</th>
            <th>Bultos</th>
            <th>Monto</th>
            <th>Estado</th>
            <th>Pago</th>
            <th>Fecha</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in parcels" :key="p.id">
            <td class="td-code">
              <code class="code-badge">{{ p.code }}</code>
            </td>
            <td class="td-route">
              <span class="route-text">{{ p.origin_stop_name }} &rarr; {{ p.dest_stop_name }}</span>
            </td>
            <td class="td-name">{{ p.sender_name }}</td>
            <td class="td-name">{{ p.receiver_name }}</td>
            <td class="td-center">{{ p.package_count }}</td>
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
            <td class="td-date">{{ formatDate(p.created_at) }}</td>
            <td class="td-actions">
              <button
                v-if="nextStatusMap[p.status]"
                class="btn-action btn-action-advance"
                :title="nextStatusLabel[p.status]"
                @click="handleAdvanceStatus(p)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>
                </svg>
                <span class="action-label">{{ nextStatusLabel[p.status] }}</span>
              </button>
              <button
                v-if="p.payment_status === 'pending' && p.status !== 'cancelled'"
                class="btn-action btn-action-pay"
                title="Cobrar"
                @click="handlePay(p)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>
                </svg>
                <span class="action-label">Cobrar</span>
              </button>
              <button class="btn-icon" title="Ver detalle" @click="goToDetail(p)">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
                </svg>
              </button>
              <button
                v-if="p.status !== 'cancelled' && p.status !== 'delivered'"
                class="btn-icon btn-icon-danger"
                title="Anular"
                @click="handleCancel(p)"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="parcels.length === 0 && !loading">
            <td colspan="10" class="td-empty">No hay encomiendas para los filtros seleccionados.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="pagination">
      <button class="btn btn-ghost btn-sm" :disabled="!hasPrev" @click="prevPage">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>
        Anterior
      </button>
      <span class="pagination-info">
        Pagina {{ page + 1 }} de {{ totalPages }}
      </span>
      <button class="btn btn-ghost btn-sm" :disabled="!hasNext" @click="nextPage">
        Siguiente
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
      </button>
    </div>

    <!-- Print Panel Overlay -->
    <Teleport to="body">
      <div v-if="showPrintPanel" class="print-overlay">
        <div class="print-panel">
          <div class="print-header">
            <div>
              <h2 class="print-title">Comprobante emitido</h2>
              <p class="print-code">Encomienda: <strong>{{ printParcelCode }}</strong></p>
            </div>
            <button class="print-close" @click="closePrintPanel">&times;</button>
          </div>
          <div class="print-formats">
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'ticket' }" @click="printPdfFormat = 'ticket'">Ticket 80mm</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4media' }" @click="printPdfFormat = 'a4media'">A4 Media</button>
            <button class="print-fmt-btn" :class="{ active: printPdfFormat === 'a4' }" @click="printPdfFormat = 'a4'">A4 Full</button>
          </div>
          <iframe v-if="printSaleId" :src="getBillingPDFUrl(printSaleId, printPdfFormat)" class="print-preview" title="Vista previa"></iframe>
          <div class="print-actions">
            <button class="print-action-btn print-btn-print" @click="openPrintPDF(printPdfFormat)">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 6 2 18 2 18 9"/><path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><rect width="12" height="8" x="6" y="14"/></svg>
              Imprimir
            </button>
            <button v-if="printWhatsappUrl" class="print-action-btn print-btn-wa" @click="openPrintWhatsApp">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/></svg>
              WhatsApp
            </button>
            <button class="print-action-btn print-btn-close" @click="closePrintPanel">Cerrar</button>
          </div>
        </div>
      </div>
    </Teleport>
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

/* -- Filters -- */
.filters-bar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: var(--radius-lg);
  padding: 1rem 1.25rem;
  box-shadow: var(--shadow-sm);
}

.filters-row {
  display: flex;
  align-items: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.filters-right {
  display: flex;
  align-items: flex-end;
  gap: 1rem;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.filter-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--slate-600);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.filter-input {
  padding: 0.55rem 0.75rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.85rem;
  font-weight: 500;
  font-family: inherit;
  transition: all 0.2s ease;
  outline: none;
  min-width: 140px;
}

.filter-input:focus {
  border-color: var(--brand-400);
  background: white;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
}

.filter-btn-row {
  display: flex;
  gap: 0.35rem;
}

.filter-count {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-500);
  padding-bottom: 0.25rem;
  white-space: nowrap;
}

.export-buttons {
  display: flex;
  gap: 0.35rem;
}

.btn-export {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.45rem 0.75rem;
  background: var(--slate-50);
  color: var(--slate-700);
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-sm);
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
}

.btn-export:hover:not(:disabled) {
  background: var(--brand-50);
  border-color: var(--brand-300);
  color: var(--brand-600);
}

.btn-export:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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

.btn-sm {
  padding: 0.45rem 0.85rem;
  font-size: 0.82rem;
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
  flex-shrink: 0;
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

.btn-action {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.3rem 0.6rem;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
  line-height: 1;
}

.btn-action-advance {
  background: var(--brand-50);
  color: var(--brand-600);
  border: 1px solid var(--brand-200);
}

.btn-action-advance:hover {
  background: var(--brand-100);
  color: var(--brand-700);
}

.btn-action-pay {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-200);
}

.btn-action-pay:hover {
  background: var(--success-100);
  color: var(--success-800);
}

/* -- Custom Select -- */
.custom-select-wrapper {
  position: relative;
  min-width: 180px;
}

.custom-select {
  width: 100%;
  padding: 0.55rem 2.5rem 0.55rem 0.75rem;
  border: 2px solid var(--slate-300);
  border-radius: var(--radius-md);
  background: var(--slate-50);
  color: var(--slate-900);
  font-size: 0.85rem;
  font-weight: 500;
  font-family: inherit;
  transition: all 0.2s ease;
  outline: none;
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
  min-width: 1000px;
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

.td-code {
  white-space: nowrap;
}

.td-route {
  max-width: 180px;
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

.td-date {
  font-size: 0.8rem;
  color: var(--slate-500);
  white-space: nowrap;
}

.td-actions {
  display: flex;
  gap: 0.4rem;
  align-items: center;
  flex-wrap: nowrap;
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

.action-label {
  display: inline;
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

.badge-blue { background: #EFF6FF; color: #2563EB; }
.badge-indigo { background: #EEF2FF; color: #4F46E5; }
.badge-yellow { background: var(--warning-50); color: var(--warning-700); }
.badge-purple { background: #F5F3FF; color: #7C3AED; }
.badge-orange { background: #FFF7ED; color: #C2410C; }
.badge-green { background: var(--success-50); color: var(--success-700); }
.badge-red { background: var(--danger-50); color: var(--danger-700); }
.badge-muted { background: var(--slate-100); color: var(--slate-500); }

/* -- Pagination -- */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-top: 1.25rem;
  padding: 0.5rem 0;
}

.pagination-info {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--slate-500);
}

/* -- Responsive -- */
@media (max-width: 768px) {
  .page-head {
    flex-direction: column;
  }

  .filters-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filters-row {
    flex-direction: column;
  }

  .filters-right {
    flex-direction: row;
    justify-content: space-between;
  }

  .custom-select-wrapper {
    min-width: unset;
  }

  .action-label {
    display: none;
  }

  .btn-action {
    padding: 0.3rem 0.4rem;
  }

  .data-table {
    min-width: 900px;
  }

  .filter-input {
    min-width: unset;
    width: 100%;
  }
}

/* ── Print Panel ── */
.print-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.print-panel {
  background: white;
  border-radius: var(--radius-xl);
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  border: 1px solid var(--slate-200);
  animation: printIn 0.25s ease;
}

@keyframes printIn {
  from { opacity: 0; transform: scale(0.95) translateY(10px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.print-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--slate-200);
}

.print-title { font-size: 1.1rem; font-weight: 800; color: var(--slate-900); }
.print-code { font-size: 0.85rem; color: var(--slate-500); margin-top: 0.15rem; }
.print-close { background: none; border: none; font-size: 1.5rem; color: var(--slate-400); cursor: pointer; line-height: 1; padding: 0.25rem; }
.print-close:hover { color: var(--slate-900); }

.print-formats {
  display: flex; gap: 0.5rem; padding: 1rem 1.5rem;
  background: var(--slate-50); border-bottom: 1px solid var(--slate-200);
}

.print-fmt-btn {
  flex: 1; padding: 0.55rem 0.75rem; border: 2px solid var(--slate-300);
  border-radius: var(--radius-md); background: white; color: var(--slate-600);
  font-size: 0.82rem; font-weight: 600; cursor: pointer; transition: all 0.15s ease;
  font-family: inherit; text-align: center;
}

.print-fmt-btn:hover { border-color: var(--brand-300); color: var(--brand-600); }

.print-fmt-btn.active {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white; border-color: var(--brand-500); box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.print-preview { width: 100%; height: 400px; border: none; display: block; }

.print-actions {
  display: flex; gap: 0.5rem; padding: 1rem 1.5rem;
  border-top: 1px solid var(--slate-200); background: var(--slate-50);
}

.print-action-btn {
  flex: 1; display: inline-flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 0.65rem 1rem; border: none; border-radius: var(--radius-md);
  font-size: 0.88rem; font-weight: 600; cursor: pointer; transition: all 0.15s ease; font-family: inherit;
}

.print-btn-print {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white; box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}
.print-btn-print:hover { background: linear-gradient(135deg, var(--brand-500), var(--brand-600)); }

.print-btn-wa { background: #25D366; color: white; box-shadow: 0 2px 8px rgba(37, 211, 102, 0.3); }
.print-btn-wa:hover { background: #1fb855; }

.print-btn-close { background: var(--slate-100); color: var(--slate-700); border: 2px solid var(--slate-300); }
.print-btn-close:hover { background: var(--slate-200); color: var(--slate-900); }
</style>
