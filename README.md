# Pasaje — Sistema de Reserva y Venta de Pasajes Interprovinciales

| Campo | Detalle |
|---|---|
| **Actividad** | Examen Final |
| **Curso** | Ingeniería Web |
| **Equipo** | Equipo 05 |
| **Empresa objeto de estudio** | Transporte Turismo Ancalla Cusco Perú S.A.C. |
| **Fecha** | Mayo de 2026 |

## Equipo 05

| Apellidos y nombres |
|---|
| ⭐ CHAMBI VILCA, Francisco Leo |
| ALAMA HERRERA, Percy Frank |
| PINO HUAMANI, Pepe |
| CUETO HENRIQUEZ, Christian Washington |
| HUARI NINANYA, Jonathan Hernan |
| GAMERO AGUILAR, Roberto Carlos |

---

## 1. Descripción del sistema

**Pasaje** es un sistema web para la **reserva y venta de pasajes interprovinciales** de la empresa **Transporte Turismo Ancalla Cusco Perú S.A.C.**, que actualmente opera la ruta **Arequipa – Cusco**. Está diseñado para crecer a nuevas rutas, agencias y canales de venta.

El problema central que resuelve es la **doble venta del mismo asiento**, causada por la falta de sincronización entre múltiples puntos de venta (web, ventanilla, agencias). Pasaje **centraliza el inventario de asientos** y garantiza consistencia transaccional sobre PostgreSQL.

### Funcionalidades principales

- Búsqueda de viajes por ruta y fecha.
- Selección de asientos sobre un mapa interactivo del bus.
- Reserva temporal con *hold* automático de 10 minutos.
- Confirmación con pago: Culqi (tarjeta) o Yape directo (voucher).
- Login de clientes con Google OAuth y autenticación interna con JWT.
- Panel administrativo: CRUD de rutas, vehículos, layouts, plantillas e instancias de viaje.
- Encomiendas (envío de paquetes asociados a viajes).
- Tarifas por segmentos de ruta intermedios.
- Integración con sistema de facturación electrónica (Venta + APIPeru).
- Auditoría de cambios críticos.
- Notificaciones en tiempo real vía WebSockets.

---

## 2. Stack tecnológico

| Capa | Tecnología |
|---|---|
| **Frontend** | Vue 3 + TypeScript + Vite, Vue Router 4, SweetAlert2 |
| **Backend** | Go 1.x · `chi` (router HTTP) · `pgx` / `pgxpool` (driver PostgreSQL) |
| **Base de datos** | PostgreSQL · PgBouncer (pool en producción) |
| **Autenticación** | JWT propio + Google OAuth (web y mobile) |
| **Pagos** | Culqi (tarjeta) · Yape directo (voucher con validación manual) |
| **Facturación electrónica** | Sistema Venta · APIPeru (Perú) |
| **Tiempo real** | WebSockets nativos (`/ws/*`) |
| **Mensajería** | SMTP (Gmail App Password) para correos |
| **Infraestructura local** | PostgreSQL local + servidor Go + dev server Vite |
| **Herramientas QA** | Playwright (capturar evidencias web) · OWASP ZAP (auditoría de seguridad) |

---

## 3. Arquitectura

**Monolito modular** por capas, sin microservicios prematuros. La base de datos es la **única fuente de verdad** para la disponibilidad de asientos.

```
┌────────────────────────────────────────────────────────────────┐
│                          CLIENTES                              │
│                                                                │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐       │
│   │ Portal       │   │ Panel        │   │ Punto de     │       │
│   │ Público      │   │ Admin        │   │ Venta (POS)  │       │
│   │ (Vue 3 + TS) │   │ (Vue 3 + TS) │   │ (Vue 3 + TS) │       │
│   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘       │
└──────────┼──────────────────┼──────────────────┼───────────────┘
           │ HTTP / JSON      │                  │
           │ WebSocket        │                  │
           ▼                  ▼                  ▼
┌────────────────────────────────────────────────────────────────┐
│                BACKEND (Go · Monolito Modular)                 │
│                                                                │
│   HTTP Router (chi):  /health/*  /api/*  /api/admin/*  /ws/*   │
│         │                                                      │
│   Middleware:  CORS · Auth (JWT) · Logging                     │
│         │                                                      │
│   Handlers (handler/):                                         │
│     auth · google_auth · routes · reservation · payment        │
│     yape_payment · billing · admin · bus_layout · parcel       │
│     user · settings · ws · health                              │
│         │                                                      │
│   Services (service/):    Lógica de negocio                    │
│         │                                                      │
│   Repositories (repository/): SQL + transacciones + locks      │
│         │                                                      │
│   pgxpool (pkg/db/):  Pool de conexiones                       │
└────────────────┼───────────────────────────────────────────────┘
                 │ TCP/5432 (vía PgBouncer en prod)
                 ▼
┌────────────────────────────────────────────────────────────────┐
│                       PostgreSQL                               │
│  Tablas con UNIQUE constraints, índices, FOREIGN KEYS,         │
│  transacciones y SELECT FOR UPDATE.                            │
│  Fuente de verdad del estado de asientos y reservas.           │
└────────────────────────────────────────────────────────────────┘

           ┌────────────────────────────┐
           │ Integraciones externas     │
           │  • Google OAuth            │
           │  • Culqi (pasarela tarjeta)│
           │  • APIPeru / Venta (SUNAT) │
           │  • SMTP (correos)          │
           │  • Yape (voucher manual)   │
           └────────────────────────────┘
```

### Capas del backend

```
Handler  →  Service  →  Repository  →  PostgreSQL
(HTTP)      (negocio)   (SQL/pgx)      (verdad)
```

- **Handler**: recibe el request HTTP, valida entrada, delega al servicio, devuelve JSON.
- **Service**: orquesta lógica de negocio y combina repositorios.
- **Repository**: ejecuta queries SQL, abre transacciones y aplica locks.
- **Domain**: structs Go que representan las entidades del sistema.

---

## 4. Estructura del repositorio

```
pasaje/
├── README.md                  Este documento
├── proyecto.md                Diseño funcional, modelo de negocio, fases
├── .gitignore                 Bloquea .env, *.sql, keystores, node_modules, etc.
├── .env.example               Plantilla raíz de variables
│
├── backend/                   API Go (chi + PostgreSQL)
│   ├── cmd/server/            Punto de entrada (main.go)
│   ├── config/                .env.example y README de configuración
│   ├── internal/
│   │   ├── config/            Lectura de variables de entorno
│   │   ├── domain/            Entidades: route, trip, vehicle, reservation, parcel, user, ...
│   │   ├── handler/           HTTP handlers (auth, reservation, payment, admin, ws, ...)
│   │   ├── repository/        Acceso SQL con pgx, transacciones, locks
│   │   ├── service/           Lógica de negocio
│   │   └── ws/                Hub de WebSockets
│   ├── migrations/            18 migraciones SQL (up/down) versionadas
│   ├── pkg/db/                pgxpool (pool de conexiones)
│   └── scripts/               Scripts de arranque
│
├── frontend/                  SPA Vue 3 + Vite
│   └── src/
│       ├── api/               Clientes HTTP del backend
│       ├── components/        Componentes reutilizables (mapa de asientos, ...)
│       ├── composables/       Lógica reactiva compartida
│       ├── layouts/           Layouts de página
│       ├── router/            Vue Router (rutas SPA)
│       ├── stores/            Estado global (auth, ...)
│       └── views/             ConsultaView · CustomerLoginView · HomeView ·
│                              LoginView · ReservaView · TrackingView · ViajesView
│
├── docs/
│   ├── arquitectura.md        Diagrama y decisiones técnicas
│   └── api.md                 Referencia detallada de endpoints
│
└── tools/
    ├── capturar_evidencias_web/   Playwright (captura de pantallas para informes)
    └── generar_anexos_*.ps1       Generación de anexos del entregable
```

---

## 5. Modelo de datos

PostgreSQL con tablas agrupadas por dominio. Las **18 migraciones** documentan la evolución del schema:

| # | Migración | Tema |
|---|---|---|
| 001 | `init_schema` | Tablas base: agencies, sales_channels, users, routes, stops, vehicles, vehicle_seats, trip_templates, trip_instances |
| 002 | `trip_seat_inventory` | Inventario por viaje + `UNIQUE(trip_instance_id, vehicle_seat_id)` |
| 003 | `reservations_tickets` | reservations, reservation_items, tickets, payments |
| 004 | `audit_logs` | Auditoría con índices por entidad y fecha |
| 005 | `seed_minimal` | Datos semilla: ruta Arequipa-Cusco + bus + plantilla |
| 006 | `vehicle_floors_layout` | Layout de pisos del bus |
| 007 | `auth` | Autenticación |
| 008 | `customer_role` + `google_oauth` | Rol cliente + login Google |
| 009 | `yape_direct_vouchers` | Vouchers de Yape directo |
| 010 | `route_pricing` | Tarifas por ruta |
| 011 | `settings_table` | Configuración del sistema |
| 012 | `route_segments` | Segmentos intermedios con tarifas |
| 013 | `parcels` | Encomiendas |
| 014 | `payment_billing_state` | Estado de facturación por pago |
| 015 | `cleanup_orphan_reservation_items` | Limpieza de items huérfanos |
| 016 | `bus_layouts` + `pos_reservation_fields` | Layouts visuales + campos POS |
| 017 | `contact_name` + `trip_templates_departure_time_nullable` | Datos de contacto + plantilla flexible |
| 018 | `seed_bus_layouts` | Datos iniciales de layouts |

### Restricción crítica

```sql
UNIQUE (trip_instance_id, vehicle_seat_id)  -- en trip_seat_inventory
```

Garantiza, **a nivel de base de datos**, que un asiento no pueda registrarse dos veces para la misma salida.

---

## 6. Estados del sistema

### Estados de un asiento (`trip_seat_inventory.status`)

```
                    reservar (hold 10 min)
    ┌──────────┐   ─────────────────────►   ┌──────────┐
    │available │                              │  held    │
    └──────────┘   ◄───── expira / cancela ── └────┬─────┘
         ▲                                         │
         │                                         │ confirmar pago
         │                                         ▼
         │            anular ticket          ┌──────────┐
         └──────────────────────────────────│  sold    │
                                             └──────────┘
    ┌──────────┐
    │ blocked  │   (bloqueado por admin)
    └──────────┘
```

### Estados de una reserva (`reservations.status`)

```
                  pago confirmado
    ┌──────────┐  ───────────────►  ┌───────────┐
    │ pending  │                     │ confirmed │
    └────┬─────┘                     └───────────┘
         ├── hold expira ───►  expired
         └── cancela cliente ►  cancelled
```

---

## 7. Flujo transaccional de reserva (núcleo del sistema)

### Hold (crear reserva)

```
POST /api/reservations { trip_instance_id, seat_ids[] }

  BEGIN TRANSACTION
  SELECT id, status FROM trip_seat_inventory
    WHERE id = ANY($seat_ids) AND trip_instance_id = $trip_id
    FOR UPDATE                           ← bloquea filas
  Verifica que TODOS estén 'available'   ← si no → ROLLBACK 409
  UPDATE trip_seat_inventory
    SET status='held', hold_expires_at = now() + interval '10 minutes'
  INSERT INTO reservations  (status='pending', expires_at=...)
  INSERT INTO reservation_items (uno por asiento)
  COMMIT
```

### Confirmación (pago)

```
POST /api/reservations/{code}/confirm { payment_method, payment_reference }

  BEGIN TRANSACTION
  SELECT * FROM reservations WHERE code = $code FOR UPDATE
  Verifica: status='pending' AND expires_at > now()
  UPDATE trip_seat_inventory  SET status='sold' WHERE id IN (...)
  UPDATE reservations          SET status='confirmed'
  INSERT INTO tickets   (uno por asiento)
  INSERT INTO payments  (uno por ticket)
  COMMIT
```

### Concurrencia: tres mecanismos complementarios

1. **`UNIQUE` constraint** — defensa pasiva en la base.
2. **`SELECT ... FOR UPDATE`** — defensa activa: bloquea filas durante la transacción.
3. **Transacciones** — atomicidad: todo o nada.

> **Regla principal:** un asiento NO puede venderse dos veces para la misma salida.

---

## 8. Endpoints HTTP

Base URL local: `http://localhost:8085`

### Health

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/health/live` | Liveness |
| GET | `/health/ready` | Readiness (incluye DB) |

### Público

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/routes` | Rutas activas |
| GET | `/api/routes/{id}/trips?from=YYYY-MM-DD` | Viajes de la ruta desde fecha |
| GET | `/api/trips/{id}/seats` | Asientos del viaje con su estado |
| POST | `/api/reservations` | Crear reserva (hold 10 min) |
| POST | `/api/reservations/{code}/confirm` | Confirmar con pago |
| GET | `/api/reservations/{code}` | Consultar reserva por código |
| POST | `/api/payments/yape/voucher` | Subir voucher de Yape |
| POST | `/api/parcels` | Registrar encomienda |
| GET | `/api/parcels/{code}` | Consultar encomienda |

### Autenticación

| Método | Ruta | Descripción |
|---|---|---|
| POST | `/api/auth/login` | Login interno con usuario/clave (devuelve JWT) |
| POST | `/api/auth/google` | Login con Google OAuth |
| POST | `/api/auth/customer/login` | Login del portal de clientes |
| POST | `/api/auth/refresh` | Refrescar token JWT |

### Admin (requiere JWT)

| Método | Ruta | Descripción |
|---|---|---|
| `*` | `/api/admin/routes` | CRUD rutas (con soft delete) |
| `*` | `/api/admin/vehicles` | CRUD vehículos |
| `*` | `/api/admin/vehicles/{id}/seats` | Asientos plantilla del vehículo |
| `*` | `/api/admin/bus-layouts` | Layouts visuales del bus |
| `*` | `/api/admin/trip-templates` | Plantillas de viaje |
| `*` | `/api/admin/trip-instances` | Instancias de viaje (genera inventario) |
| GET | `/api/admin/stats` | Dashboard: rutas, vehículos, viajes, reservas |
| `*` | `/api/admin/parcels` | Encomiendas |
| `*` | `/api/admin/yape-vouchers` | Validar/rechazar vouchers de Yape |
| `*` | `/api/admin/users` | Usuarios del sistema |
| `*` | `/api/admin/settings` | Configuración (mail, vouchers, etc.) |
| `*` | `/api/admin/billing` | Facturación electrónica (consultar / reintentar) |

### WebSocket

| Ruta | Descripción |
|---|---|
| `/ws/trips/{id}/seats` | Actualizaciones en tiempo real del estado de asientos |

> El detalle completo (request/response, códigos de error) está en `docs/api.md`.

---

## 9. Frontend — vistas principales

| Vista | Pantalla |
|---|---|
| `HomeView` | Página inicial pública |
| `ViajesView` | Búsqueda de viajes por ruta y fecha |
| `ReservaView` | Selección de asientos + datos del pasajero + pago |
| `ConsultaView` | Consulta de reserva/ticket por código |
| `TrackingView` | Seguimiento de viaje en curso |
| `LoginView` | Login de operadores administrativos |
| `CustomerLoginView` | Login de clientes (con Google) |

Estado global con `stores/auth.ts` (token JWT, datos de usuario).
WebSocket cliente para refrescar el mapa de asientos cuando otro punto de venta reserva.

---

## 10. Cómo ejecutar localmente

### Pre-requisitos

- Go 1.21+
- Node.js 20.19+ o 22.12+
- PostgreSQL 14+
- (Opcional) PgBouncer

### 1. Base de datos

Crear la base `pasaje` y aplicar las migraciones de `backend/migrations/` en orden ascendente. Cualquier herramienta sirve (golang-migrate, manualmente con `psql`, etc.).

### 2. Backend

```bash
cd backend
cp config/.env.example config/.env       # editar con credenciales reales
./scripts/run.sh                          # o: go run ./cmd/server/
```

API expuesta en `http://localhost:8085`.

### 3. Frontend

```bash
cd frontend
cp .env.example .env                      # ajustar VITE_API_URL si hace falta
npm install
npm run dev
```

UI en `http://localhost:5173`.

### 4. Verificar

```bash
curl http://localhost:8085/health/ready
# → {"status":"ok"}
```

---

## 11. Variables de entorno

Plantillas en `.env.example` (raíz), `backend/config/.env.example` y `frontend/.env.example`. **No commitear los `.env` reales** — el `.gitignore` ya los bloquea.

### Backend (`backend/config/.env`)

```ini
# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=change_me
DB_NAME=pasaje
DB_SSLMODE=disable

# API
API_PORT=8085

# Seguridad
JWT_SECRET=change_me_long_random_secret
DEFAULT_ADMIN_PASSWORD=change_me_admin_password

# Google OAuth
GOOGLE_CLIENT_ID=...
GOOGLE_MOBILE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=change_me

# Facturación (Venta + APIPeru)
BILLING_GO_SERVICE_URL=https://goventa.example.com
BILLING_TENANT_SLUG=ancalla
BILLING_APIPERU_URL=https://apiperu.example.com
BILLING_APIPERU_TOKEN=change_me
BILLING_DB_HOST=localhost
BILLING_DB_PORT=5432
BILLING_DB_USER=postgres
BILLING_DB_PASSWORD=change_me
BILLING_TENANT_EMAIL=admin@example.com
BILLING_TENANT_PASS=change_me

# Pagos Culqi
CULQI_PUBLIC_KEY=pk_test_...
CULQI_PRIVATE_KEY=sk_test_...

# Correo SMTP
MAIL_HOST=smtp.gmail.com
MAIL_PORT=465
MAIL_USERNAME=notificaciones@example.com
MAIL_PASSWORD=change_me_app_password

# Yape directo / vouchers
VOUCHER_STORAGE_PATH=/data/vouchers
VOUCHER_MAX_SIZE_MB=10
YAPE_HOLD_MINUTES=30
YAPE_BUSINESS_NUMBER=999888777
WHATSAPP_NOTIFY_PHONE=51999888777
```

### Frontend (`frontend/.env`)

```ini
VITE_API_URL=http://localhost:8085
VITE_WS_URL=ws://localhost:8085
VITE_GOOGLE_CLIENT_ID=...
```

---

## 12. Decisiones técnicas

1. **PostgreSQL como fuente de verdad** — la disponibilidad de asientos se resuelve siempre en la base, nunca en cache ni en el frontend.
2. **Monolito modular por capas** — handler / service / repository, sin microservicios prematuros.
3. **`chi` como router** — ligero, compatible con `net/http`, ergonomía RESTful.
4. **`pgx` / `pgxpool`** — soporte nativo de tipos PostgreSQL, pool integrado, alto rendimiento.
5. **Códigos legibles** — reservas (`A3K9M2PX`) y tickets (`TKT-B7X4R9WQ`) generados con `crypto/rand`.
6. **Soft delete** — rutas y vehículos se desactivan (`active=false`), preservando integridad referencial.
7. **Hold con expiración** — `hold_expires_at` libera asientos automáticamente si el cliente no paga.
8. **WebSockets sólo como reflejo** — la verdad sigue siendo la base; el WS notifica cambios para refrescar la UI.
9. **JWT corto + Google OAuth** — autenticación simple para operadores, OAuth para clientes finales.
10. **Vouchers Yape con validación humana** — captura de imagen + workflow de aprobación, evita falsos positivos.

---

## 13. Documentación complementaria

| Documento | Contenido |
|---|---|
| `proyecto.md` | Diseño funcional, modelo de negocio, fases del proyecto |
| `docs/arquitectura.md` | Arquitectura detallada, flujos transaccionales, decisiones |
| `docs/api.md` | Referencia completa de endpoints (request/response/errores) |
| `backend/README.md` | Estructura del backend, dominio, migraciones |
| `frontend/README.md` | Setup de Vue 3 + Vite |
| `tools/capturar_evidencias_web/` | Scripts Playwright para capturar evidencias |

---

## 14. Repositorio

- **GitHub**: https://github.com/Franciscoleochambiv/pasaje_pa2
- **Rama principal**: `main`

---

*Trabajo presentado por el **Equipo 05** — Curso de Ingeniería Web — Mayo de 2026.*
