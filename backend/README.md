# Backend - Pasaje API (Go)

API REST del sistema de reserva y venta de pasajes interprovinciales.

## Como ejecutar

### Opcion 1: Script con archivo .env

```bash
./scripts/run.sh
```

El script carga las variables de entorno desde `config/.env` y ejecuta `go run ./cmd/server/`.

### Opcion 2: Variables de entorno directas

```bash
DB_PASSWORD=tu_password go run ./cmd/server/
```

El servidor inicia en el puerto configurado (por defecto `8085`).

## Configuracion

Todas las variables se leen del entorno. Valores por defecto entre parentesis:

| Variable | Descripcion | Default |
|----------|-------------|---------|
| `DB_HOST` | Host de PostgreSQL | `localhost` |
| `DB_PORT` | Puerto de PostgreSQL | `5432` |
| `DB_USER` | Usuario de PostgreSQL | `postgres` |
| `DB_PASSWORD` | Password de PostgreSQL | *(vacio)* |
| `DB_NAME` | Nombre de la base de datos | `pasaje` |
| `DB_SSLMODE` | Modo SSL de PostgreSQL | `disable` |
| `API_PORT` | Puerto HTTP del servidor | `8085` |

## Estructura de carpetas

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada de la aplicacion
├── internal/
│   ├── config/
│   │   └── config.go            # Lectura de configuracion desde entorno
│   ├── domain/
│   │   ├── reservation.go       # Reservation, ReservationItem, Ticket, Payment
│   │   ├── route.go             # Route
│   │   ├── trip.go              # TripInstance, TripInstanceAdmin, SeatInfo
│   │   ├── trip_template.go     # TripTemplate
│   │   └── vehicle.go           # Vehicle, VehicleSeat
│   ├── handler/
│   │   ├── cors.go              # Middleware CORS
│   │   ├── health.go            # Health check handlers
│   │   ├── router.go            # Registro de rutas HTTP (chi)
│   │   └── routes.go            # Handlers publicos (rutas, viajes, asientos)
│   ├── repository/
│   │   ├── codegen.go           # Generacion de codigos aleatorios
│   │   ├── reservation.go       # CRUD reservas, confirmacion, tickets, pagos
│   │   ├── route.go             # CRUD rutas
│   │   ├── stats.go             # Estadisticas del dashboard
│   │   ├── trip.go              # Instancias de viaje, inventario de asientos
│   │   ├── trip_template.go     # CRUD plantillas de viaje
│   │   └── vehicle.go           # CRUD vehiculos y asientos
│   └── service/
│       └── routes.go            # Logica de negocio (rutas, viajes, asientos)
├── pkg/
│   └── db/
│       └── db.go                # Pool de conexiones PostgreSQL (pgxpool)
├── migrations/
│   ├── 000001_init_schema.up.sql        # Tablas base del dominio
│   ├── 000001_init_schema.down.sql
│   ├── 000002_trip_seat_inventory.up.sql # Inventario de asientos por viaje
│   ├── 000002_trip_seat_inventory.down.sql
│   ├── 000003_reservations_tickets.up.sql # Reservas, tickets, pagos
│   ├── 000003_reservations_tickets.down.sql
│   ├── 000004_audit_logs.up.sql          # Tabla de auditoria
│   ├── 000004_audit_logs.down.sql
│   ├── 000005_seed_minimal.up.sql        # Datos semilla para desarrollo
│   └── 000005_seed_minimal.down.sql
├── scripts/
│   └── run.sh                   # Script de arranque con .env
├── config/
│   └── README.md
├── go.mod
└── go.sum
```

## Endpoints API

### Health

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `GET` | `/health/live` | Liveness check (servidor vivo) |
| `GET` | `/health/ready` | Readiness check (conexion a DB activa) |

### Publico

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `GET` | `/api/routes` | Listar rutas activas |
| `GET` | `/api/routes/{id}/trips` | Listar viajes de una ruta (query: `?from=2026-03-15`) |
| `GET` | `/api/trips/{id}/seats` | Ver asientos de un viaje con su estado |

### Reservas (repositorio implementado, handler pendiente de conectar al router)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `POST` | `/api/reservations` | Crear reserva (hold temporal de 10 min) |
| `POST` | `/api/reservations/{code}/confirm` | Confirmar reserva con pago |
| `GET` | `/api/reservations/{code}` | Consultar reserva por codigo |

### Admin (repositorio implementado, handlers pendientes de conectar al router)

| Metodo | Ruta | Descripcion |
|--------|------|-------------|
| `GET` | `/api/admin/routes` | Listar todas las rutas |
| `POST` | `/api/admin/routes` | Crear ruta |
| `PUT` | `/api/admin/routes/{id}` | Actualizar ruta |
| `DELETE` | `/api/admin/routes/{id}` | Desactivar ruta (soft delete) |
| `GET` | `/api/admin/vehicles` | Listar vehiculos |
| `POST` | `/api/admin/vehicles` | Crear vehiculo |
| `GET` | `/api/admin/vehicles/{id}/seats` | Listar asientos de un vehiculo |
| `POST` | `/api/admin/vehicles/{id}/seats` | Crear asiento de vehiculo |
| `GET` | `/api/admin/trip-templates` | Listar plantillas de viaje |
| `POST` | `/api/admin/trip-templates` | Crear plantilla de viaje |
| `GET` | `/api/admin/trip-instances` | Listar instancias de viaje (query: `?from=`) |
| `POST` | `/api/admin/trip-instances` | Crear instancia de viaje desde plantilla |
| `PUT` | `/api/admin/trip-instances/{id}` | Actualizar estado de instancia |
| `GET` | `/api/admin/stats` | Estadisticas del dashboard |

## Migraciones

Las migraciones se encuentran en `migrations/` y siguen numeracion secuencial:

| # | Archivo | Contenido |
|---|---------|-----------|
| 1 | `000001_init_schema` | Tablas base: agencies, sales_channels, users, routes, stops, vehicles, vehicle_seats, trip_templates, trip_instances |
| 2 | `000002_trip_seat_inventory` | Tabla `trip_seat_inventory` con UNIQUE(trip_instance_id, vehicle_seat_id) |
| 3 | `000003_reservations_tickets` | Tablas: reservations, reservation_items, tickets, payments |
| 4 | `000004_audit_logs` | Tabla `audit_logs` con indices por entidad y fecha |
| 5 | `000005_seed_minimal` | Datos semilla: ruta Arequipa-Cusco, bus de 12 asientos, plantilla, instancia |

## Modelo de dominio

### Route
Ruta entre ciudades (ej. Arequipa - Cusco). Campos: `id`, `name`, `code`, `active`.

### Vehicle
Vehiculo/bus. Campos: `id`, `plate`, `name`, `capacity`, `active`, `created_at`, `updated_at`.

### VehicleSeat
Asiento plantilla de un vehiculo. Campos: `id`, `vehicle_id`, `label`, `position`.

### TripTemplate
Plantilla de viaje: ruta + vehiculo + horario base. Campos: `id`, `route_id`, `vehicle_id`, `name`, `departure_time`, `active`, `created_at`, `updated_at`.

### TripInstance
Instancia de viaje en fecha concreta. Campos: `id`, `trip_template_id`, `route_id`, `route_name`, `departure_at`, `status`. Estados: `scheduled`, `in_progress`, `completed`, `cancelled`.

### SeatInfo
Asiento del inventario de un viaje. Campos: `id`, `vehicle_seat_id`, `label`, `position`, `status`, `hold_expires_at`. Estados: `available`, `held`, `sold`, `blocked`.

### Reservation
Reserva de asientos. Campos: `id`, `code`, `user_id`, `agency_id`, `sales_channel_id`, `status`, `expires_at`, `created_at`, `updated_at`. Estados: `pending`, `confirmed`, `expired`, `cancelled`.

### Ticket
Boleto emitido tras confirmar reserva. Campos: `id`, `code`, `reservation_id`, `trip_instance_id`, `trip_seat_inventory_id`, `user_id`, `agency_id`, `sales_channel_id`, `status`, `created_at`, `updated_at`.

### Payment
Pago asociado a un ticket. Campos: `id`, `ticket_id`, `amount_cents`, `currency`, `method`, `reference`, `status`, `created_at`, `updated_at`.
