# Pasaje - Sistema de Reserva y Venta de Pasajes Interprovinciales

## Descripcion del negocio

Pasaje es un sistema web de reserva y venta de pasajes para una empresa de transporte interprovincial que actualmente opera la ruta Arequipa - Cusco. El sistema esta disenado para crecer a nuevas rutas, ciudades, agencias y canales de venta.

El problema principal que resuelve es la **doble venta del mismo asiento**, causada por la falta de sincronizacion entre multiples puntos de venta (fisicos y online). Pasaje centraliza el inventario de asientos y garantiza consistencia transaccional mediante PostgreSQL.

## Stack tecnologico

| Capa | Tecnologia |
|------|-----------|
| Frontend | Vue 3 + TypeScript |
| Backend | Go (chi router, pgx) |
| Base de datos | PostgreSQL |
| Tiempo real | WebSockets (planificado) |
| Mensajeria asincrona | RabbitMQ (planificado, solo procesos no criticos) |

## Arquitectura

**Monolito modular**, listo para escalar sin microservicios prematuros.

- La base de datos es la fuente de verdad.
- La consistencia de asientos se resuelve con transacciones, locks de fila (`SELECT FOR UPDATE`) y restricciones unicas en PostgreSQL.
- WebSockets reflejaran cambios en tiempo real, pero no son la fuente de verdad.
- RabbitMQ se usara unicamente para tareas asincronas desacopladas (notificaciones, reportes, auditoria).

## Modulos del sistema

### 1. Portal Publico

Interfaz para el pasajero final:

- Busqueda de viajes por ruta y fecha
- Visualizacion del mapa de asientos con estados en tiempo real
- Seleccion de asientos
- Registro de datos del pasajero
- Creacion de reserva (hold temporal de 10 minutos)
- Confirmacion con pago
- Consulta de reserva por codigo

### 2. Panel Administrativo

Interfaz para operadores de la empresa:

- Gestion de rutas (CRUD con soft delete)
- Gestion de vehiculos y sus asientos (CRUD)
- Gestion de plantillas de viaje (ruta + vehiculo + horario base)
- Generacion de instancias de viaje (fecha concreta a partir de plantilla)
- Dashboard con estadisticas (rutas activas, vehiculos, viajes proximos, reservas del dia)

### 3. Motor de Reservas

Logica transaccional critica:

- Retencion temporal de asientos (`held`) con expiracion automatica (`hold_expires_at`)
- Confirmacion de reserva: convierte `held` a `sold`, crea tickets y registra pago
- Expiracion automatica de holds vencidos (proceso programado, pendiente)
- Consulta de reserva por codigo

### 4. Auditoria

- Tabla `audit_logs` con registro de cambios criticos
- Campos: entidad, ID de entidad, accion, actor, datos anteriores/nuevos (JSONB), timestamp

## Modelo de datos

El sistema usa 15 tablas en PostgreSQL:

| # | Tabla | Proposito |
|---|-------|-----------|
| 1 | `agencies` | Agencias de venta |
| 2 | `sales_channels` | Canales de venta (web, ventanilla, etc.) |
| 3 | `users` | Operadores/usuarios del sistema |
| 4 | `routes` | Rutas (ej. Arequipa - Cusco) |
| 5 | `stops` | Paradas de cada ruta |
| 6 | `vehicles` | Vehiculos/buses |
| 7 | `vehicle_seats` | Asientos plantilla de cada vehiculo |
| 8 | `trip_templates` | Plantillas de viaje (ruta + vehiculo + horario) |
| 9 | `trip_instances` | Instancias de viaje (fecha concreta) |
| 10 | `trip_seat_inventory` | Inventario de asientos por viaje (fuente de verdad) |
| 11 | `reservations` | Reservas de pasajeros |
| 12 | `reservation_items` | Asientos incluidos en cada reserva |
| 13 | `tickets` | Boletos emitidos tras confirmar reserva |
| 14 | `payments` | Pagos asociados a tickets |
| 15 | `audit_logs` | Logs de auditoria |

### Restriccion critica

```
UNIQUE(trip_instance_id, vehicle_seat_id)  -- en trip_seat_inventory
```

Esta restriccion, combinada con `SELECT ... FOR UPDATE` y transacciones, garantiza que **un asiento NO puede venderse dos veces** para la misma salida.

## Flujo de reserva (pasajero)

```
1. Buscar viajes       GET /api/routes → GET /api/routes/{id}/trips
2. Ver asientos        GET /api/trips/{id}/seats
3. Seleccionar asientos (frontend: mapa de asientos interactivo)
4. Datos del pasajero  (formulario frontend)
5. Reservar            POST /api/reservations
   → Backend: BEGIN TX → SELECT FOR UPDATE → verificar available
   → UPDATE status='held', hold_expires_at=now()+10min
   → INSERT reservations + reservation_items → COMMIT
6. Pagar               POST /api/reservations/{code}/confirm
   → Backend: BEGIN TX → SELECT reservation FOR UPDATE → verificar pending + no expirada
   → UPDATE seats status='sold' → UPDATE reservation status='confirmed'
   → INSERT tickets + payments → COMMIT
7. Ticket emitido      Codigo TKT-XXXXXXXX
```

## Flujo administrativo

```
1. Crear ruta          POST /api/admin/routes
2. Crear vehiculo      POST /api/admin/vehicles
3. Crear plantilla     POST /api/admin/trip-templates (ruta + vehiculo + horario)
4. Generar instancia   POST /api/admin/trip-instances (plantilla + fecha)
   → Backend: BEGIN TX → INSERT trip_instance
   → INSERT trip_seat_inventory (un registro por cada vehicle_seat)
   → COMMIT
5. Inventario listo    Los asientos quedan en estado 'available'
```

## Regla principal de concurrencia

> **Un asiento NO puede venderse dos veces para la misma salida.**

Se garantiza mediante tres mecanismos complementarios:

1. **UNIQUE constraint** en `trip_seat_inventory(trip_instance_id, vehicle_seat_id)`: impide duplicidad a nivel de base de datos.
2. **SELECT ... FOR UPDATE**: bloquea las filas de asientos durante la transaccion de reserva/confirmacion.
3. **Transacciones**: toda operacion critica (hold, confirm, expire) se ejecuta dentro de una transaccion.

## Plan de fases

### Fase 1 - Fundacion (completada)

- Schema de base de datos (15 tablas, restricciones, indices)
- API de lectura publica (rutas, viajes, asientos)
- Frontend basico con Vue 3 + TypeScript
- Health checks

### Fase 2 - Funcionalidad core (completada)

- CRUD administrativo (rutas, vehiculos, plantillas, instancias)
- Motor de reservas (hold + confirmacion con pago)
- Portal publico con seleccion de asientos interactiva
- Dashboard administrativo con estadisticas
- Datos semilla para desarrollo

### Fase 3 - Tiempo real y seguridad (pendiente)

- WebSockets para actualizacion en tiempo real del mapa de asientos
- Proceso automatico de expiracion de holds vencidos
- Autenticacion y autorizacion de usuarios
- Roles (admin, operador, pasajero)

### Fase 4 - Pagos y comunicaciones (pendiente)

- Integracion con pasarela de pagos (Mercado Pago / Niubiz)
- Notificaciones por email y SMS
- Reportes de ventas y ocupacion
- Emision de comprobantes electronicos
