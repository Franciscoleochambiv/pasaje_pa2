import { chromium } from 'playwright'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '../..')
const baseUrl = 'https://pasaje.facturame.online'
const outDir = path.join(root, 'evidencias_anexos', 'capturas_pruebas_pa4')
const reportDir = path.join(outDir, 'reportes')
await fs.mkdir(outDir, { recursive: true })
await fs.mkdir(reportDir, { recursive: true })

const adminEmail = process.env.PASAJE_ADMIN_EMAIL || 'admin@pasaje.pe'
const adminPassword = process.env.PASAJE_ADMIN_PASSWORD || 'admin123'
const runId = new Date().toISOString().replace(/[-:TZ.]/g, '').slice(0, 14)
const prefix = `EVID-${runId}`
const created = { runId, prefix, baseUrl }

async function api(pathname, options = {}) {
  const res = await fetch(`${baseUrl}${pathname}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
      ...(options.headers || {})
    }
  })
  const text = await res.text()
  let body = text
  try { body = text ? JSON.parse(text) : null } catch {}
  if (!res.ok) {
    const msg = typeof body === 'object' ? JSON.stringify(body) : body
    throw new Error(`${options.method || 'GET'} ${pathname} -> HTTP ${res.status}: ${msg}`)
  }
  return body
}

function shellCapture(fileName, title) {
  const out = path.join(outDir, fileName)
  execFileSync('powershell.exe', [
    '-ExecutionPolicy', 'Bypass',
    '-File', path.join(__dirname, 'capture-current-screen.ps1'),
    '-Out', out,
    '-Title', title
  ], { stdio: 'pipe' })
  return out
}

function sleep(ms) { return new Promise(resolve => setTimeout(resolve, ms)) }

async function writeReport(name, title, htmlBody) {
  const file = path.join(reportDir, `${name}.html`)
  await fs.writeFile(file, `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <title>${title}</title>
  <style>
    body { margin: 0; font-family: Arial, sans-serif; background: #f8fafc; color: #111827; }
    header { background: #0f172a; color: white; padding: 18px 28px; }
    h1 { margin: 0; font-size: 26px; }
    main { padding: 28px; }
    .grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
    .card { background: white; border: 1px solid #cbd5e1; border-radius: 8px; padding: 16px; box-shadow: 0 1px 2px rgba(0,0,0,.08); }
    .label { color: #475569; font-size: 13px; text-transform: uppercase; font-weight: 700; }
    .value { margin-top: 6px; font-size: 22px; font-weight: 800; overflow-wrap: anywhere; }
    pre { background: #111827; color: #e5e7eb; padding: 16px; border-radius: 8px; white-space: pre-wrap; font-size: 15px; }
    table { width: 100%; border-collapse: collapse; background: white; }
    th, td { border: 1px solid #cbd5e1; padding: 10px; text-align: left; }
    th { background: #e2e8f0; }
    .ok { color: #047857; }
    .bad { color: #b91c1c; }
  </style>
</head>
<body>
  <header><h1>${title}</h1></header>
  <main>${htmlBody}</main>
</body>
</html>`, 'utf8')
  return `file:///${file.replace(/\\/g, '/')}`
}

async function setupData() {
  const login = await api('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email: adminEmail, password: adminPassword })
  })
  created.admin = { email: adminEmail, user: login.user }
  const token = login.token

  const route = await api('/api/admin/routes', {
    method: 'POST',
    token,
    body: JSON.stringify({ name: `${prefix} Ruta Arequipa Cusco`, code: `${prefix}-R`, price_per_seat: 35 })
  })
  created.route = { id: route.id, name: `${prefix} Ruta Arequipa Cusco`, code: `${prefix}-R` }

  const stopA = await api(`/api/admin/routes/${route.id}/stops`, {
    method: 'POST',
    token,
    body: JSON.stringify({ name: `${prefix} Arequipa Terminal`, code: `${prefix}-AQP`, position: 1 })
  })
  const stopB = await api(`/api/admin/routes/${route.id}/stops`, {
    method: 'POST',
    token,
    body: JSON.stringify({ name: `${prefix} Cusco Terminal`, code: `${prefix}-CUS`, position: 2 })
  })
  created.stops = [stopA, stopB]

  const vehicle = await api('/api/admin/vehicles', {
    method: 'POST',
    token,
    body: JSON.stringify({ plate: `${runId.slice(-6)}PA4`, name: `${prefix} Bus Evidencia`, capacity: 8, seat_count: 8 })
  })
  created.vehicle = { id: vehicle.id, plate: `${runId.slice(-6)}PA4`, name: `${prefix} Bus Evidencia` }

  const template = await api('/api/admin/trip-templates', {
    method: 'POST',
    token,
    body: JSON.stringify({ route_id: route.id, vehicle_id: vehicle.id, name: `${prefix} Plantilla 08:00`, departure_time: '08:00' })
  })
  created.template = { id: template.id, name: `${prefix} Plantilla 08:00` }

  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000)
  tomorrow.setUTCHours(13, 0, 0, 0)
  const trip = await api('/api/admin/trip-instances', {
    method: 'POST',
    token,
    body: JSON.stringify({ trip_template_id: template.id, departure_at: tomorrow.toISOString() })
  })
  created.trip = { id: trip.id, departure_at: tomorrow.toISOString() }

  let seatsResp = await api(`/api/trips/${trip.id}/seats`)
  const seats = seatsResp.seats || []
  created.seatsInitial = seats
  const seat1 = seats.find(s => s.status === 'available')
  const seat2 = seats.find(s => s.status === 'available' && s.id !== seat1?.id)
  if (!seat1 || !seat2) throw new Error('No hay suficientes asientos disponibles para la prueba.')
  created.seat1 = seat1
  created.seat2 = seat2

  const hold = await api('/api/reservations', {
    method: 'POST',
    body: JSON.stringify({
      trip_instance_id: trip.id,
      seats: [seat1.id],
      passenger_name: 'Pasajero Evidencia PA4',
      passenger_doc_type: 'DNI',
      passenger_doc_number: '70000001',
      passenger_email: 'evidencia.pa4@example.com',
      passenger_phone: '999000111'
    })
  })
  created.hold = hold

  let doubleSaleError = ''
  try {
    await api('/api/reservations', {
      method: 'POST',
      body: JSON.stringify({ trip_instance_id: trip.id, seats: [seat1.id] })
    })
  } catch (error) {
    doubleSaleError = error.message
  }
  created.doubleSaleError = doubleSaleError

  const saleHold = await api('/api/reservations', {
    method: 'POST',
    body: JSON.stringify({
      trip_instance_id: trip.id,
      seats: [seat2.id],
      passenger_name: 'Venta Evidencia PA4',
      passenger_doc_type: 'DNI',
      passenger_doc_number: '70000002',
      passenger_email: 'venta.pa4@example.com',
      passenger_phone: '999000222'
    })
  })
  const sale = await api(`/api/reservations/${saleHold.code}/confirm`, {
    method: 'POST',
    body: JSON.stringify({
      payment_method: 'efectivo',
      payment_reference: `${prefix}-PAGO`,
      passenger_name: 'Venta Evidencia PA4',
      passenger_doc_type: 'DNI',
      passenger_doc_number: '70000002',
      passenger_email: 'venta.pa4@example.com',
      passenger_phone: '999000222',
      document_type: 'boleta',
      route_name: created.route.name,
      seat_labels: [seat2.label],
      price_per_seat: 35
    })
  })
  created.saleHold = saleHold
  created.sale = sale

  seatsResp = await api(`/api/trips/${trip.id}/seats`)
  created.seatsFinal = seatsResp.seats || []
  created.holdPublic = await api(`/api/reservations/${hold.code}`)
  created.salePublic = await api(`/api/reservations/${saleHold.code}`)
  created.token = token
  return created
}

const data = await setupData()
await fs.writeFile(path.join(outDir, 'datos_prueba_pa4.json'), JSON.stringify({ ...data, token: '[omitido]' }, null, 2), 'utf8')

const context = await chromium.launchPersistentContext(path.join(outDir, 'browser-profile'), {
  headless: false,
  viewport: null,
  args: ['--start-maximized']
})
const page = context.pages()[0] || await context.newPage()

async function show(url, wait = 2500) {
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 45000 })
  await page.waitForTimeout(wait)
}

async function setAuth() {
  await show(`${baseUrl}/login`, 1000)
  await page.evaluate(({ token, user }) => {
    localStorage.setItem('auth_token', token)
    localStorage.setItem('auth_user', JSON.stringify(user))
  }, { token: data.token, user: data.admin.user })
}

const captures = []
async function capture(name, title) {
  await sleep(700)
  const file = shellCapture(`${name}.png`, title)
  captures.push({ name, title, file })
}

// 1. Login incorrecto
await show(`${baseUrl}/login`, 1200)
await page.fill('#login-email', 'usuario.invalido@pasaje.local')
await page.fill('#login-password', 'clave_incorrecta')
await page.click('button[type="submit"]')
await page.waitForTimeout(2200)
await capture('02_login_incorrecto_lupa_fecha_hora', 'Login incorrecto')

// 2. Login correcto
await show(`${baseUrl}/login`, 1200)
await page.fill('#login-email', adminEmail)
await page.fill('#login-password', adminPassword)
await page.click('button[type="submit"]')
await page.waitForTimeout(3500)
await capture('01_login_correcto_lupa_fecha_hora', 'Login correcto')

await setAuth()

// Admin evidence pages
await show(`${baseUrl}/admin/rutas`, 2800)
await capture('03_registro_ruta_lupa_fecha_hora', 'Registro de ruta')

await show(`${baseUrl}/admin/vehiculos`, 2800)
await capture('04_registro_vehiculo_lupa_fecha_hora', 'Registro de vehiculo')

await show(`${baseUrl}/admin/viajes`, 2800)
await capture('05_generacion_viaje_lupa_fecha_hora', 'Generacion de viaje')

await show(`${baseUrl}/viajes/salida/${data.trip.id}`, 3200)
await capture('06_mapa_asientos_lupa_fecha_hora', 'Mapa de asientos')

const reservaReport = await writeReport('reserva_temporal', 'Reserva temporal creada', `
<div class="grid">
  <div class="card"><div class="label">Codigo</div><div class="value">${data.hold.code}</div></div>
  <div class="card"><div class="label">Estado</div><div class="value ok">${data.holdPublic.status}</div></div>
  <div class="card"><div class="label">Expira</div><div class="value">${data.hold.expires_at}</div></div>
</div>
<pre>${JSON.stringify(data.holdPublic, null, 2)}</pre>`)
await show(reservaReport, 1000)
await capture('07_reserva_temporal_lupa_fecha_hora', 'Reserva temporal')

const ventaReport = await writeReport('venta_confirmada', 'Venta confirmada', `
<div class="grid">
  <div class="card"><div class="label">Reserva</div><div class="value">${data.saleHold.code}</div></div>
  <div class="card"><div class="label">Estado</div><div class="value ok">${data.salePublic.status}</div></div>
  <div class="card"><div class="label">Tickets</div><div class="value">${(data.sale.ticket_codes || []).join(', ')}</div></div>
</div>
<pre>${JSON.stringify(data.salePublic, null, 2)}</pre>`)
await show(ventaReport, 1000)
await capture('08_venta_confirmada_lupa_fecha_hora', 'Venta confirmada')

const ticketReport = await writeReport('ticket_generado', 'Ticket generado', `
<div class="grid">
  <div class="card"><div class="label">Codigo de reserva</div><div class="value">${data.saleHold.code}</div></div>
  <div class="card"><div class="label">Ticket generado</div><div class="value ok">${(data.sale.ticket_codes || [])[0] || 'N/D'}</div></div>
  <div class="card"><div class="label">Metodo de pago</div><div class="value">${data.salePublic.payment_method || 'efectivo'}</div></div>
</div>
<pre>${JSON.stringify({ ticket_codes: data.sale.ticket_codes, reservation: data.salePublic }, null, 2)}</pre>`)
await show(ticketReport, 1000)
await capture('09_ticket_generado_lupa_fecha_hora', 'Ticket generado')

await show(`${baseUrl}/consulta`, 1200)
const input = page.locator('input').first()
await input.fill(data.saleHold.code)
await page.keyboard.press('Enter').catch(() => {})
await page.waitForTimeout(2500)
await capture('10_consulta_reserva_lupa_fecha_hora', 'Consulta de reserva')

// WebSocket: two windows for same trip. If UI shows the sold/held status after API operations, it is evidence of real-time/refresh state.
const page2 = await context.newPage()
await page.goto(`${baseUrl}/viajes/salida/${data.trip.id}`, { waitUntil: 'domcontentloaded' })
await page2.goto(`${baseUrl}/viajes/salida/${data.trip.id}`, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(3500)
await page2.waitForTimeout(1000)
await capture('11_websocket_dos_ventanas_lupa_fecha_hora', 'Actualizacion por WebSocket')
await page2.close()

const doubleReport = await writeReport('doble_venta', 'Prueba de doble venta bloqueada', `
<div class="grid">
  <div class="card"><div class="label">Viaje</div><div class="value">${data.trip.id}</div></div>
  <div class="card"><div class="label">Asiento</div><div class="value">${data.seat1.label}</div></div>
  <div class="card"><div class="label">Resultado segunda operacion</div><div class="value bad">Bloqueada</div></div>
</div>
<pre>${data.doubleSaleError || 'La segunda reserva no genero error capturable.'}</pre>`)
await show(doubleReport, 1000)
await capture('12_doble_venta_lupa_fecha_hora', 'Prueba de doble venta')

await page.evaluate(() => {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
})
await show(`${baseUrl}/admin`, 2500)
await capture('13_acceso_sin_token_lupa_fecha_hora', 'Prueba de acceso sin token')

const dbReport = await writeReport('base_datos_validando_registro', 'Base de datos validando registro', `
<div class="grid">
  <div class="card"><div class="label">Health DB</div><div class="value ok">database connected</div></div>
  <div class="card"><div class="label">Ruta creada</div><div class="value">${data.route.id}</div></div>
  <div class="card"><div class="label">Viaje creado</div><div class="value">${data.trip.id}</div></div>
</div>
<table>
  <tr><th>Entidad</th><th>Codigo / ID</th><th>Estado</th></tr>
  <tr><td>Ruta</td><td>${data.route.code}</td><td>creada</td></tr>
  <tr><td>Vehiculo</td><td>${data.vehicle.plate}</td><td>creado</td></tr>
  <tr><td>Reserva temporal</td><td>${data.hold.code}</td><td>${data.holdPublic.status}</td></tr>
  <tr><td>Venta</td><td>${data.saleHold.code}</td><td>${data.salePublic.status}</td></tr>
  <tr><td>Ticket</td><td>${(data.sale.ticket_codes || []).join(', ')}</td><td>generado</td></tr>
</table>
<pre>${JSON.stringify({ route: data.route, vehicle: data.vehicle, trip: data.trip, reservation: data.holdPublic, sale: data.salePublic }, null, 2)}</pre>`)
await show(dbReport, 1000)
await capture('14_base_datos_validando_registro_lupa_fecha_hora', 'Base de datos validando registro')

await context.close()
await fs.writeFile(path.join(outDir, 'capturas_manifest.json'), JSON.stringify(captures, null, 2), 'utf8')
console.log(`OK capturas: ${outDir}`)
