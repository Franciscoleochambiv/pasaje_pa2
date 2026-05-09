# Pasaje

**Examen Final** — Sistema web de reserva y venta de pasajes interprovinciales.

Empresa de transporte que opera la ruta **Arequipa – Cusco**. El sistema centraliza el inventario de asientos para evitar la **doble venta** en múltiples canales (web, ventanilla, agencias).

## Stack

| Capa | Tecnología |
|------|-----------|
| Frontend | Vue 3 + TypeScript + Vite, Vue Router, SweetAlert2 |
| Backend | Go (chi router, pgx/pgxpool) |
| Base de datos | PostgreSQL (con PgBouncer en producción) |
| Tiempo real | WebSockets |
| Autenticación | JWT + Google OAuth |
| Pagos | Culqi (tarjeta) + Yape directo (voucher) |
| Facturación | Integración con sistema Venta (APIPeru) |

## Estructura del repositorio

```
pasaje/
├── backend/            API Go (chi + PostgreSQL)
│   ├── cmd/server/     Punto de entrada
│   ├── internal/       config, domain, handler, repository, service, ws
│   ├── migrations/     18 migraciones SQL (up/down)
│   ├── pkg/db/         Pool de conexiones pgx
│   └── scripts/        Scripts de arranque
├── frontend/           SPA Vue 3 + Vite
│   └── src/            api, components, composables, layouts, router, stores, views
├── docs/               Documentación funcional y técnica
├── tools/              Utilidades (capturar evidencias con Playwright, generar anexos)
├── proyecto.md         Documento de diseño (modelo de negocio, fases)
└── .env.example        Plantilla de variables de entorno
```

## Módulos funcionales

1. **Portal público** – Búsqueda de viajes, mapa de asientos en tiempo real, reserva con hold de 10 min, confirmación con pago, consulta por código.
2. **Panel administrativo** – CRUD de rutas, vehículos, layouts de bus, plantillas y instancias de viaje, dashboard con estadísticas.
3. **Motor de reservas** – Hold transaccional con `SELECT FOR UPDATE`, confirmación, expiración automática, restricción `UNIQUE(trip_instance_id, vehicle_seat_id)` en `trip_seat_inventory`.
4. **Pagos Yape directo** – Subida de voucher por el cliente, validación manual o automática por el operador.
5. **Encomiendas (parcels)** – Envío de paquetes asociados a viajes.
6. **Segmentación de rutas** – Tarifas por segmentos intermedios (`route_segments`).
7. **Auditoría** – Tabla `audit_logs` con cambios críticos.

## Cómo ejecutar local

### 1. Base de datos

PostgreSQL local en `localhost:5432`, base `pasaje`. Aplicar migraciones de `backend/migrations/` en orden.

### 2. Backend

```bash
cd backend
cp config/.env.example config/.env   # editar con credenciales reales
./scripts/run.sh                      # o: go run ./cmd/server/
```

API disponible en `http://localhost:8085`.

### 3. Frontend

```bash
cd frontend
cp .env.example .env                  # ajustar VITE_API_URL si hace falta
npm install
npm run dev
```

UI en `http://localhost:5173`.

## Variables de entorno

Ver `.env.example` (raíz), `backend/config/.env.example` y `frontend/.env.example`. **No commitear los `.env` reales** — el `.gitignore` ya los bloquea.

Variables clave del backend:

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `API_PORT` (default `8085`)
- `JWT_SECRET`, `DEFAULT_ADMIN_PASSWORD`
- `GOOGLE_CLIENT_ID`, `GOOGLE_MOBILE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
- `BILLING_GO_SERVICE_URL`, `BILLING_APIPERU_TOKEN`, `BILLING_TENANT_*`
- `CULQI_PUBLIC_KEY`, `CULQI_PRIVATE_KEY`
- `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD`
- `VOUCHER_STORAGE_PATH`, `YAPE_HOLD_MINUTES`, `YAPE_BUSINESS_NUMBER`

## Endpoints principales

### Público
- `GET /api/routes` – Rutas activas
- `GET /api/routes/{id}/trips?from=YYYY-MM-DD` – Viajes de una ruta
- `GET /api/trips/{id}/seats` – Estado de asientos
- `POST /api/reservations` – Crear reserva (hold 10 min)
- `POST /api/reservations/{code}/confirm` – Confirmar con pago
- `GET /api/reservations/{code}` – Consultar por código

### Admin
- `/api/admin/routes`, `/api/admin/vehicles`, `/api/admin/trip-templates`, `/api/admin/trip-instances`
- `/api/admin/stats` – Dashboard
- `/api/admin/parcels` – Encomiendas
- `/api/admin/yape-vouchers` – Validación de vouchers

### Health
- `GET /health/live` – Liveness
- `GET /health/ready` – Readiness (incluye DB)

## Migraciones (18)

Las migraciones cubren la evolución del sistema:

| # | Tema |
|---|------|
| 001 | Schema inicial (rutas, vehículos, plantillas, instancias) |
| 002 | Inventario de asientos por viaje |
| 003 | Reservas, tickets, pagos |
| 004 | Logs de auditoría |
| 005 | Datos semilla mínimos |
| 006 | Layout de pisos del vehículo |
| 007 | Autenticación |
| 008 | Rol customer + Google OAuth |
| 009 | Vouchers de Yape directo |
| 010 | Tarifas por ruta |
| 011 | Tabla de configuración |
| 012 | Segmentos de ruta |
| 013 | Encomiendas (parcels) |
| 014 | Estado de facturación de pagos |
| 015 | Limpieza de items huérfanos |
| 016 | Bus layouts + campos POS |
| 017 | Contact name + departure_time nullable |
| 018 | Seed de bus layouts |

## Regla principal de concurrencia

> Un asiento NO puede venderse dos veces para la misma salida.

Garantizada por:
1. `UNIQUE(trip_instance_id, vehicle_seat_id)` en `trip_seat_inventory`
2. `SELECT ... FOR UPDATE` durante el hold/confirm
3. Transacciones en toda operación crítica (hold, confirm, expire)

## Documentación adicional

- `proyecto.md` – Diseño funcional, modelo de datos, fases
- `backend/README.md` – Detalle del backend (estructura, endpoints, dominio)
- `frontend/README.md` – Setup de Vue 3 + Vite
- `docs/` – Documentación de soporte
