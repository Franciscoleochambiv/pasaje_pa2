import type { BusLayoutSeat, BusLayoutElement } from '../api/client'

export interface DefaultBusLayoutSeed {
  floors: number
  layout_cols: number
  seats: BusLayoutSeat[]
  elements: BusLayoutElement[]
}

// Plantilla "estándar 58 asientos doble piso" reproduciendo el bus de
// referencia (foto WhatsApp). Floor 2 = cabina superior (46 asientos);
// Floor 1 = cabina inferior (12 asientos). Las dos se renderizan
// lado a lado en una sola imagen para que el usuario solo arrastre/borre.
//
// Colores tomados de la foto:
//   - purple: asientos morados con figura (preferenciales)
//   - yape:   asientos cian con badge "YP"
//   - orange: asientos naranja del sector inferior izquierdo
//   - regular: blanco
const PURPLE_SEATS = new Set([
  '16', '25', '26', '29', '30', '33',
  '1', '2', '4', '5', '6', '7', '8', '9', '10', '11', '12',
])
const YAPE_SEATS = new Set(['13', '14', '15', '28', '3'])
const ORANGE_SEATS = new Set(['17', '18', '19', '20', '21', '22', '23', '24'])

function colorFor(label: string): string {
  if (PURPLE_SEATS.has(label)) return 'purple'
  if (YAPE_SEATS.has(label)) return 'yape'
  if (ORANGE_SEATS.has(label)) return 'orange'
  return 'regular'
}

export function getDefaultBusLayoutSeed(): DefaultBusLayoutSeed {
  const seats: BusLayoutSeat[] = []
  const elements: BusLayoutElement[] = []
  let position = 0

  const pushSeat = (floor: number, row: number, col: number, label: string) => {
    seats.push({
      label,
      position: ++position,
      floor,
      row_num: row,
      col_num: col,
      seat_type: colorFor(label),
    })
  }

  // ── Floor 2 (segundo piso / parte superior) ──────────────────────
  // Fila 1 (atrás): col 1: 16, col 2: escalera, cols 3-11: 26..58
  pushSeat(2, 1, 1, '16')
  pushSeat(2, 1, 3, '26'); pushSeat(2, 1, 4, '30')
  pushSeat(2, 1, 5, '34'); pushSeat(2, 1, 6, '38')
  pushSeat(2, 1, 7, '42'); pushSeat(2, 1, 8, '46')
  pushSeat(2, 1, 9, '50'); pushSeat(2, 1, 10, '54')
  pushSeat(2, 1, 11, '58')

  // Fila 2
  pushSeat(2, 2, 1, '15')
  pushSeat(2, 2, 3, '25'); pushSeat(2, 2, 4, '29')
  pushSeat(2, 2, 5, '33'); pushSeat(2, 2, 6, '37')
  pushSeat(2, 2, 7, '41'); pushSeat(2, 2, 8, '45')
  pushSeat(2, 2, 9, '49'); pushSeat(2, 2, 10, '53')
  pushSeat(2, 2, 11, '57')

  // Fila 4 (después del pasillo intermedio en fila 3)
  pushSeat(2, 4, 1, '14');  pushSeat(2, 4, 2, '18')
  pushSeat(2, 4, 3, '20');  pushSeat(2, 4, 4, '22')
  pushSeat(2, 4, 5, '24');  pushSeat(2, 4, 6, '28')
  pushSeat(2, 4, 7, '32');  pushSeat(2, 4, 8, '36')
  pushSeat(2, 4, 9, '40');  pushSeat(2, 4, 10, '44')
  pushSeat(2, 4, 11, '48'); pushSeat(2, 4, 12, '52')
  pushSeat(2, 4, 13, '56')

  // Fila 5
  pushSeat(2, 5, 1, '13');  pushSeat(2, 5, 2, '17')
  pushSeat(2, 5, 3, '19');  pushSeat(2, 5, 4, '21')
  pushSeat(2, 5, 5, '23');  pushSeat(2, 5, 6, '27')
  pushSeat(2, 5, 7, '31');  pushSeat(2, 5, 8, '35')
  pushSeat(2, 5, 9, '39');  pushSeat(2, 5, 10, '43')
  pushSeat(2, 5, 11, '47'); pushSeat(2, 5, 12, '51')
  pushSeat(2, 5, 13, '55')

  // Elementos floor 2
  elements.push({ floor: 2, row_num: 1, col_num: 2, kind: 'stairs',  text: '' })
  elements.push({ floor: 2, row_num: 2, col_num: 2, kind: 'stairs',  text: '' })
  elements.push({ floor: 2, row_num: 3, col_num: 3, kind: 'aisle',   text: 'PASILLO' })
  elements.push({ floor: 2, row_num: 3, col_num: 6, kind: 'aisle',   text: 'PASILLO' })
  elements.push({ floor: 2, row_num: 3, col_num: 7, kind: 'driver',  text: '' })
  elements.push({ floor: 2, row_num: 3, col_num: 9, kind: 'tv',      text: '' })

  // ── Floor 1 (primer piso / cabina inferior) ──────────────────────
  // 5 cols: col 1 = TV, cols 2-5 = 12 asientos (3 filas × 4 cols)
  pushSeat(1, 1, 2, '3');  pushSeat(1, 1, 3, '6');  pushSeat(1, 1, 4, '9');  pushSeat(1, 1, 5, '12')
  pushSeat(1, 2, 2, '2');  pushSeat(1, 2, 3, '5');  pushSeat(1, 2, 4, '8');  pushSeat(1, 2, 5, '11')
  pushSeat(1, 3, 2, '1');  pushSeat(1, 3, 3, '4');  pushSeat(1, 3, 4, '7');  pushSeat(1, 3, 5, '10')

  elements.push({ floor: 1, row_num: 1, col_num: 1, kind: 'tv', text: '' })

  return { floors: 2, layout_cols: 13, seats, elements }
}
