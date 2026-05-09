# Referencia de API - Pasaje

Base URL: `http://localhost:8085`

Todos los endpoints devuelven JSON con `Content-Type: application/json; charset=utf-8`.

En caso de error, la respuesta tiene el formato:

```json
{ "error": "mensaje descriptivo del error" }
```

---

## Health

### GET /health/live

Verifica que el servidor esta en ejecucion.

**Respuesta exitosa** `200 OK`:

```json
{ "status": "ok" }
```

---

### GET /health/ready

Verifica que el servidor esta listo (conexion a base de datos activa).

**Respuesta exitosa** `200 OK`:

```json
{ "status": "ok" }
```

**Respuesta de error** `503 Service Unavailable`:

```json
{ "status": "unavailable" }
```

---

## Publico

### GET /api/routes

Lista todas las rutas activas.

**Parametros**: ninguno.

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "name": "Arequipa - Cusco",
    "code": "ARQ-CUS",
    "active": true
  }
]
```

Devuelve un array vacio `[]` si no hay rutas.

---

### GET /api/routes/{id}/trips

Lista los viajes programados (instancias) de una ruta, filtrados desde una fecha.

**Parametros de ruta**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `id` | int | si | ID de la ruta |

**Query parameters**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `from` | string | no | Fecha minima en formato `YYYY-MM-DD`. Si no se envia, usa la fecha actual. |

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "trip_template_id": 1,
    "route_id": 1,
    "route_name": "Arequipa - Cusco",
    "departure_at": "2026-03-15T08:00:00Z",
    "status": "scheduled"
  }
]
```

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `400` | ID de ruta invalido |
| `500` | Error interno |

---

### GET /api/trips/{id}/seats

Devuelve el inventario de asientos de un viaje con su estado actual.

**Parametros de ruta**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `id` | int | si | ID de la instancia de viaje |

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 101,
    "vehicle_seat_id": 1,
    "label": "1",
    "position": 1,
    "status": "available"
  },
  {
    "id": 102,
    "vehicle_seat_id": 2,
    "label": "2",
    "position": 2,
    "status": "held",
    "hold_expires_at": "2026-03-14T16:30:00Z"
  },
  {
    "id": 103,
    "vehicle_seat_id": 3,
    "label": "3",
    "position": 3,
    "status": "sold"
  }
]
```

**Estados posibles de un asiento**: `available`, `held`, `sold`, `blocked`.

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `400` | ID de viaje invalido |
| `404` | Viaje no encontrado |
| `500` | Error interno |

---

## Reservas

> **Nota**: La logica de repositorio esta implementada. Los handlers HTTP estan pendientes de conectar al router.

### POST /api/reservations

Crea una reserva temporal (hold) de uno o mas asientos.

**Request body**:

```json
{
  "trip_instance_id": 1,
  "seat_ids": [101, 102]
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `trip_instance_id` | int | si | ID de la instancia de viaje |
| `seat_ids` | int[] | si | IDs de los asientos del inventario (`trip_seat_inventory.id`) |

**Respuesta exitosa** `201 Created`:

```json
{
  "id": 1,
  "code": "A3K9M2PX",
  "sales_channel_id": 1,
  "status": "pending",
  "expires_at": "2026-03-14T16:30:00Z",
  "created_at": "2026-03-14T16:20:00Z",
  "updated_at": "2026-03-14T16:20:00Z"
}
```

**Comportamiento**:
- Bloquea los asientos con `SELECT FOR UPDATE`.
- Verifica que todos esten en estado `available`.
- Cambia su estado a `held` con `hold_expires_at = now() + 10 minutos`.
- Crea la reserva con status `pending`.
- Todo ocurre en una unica transaccion.

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `400` | Datos invalidos o asientos no encontrados |
| `409` | Uno o mas asientos no estan disponibles |
| `500` | Error interno |

---

### POST /api/reservations/{code}/confirm

Confirma una reserva pendiente, marcando los asientos como vendidos y emitiendo tickets.

**Parametros de ruta**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `code` | string | si | Codigo de la reserva (ej. `A3K9M2PX`) |

**Request body**:

```json
{
  "payment_method": "cash",
  "payment_reference": "REC-001"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `payment_method` | string | si | Metodo de pago (`cash`, `card`, `transfer`, etc.) |
| `payment_reference` | string | no | Referencia externa del pago |

**Respuesta exitosa** `200 OK`:

```json
{
  "ticket_codes": ["TKT-B7X4R9WQ", "TKT-K2M8N5PL"]
}
```

**Comportamiento**:
- Bloquea la reserva con `SELECT FOR UPDATE`.
- Verifica que el status sea `pending` y que no haya expirado.
- Cambia los asientos de `held` a `sold`.
- Cambia la reserva de `pending` a `confirmed`.
- Crea un ticket y un registro de pago por cada asiento.
- Todo ocurre en una unica transaccion.

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `404` | Reserva no encontrada |
| `409` | Reserva no esta en estado `pending` |
| `410` | Reserva expirada |
| `500` | Error interno |

---

### GET /api/reservations/{code}

Consulta una reserva por su codigo.

**Parametros de ruta**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `code` | string | si | Codigo de la reserva |

**Respuesta exitosa** `200 OK`:

```json
{
  "id": 1,
  "code": "A3K9M2PX",
  "user_id": null,
  "agency_id": null,
  "sales_channel_id": 1,
  "status": "confirmed",
  "expires_at": "2026-03-14T16:30:00Z",
  "created_at": "2026-03-14T16:20:00Z",
  "updated_at": "2026-03-14T16:25:00Z"
}
```

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `404` | Reserva no encontrada |
| `500` | Error interno |

---

## Admin

> **Nota**: La logica de repositorio esta implementada. Los handlers HTTP estan pendientes de conectar al router.

### GET /api/admin/routes

Lista todas las rutas (incluyendo inactivas).

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "name": "Arequipa - Cusco",
    "code": "ARQ-CUS",
    "active": true
  }
]
```

---

### POST /api/admin/routes

Crea una nueva ruta.

**Request body**:

```json
{
  "name": "Arequipa - Cusco",
  "code": "ARQ-CUS"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `name` | string | si | Nombre de la ruta |
| `code` | string | si | Codigo unico de la ruta |

**Respuesta exitosa** `201 Created`:

```json
{ "id": 1 }
```

**Errores**:

| Codigo | Condicion |
|--------|-----------|
| `400` | Datos invalidos |
| `409` | Codigo de ruta duplicado |
| `500` | Error interno |

---

### PUT /api/admin/routes/{id}

Actualiza una ruta existente.

**Request body**:

```json
{
  "name": "Arequipa - Cusco Express",
  "code": "ARQ-CUS",
  "active": true
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `name` | string | si | Nombre de la ruta |
| `code` | string | si | Codigo de la ruta |
| `active` | bool | si | Si la ruta esta activa |

**Respuesta exitosa** `200 OK`:

```json
{ "ok": true }
```

---

### DELETE /api/admin/routes/{id}

Desactiva una ruta (soft delete). No elimina el registro.

**Respuesta exitosa** `200 OK`:

```json
{ "ok": true }
```

---

### GET /api/admin/vehicles

Lista todos los vehiculos.

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "plate": "ABC-123",
    "name": "Bus estandar",
    "capacity": 12,
    "active": true,
    "created_at": "2026-03-14T10:00:00Z",
    "updated_at": "2026-03-14T10:00:00Z"
  }
]
```

---

### POST /api/admin/vehicles

Crea un nuevo vehiculo.

**Request body**:

```json
{
  "plate": "ABC-123",
  "name": "Bus estandar",
  "capacity": 40
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `plate` | string | si | Placa del vehiculo (unica) |
| `name` | string | no | Nombre descriptivo |
| `capacity` | int | si | Capacidad de asientos |

**Respuesta exitosa** `201 Created`:

```json
{ "id": 1 }
```

---

### GET /api/admin/vehicles/{id}/seats

Lista los asientos plantilla de un vehiculo.

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "vehicle_id": 1,
    "label": "1",
    "position": 1
  },
  {
    "id": 2,
    "vehicle_id": 1,
    "label": "2",
    "position": 2
  }
]
```

---

### POST /api/admin/vehicles/{id}/seats

Crea un asiento en la plantilla del vehiculo.

**Request body**:

```json
{
  "label": "1A",
  "position": 1
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `label` | string | si | Etiqueta visible del asiento |
| `position` | int | si | Posicion ordinal |

**Respuesta exitosa** `201 Created`:

```json
{ "id": 1 }
```

---

### GET /api/admin/trip-templates

Lista todas las plantillas de viaje con datos de ruta y vehiculo.

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "route_id": 1,
    "vehicle_id": 1,
    "name": "Salida matinal",
    "departure_time": "08:00:00",
    "active": true,
    "route_name": "Arequipa - Cusco",
    "vehicle_name": "Bus estandar",
    "created_at": "2026-03-14T10:00:00Z",
    "updated_at": "2026-03-14T10:00:00Z"
  }
]
```

---

### POST /api/admin/trip-templates

Crea una plantilla de viaje.

**Request body**:

```json
{
  "route_id": 1,
  "vehicle_id": 1,
  "name": "Salida matinal",
  "departure_time": "08:00"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `route_id` | int | si | ID de la ruta |
| `vehicle_id` | int | si | ID del vehiculo |
| `name` | string | si | Nombre descriptivo |
| `departure_time` | string | si | Hora de salida en formato `HH:MM` |

**Respuesta exitosa** `201 Created`:

```json
{ "id": 1 }
```

---

### GET /api/admin/trip-instances

Lista las instancias de viaje con datos extendidos.

**Query parameters**:

| Parametro | Tipo | Requerido | Descripcion |
|-----------|------|-----------|-------------|
| `from` | string | no | Fecha minima en formato `YYYY-MM-DD` |

**Respuesta exitosa** `200 OK`:

```json
[
  {
    "id": 1,
    "trip_template_id": 1,
    "route_id": 1,
    "route_name": "Arequipa - Cusco",
    "vehicle_id": 1,
    "vehicle_name": "Bus estandar",
    "departure_at": "2026-03-15T08:00:00Z",
    "status": "scheduled",
    "created_at": "2026-03-14T10:00:00Z",
    "updated_at": "2026-03-14T10:00:00Z"
  }
]
```

---

### POST /api/admin/trip-instances

Crea una instancia de viaje a partir de una plantilla. Genera automaticamente el inventario de asientos (un registro por cada asiento del vehiculo, todos en estado `available`).

**Request body**:

```json
{
  "trip_template_id": 1,
  "departure_at": "2026-03-15T08:00:00Z"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `trip_template_id` | int | si | ID de la plantilla de viaje |
| `departure_at` | string | si | Fecha y hora de salida (ISO 8601) |

**Respuesta exitosa** `201 Created`:

```json
{ "id": 1 }
```

**Comportamiento**:
1. Obtiene el `vehicle_id` de la plantilla.
2. Crea la instancia de viaje con status `scheduled`.
3. Genera una fila en `trip_seat_inventory` por cada `vehicle_seat` del vehiculo, con status `available`.
4. Todo ocurre en una unica transaccion.

---

### PUT /api/admin/trip-instances/{id}

Actualiza el estado de una instancia de viaje.

**Request body**:

```json
{
  "status": "cancelled"
}
```

| Campo | Tipo | Requerido | Descripcion |
|-------|------|-----------|-------------|
| `status` | string | si | Nuevo estado (`scheduled`, `in_progress`, `completed`, `cancelled`) |

**Respuesta exitosa** `200 OK`:

```json
{ "ok": true }
```

---

### GET /api/admin/stats

Devuelve estadisticas del dashboard administrativo.

**Respuesta exitosa** `200 OK`:

```json
{
  "total_routes": 3,
  "total_vehicles": 5,
  "upcoming_trips": 12,
  "reservations_today": 8
}
```

| Campo | Tipo | Descripcion |
|-------|------|-------------|
| `total_routes` | int | Rutas activas |
| `total_vehicles` | int | Vehiculos activos |
| `upcoming_trips` | int | Viajes programados con salida futura |
| `reservations_today` | int | Reservas creadas hoy |
