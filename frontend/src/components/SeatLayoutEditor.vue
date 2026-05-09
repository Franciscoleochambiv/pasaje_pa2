<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import type { BusLayout, BusLayoutSeat, BusLayoutElement } from '../api/client'
import { getDefaultBusLayoutSeed } from '../composables/defaultBusLayout'

interface Props {
  layout: BusLayout
  seats: BusLayoutSeat[]
  elements: BusLayoutElement[]
}
const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'save', payload: { layout: BusLayout; seats: BusLayoutSeat[]; elements: BusLayoutElement[] }): void
  (e: 'cancel'): void
}>()

// ── Estado local editable ──
const layout = reactive<BusLayout>({ ...props.layout })
const seats = ref<BusLayoutSeat[]>(props.seats.map(s => ({ ...s })))
const elements = ref<BusLayoutElement[]>(props.elements.map(e => ({ ...e })))

watch(() => props.layout, v => Object.assign(layout, v))
watch(() => props.seats, v => { seats.value = v.map(s => ({ ...s })) })
watch(() => props.elements, v => { elements.value = v.map(e => ({ ...e })) })

// ── Editor state ──
const activeFloor = ref(1)
type Tool =
  | { kind: 'select' }
  | { kind: 'erase' }
  | { kind: 'seat' }
  | { kind: 'element'; elementKind: string; defaultText?: string }
const tool = ref<Tool>({ kind: 'select' })

const floorList = computed(() => {
  const arr: number[] = []
  for (let i = 1; i <= Math.max(1, layout.floors || 1); i++) arr.push(i)
  return arr
})

// Tamaño visible del grid (cols globales del layout, filas calculadas por piso)
const visibleCols = computed(() => layout.layout_cols || 4)

// Mínimo de filas que el usuario "fijó" manualmente con el botón "+ fila"
// (el grid puede crecer también automáticamente si hay un asiento en una fila más abajo)
const minRowsPerFloor = reactive<Record<number, number>>({})

function visibleRowsFor(floor: number): number {
  let maxR = 0
  for (const s of seats.value) if (s.floor === floor && s.row_num > maxR) maxR = s.row_num
  for (const e of elements.value) if (e.floor === floor && e.row_num > maxR) maxR = e.row_num
  const min = minRowsPerFloor[floor] ?? 5
  return Math.max(min, maxR + 1)
}

function addRowToFloor(f: number) {
  minRowsPerFloor[f] = visibleRowsFor(f) + 1
}

// ── Grid construcción ──
interface Cell {
  floor: number
  row: number
  col: number
  seat?: BusLayoutSeat
  element?: BusLayoutElement
}

function buildGrid(floor: number): Cell[][] {
  const rowsCount = visibleRowsFor(floor)
  const colsCount = visibleCols.value
  const rows: Cell[][] = []
  for (let r = 1; r <= rowsCount; r++) {
    const row: Cell[] = []
    for (let c = 1; c <= colsCount; c++) row.push({ floor, row: r, col: c })
    rows.push(row)
  }
  for (const s of seats.value) {
    if (s.floor !== floor) continue
    if (s.row_num >= 1 && s.row_num <= rowsCount && s.col_num >= 1 && s.col_num <= colsCount) {
      const cell = rows[s.row_num - 1]?.[s.col_num - 1]
      if (cell) cell.seat = s
    }
  }
  for (const e of elements.value) {
    if (e.floor !== floor) continue
    if (e.row_num >= 1 && e.row_num <= rowsCount && e.col_num >= 1 && e.col_num <= colsCount) {
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

// ── Numeración automática ──
function nextSeatLabel(): string {
  const used = new Set(seats.value.map(s => parseInt(s.label, 10)).filter(n => !isNaN(n)))
  let n = 1
  while (used.has(n)) n++
  return String(n)
}

function nextSeatPosition(): number {
  let max = 0
  for (const s of seats.value) if (s.position > max) max = s.position
  return max + 1
}

function renumberByPosition() {
  // Asigna labels 1..N siguiendo orden floor → row → col
  const sorted = [...seats.value].sort((a, b) => {
    if (a.floor !== b.floor) return a.floor - b.floor
    if (a.row_num !== b.row_num) return a.row_num - b.row_num
    return a.col_num - b.col_num
  })
  sorted.forEach((s, i) => { s.label = String(i + 1); s.position = i + 1 })
  seats.value = sorted
}

// ── Acciones de celda ──
function placeAt(floor: number, row: number, col: number) {
  const t = tool.value
  if (t.kind === 'select' || t.kind === 'erase') return

  // Ya hay algo en esa celda → no apilar; solo reemplazar si herramienta es distinta
  removeAt(floor, row, col, true)

  if (t.kind === 'seat') {
    seats.value.push({
      label: nextSeatLabel(),
      position: nextSeatPosition(),
      floor, row_num: row, col_num: col,
      seat_type: layout.seat_type || 'regular',
    })
  } else if (t.kind === 'element') {
    elements.value.push({
      floor, row_num: row, col_num: col,
      kind: t.elementKind,
      text: t.defaultText ?? '',
    })
  }
}

function removeAt(floor: number, row: number, col: number, silent = false) {
  const sIdx = seats.value.findIndex(s => s.floor === floor && s.row_num === row && s.col_num === col)
  if (sIdx >= 0) seats.value.splice(sIdx, 1)
  const eIdx = elements.value.findIndex(e => e.floor === floor && e.row_num === row && e.col_num === col)
  if (eIdx >= 0) elements.value.splice(eIdx, 1)
  if (!silent && sIdx < 0 && eIdx < 0) {
    // nada
  }
}

function handleCellClick(cell: Cell) {
  if (tool.value.kind === 'erase') {
    removeAt(cell.floor, cell.row, cell.col)
    return
  }
  if (cell.seat || cell.element) {
    if (tool.value.kind === 'select') {
      openCellEditor(cell)
    }
    return
  }
  placeAt(cell.floor, cell.row, cell.col)
}

// ── Drag & drop ──
const dragSrc = ref<{ floor: number; row: number; col: number } | null>(null)

function onDragStart(ev: DragEvent, cell: Cell) {
  if (!cell.seat && !cell.element) { ev.preventDefault(); return }
  dragSrc.value = { floor: cell.floor, row: cell.row, col: cell.col }
  ev.dataTransfer?.setData('text/plain', `${cell.floor},${cell.row},${cell.col}`)
  if (ev.dataTransfer) ev.dataTransfer.effectAllowed = 'move'
}

function onDragOver(ev: DragEvent) {
  if (!dragSrc.value) return
  ev.preventDefault()
  if (ev.dataTransfer) ev.dataTransfer.dropEffect = 'move'
}

function onDrop(ev: DragEvent, cell: Cell) {
  ev.preventDefault()
  if (!dragSrc.value) return
  const src = dragSrc.value
  dragSrc.value = null
  if (src.floor === cell.floor && src.row === cell.row && src.col === cell.col) return

  const sIdx = seats.value.findIndex(s => s.floor === src.floor && s.row_num === src.row && s.col_num === src.col)
  const eIdx = elements.value.findIndex(e => e.floor === src.floor && e.row_num === src.row && e.col_num === src.col)

  const dstS = seats.value.findIndex(s => s.floor === cell.floor && s.row_num === cell.row && s.col_num === cell.col)
  const dstE = elements.value.findIndex(e => e.floor === cell.floor && e.row_num === cell.row && e.col_num === cell.col)

  // Helper para intercambiar posición/piso entre dos entidades arbitrarias.
  const swap = (a: { row_num: number; col_num: number; floor: number }, b: { row_num: number; col_num: number; floor: number }) => {
    const tmp = { row: a.row_num, col: a.col_num, floor: a.floor }
    a.row_num = b.row_num; a.col_num = b.col_num; a.floor = b.floor
    b.row_num = tmp.row; b.col_num = tmp.col; b.floor = tmp.floor
  }

  if (sIdx >= 0) {
    const a = seats.value[sIdx]
    if (!a) return
    if (dstS >= 0) {
      const b = seats.value[dstS]
      if (b) swap(a, b)
    } else if (dstE >= 0) {
      const b = elements.value[dstE]
      if (b) swap(a, b)
    } else {
      a.row_num = cell.row
      a.col_num = cell.col
      a.floor = cell.floor
    }
  } else if (eIdx >= 0) {
    const a = elements.value[eIdx]
    if (!a) return
    if (dstS >= 0) {
      const b = seats.value[dstS]
      if (b) swap(a, b)
    } else if (dstE >= 0) {
      const b = elements.value[dstE]
      if (b) swap(a, b)
    } else {
      a.row_num = cell.row
      a.col_num = cell.col
      a.floor = cell.floor
    }
  }
}

// ── Editor de celda (label / texto / tipo) ──
const editing = ref<Cell | null>(null)
const editLabel = ref('')
const editText = ref('')
const editSeatType = ref('regular')

function openCellEditor(cell: Cell) {
  editing.value = cell
  if (cell.seat) {
    editLabel.value = cell.seat.label
    editSeatType.value = cell.seat.seat_type
  } else if (cell.element) {
    editText.value = cell.element.text
  }
}

function applyEdit() {
  if (!editing.value) return
  const c = editing.value
  if (c.seat) {
    const s = seats.value.find(s => s.floor === c.floor && s.row_num === c.row && s.col_num === c.col)
    if (s) {
      s.label = editLabel.value.trim() || s.label
      s.seat_type = editSeatType.value
    }
  } else if (c.element) {
    const e = elements.value.find(e => e.floor === c.floor && e.row_num === c.row && e.col_num === c.col)
    if (e) e.text = editText.value
  }
  editing.value = null
}

function deleteEdited() {
  if (!editing.value) return
  removeAt(editing.value.floor, editing.value.row, editing.value.col)
  editing.value = null
}

// ── Acciones globales ──
function addCol() { layout.layout_cols = (layout.layout_cols || 4) + 1 }
function removeCol() { if ((layout.layout_cols || 4) > 1) layout.layout_cols = (layout.layout_cols || 4) - 1 }
function setFloors(n: number) {
  if (n < 1) n = 1
  if (n > 2) n = 2
  layout.floors = n
  if (activeFloor.value > n) activeFloor.value = 1
}

function fillSeats(rowsCount: number, colsCount: number, skipCols: number[] = []) {
  // Helper para seed: rellena el piso actual con asientos numerados, saltando cols especificadas (pasillos).
  for (let r = 1; r <= rowsCount; r++) {
    for (let c = 1; c <= colsCount; c++) {
      if (skipCols.includes(c)) continue
      placeAt(activeFloor.value, r, c)
    }
  }
}

function applyDefaultSeed() {
  const seed = getDefaultBusLayoutSeed()
  layout.floors = seed.floors
  layout.layout_cols = seed.layout_cols
  seats.value = seed.seats.map(s => ({ ...s }))
  elements.value = seed.elements.map(e => ({ ...e }))
  activeFloor.value = 2
}

async function loadDefaultSeed() {
  if (seats.value.length > 0 || elements.value.length > 0) {
    const ok = window.confirm('Esto reemplazará todos los asientos y elementos actuales con la plantilla por defecto (58 asientos, 2 pisos). ¿Continuar?')
    if (!ok) return
  }
  applyDefaultSeed()
}

// Si el editor abre con un layout vacío (recién creado), precargá la plantilla
// por defecto basada en la imagen de referencia, así el usuario solo mueve.
onMounted(() => {
  if (seats.value.length === 0 && elements.value.length === 0) {
    applyDefaultSeed()
  }
})

defineExpose({ fillSeats, loadDefaultSeed })

function save() {
  emit('save', {
    layout: { ...layout },
    seats: seats.value.map(s => ({ ...s })),
    elements: elements.value.map(e => ({ ...e })),
  })
}

const tools: Array<{ key: string; label: string; tool: Tool }> = [
  { key: 'select', label: 'Seleccionar', tool: { kind: 'select' } },
  { key: 'seat', label: 'Asiento', tool: { kind: 'seat' } },
  { key: 'aisle', label: 'Pasillo', tool: { kind: 'element', elementKind: 'aisle', defaultText: 'PASILLO' } },
  { key: 'stairs', label: 'Escalera', tool: { kind: 'element', elementKind: 'stairs' } },
  { key: 'tv', label: 'TV', tool: { kind: 'element', elementKind: 'tv' } },
  { key: 'wc', label: 'Baño', tool: { kind: 'element', elementKind: 'wc' } },
  { key: 'driver', label: 'Conductor', tool: { kind: 'element', elementKind: 'driver' } },
  { key: 'icon_yape', label: 'Yape', tool: { kind: 'element', elementKind: 'icon_yape', defaultText: 'YP' } },
  { key: 'seat_woman', label: 'Mujer', tool: { kind: 'element', elementKind: 'seat_woman' } },
  { key: 'seat_man', label: 'Hombre', tool: { kind: 'element', elementKind: 'seat_man' } },
  { key: 'bed', label: 'Cama', tool: { kind: 'element', elementKind: 'bed' } },
  { key: 'screen', label: 'Pantalla', tool: { kind: 'element', elementKind: 'screen' } },
  { key: 'label', label: 'Texto', tool: { kind: 'element', elementKind: 'label', defaultText: 'TEXTO' } },
  { key: 'erase', label: 'Borrar', tool: { kind: 'erase' } },
]

const activeToolKey = computed(() => {
  const t = tool.value
  if (t.kind === 'select') return 'select'
  if (t.kind === 'erase') return 'erase'
  if (t.kind === 'seat') return 'seat'
  return t.elementKind
})

function pickTool(t: Tool, key: string) {
  tool.value = t
  // efecto visual: scroll? no
  void key
}

// ── Cell helpers para render ──
function seatClass(seat: BusLayoutSeat): string {
  const out = ['seat-cell']
  if (seat.seat_type && seat.seat_type !== 'regular') out.push(`type-${seat.seat_type}`)
  return out.join(' ')
}
</script>

<template>
  <div class="editor">
    <!-- Header: metadata del layout -->
    <div class="header">
      <div class="header-row">
        <div class="form-group" style="flex: 2">
          <label class="form-label">Nombre</label>
          <input v-model="layout.name" type="text" class="form-input" placeholder="Ej. Scania K410 - 58 asientos 2 pisos" />
        </div>
        <div class="form-group">
          <label class="form-label">Marca</label>
          <input v-model="layout.brand" type="text" class="form-input" placeholder="Scania" />
        </div>
        <div class="form-group">
          <label class="form-label">Modelo</label>
          <input v-model="layout.model" type="text" class="form-input" placeholder="K410" />
        </div>
      </div>
      <div class="header-row">
        <div class="form-group">
          <label class="form-label">Tipo</label>
          <select v-model="layout.seat_type" class="form-input">
            <option value="regular">Regular</option>
            <option value="semi_cama">Semi cama</option>
            <option value="cama">Cama</option>
            <option value="suite">Suite</option>
            <option value="purple">Morado</option>
            <option value="yape">Yape</option>
            <option value="orange">Naranja</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">Pisos</label>
          <select :value="layout.floors" class="form-input" @change="(e) => setFloors(Number((e.target as HTMLSelectElement).value))">
            <option :value="1">1 piso</option>
            <option :value="2">2 pisos</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">Columnas</label>
          <div style="display: flex; gap: 4px; align-items: center">
            <button type="button" class="btn-mini" @click="removeCol">-</button>
            <input v-model.number="layout.layout_cols" type="number" min="1" max="20" class="form-input" style="width: 64px; text-align: center" />
            <button type="button" class="btn-mini" @click="addCol">+</button>
          </div>
        </div>
        <div class="form-group" style="flex: 2">
          <label class="form-label">Imagen de referencia (URL)</label>
          <input v-model="layout.preview_image_url" type="text" class="form-input" placeholder="https://..." />
        </div>
      </div>
    </div>

    <div class="workspace">
      <!-- Toolbar -->
      <aside class="toolbar">
        <div class="toolbar-title">Herramientas</div>
        <div class="tool-grid">
          <button
            v-for="t in tools" :key="t.key"
            type="button"
            class="tool"
            :class="{ active: activeToolKey === t.key }"
            @click="pickTool(t.tool, t.key)"
          >
            <span class="tool-icon" :class="`tool-icon-${t.key}`">
              <template v-if="t.key === 'seat'">🪑</template>
              <template v-else-if="t.key === 'aisle'">↔</template>
              <template v-else-if="t.key === 'stairs'">⇗</template>
              <template v-else-if="t.key === 'tv' || t.key === 'screen'">📺</template>
              <template v-else-if="t.key === 'wc'">🚻</template>
              <template v-else-if="t.key === 'driver'">🚍</template>
              <template v-else-if="t.key === 'icon_yape'">YP</template>
              <template v-else-if="t.key === 'seat_woman'">♀</template>
              <template v-else-if="t.key === 'seat_man'">♂</template>
              <template v-else-if="t.key === 'bed'">🛏</template>
              <template v-else-if="t.key === 'label'">T</template>
              <template v-else-if="t.key === 'erase'">✕</template>
              <template v-else>↖</template>
            </span>
            <span class="tool-label">{{ t.label }}</span>
          </button>
        </div>

        <div class="toolbar-section">
          <div class="toolbar-title">Acciones</div>
          <button type="button" class="btn-action" @click="renumberByPosition">Renumerar 1..N</button>
          <button type="button" class="btn-action btn-action-seed" @click="loadDefaultSeed">
            Plantilla 58 asientos
          </button>
        </div>

        <div v-if="layout.preview_image_url" class="preview-img">
          <div class="toolbar-title">Referencia</div>
          <img :src="layout.preview_image_url" alt="Referencia" />
        </div>
      </aside>

      <!-- Grid: ambos pisos en una sola imagen, lado a lado -->
      <div class="canvas">
        <div class="bus">
          <div class="bus-front">FRENTE</div>
          <div class="floors-row">
            <div
              v-for="f in floorList.slice().sort((a, b) => b - a)"
              :key="f"
              class="floor-block"
            >
              <div class="floor-label">{{ floorLabel(f) }}</div>
              <div class="grid">
                <div v-for="(row, ri) in grids[f]" :key="`f${f}-r${ri}`" class="grid-row">
                  <div
                    v-for="cell in row" :key="`f${f}-${cell.row}-${cell.col}`"
                    class="grid-cell"
                    :class="{
                      empty: !cell.seat && !cell.element,
                      has: !!cell.seat || !!cell.element,
                      highlight: tool.kind !== 'select' && !cell.seat && !cell.element,
                    }"
                    @click="handleCellClick(cell)"
                    @dragover="onDragOver"
                    @drop="onDrop($event, cell)"
                  >
                    <button
                      v-if="cell.seat"
                      type="button"
                      draggable="true"
                      :class="seatClass(cell.seat)"
                      @click.stop="openCellEditor(cell)"
                      @dragstart="onDragStart($event, cell)"
                      :title="`Asiento ${cell.seat.label}`"
                    >{{ cell.seat.label }}</button>
                    <div
                      v-else-if="cell.element"
                      draggable="true"
                      class="el" :class="`el-${cell.element.kind}`"
                      @click.stop="openCellEditor(cell)"
                      @dragstart="onDragStart($event, cell)"
                      :title="cell.element.kind"
                    >
                      <span v-if="cell.element.kind === 'aisle' || cell.element.kind === 'label'" class="el-text">{{ cell.element.text || cell.element.kind }}</span>
                      <span v-else-if="cell.element.kind === 'icon_yape'" class="el-yape">YP</span>
                      <template v-else>
                        <span class="el-icon">
                          <template v-if="cell.element.kind === 'stairs'">⇗</template>
                          <template v-else-if="cell.element.kind === 'tv' || cell.element.kind === 'screen'">📺</template>
                          <template v-else-if="cell.element.kind === 'wc'">🚻</template>
                          <template v-else-if="cell.element.kind === 'driver'">🚍</template>
                          <template v-else-if="cell.element.kind === 'seat_woman'">♀</template>
                          <template v-else-if="cell.element.kind === 'seat_man'">♂</template>
                          <template v-else-if="cell.element.kind === 'bed'">🛏</template>
                        </span>
                        <span v-if="cell.element.text" class="el-sub">{{ cell.element.text }}</span>
                      </template>
                    </div>
                    <span v-else class="cell-coord">{{ cell.row }},{{ cell.col }}</span>
                  </div>
                </div>
              </div>
              <button type="button" class="btn-row-add" @click="addRowToFloor(f)">+ fila</button>
            </div>
          </div>
          <div class="bus-back">ATRÁS</div>
        </div>

        <div class="hint">
          Hacé click en una celda vacía con la herramienta activa para colocar un asiento o elemento.
          Arrastrá un asiento para moverlo (intercambia si el destino está ocupado, incluso entre pisos).
          Click sobre un asiento para editar su etiqueta/tipo o eliminarlo.
        </div>
      </div>
    </div>

    <!-- Modal edición de celda -->
    <div v-if="editing" class="modal-overlay" @click.self="editing = null">
      <div class="modal-content">
        <h3 class="modal-title">
          {{ editing.seat ? `Asiento (fila ${editing.row}, col ${editing.col})` : `Elemento (fila ${editing.row}, col ${editing.col})` }}
        </h3>
        <div v-if="editing.seat" class="form-group">
          <label class="form-label">Etiqueta</label>
          <input v-model="editLabel" type="text" class="form-input" />
          <label class="form-label" style="margin-top: 0.7rem">Tipo de asiento</label>
          <select v-model="editSeatType" class="form-input">
            <option value="regular">Regular (blanco)</option>
            <option value="purple">Morado</option>
            <option value="yape">Yape (cian + YP)</option>
            <option value="orange">Naranja</option>
            <option value="semi_cama">Semi cama</option>
            <option value="cama">Cama</option>
            <option value="suite">Suite</option>
          </select>
        </div>
        <div v-if="editing.element" class="form-group">
          <label class="form-label">Texto (opcional)</label>
          <input v-model="editText" type="text" class="form-input" />
          <p class="form-hint">Tipo: <code>{{ editing.element.kind }}</code></p>
        </div>
        <div class="modal-actions">
          <button type="button" class="btn-ghost" @click="deleteEdited">Eliminar</button>
          <button type="button" class="btn-ghost" @click="editing = null">Cancelar</button>
          <button type="button" class="btn-primary" @click="applyEdit">Aplicar</button>
        </div>
      </div>
    </div>

    <!-- Footer fijo -->
    <div class="footer">
      <div class="counts">
        <span><strong>{{ seats.length }}</strong> asientos</span>
        <span><strong>{{ elements.length }}</strong> elementos</span>
      </div>
      <div class="actions">
        <button type="button" class="btn-ghost" @click="emit('cancel')">Cancelar</button>
        <button type="button" class="btn-primary" @click="save">Guardar plantilla</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.editor { display: flex; flex-direction: column; gap: 0.85rem; }

.header {
  display: flex; flex-direction: column; gap: 0.6rem;
  padding: 0.85rem 1rem;
  background: var(--slate-50);
  border: 1px solid var(--slate-200);
  border-radius: 12px;
}
.header-row { display: flex; gap: 0.85rem; flex-wrap: wrap; }
.form-group { display: flex; flex-direction: column; gap: 0.25rem; flex: 1; min-width: 120px; }
.form-label { font-size: 0.7rem; font-weight: 700; color: var(--slate-600); text-transform: uppercase; letter-spacing: 0.04em; }
.form-input {
  padding: 0.5rem 0.7rem;
  border: 2px solid var(--slate-300);
  border-radius: 8px;
  background: white;
  font-size: 0.85rem;
  font-family: inherit;
  color: var(--slate-900);
}
.form-input:focus { outline: none; border-color: var(--brand-400); box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15); }
.form-hint { font-size: 0.72rem; color: var(--slate-500); margin-top: 0.4rem; }

.workspace { display: grid; grid-template-columns: 220px 1fr; gap: 1rem; }
@media (max-width: 900px) { .workspace { grid-template-columns: 1fr; } }

.toolbar {
  display: flex; flex-direction: column; gap: 0.6rem;
  padding: 0.85rem;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: 12px;
}
.toolbar-title { font-size: 0.7rem; font-weight: 800; color: var(--slate-700); text-transform: uppercase; letter-spacing: 0.05em; }
.tool-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.4rem; }
.tool {
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  padding: 0.55rem 0.4rem;
  background: var(--slate-50);
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--slate-700);
  transition: all 0.15s;
}
.tool:hover { background: white; border-color: var(--slate-300); }
.tool.active { background: linear-gradient(135deg, var(--brand-400), var(--brand-500)); color: white; border-color: var(--brand-500); }
.tool-icon { font-size: 1.05rem; line-height: 1; }
.tool-label { font-size: 0.65rem; }
.toolbar-section { display: flex; flex-direction: column; gap: 0.4rem; padding-top: 0.6rem; border-top: 1px solid var(--slate-200); }
.btn-action {
  padding: 0.5rem; border: 1px solid var(--slate-300); background: white;
  border-radius: 6px; cursor: pointer; font-size: 0.78rem; font-weight: 600;
  color: var(--slate-700); font-family: inherit;
}
.btn-action:hover { background: var(--slate-50); }
.btn-action-seed {
  background: linear-gradient(135deg, #f0fdf4, #ecfccb);
  border-color: #86efac;
  color: #166534;
}
.btn-action-seed:hover { background: linear-gradient(135deg, #dcfce7, #d9f99d); }
.btn-mini {
  width: 26px; height: 30px; padding: 0;
  border: 1px solid var(--slate-300); background: white;
  border-radius: 6px; cursor: pointer; font-weight: 700;
}
.btn-mini:hover { background: var(--slate-50); }

.preview-img { display: flex; flex-direction: column; gap: 0.3rem; padding-top: 0.6rem; border-top: 1px solid var(--slate-200); }
.preview-img img { width: 100%; border-radius: 8px; border: 1px solid var(--slate-200); }

.canvas { display: flex; flex-direction: column; gap: 0.7rem; }

.floors-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1.2rem;
  justify-content: center;
  align-items: flex-start;
}
.floor-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.7rem;
  background: white;
  border: 1px solid var(--slate-200);
  border-radius: 14px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.05);
}
.floor-label {
  font-size: 0.7rem;
  font-weight: 800;
  color: var(--slate-700);
  letter-spacing: 0.12em;
  background: var(--slate-100);
  padding: 0.25rem 0.85rem;
  border-radius: 999px;
}
.btn-row-add {
  border: 1px dashed var(--slate-300);
  background: transparent;
  border-radius: 8px;
  padding: 0.3rem 0.7rem;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--slate-500);
  cursor: pointer;
  font-family: inherit;
}
.btn-row-add:hover { background: var(--slate-50); color: var(--slate-700); border-color: var(--slate-400); }

.bus {
  position: relative;
  background: linear-gradient(180deg, #fafafa 0%, #f1f5f9 100%);
  border: 2px solid var(--slate-300);
  border-radius: 22px;
  padding: 2.2rem 1rem 2rem;
  overflow-x: auto;
}
.bus-front, .bus-back {
  position: absolute; left: 50%; transform: translateX(-50%);
  font-size: 0.65rem; font-weight: 800; color: var(--slate-500); letter-spacing: 0.08em;
}
.bus-front { top: 6px; }
.bus-back { bottom: 6px; }

.grid { display: flex; flex-direction: column; gap: 6px; align-items: center; width: max-content; margin: 0 auto; }
.grid-row { display: flex; gap: 6px; }
.grid-cell {
  width: 48px; height: 48px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 8px;
  position: relative;
}
.grid-cell.empty { border: 1px dashed var(--slate-200); }
.grid-cell.empty:hover { background: var(--brand-50); border-color: var(--brand-300); cursor: pointer; }
.grid-cell.highlight { border-color: var(--brand-400); }
.cell-coord { font-size: 0.55rem; color: var(--slate-300); font-weight: 500; }

.seat-cell {
  width: 100%; height: 100%;
  background: white;
  border: 2px solid var(--slate-300);
  border-radius: 8px;
  font-weight: 700; font-size: 0.78rem; color: var(--slate-800);
  cursor: grab; font-family: inherit; padding: 0;
}
.seat-cell:active { cursor: grabbing; }
.seat-cell:hover { border-color: var(--brand-400); background: var(--brand-50); }
.seat-cell.type-semi_cama { border-color: #a78bfa; }
.seat-cell.type-cama { border-color: #f472b6; }
.seat-cell.type-suite { border-color: #facc15; }

/* Colores de la plantilla WhatsApp */
.seat-cell.type-purple {
  background: #7c3aed;
  color: white;
  border-color: #6d28d9;
}
.seat-cell.type-purple:hover { background: #6d28d9; border-color: #5b21b6; }
.seat-cell.type-yape {
  background: #06b6d4;
  color: white;
  border-color: #0891b2;
  position: relative;
}
.seat-cell.type-yape:hover { background: #0891b2; border-color: #0e7490; }
.seat-cell.type-yape::before {
  content: 'YP';
  position: absolute;
  top: -3px; left: -3px;
  background: #06b6d4;
  color: white;
  font-size: 0.5rem;
  font-weight: 800;
  padding: 1px 3px;
  border-radius: 3px;
  border: 1px solid white;
  letter-spacing: 0.04em;
  line-height: 1;
}
.seat-cell.type-orange {
  background: #fb923c;
  color: white;
  border-color: #ea580c;
}
.seat-cell.type-orange:hover { background: #f97316; border-color: #c2410c; }

.el {
  width: 100%; height: 100%;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  border-radius: 6px; cursor: grab;
  font-size: 0.62rem; font-weight: 700; text-align: center; line-height: 1;
}
.el:active { cursor: grabbing; }
.el-icon { font-size: 1rem; }
.el-text { font-size: 0.6rem; padding: 0 2px; }
.el-sub { font-size: 0.52rem; color: var(--slate-500); margin-top: 1px; font-weight: 500; }
.el-aisle { background: #ecfeff; color: #0e7490; border: 1px dashed #67e8f9; }
.el-stairs { background: #f5f3ff; color: #6d28d9; border: 1px solid #c4b5fd; }
.el-tv, .el-screen { background: #1e293b; color: white; border: 1px solid #334155; }
.el-wc { background: #ecfdf5; color: #047857; border: 1px solid #86efac; }
.el-driver { background: #fffbeb; color: #b45309; border: 1px solid #fcd34d; }
.el-seat_woman { background: #fdf2f8; color: #be185d; border: 1px solid #f9a8d4; }
.el-seat_man { background: #eff6ff; color: #1d4ed8; border: 1px solid #93c5fd; }
.el-bed { background: #fef3c7; color: #92400e; border: 1px solid #fcd34d; }
.el-label { background: transparent; color: var(--slate-700); font-weight: 800; }
.el-yape { background: #6b21a8; color: white; font-weight: 800; font-size: 0.7rem; width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; border-radius: 6px; }

.hint { font-size: 0.75rem; color: var(--slate-500); padding: 0.5rem 0; }

.footer {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--slate-200);
  background: white;
  position: sticky; bottom: 0;
}
.counts { display: flex; gap: 1rem; font-size: 0.85rem; color: var(--slate-600); }
.actions { display: flex; gap: 0.5rem; }
.btn-ghost {
  padding: 0.55rem 1rem;
  background: transparent; color: var(--slate-700);
  border: 2px solid var(--slate-300); border-radius: 8px; cursor: pointer; font-weight: 600;
}
.btn-ghost:hover { background: var(--slate-50); }
.btn-primary {
  padding: 0.55rem 1.2rem;
  background: linear-gradient(135deg, var(--brand-400), var(--brand-500));
  color: white; border: none; border-radius: 8px; cursor: pointer; font-weight: 700;
}
.btn-primary:hover { box-shadow: 0 6px 20px rgba(59,130,246,0.35); transform: translateY(-1px); }

/* Modal edit */
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 1100; padding: 1rem;
}
.modal-content {
  background: white; border-radius: 16px; padding: 1.5rem; width: 100%; max-width: 380px;
  box-shadow: 0 20px 50px rgba(0,0,0,0.25);
}
.modal-title { font-size: 1.05rem; font-weight: 800; margin-bottom: 1rem; color: var(--slate-900); }
.modal-actions { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1rem; }
</style>
