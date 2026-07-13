# Flujo de dos pasos: reservar y cobrar paquetes

**Fecha:** 2026-07-12
**Repos afectados:** `AtrevidaBack` (Go) + `AtrevidaFront` (Next.js)

## Contexto y problema

Hoy un plan (paquete adquirido por un cliente) se crea **únicamente desde caja**:
`crearPlan(combo_id, cliente_id, local_id, pago_codigo)` crea el plan, adjunta el pago
y luego —en una **segunda llamada** `cambiarEstadoPlan(id, 'ACTIVO')`— lo activa. Todo plan
nace en estado `BORRADOR` y caja lo activa acto seguido.

Dos problemas:

1. **No existe "reservar sin cobrar".** Recepción no puede apartar un paquete para un
   cliente que pagará después.
2. **BORRADOR colgado.** Si la segunda llamada (`cambiarEstadoPlan`) falla tras un pago
   exitoso, el plan queda en `BORRADOR` pagado-pero-inactivo. El nombre "BORRADOR" además
   confunde: sugiere "a medio armar" cuando en realidad está completo.

## Objetivo

Habilitar un flujo de **dos pasos** — **reservar** ahora (sin pago) y **cobrar** después —
sin crear planes duplicados, y renombrar el estado pre-pago a algo que signifique
"reservado, esperando cobro".

## Decisiones (cerradas en brainstorming)

- **Caso de uso:** cliente paga después. Recepción reserva; el cobro llega luego.
- **Estado pre-pago:** renombrar `BORRADOR` → **`RESERVADO`** (label UI "Reservado").
  Un solo estado pre-activo; no hay "draft a medio armar" aparte.
- **Reservar** desde un botón en la página de paquetes; el paquete sale **solo de un combo**
  del catálogo (no composición manual, por ahora).
- **Cobrar** una reserva desde **dos** entradas: la fila del plan y caja.
- **Endpoint dedicado** para cobrar (enfoque A), separado de crear/estado.

## Alcance explícito

- **NO se toca `pagos.estado`** (tabla `pagos`), que tiene su propio valor `'BORRADOR'`
  con otro significado. El rename es exclusivo de `planes.estado`.
- Reservar es **solo desde combo**. Composición manual queda fuera de este spec.
- El pago es de contado (`tipo_pago = 'UNICO'`). Cuotas/planes en cuotas fuera de alcance.

## Diseño

### 1. Estados — rename `BORRADOR` → `RESERVADO`

**Backend (`services/planes_service.go`)**
- Constante `EstadoPlanReservado = "RESERVADO"` (reemplaza `EstadoPlanBorrador`).
- `CrearPlan`: default `EstadoPlanReservado`.
- `CambiarEstado`: sin cambios de reglas más allá del rename.

**Backend (`repositories/pgsql/planes_repo.go`)**
- `esTransicionValida`: caso `"RESERVADO"` → permite `ACTIVO` o `CANCELADO`
  (idéntico a lo que hoy permite `BORRADOR`).

**Migración `000043_rename_estado_borrador_a_reservado`**
```sql
-- up
ALTER TABLE planes ALTER COLUMN estado DROP DEFAULT;
ALTER TABLE planes DROP CONSTRAINT IF EXISTS chk_planes_estado;
UPDATE planes SET estado = 'RESERVADO' WHERE estado = 'BORRADOR';
ALTER TABLE planes ADD CONSTRAINT chk_planes_estado
    CHECK (estado IN ('RESERVADO','ACTIVO','COMPLETADO','VENCIDO','CANCELADO'));
ALTER TABLE planes ALTER COLUMN estado SET DEFAULT 'RESERVADO';
COMMENT ON COLUMN planes.estado IS
    'Estado contractual: RESERVADO, ACTIVO, COMPLETADO, VENCIDO, CANCELADO.';
-- down: inverso (RESERVADO -> BORRADOR, constraint y default originales).
```
La migración es idempotente en la práctica: `UPDATE ... WHERE estado='BORRADOR'` es no-op
si ya no quedan BORRADOR.

### 2. Reservar (paso 1)

**Frontend** — página de paquetes: botón **"Reservar paquete"** abre un modal:
- Campos: cliente (buscador existente), local, combo del catálogo.
- Al confirmar: `crearPlan({ combo_id, cliente_id, local_id, tipo_pago: 'UNICO' })`
  **sin** `pago_codigo` y **sin** llamar a `cambiarEstadoPlan`.
- Resultado: plan nace `RESERVADO`, aparece en la lista.

**Backend** — ningún cambio adicional: `crearPlan` sin `pago_codigo` ya deja el plan en el
estado default (ahora `RESERVADO`) con sus cuotas `PENDIENTE`.

### 3. Cobrar (paso 2) — endpoint dedicado

**Backend — nuevo endpoint** `POST /bd/planes/{id}/cobrar`
- Body: `{ "pago_codigo": "PAGO-000123" }`
- Auth: `GerenciaRequired` (igual que crear/activar planes).
- Servicio `CobrarPlan(planID, pagoCodigo)`:
  1. En una transacción, verifica que el plan exista y esté en `RESERVADO`
     (si no → error de transición/estado; evita doble cobro).
  2. Adjunta el pago a la cuota del plan reutilizando la lógica de `aplicarPagoUnicoTx`.
  3. Marca `estado_cobranza = 'PAGADO'` (derivado de cuotas) y `estado = 'ACTIVO'`.
- Swagger: anotaciones nuevas + regenerar `docs/`.

**Frontend — dos entradas (ambas llaman al mismo endpoint):**
- **Fila "Cobrar"** en un plan `RESERVADO` (menú de acciones): abre un modal de pago
  (tipo `qr`/`efectivo`, monto = `precio_total`) → `crearPagoDB(...)` para obtener
  `codigo_pago` → `cobrarPlan(id, codigo_pago)`.
- **Caja, modo "cobrar reserva"**: buscar cliente → listar sus planes `RESERVADO` →
  elegir uno → mismo `crearPagoDB` + `cobrarPlan`.
- Nueva función en `lib/api/planes.ts`: `cobrarPlan(id, pago_codigo)`.

### 4. Venta directa atómica (robustez)

`crearPlan` cuando **viene** `pago_codigo` (venta directa walk-in): crear el plan
**ya en `ACTIVO`** dentro de la misma transacción donde se aplica el pago
(`services.CrearPlan` / `repo.CreatePlan`). Caja **deja de** llamar a `cambiarEstadoPlan`
tras el pago. Elimina la ventana de fallo que dejaba planes colgados.

### 5. Página: rename + filtro/orden

**Frontend** (`app/atrevida-gestion/paquetes-activos/`)
- Título "Paquetes Activos" → **"Paquetes de clientes"**. (Ruta puede quedar igual para no
  romper enlaces; solo cambia el texto visible y el `PageHeader`.)
- Filtro de estado: **todos los estados** (Reservado, Activo, Completado, Cancelado, Todos),
  **por defecto "Todos"**.
- Orden de la lista: primero `RESERVADO` y `ACTIVO`, luego el resto; desempate por
  fecha de creación/modificación descendente.

### 6. Duplicados, guards y errores

- **Sin duplicados:** reservar crea 1 plan; cobrar actúa sobre ese `id` exacto. La venta
  directa (walk-in sin reserva previa) sigue creando un plan nuevo — es correcto.
- **Guard anti doble-cobro:** `cobrar` solo procede si el plan está `RESERVADO`; rechaza
  `ACTIVO`/`COMPLETADO`/`CANCELADO`.
- **Fallo parcial en cobro:** el front crea el pago primero; si `cobrar` falla, el plan
  queda `RESERVADO` y el cajero reintenta (idempotente por el guard de estado). El pago
  ya creado se reaprovecha en el reintento.

## Componentes tocados

| Repo | Archivo | Cambio |
|------|---------|--------|
| Back | `services/planes_service.go` | const RESERVADO, default, nuevo `CobrarPlan`, venta directa ACTIVA |
| Back | `repositories/pgsql/planes_repo.go` | `esTransicionValida`, `CreatePlan` (estado según pago), nuevo método cobrar/reuso `aplicarPagoUnicoTx` |
| Back | `handlers/planes_handler.go` | handler `CobrarPlan` + swagger |
| Back | `router/router.go` | ruta `POST /bd/planes/:id/cobrar` (`GerenciaRequired`) |
| Back | `migrations/000043_*` | rename estado + constraint + default |
| Front | `lib/api/planes.ts` | `cobrarPlan(id, pago_codigo)` |
| Front | `app/atrevida-gestion/paquetes-activos/page.tsx` | rename página, botón reservar, acción cobrar, filtro/orden |
| Front | modal reservar + modal/flujo cobrar | nuevos componentes UI |
| Front | `app/atrevida-gestion/caja/page.tsx` | modo "cobrar reserva"; quitar 2da llamada en venta directa |

## Verificación

- **Migración:** correr `cmd/migrate/main.go` en DB fresca y en DB con BORRADOR existentes;
  verificar que todos quedan `RESERVADO`, constraint acepta RESERVADO, rechaza BORRADOR.
- **Reservar:** crear reserva desde la página → plan `RESERVADO`, sin pago, con cuota PENDIENTE.
- **Cobrar (fila y caja):** cobrar la reserva → `estado=ACTIVO`, `estado_cobranza=PAGADO`,
  pago adjunto; reintentar cobrar sobre un ACTIVO → rechazado.
- **Venta directa:** vender en caja con pago → plan nace `ACTIVO` en una sola operación
  (sin BORRADOR intermedio).
- **No duplicados:** reservar + cobrar produce exactamente 1 plan.
- `go build ./...`, `go generate ./...` (swagger en sync), `npm run lint`.
