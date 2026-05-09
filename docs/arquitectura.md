# Arquitectura del Sistema Pasaje

## Diagrama de componentes

```
┌─────────────────────────────────────────────────────────┐
│                      CLIENTES                           │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Portal       │  │ Panel        │  │ Puntos de    │  │
│  │ Publico      │  │ Admin        │  │ Venta        │  │
│  │ (Vue 3 + TS) │  │ (Vue 3 + TS) │  │ (futuro)     │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
└─────────┼─────────────────┼─────────────────┼───────────┘
          │ HTTP/JSON       │ HTTP/JSON       │ HTTP/JSON
          │ (WebSocket*)    │                 │
          ▼                 ▼                 ▼
┌─────────────────────────────────────────────────────────┐
│              BACKEND (Go - Monolito Modular)            │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │                  HTTP Router (chi)                │   │
│  │  /health/*  /api/*  /api/admin/*  /ws/* (futuro) │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                               │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │              Handlers (handler/)                  │   │
│  │  HealthHandler  RoutesHandler  (Admin pendiente)  │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                               │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │              Servicios (service/)                 │   │
│  │              Logica de negocio                    │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                               │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │            Repositorios (repository/)             │   │
│  │  RouteRepo  VehicleRepo  TripRepo  ReservationRepo│  │
│  │  TripTemplateRepo  StatsRepo                      │  │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                               │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │              pgxpool (pkg/db/)                    │   │
│  └──────────────────────┬───────────────────────────┘   │
└─────────────────────────┼───────────────────────────────┘
                          │ TCP/5432
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    PostgreSQL                           │
│                                                         │
│  15 tablas con restricciones UNIQUE, indices,           │
│  transacciones y SELECT FOR UPDATE                      │
│                                                         │
│  Fuente de verdad para el estado de asientos            │
└─────────────────────────────────────────────────────────┘

                          │ (futuro)
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 RabbitMQ (futuro)                        │
│  Solo para tareas asincronas:                           │
│  - Notificaciones email/SMS                             │
│  - Emision de comprobantes                              │
│  - Eventos de auditoria                                 │
│  - Proyecciones/reportes                                │
│  NO para control de asientos                            │
└─────────────────────────────────────────────────────────┘

* WebSocket planificado para Fase 3
```

## Capas del backend

```
Handler  →  Service  →  Repository  →  PostgreSQL
(HTTP)      (logica)    (SQL/pgx)      (fuente de verdad)
```

- **Handler**: recibe HTTP request, valida entrada, delega al servicio, devuelve JSON.
- **Service**: orquesta logica de negocio, combina repositorios.
- **Repository**: ejecuta queries SQL, maneja transacciones y locks.
- **Domain**: structs Go que representan las entidades del sistema.

## Flujo transaccional de reserva

Este es el flujo critico del sistema. Cada paso ocurre dentro de una transaccion PostgreSQL.

### Paso 1: Crear reserva (hold temporal)

```
Cliente: POST /api/reservations
Body: { trip_instance_id: 1, seat_ids: [5, 6] }

Backend:
  1. BEGIN TRANSACTION
  2. SELECT id, status FROM trip_seat_inventory
     WHERE id = ANY($seat_ids) AND trip_instance_id = $trip_id
     FOR UPDATE                            ← bloquea filas
  3. Verificar que TODOS los asientos existen y status = 'available'
     Si alguno no esta disponible → ROLLBACK + error 409
  4. UPDATE trip_seat_inventory
     SET status = 'held',
         hold_expires_at = now() + interval '10 minutes',
         held_by = $reservation_code
     WHERE id = ANY($seat_ids)
  5. INSERT INTO reservations (code, sales_channel_id, status, expires_at)
     VALUES ($code, $channel, 'pending', $expires_at)
  6. INSERT INTO reservation_items (reservation_id, trip_seat_inventory_id)
     VALUES ($res_id, $seat_id)             ← uno por asiento
  7. COMMIT

Resultado: Reserva con codigo (ej. "A3K9M2PX"), status 'pending', expira en 10 min
```

### Paso 2: Confirmar reserva (pago)

```
Cliente: POST /api/reservations/{code}/confirm
Body: { payment_method: "cash", payment_reference: "..." }

Backend:
  1. BEGIN TRANSACTION
  2. SELECT id, status, expires_at FROM reservations
     WHERE code = $code
     FOR UPDATE                            ← bloquea la reserva
  3. Verificar status = 'pending' Y no expirada
     Si expirada o no pending → ROLLBACK + error
  4. SELECT reservation_items + trip_seat_inventory
  5. UPDATE trip_seat_inventory
     SET status = 'sold', hold_expires_at = NULL
     WHERE id = ANY($seat_ids)
  6. UPDATE reservations SET status = 'confirmed'
  7. Para cada asiento:
     INSERT INTO tickets (code, reservation_id, trip_instance_id, ...)
     INSERT INTO payments (ticket_id, amount_cents, method, ...)
  8. COMMIT

Resultado: Tickets emitidos (ej. "TKT-B7X4R9WQ"), asientos marcados como 'sold'
```

### Paso 3: Expiracion de holds (pendiente de implementar)

```
Proceso programado (cron/goroutine):
  1. BEGIN TRANSACTION
  2. SELECT id FROM trip_seat_inventory
     WHERE status = 'held' AND hold_expires_at < now()
     FOR UPDATE
  3. UPDATE trip_seat_inventory
     SET status = 'available', hold_expires_at = NULL, held_by = NULL
     WHERE id = ANY($expired_ids)
  4. UPDATE reservations SET status = 'expired'
     WHERE id IN (SELECT reservation_id FROM reservation_items
                  WHERE trip_seat_inventory_id = ANY($expired_ids))
     AND status = 'pending'
  5. COMMIT
```

## Estrategia de concurrencia

### Problema

Multiples usuarios o puntos de venta intentan reservar el mismo asiento simultaneamente. Sin control, se produce doble venta.

### Solucion: tres mecanismos complementarios

#### 1. UNIQUE constraint (defensa pasiva)

```sql
UNIQUE(trip_instance_id, vehicle_seat_id)  -- en trip_seat_inventory
UNIQUE(trip_seat_inventory_id)             -- en reservation_items
```

Impide que existan dos registros de inventario para el mismo asiento en el mismo viaje, y que un asiento aparezca en dos reservas activas.

#### 2. SELECT ... FOR UPDATE (defensa activa)

```sql
SELECT id, status FROM trip_seat_inventory
WHERE id = ANY($seat_ids) AND trip_instance_id = $trip_id
FOR UPDATE
```

Bloquea las filas seleccionadas hasta que la transaccion termine. Si otra transaccion intenta leer las mismas filas con `FOR UPDATE`, queda en espera hasta que la primera haga COMMIT o ROLLBACK.

#### 3. Transacciones (atomicidad)

Toda operacion critica (hold, confirm, expire, cancel) se ejecuta dentro de una transaccion PostgreSQL. Si cualquier paso falla, se hace ROLLBACK y ninguno de los cambios se aplica.

### Flujo de concurrencia (ejemplo)

```
Usuario A                         Usuario B
   │                                 │
   ├─ BEGIN TX                       │
   ├─ SELECT FOR UPDATE seat 5       │
   │  (obtiene lock)                 │
   │                                 ├─ BEGIN TX
   │                                 ├─ SELECT FOR UPDATE seat 5
   │                                 │  (BLOQUEADO, espera...)
   ├─ UPDATE seat 5 → held           │
   ├─ INSERT reservation             │
   ├─ COMMIT                         │
   │  (libera lock)                  │
   │                                 │  (desbloqueo)
   │                                 ├─ Lee seat 5 → status='held'
   │                                 ├─ ROLLBACK (no disponible)
   │                                 ├─ Error: "asiento no disponible"
```

## Modelo de estados de asientos

Los asientos del inventario (`trip_seat_inventory.status`) siguen este modelo de estados:

```
                    reservar (hold)
    ┌──────────┐  ────────────────►  ┌──────────┐
    │          │                     │          │
    │available │                     │  held    │
    │          │  ◄────────────────  │          │
    └──────────┘    expirar hold     └────┬─────┘
         ▲                                │
         │                                │ confirmar pago
         │                                │
         │         anular ticket          ▼
         │       ◄────────────────  ┌──────────┐
         │                          │          │
         └──────────────────────────│  sold    │
                                    │          │
                                    └──────────┘

    ┌──────────┐
    │ blocked  │  (reservado por admin, fuera de venta)
    └──────────┘
```

### Transiciones validas

| Desde | Hacia | Evento | Operacion SQL |
|-------|-------|--------|---------------|
| `available` | `held` | Pasajero crea reserva | `UPDATE SET status='held', hold_expires_at=...` |
| `held` | `sold` | Pago confirmado | `UPDATE SET status='sold', hold_expires_at=NULL` |
| `held` | `available` | Hold expirado | `UPDATE SET status='available', hold_expires_at=NULL` |
| `held` | `available` | Reserva cancelada | `UPDATE SET status='available', hold_expires_at=NULL` |
| `sold` | `available` | Ticket anulado (flujo admin) | `UPDATE SET status='available'` |
| `available` | `blocked` | Admin bloquea asiento | `UPDATE SET status='blocked'` |
| `blocked` | `available` | Admin desbloquea | `UPDATE SET status='available'` |

## Modelo de estados de reserva

```
                     confirmar pago
    ┌──────────┐  ────────────────►  ┌───────────┐
    │          │                     │           │
    │ pending  │                     │ confirmed │
    │          │                     │           │
    └────┬─────┘                     └───────────┘
         │
         ├─── hold expira ──────►  ┌───────────┐
         │                          │  expired  │
         │                          └───────────┘
         │
         └─── usuario cancela ──►  ┌───────────┐
                                    │ cancelled │
                                    └───────────┘
```

### Transiciones validas

| Desde | Hacia | Evento |
|-------|-------|--------|
| `pending` | `confirmed` | Pago confirmado dentro del plazo |
| `pending` | `expired` | `hold_expires_at` superado sin pago |
| `pending` | `cancelled` | Pasajero cancela antes de pagar |

Una reserva `confirmed`, `expired` o `cancelled` es un estado final y no admite mas transiciones.

## Decisiones tecnicas

1. **PostgreSQL como fuente de verdad**: toda logica de disponibilidad de asientos se resuelve en la base de datos, no en cache ni en el frontend.

2. **Monolito modular**: estructura clara por capas (handler/service/repository) sin microservicios prematuros. Cada capa tiene responsabilidad definida.

3. **chi como router HTTP**: ligero, compatible con `net/http`, buena ergonomia para rutas RESTful.

4. **pgx como driver PostgreSQL**: soporte nativo para tipos PostgreSQL, pool de conexiones integrado, buen rendimiento.

5. **Codigos alfanumericos**: reservas (`A3K9M2PX`) y tickets (`TKT-B7X4R9WQ`) usan codigos legibles generados con `crypto/rand`.

6. **Soft delete para entidades administrativas**: rutas y vehiculos se desactivan (`active=false`) en lugar de eliminarse, preservando integridad referencial e historial.

7. **Hold con expiracion**: las reservas temporales tienen un campo `hold_expires_at` que permite liberar asientos automaticamente si el pasajero no paga a tiempo.
