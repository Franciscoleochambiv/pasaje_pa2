<script setup lang="ts">
import { computed } from 'vue'

export interface RendererSeat {
  id?: number
  label: string
  floor: number
  row_num: number
  col_num: number
  seat_type: string
  status?: string
}

export interface RendererElement {
  floor: number
  row_num: number
  col_num: number
  kind: string
  text?: string
}

const props = withDefaults(defineProps<{
  floors: number
  seats: RendererSeat[]
  elements: RendererElement[]
  selectedSeatIds?: Set<number>
  selectedSeatLabels?: Set<string>
  // Cuando es true, los asientos disparan @seat-click
  interactive?: boolean
  // Cuando es true, muestra una guía de cuadrícula (para edición)
  showGrid?: boolean
  // Forzar # de cols visibles (overrride). Si no, calcula desde los datos.
  layoutCols?: number
  // Filas mínimas a mostrar (para que el editor tenga espacio aunque esté vacío)
  minRows?: number
}>(), {
  selectedSeatIds: () => new Set<number>(),
  selectedSeatLabels: () => new Set<string>(),
  interactive: false,
  showGrid: false,
  minRows: 0,
})

const emit = defineEmits<{
  (e: 'seat-click', seat: RendererSeat): void
  (e: 'cell-click', payload: { floor: number; row: number; col: number }): void
}>()

// Lista de pisos a renderizar (descendente: segundo piso a la izquierda).
const floorList = computed(() => {
  const arr: number[] = []
  for (let i = 1; i <= Math.max(1, props.floors); i++) arr.push(i)
  // Mostramos primero el más alto (segundo piso a la izquierda como en la foto WhatsApp).
  return arr.sort((a, b) => b - a)
})

interface Cell {
  floor: number
  row: number
  col: number
  seat?: RendererSeat
  element?: RendererElement
}

function buildGrid(floor: number): Cell[][] {
  const fSeats = props.seats.filter(s => s.floor === floor)
  const fElems = props.elements.filter(e => e.floor === floor)

  let maxRow = props.minRows
  let maxCol = props.layoutCols ?? 0
  for (const s of fSeats) { if (s.row_num > maxRow) maxRow = s.row_num; if (s.col_num > maxCol) maxCol = s.col_num }
  for (const e of fElems) { if (e.row_num > maxRow) maxRow = e.row_num; if (e.col_num > maxCol) maxCol = e.col_num }
  if (maxRow < 1) maxRow = 1
  if (maxCol < 1) maxCol = props.layoutCols ?? 4

  const rows: Cell[][] = []
  for (let r = 1; r <= maxRow; r++) {
    const row: Cell[] = []
    for (let c = 1; c <= maxCol; c++) row.push({ floor, row: r, col: c })
    rows.push(row)
  }
  for (const s of fSeats) {
    if (s.row_num >= 1 && s.row_num <= maxRow && s.col_num >= 1 && s.col_num <= maxCol) {
      const cell = rows[s.row_num - 1]?.[s.col_num - 1]
      if (cell) cell.seat = s
    }
  }
  for (const e of fElems) {
    if (e.row_num >= 1 && e.row_num <= maxRow && e.col_num >= 1 && e.col_num <= maxCol) {
      const cell = rows[e.row_num - 1]?.[e.col_num - 1]
      if (cell) cell.element = e
    }
  }
  return rows
}

const grids = computed<Record<number, Cell[][]>>(() => {
  const out: Record<number, Cell[][]> = {}
  for (const f of floorList.value) out[f] = buildGrid(f)
  return out
})

function floorLabel(f: number): string {
  if (f === 1) return 'PRIMER PISO'
  if (f === 2) return 'SEGUNDO PISO'
  return `PISO ${f}`
}

function seatClasses(seat: RendererSeat): string {
  const out = ['seat']
  if (seat.id != null && props.selectedSeatIds.has(seat.id)) out.push('selected')
  else if (props.selectedSeatLabels.has(seat.label)) out.push('selected')
  if (seat.status === 'sold') out.push('sold')
  else if (seat.status === 'held') out.push('held')
  else if (seat.status === 'blocked') out.push('blocked')
  else out.push('available')
  if (seat.seat_type && seat.seat_type !== 'regular') out.push(`type-${seat.seat_type}`)
  return out.join(' ')
}

function handleCellClick(cell: Cell) {
  if (!props.interactive) return
  if (cell.seat) {
    emit('seat-click', cell.seat)
    return
  }
  emit('cell-click', { floor: cell.floor, row: cell.row, col: cell.col })
}
</script>

<template>
  <div class="seat-layout">
    <div class="bus-frame">
      <div class="bus-front">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <span>FRENTE</span>
      </div>

      <div class="floors-row">
        <div v-for="f in floorList" :key="f" class="floor-block">
          <div v-if="floorList.length > 1" class="floor-label">{{ floorLabel(f) }}</div>
          <div class="grid">
            <div v-for="(row, ri) in grids[f]" :key="`f${f}-r${ri}`" class="grid-row">
              <div
                v-for="cell in row" :key="`f${f}-${cell.row}-${cell.col}`"
                class="grid-cell"
                :class="{
                  empty: !cell.seat && !cell.element,
                  'show-grid': showGrid && !cell.seat && !cell.element,
                  clickable: interactive,
                }"
                @click="handleCellClick(cell)"
              >
                <button
                  v-if="cell.seat"
                  type="button"
                  class="seat-cell"
                  :class="seatClasses(cell.seat)"
                  :disabled="!interactive"
                  @click.stop="interactive && emit('seat-click', cell.seat!)"
                >
                  <span class="seat-label">{{ cell.seat.label }}</span>
                </button>

                <div v-else-if="cell.element" class="el" :class="`el-${cell.element.kind}`">
                  <span v-if="cell.element.kind === 'aisle'" class="el-text">{{ cell.element.text || 'PASILLO' }}</span>

                  <svg v-else-if="cell.element.kind === 'stairs'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 20 4 16 8 16 8 12 12 12 12 8 16 8 16 4 20 4"/></svg>

                  <svg v-else-if="cell.element.kind === 'tv' || cell.element.kind === 'screen'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="15" x="2" y="3" rx="2"/><polyline points="8 21 12 17 16 21"/></svg>

                  <svg v-else-if="cell.element.kind === 'wc'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3v18"/><path d="M17 3v18"/><circle cx="7" cy="7" r="2"/><circle cx="17" cy="7" r="2"/></svg>

                  <svg v-else-if="cell.element.kind === 'driver'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 12 9 9"/><path d="M12 12l3-3"/></svg>

                  <span v-else-if="cell.element.kind === 'icon_yape'" class="el-yape">YP</span>

                  <svg v-else-if="cell.element.kind === 'seat_woman'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="6" r="3"/><path d="M12 9v6"/><path d="M9 21l3-6 3 6"/></svg>

                  <svg v-else-if="cell.element.kind === 'seat_man'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="6" r="3"/><path d="M12 9v8"/><path d="M9 13h6"/><path d="M9 21l3-4 3 4"/></svg>

                  <svg v-else-if="cell.element.kind === 'bed'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 4v16"/><path d="M2 8h18a2 2 0 0 1 2 2v10"/><path d="M2 17h20"/><path d="M6 8v9"/></svg>

                  <span v-else-if="cell.element.kind === 'label'" class="el-text">{{ cell.element.text }}</span>

                  <span v-else class="el-text">{{ cell.element.kind }}</span>

                  <span v-if="cell.element.text && cell.element.kind !== 'aisle' && cell.element.kind !== 'label'" class="el-sub">{{ cell.element.text }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="bus-back">ATRÁS</div>
    </div>

    <slot name="legend">
      <div v-if="!interactive" class="legend">
        <span class="leg-item"><span class="dot dot-available"></span>Disponible</span>
        <span class="leg-item"><span class="dot dot-selected"></span>Seleccionado</span>
        <span class="leg-item"><span class="dot dot-sold"></span>Vendido</span>
        <span class="leg-item"><span class="dot dot-held"></span>Reservado</span>
      </div>
    </slot>
  </div>
</template>

<style scoped>
.seat-layout { display: flex; flex-direction: column; gap: 0.85rem; }

.floor-tabs { display: flex; gap: 0.5rem; }
.floor-tab {
  padding: 0.45rem 0.95rem;
  border-radius: 999px;
  border: 2px solid var(--slate-300);
  background: white;
  color: var(--slate-700);
  font-weight: 600;
  font-size: 0.82rem;
  cursor: pointer;
  transition: all 0.15s ease;
}
.floor-tab:hover { background: var(--slate-50); }
.floor-tab.active {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-color: transparent;
  box-shadow: var(--shadow-sm);
}

.bus-frame {
  position: relative;
  background: linear-gradient(180deg, #fafafa 0%, #f1f5f9 100%);
  border: 2px solid var(--slate-300);
  border-radius: 24px;
  padding: 2.4rem 1rem 2rem;
  overflow-x: auto;
}
.floors-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
  justify-content: center;
  align-items: flex-start;
}
.floor-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}
.floor-label {
  font-size: 0.7rem;
  font-weight: 800;
  color: var(--slate-700);
  letter-spacing: 0.12em;
  background: white;
  padding: 0.25rem 0.85rem;
  border-radius: 999px;
  border: 1px solid var(--slate-200);
}
.bus-front {
  position: absolute;
  top: 6px; left: 50%;
  transform: translateX(-50%);
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--slate-500);
  letter-spacing: 0.08em;
}
.bus-back {
  position: absolute;
  bottom: 6px; left: 50%;
  transform: translateX(-50%);
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--slate-400);
  letter-spacing: 0.08em;
}

.grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  width: max-content;
  margin: 0 auto;
}
.grid-row { display: flex; gap: 6px; }
.grid-cell {
  width: 44px; height: 44px;
  display: flex; align-items: center; justify-content: center;
}
.grid-cell.show-grid { border: 1px dashed var(--slate-200); border-radius: 6px; }
.grid-cell.clickable { cursor: pointer; }
.grid-cell.clickable.empty:hover { background: var(--brand-50); border: 1px dashed var(--brand-400); border-radius: 6px; }

/* Asiento */
.seat-cell {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid var(--slate-300);
  border-radius: 8px;
  background: white;
  font-size: 0.78rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
  color: var(--slate-800);
  padding: 0;
}
.seat-cell:disabled { cursor: default; }
.seat-cell.available { background: white; }
.seat-cell.available:not(:disabled):hover {
  border-color: var(--brand-400);
  background: var(--brand-50);
  transform: scale(1.05);
}
.seat-cell.selected {
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white;
  border-color: var(--brand-500);
  box-shadow: 0 4px 10px rgba(59, 130, 246, 0.35);
}
.seat-cell.sold {
  background: #f1f5f9;
  color: #94a3b8;
  border-color: #e2e8f0;
  text-decoration: line-through;
}
.seat-cell.held {
  background: #fef3c7;
  color: #92400e;
  border-color: #fcd34d;
}
.seat-cell.blocked {
  background: #fee2e2;
  color: #991b1b;
  border-color: #fca5a5;
}

/* Tipos de asiento (color de borde decorativo) */
.seat-cell.type-semi_cama { border-color: #a78bfa; }
.seat-cell.type-cama      { border-color: #f472b6; }
.seat-cell.type-suite     { border-color: #facc15; }

/* Tipos con relleno completo (colores plantilla WhatsApp). Solo se aplican
   cuando el asiento está disponible: si está vendido/reservado/seleccionado,
   ese estado pisa al tipo. */
.seat-cell.type-purple.available { background: #7c3aed; color: white; border-color: #6d28d9; }
.seat-cell.type-purple.available:not(:disabled):hover { background: #6d28d9; }
.seat-cell.type-yape.available {
  background: #06b6d4; color: white; border-color: #0891b2;
  position: relative;
}
.seat-cell.type-yape.available:not(:disabled):hover { background: #0891b2; }
.seat-cell.type-yape.available::before {
  content: 'YP';
  position: absolute;
  top: -3px; left: -3px;
  background: #06b6d4; color: white;
  font-size: 0.5rem; font-weight: 800;
  padding: 1px 3px; border-radius: 3px;
  border: 1px solid white;
  letter-spacing: 0.04em; line-height: 1;
}
.seat-cell.type-orange.available { background: #fb923c; color: white; border-color: #ea580c; }
.seat-cell.type-orange.available:not(:disabled):hover { background: #f97316; }

/* Elementos decorativos */
.el {
  width: 100%; height: 100%;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  border-radius: 6px;
  font-size: 0.62rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-align: center;
  color: var(--slate-600);
  line-height: 1;
}
.el-text { font-size: 0.6rem; }
.el-sub { font-size: 0.55rem; color: var(--slate-500); margin-top: 1px; font-weight: 500; }

.el-aisle { background: #ecfeff; color: #0e7490; border: 1px dashed #67e8f9; }
.el-stairs { background: #f5f3ff; color: #6d28d9; border: 1px solid #c4b5fd; }
.el-tv, .el-screen { background: #1e293b; color: white; border: 1px solid #334155; }
.el-wc { background: #ecfdf5; color: #047857; border: 1px solid #86efac; }
.el-driver { background: #fffbeb; color: #b45309; border: 1px solid #fcd34d; }
.el-seat_woman { background: #fdf2f8; color: #be185d; border: 1px solid #f9a8d4; }
.el-seat_man { background: #eff6ff; color: #1d4ed8; border: 1px solid #93c5fd; }
.el-bed { background: #fef3c7; color: #92400e; border: 1px solid #fcd34d; }
.el-label { background: transparent; color: var(--slate-700); font-weight: 800; }

.el-yape {
  background: #6b21a8;
  color: white;
  font-weight: 800;
  font-size: 0.7rem;
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
  border-radius: 6px;
  letter-spacing: 0.04em;
}

/* Leyenda */
.legend { display: flex; flex-wrap: wrap; gap: 0.85rem; padding-top: 0.5rem; }
.leg-item { display: inline-flex; align-items: center; gap: 0.4rem; font-size: 0.78rem; color: var(--slate-600); font-weight: 500; }
.dot { width: 14px; height: 14px; border-radius: 4px; border: 2px solid var(--slate-300); display: inline-block; }
.dot-available { background: white; }
.dot-selected { background: linear-gradient(135deg, var(--brand-400), var(--brand-500)); border-color: var(--brand-500); }
.dot-sold { background: #f1f5f9; border-color: #e2e8f0; }
.dot-held { background: #fef3c7; border-color: #fcd34d; }
</style>
