// En desarrollo Vite hace proxy de /api y /health al backend (vite.config.ts). Usar '' = misma origen.
const apiBase = import.meta.env.VITE_API_URL ?? ''

function authHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  if (token) return { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }
  return { 'Content-Type': 'application/json' }
}

async function extractErr(res: Response): Promise<string> {
  const text = await res.text()
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/json')) {
    try {
      const j = JSON.parse(text)
      const msg = j?.error || j?.message
      if (typeof msg === 'string' && msg.trim()) return msg
    } catch { /* fallthrough */ }
  }
  if (/^\s*<(!doctype|html)/i.test(text)) {
    if (res.status === 502 || res.status === 503 || res.status === 504) {
      return `Servicio temporalmente no disponible (HTTP ${res.status}). Intente nuevamente en unos minutos.`
    }
    return `Error del servidor (HTTP ${res.status}).`
  }
  const snippet = text.trim().slice(0, 240)
  return snippet || `Error HTTP ${res.status}`
}

export interface Route {
  id: number
  name: string
  code: string
  active: boolean
  price_per_seat: number
}

export interface TripInstance {
  id: number
  trip_template_id: number
  route_id: number
  route_name: string
  price_per_seat: number
  departure_at: string
  status: string
}

export interface SeatInfo {
  id: number
  vehicle_seat_id: number
  label: string
  position: number
  floor: number
  row_num: number
  col_num: number
  seat_type: string
  status: string
  hold_expires_at?: string
}

export interface VehicleLayoutElement {
  id: number
  vehicle_id: number
  floor: number
  row_num: number
  col_num: number
  kind: string
  text: string
}

export interface TripSeatResponse {
  vehicle_floors: number
  vehicle_layout_cols: number
  vehicle_seat_type: string
  vehicle_name: string
  route_name: string
  price_per_seat: number
  seats: SeatInfo[]
  layout_elements: VehicleLayoutElement[]
}

export interface Vehicle {
  id: number
  plate: string
  name: string
  capacity: number
  floors: number
  layout_cols: number
  seat_type: string
  layout_id: number | null
  active: boolean
}

export interface BusLayout {
  id: number
  name: string
  brand: string
  model: string
  description: string
  floors: number
  layout_cols: number
  seat_type: string
  preview_image_url: string
  active: boolean
  created_at: string
  updated_at: string
}

export interface BusLayoutSeat {
  id?: number
  layout_id?: number
  label: string
  position: number
  floor: number
  row_num: number
  col_num: number
  seat_type: string
}

export interface BusLayoutElement {
  id?: number
  layout_id?: number
  floor: number
  row_num: number
  col_num: number
  kind: string
  text: string
}

export interface BusLayoutFull {
  layout: BusLayout
  seats: BusLayoutSeat[]
  elements: BusLayoutElement[]
}

export interface VehicleSeat {
  id: number
  vehicle_id: number
  label: string
  position: number
  floor: number
  row_num: number
  col_num: number
  seat_type: string
}

export interface TripTemplate {
  id: number
  route_id: number
  vehicle_id: number
  name: string
  departure_time: string
  route_name?: string
  vehicle_name?: string
}

export interface AdminStats {
  total_routes: number
  total_vehicles: number
  upcoming_trips: number
  reservations_today: number
  seats_available: number
  seats_sold: number
  seats_held: number
  seats_blocked: number
  total_sales_amount: number
  sales_today: number
  sales_week: number
  sales_month: number
  confirmed_today: number
  pending_today: number
  expired_today: number
  cancelled_today: number
}

export interface ReservationItem {
  id: number
  seat_label: string
  seat_position?: number
  ticket_code?: string
  ticket_status?: string
}

export interface Reservation {
  id?: number
  code: string
  status: string
  expires_at: string
  created_at?: string
  passenger_name: string
  passenger_doc_type?: string
  passenger_doc_number?: string
  passenger_email: string
  passenger_phone?: string
  document_type?: string
  payment_method?: string
  payment_status?: string
  billing_sale_id?: number
  total_amount?: number
  route_name?: string
  origin_stop?: string
  dest_stop?: string
  departure_at?: string
  trip_instance_id?: number
  items: ReservationItem[]
  contact_doc_number?: string
  contact_phone?: string
}

export interface PendingReservationView {
  id: number
  code: string
  status: string
  expires_at: string
  created_at: string
  contact_name: string
  contact_doc_number: string
  contact_phone: string
  seat_labels: string[]
}

export interface ConfirmResponse {
  ticket_codes: string[]
}

export interface User {
  id: number
  email: string
  name: string
  role: string
}

export interface UserInfo {
  id: number
  email: string
  name: string
  role: string
  active: boolean
  created_at: string
}

// ── Auth endpoints ──

export async function login(email: string, password: string): Promise<{ token: string; user: User }> {
  const res = await fetch(`${apiBase}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getMe(): Promise<User> {
  const res = await fetch(`${apiBase}/api/auth/me`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Public endpoints ──

export async function getRoutes(): Promise<Route[]> {
  const res = await fetch(`${apiBase}/api/routes`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getTripsByRoute(routeId: number, from?: string): Promise<TripInstance[]> {
  const url = new URL(`${apiBase}/api/routes/${routeId}/trips`, window.location.origin)
  if (from) url.searchParams.set('from', from)
  const res = await fetch(url.toString())
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getTripSeats(tripId: number): Promise<TripSeatResponse> {
  const res = await fetch(`${apiBase}/api/trips/${tripId}/seats`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Reservation endpoints ──

export async function createReservation(data: {
  trip_instance_id: number
  seats: number[]
  passenger_name: string
  passenger_doc_type: string
  passenger_doc_number: string
  passenger_email: string
  passenger_phone: string
}): Promise<{ id: number; code: string; expires_at: string }> {
  const res = await fetch(`${apiBase}/api/reservations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function confirmReservation(
  code: string,
  data: {
    payment_method: string
    payment_reference: string
    passenger_name?: string
    passenger_doc_type?: string
    passenger_doc_number?: string
    passenger_email?: string
    passenger_phone?: string
    passenger_address?: string
    document_type?: string
    route_name?: string
    seat_labels?: string[]
    price_per_seat?: number
  }
): Promise<ConfirmResponse> {
  const res = await fetch(`${apiBase}/api/reservations/${code}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getReservation(code: string): Promise<Reservation> {
  const res = await fetch(`${apiBase}/api/reservations/${code}`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createAdminReservation(data: {
  trip_instance_id: number
  seats: number[]
  contact_name: string
  contact_doc_number: string
  contact_phone: string
  hold_hours: number
}): Promise<{ reservation_id: number; code: string; expires_at: string }> {
  const res = await fetch(`${apiBase}/api/admin/reservations`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getAdminReservations(tripInstanceId: number): Promise<PendingReservationView[]> {
  const res = await fetch(`${apiBase}/api/admin/reservations?trip_instance_id=${tripInstanceId}`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Google Auth (customer) ──

export async function loginWithGoogle(credential: string): Promise<{ token: string; user: any }> {
  const res = await fetch(`${apiBase}/api/auth/google`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ credential }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Admin endpoints ──

export async function getAdminStats(): Promise<AdminStats> {
  const res = await fetch(`${apiBase}/api/admin/stats`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createRoute(data: { name: string; code: string; price_per_seat?: number }): Promise<Route> {
  const res = await fetch(`${apiBase}/api/admin/routes`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateRoute(id: number, data: { name: string; code: string; price_per_seat?: number }): Promise<Route> {
  const res = await fetch(`${apiBase}/api/admin/routes/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function deleteRoute(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/routes/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function getVehicles(): Promise<Vehicle[]> {
  const res = await fetch(`${apiBase}/api/admin/vehicles`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateVehicle(id: number, data: { plate: string; name: string; capacity: number; active?: boolean; layout_id?: number }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/vehicles/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteVehicle(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/vehicles/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function createVehicle(data: {
  plate: string
  name: string
  capacity: number
  seat_count?: number
  layout_id?: number
}): Promise<Vehicle> {
  const res = await fetch(`${apiBase}/api/admin/vehicles`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Bus Layouts (plantillas reutilizables) ──

export async function getBusLayouts(): Promise<BusLayout[]> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getBusLayoutFull(id: number): Promise<BusLayoutFull> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts/${id}/full`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createBusLayout(data: {
  name: string
  brand?: string
  model?: string
  description?: string
  floors: number
  layout_cols: number
  seat_type: string
  preview_image_url?: string
}): Promise<{ id: number }> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateBusLayout(id: number, data: Partial<BusLayout>): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteBusLayout(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function saveBusLayoutFull(id: number, data: BusLayoutFull): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/bus-layouts/${id}/full`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function saveVehicleAsLayout(vehicleId: number, data: {
  name: string
  brand?: string
  model?: string
  description?: string
  preview_image_url?: string
}): Promise<{ id: number }> {
  const res = await fetch(`${apiBase}/api/admin/vehicles/${vehicleId}/save-as-layout`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getVehicleSeats(vehicleId: number): Promise<VehicleSeat[]> {
  const res = await fetch(`${apiBase}/api/admin/vehicles/${vehicleId}/seats`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getTripTemplates(): Promise<TripTemplate[]> {
  const res = await fetch(`${apiBase}/api/admin/trip-templates`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateTripTemplate(id: number, data: { route_id: number; vehicle_id: number; name: string; departure_time: string }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/trip-templates/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteTripTemplate(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/trip-templates/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function updateTripInstance(id: number, data: { status?: string; departure_at?: string }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/trip-instances/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteTripInstance(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/trip-instances/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function createTripTemplate(data: {
  route_id: number
  vehicle_id: number
  name: string
  departure_time: string
}): Promise<TripTemplate> {
  const res = await fetch(`${apiBase}/api/admin/trip-templates`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getTripInstances(from?: string): Promise<TripInstance[]> {
  const url = new URL(`${apiBase}/api/admin/trip-instances`, window.location.origin)
  if (from) url.searchParams.set('from', from)
  const res = await fetch(url.toString(), {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createTripInstance(data: {
  trip_template_id: number
  departure_at: string
}): Promise<TripInstance> {
  const res = await fetch(`${apiBase}/api/admin/trip-instances`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── User CRUD (admin) ──

export async function getUsers(): Promise<UserInfo[]> {
  const res = await fetch(`${apiBase}/api/admin/users`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createUser(data: { email: string; name: string; password: string; role: string }): Promise<UserInfo> {
  const res = await fetch(`${apiBase}/api/admin/users`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateUser(id: number, data: { email: string; name: string; role: string }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/users/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function updateUserPassword(id: number, password: string): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/users/${id}/password`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify({ password }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteUser(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/users/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

// ── Billing endpoints ──

export interface DNIResult {
  nombres: string
  apellido_paterno: string
  apellido_materno: string
  nombre_completo: string
  dni: string
}

export interface RUCResult {
  ruc: string
  razon_social: string
  nombre_o_razon_social: string
  estado: string
  condicion: string
  direccion: string
}

export interface BillingSaleResponse {
  billing_sale_id: number
  status: string
  message: string
  warnings?: string[]
}

export async function lookupDNI(dni: string): Promise<DNIResult> {
  const res = await fetch(`${apiBase}/api/billing/dni/${dni}`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function lookupRUC(ruc: string): Promise<RUCResult> {
  const res = await fetch(`${apiBase}/api/billing/ruc/${ruc}`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createBillingSale(data: {
  document_type: 'boleta' | 'factura' | 'pedido'
  trip_instance_id: number
  reservation_code: string
  passenger_name: string
  passenger_doc_type: string
  passenger_doc_number: string
  passenger_address?: string
  price_per_seat: number
  route_name: string
  seat_labels: string[]
}): Promise<BillingSaleResponse> {
  const res = await fetch(`${apiBase}/api/billing/sale`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export function getBillingPDFUrl(ventaId: number, format: string): string {
  return `${apiBase}/api/billing/pdf/${ventaId}?format=${format}`
}

export async function sendBillingEmail(data: {
  email: string
  passenger_name: string
  reservation_code: string
  ticket_codes: string[]
  total_amount: number
  seat_labels: string[]
  route_name: string
  billing_sale_id: number
  payment_ref: string
}): Promise<{ status: string; message: string }> {
  const res = await fetch(`${apiBase}/api/billing/send-email`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Yape Direct endpoints ──

export interface YapeConfig {
  yape_number: string
  whatsapp_phone: string
}

export interface YapeDirectResponse {
  reservation_code: string
  payment_status: string
  hold_expires_at: string
  yape_number: string
  whatsapp_phone: string
}

export interface YapePaymentStatusResponse {
  payment_status: string
  reservation_status: string
  rejection_reason?: string
}

export interface PendingVoucher {
  payment_id: number
  reservation_code: string
  reservation_id: number
  amount_cents: number
  passenger_name: string
  passenger_doc_type: string
  passenger_doc_number: string
  passenger_email: string
  passenger_phone: string
  document_type: string
  route_name: string
  seat_labels: string[]
  trip_instance_id: number | null
  price_per_seat: number
  passenger_address: string
  created_at: string
  hold_expires_at: string
}

export async function getYapeConfig(): Promise<YapeConfig> {
  const res = await fetch(`${apiBase}/api/payment/yape-config`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function submitYapeDirectPayment(data: FormData): Promise<YapeDirectResponse> {
  const res = await fetch(`${apiBase}/api/payment/yape-direct`, {
    method: 'POST',
    body: data,
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getYapePaymentStatus(code: string): Promise<YapePaymentStatusResponse> {
  const res = await fetch(`${apiBase}/api/payment/yape-direct/${code}/status`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getPendingVouchers(): Promise<PendingVoucher[]> {
  const res = await fetch(`${apiBase}/api/admin/vouchers/pending`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function approveVoucher(paymentId: number): Promise<{ status: string; reservation_code: string; ticket_codes: string[]; billing_sale_id?: number; billing_error?: string }> {
  const res = await fetch(`${apiBase}/api/admin/vouchers/${paymentId}/approve`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function retryComprobante(paymentId: number): Promise<{ status: string; billing_sale_id?: number | null; billing_attempts?: number; billing_error?: string | null; comprobante_email_sent_at?: string | null }> {
  const res = await fetch(`${apiBase}/api/admin/payments/${paymentId}/retry-comprobante`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export interface SeatHolder {
  seat_label: string
  seat_status: string
  reservation_code: string
  reservation_status: string
  ticket_code?: string
  ticket_status?: string
  passenger_name: string
  passenger_doc_type: string
  passenger_doc_number: string
  passenger_email: string
  passenger_phone: string
  payment_method: string
  payment_status: string
  billing_sale_id?: number | null
  created_at?: string | null
}

export async function voidReservation(code: string, reason: string): Promise<{ status: string; reservation_code: string; trip_instance_ids: number[] }> {
  const res = await fetch(`${apiBase}/api/admin/reservations/${code}/void`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ reason }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getSeatHolder(tripId: number, seatInventoryId: number): Promise<SeatHolder> {
  const res = await fetch(`${apiBase}/api/admin/trips/${tripId}/seats/${seatInventoryId}/holder`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function rejectVoucher(paymentId: number, reason: string): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/vouchers/${paymentId}/reject`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ reason }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export function getVoucherImageUrl(paymentId: number): string {
  const token = localStorage.getItem('auth_token')
  return `${apiBase}/api/admin/vouchers/${paymentId}/image${token ? `?token=${token}` : ''}`
}

// ── Stops & Segments (admin) ──

export interface Stop {
  id: number
  route_id: number
  name: string
  code: string
  position: number
}

export interface RouteSegment {
  id: number
  route_id: number
  origin_stop_id: number
  dest_stop_id: number
  price: number
  origin_stop_name?: string
  dest_stop_name?: string
}

export async function getStops(routeId: number): Promise<Stop[]> {
  const res = await fetch(`${apiBase}/api/routes/${routeId}/stops`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createStop(routeId: number, data: { name: string; code: string; position: number }): Promise<{ id: number }> {
  const res = await fetch(`${apiBase}/api/admin/routes/${routeId}/stops`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateStop(id: number, data: { name: string; code: string; position: number; route_id?: number }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/stops/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function deleteStop(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/stops/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function reorderStops(routeId: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/routes/${routeId}/stops/reorder`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function getSegments(routeId: number): Promise<RouteSegment[]> {
  const res = await fetch(`${apiBase}/api/admin/routes/${routeId}/segments`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function upsertSegment(routeId: number, data: { origin_stop_id: number; dest_stop_id: number; price: number }): Promise<{ id: number }> {
  const res = await fetch(`${apiBase}/api/admin/routes/${routeId}/segments`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function generateSegments(routeId: number): Promise<{ created: number }> {
  const res = await fetch(`${apiBase}/api/admin/routes/${routeId}/segments/generate`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function deleteSegment(id: number): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/segments/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

// ── Parcels (encomiendas) ──

export interface Parcel {
  id: number
  code: string
  trip_instance_id: number
  origin_stop_id: number
  dest_stop_id: number
  origin_stop_name: string
  dest_stop_name: string
  sender_name: string
  sender_doc_type: string
  sender_doc_number: string
  sender_phone: string
  receiver_name: string
  receiver_doc_type: string
  receiver_doc_number: string
  receiver_phone: string
  package_count: number
  weight_kg: number
  description: string
  amount_cents: number
  currency: string
  payment_mode: string
  payment_status: string
  payment_method: string
  billing_doc_type: string
  billing_email: string
  billing_ruc: string
  billing_razon_social: string
  billing_address: string
  billing_sale_id: number | null
  status: string
  registered_by: number | null
  route_name: string
  departure_at: string
  created_at: string
  updated_at: string
}

export interface ParcelTracking {
  id: number
  parcel_id: number
  status: string
  location: string
  notes: string
  user_id: number | null
  user_name: string
  created_at: string
}

export interface ParcelPublicInfo {
  code: string
  status: string
  origin_stop: string
  dest_stop: string
  receiver_name: string
  package_count: number
  weight_kg: number
  description: string
  payment_mode: string
  payment_status: string
  created_at: string
  tracking: ParcelTracking[]
}

// Público
export async function trackParcel(code: string): Promise<ParcelPublicInfo> {
  const res = await fetch(`${apiBase}/api/parcels/track/${code}`)
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// Admin
export async function getParcels(params?: { status?: string; trip_instance_id?: number; date_from?: string; date_to?: string; limit?: number; offset?: number }): Promise<{ data: Parcel[]; total: number }> {
  const url = new URL(`${apiBase}/api/admin/parcels`, window.location.origin)
  if (params?.status) url.searchParams.set('status', params.status)
  if (params?.trip_instance_id) url.searchParams.set('trip_instance_id', String(params.trip_instance_id))
  if (params?.date_from) url.searchParams.set('date_from', params.date_from)
  if (params?.date_to) url.searchParams.set('date_to', params.date_to)
  if (params?.limit) url.searchParams.set('limit', String(params.limit))
  if (params?.offset) url.searchParams.set('offset', String(params.offset))
  const res = await fetch(url.toString(), { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function getParcel(id: number): Promise<{ parcel: Parcel; tracking: ParcelTracking[] }> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function createParcel(data: {
  trip_instance_id: number
  origin_stop_id: number
  dest_stop_id: number
  sender_name: string
  sender_doc_type: string
  sender_doc_number: string
  sender_phone?: string
  receiver_name: string
  receiver_doc_type: string
  receiver_doc_number: string
  receiver_phone: string
  package_count: number
  weight_kg?: number
  description?: string
  amount_cents: number
  payment_mode: string
  payment_method?: string
  billing_doc_type: string
  billing_email: string
  billing_ruc?: string
  billing_razon_social?: string
  billing_address?: string
}): Promise<{ parcel: Parcel; pdf_url?: string; whatsapp_url?: string; billing_error?: string }> {
  const res = await fetch(`${apiBase}/api/admin/parcels`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function updateParcel(id: number, data: Record<string, any>): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function updateParcelStatus(id: number, data: { status: string; location?: string; notes?: string }): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}/status`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function payParcel(id: number, paymentMethod: string): Promise<{ status: string; parcel: Parcel; pdf_url?: string; whatsapp_url?: string; billing_error?: string }> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}/pay`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ payment_method: paymentMethod }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function retryParcelBilling(id: number): Promise<{ parcel: Parcel; pdf_url?: string; whatsapp_url?: string; billing_error?: string }> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}/retry-billing`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

export async function cancelParcel(id: number, notes?: string): Promise<void> {
  const res = await fetch(`${apiBase}/api/admin/parcels/${id}/cancel`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ notes }),
  })
  if (!res.ok) throw new Error(await extractErr(res))
}

export async function getParcelsByTrip(tripId: number): Promise<Parcel[]> {
  const res = await fetch(`${apiBase}/api/admin/parcels/by-trip/${tripId}`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Settings ──

export async function getAdminSettings(): Promise<Record<string, string>> {
  const res = await fetch(`${apiBase}/api/admin/settings`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  const list: { key: string; value: string }[] = await res.json()
  const map: Record<string, string> = {}
  for (const s of list) map[s.key] = s.value
  return map
}

// ── Manifest (hoja de ruta) ──

export interface TripManifest {
  trip: { id: number; route_name: string; vehicle_name: string; price_per_seat: number }
  seats: SeatInfo[]
  parcels: Parcel[]
  totals: { passengers: number; parcels: number; revenue_passengers: number; revenue_parcels: number }
}

export async function getTripManifest(tripId: number): Promise<TripManifest> {
  const res = await fetch(`${apiBase}/api/admin/trips/${tripId}/manifest`, { headers: authHeaders() })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}

// ── Payment / Culqi endpoints ──

export interface PaymentChargeResponse {
  charge_id: string
  reservation_code: string
  ticket_codes: string[]
  billing_sale_id: number
  billing_error?: string
}

export async function processPayment(data: {
  token_id: string
  amount: number
  email: string
  description: string
  trip_instance_id: number
  reservation_code?: string
  seats: number[]
  passenger_name: string
  passenger_doc_type: string
  passenger_doc_number: string
  passenger_address?: string
  document_type: 'boleta' | 'factura'
  route_name: string
  seat_labels: string[]
  price_per_seat: number
  passenger_phone?: string
}): Promise<PaymentChargeResponse> {
  const res = await fetch(`${apiBase}/api/payment/charge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await extractErr(res))
  return res.json()
}
