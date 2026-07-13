# Flujo Reservar/Cobrar Paquetes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Habilitar reservar un paquete sin pago (estado `RESERVADO`) y cobrarlo después vía endpoint dedicado, sin duplicar planes.

**Architecture:** Backend Go/Gin/Postgres: rename del estado `BORRADOR→RESERVADO`, endpoint `POST /bd/planes/{id}/cobrar` que adjunta un pago existente a un plan reservado y lo activa (reusa `aplicarPagoUnicoTx`), y venta directa atómica en `CrearPlan`. Frontend Next.js: función `cobrarPlan`, modal de reserva y acción/modal de cobro en la página de paquetes, más modo "cobrar reserva" en caja.

**Tech Stack:** Go 1.26, Gin, sqlx, golang-migrate, swaggo. Next.js 16, React 19, TypeScript.

## Global Constraints

- Backend: cualquier cambio de endpoint exige anotaciones swaggo + `go generate ./...` (hooks lo bloquean si `docs/` queda out-of-sync). Copiado verbatim del CLAUDE.md.
- El rename es **solo** `planes.estado`. NO tocar `pagos.estado` (valor `'BORRADOR'` propio, otro concepto).
- Auth de mutaciones de planes: `GerenciaRequired`.
- JSON de respuesta serializa modelos con tags `*_texto` (contrato ya vigente); JSON de request de servicios manuales sigue en `*_snapshot`.
- Frontend no tiene framework de test: el gate es `npm run lint` (0 errores). Verificación funcional = build + prueba manual descrita.
- Pago de contado: `tipo_pago = 'UNICO'`.
- Migraciones corren por `go run cmd/migrate/main.go` (separado del app).
- DB local de prueba: usar `atrevida_migtest` (descartable) para probar cadenas de migración; nunca `atrevida_db` para cosas destructivas.

---

### Task 1: Migración 000043 — rename estado `BORRADOR → RESERVADO`

**Files:**
- Create: `migrations/000043_rename_estado_borrador_a_reservado.up.sql`
- Create: `migrations/000043_rename_estado_borrador_a_reservado.down.sql`

**Interfaces:**
- Produces: columna `planes.estado` con valores `{RESERVADO,ACTIVO,COMPLETADO,VENCIDO,CANCELADO}`, default `RESERVADO`, constraint `chk_planes_estado` actualizado.

- [ ] **Step 1: Escribir la migración up**

`migrations/000043_rename_estado_borrador_a_reservado.up.sql`:
```sql
-- Renombra el estado pre-pago BORRADOR -> RESERVADO (reservado, esperando cobro).
-- Solo afecta planes.estado; pagos.estado tiene su propio 'BORRADOR' sin relacion.
ALTER TABLE planes ALTER COLUMN estado DROP DEFAULT;
ALTER TABLE planes DROP CONSTRAINT IF EXISTS chk_planes_estado;
UPDATE planes SET estado = 'RESERVADO' WHERE estado = 'BORRADOR';
ALTER TABLE planes ADD CONSTRAINT chk_planes_estado
    CHECK (estado IN ('RESERVADO', 'ACTIVO', 'COMPLETADO', 'VENCIDO', 'CANCELADO'));
ALTER TABLE planes ALTER COLUMN estado SET DEFAULT 'RESERVADO';
COMMENT ON COLUMN planes.estado IS
    'Estado contractual: RESERVADO, ACTIVO, COMPLETADO, VENCIDO, CANCELADO.';
```

- [ ] **Step 2: Escribir la migración down**

`migrations/000043_rename_estado_borrador_a_reservado.down.sql`:
```sql
ALTER TABLE planes ALTER COLUMN estado DROP DEFAULT;
ALTER TABLE planes DROP CONSTRAINT IF EXISTS chk_planes_estado;
UPDATE planes SET estado = 'BORRADOR' WHERE estado = 'RESERVADO';
ALTER TABLE planes ADD CONSTRAINT chk_planes_estado
    CHECK (estado IN ('BORRADOR', 'ACTIVO', 'COMPLETADO', 'VENCIDO', 'CANCELADO'));
ALTER TABLE planes ALTER COLUMN estado SET DEFAULT 'BORRADOR';
COMMENT ON COLUMN planes.estado IS
    'Estado contractual: BORRADOR, ACTIVO, COMPLETADO, VENCIDO, CANCELADO.';
```

- [ ] **Step 3: Probar la cadena completa en DB fresca descartable**

```bash
cd AtrevidaBack
export PGPASSWORD=$(grep -E '^DB_PASSWORD=' .env | cut -d= -f2- | tr -d '\r"'"'"' ')
psql -h localhost -p 5432 -U postgres -d postgres -tAc "DROP DATABASE IF EXISTS atrevida_migtest; CREATE DATABASE atrevida_migtest;"
DB_NAME=atrevida_migtest go run cmd/migrate/main.go
```
Expected: `versión actual: 43 (dirty: false)`.

- [ ] **Step 4: Verificar esquema resultante**

```bash
psql -h localhost -p 5432 -U postgres -d atrevida_migtest -tAc "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname='chk_planes_estado';"
psql -h localhost -p 5432 -U postgres -d atrevida_migtest -tAc "SELECT column_default FROM information_schema.columns WHERE table_name='planes' AND column_name='estado';"
psql -h localhost -p 5432 -U postgres -d postgres -tAc "DROP DATABASE atrevida_migtest;"
```
Expected: constraint incluye `RESERVADO` y NO `BORRADOR`; default `'RESERVADO'::character varying`.

- [ ] **Step 5: Aplicar a la DB local real (tiene datos)**

```bash
go run cmd/migrate/main.go
psql -h localhost -p 5432 -U postgres -d atrevida_db -tAc "SELECT estado, COUNT(*) FROM planes GROUP BY estado;"
```
Expected: `versión 43`; ningún plan con estado `BORRADOR`.

- [ ] **Step 6: Commit**

```bash
git add migrations/000043_rename_estado_borrador_a_reservado.up.sql migrations/000043_rename_estado_borrador_a_reservado.down.sql
git commit -m "feat(db): migracion 000043 rename planes.estado BORRADOR -> RESERVADO"
```

---

### Task 2: Backend — rename constante, transición y estado inicial (venta directa atómica)

**Files:**
- Modify: `services/planes_service.go` (const `EstadoPlanBorrador`, `CrearPlan` estado inicial)
- Modify: `repositories/pgsql/planes_repo.go:333` (`esTransicionValida`)
- Test: `repositories/pgsql/planes_repo_test.go` (nuevo, solo función pura)

**Interfaces:**
- Consumes: constante estado del Task 1 (`RESERVADO`).
- Produces: `services.EstadoPlanReservado = "RESERVADO"`; `CrearPlan` deja el plan en `ACTIVO` cuando viene `PagoCodigo` (UNICO, precio>0), si no en `RESERVADO`.

- [ ] **Step 1: Escribir test de transición (RESERVADO)**

`repositories/pgsql/planes_repo_test.go`:
```go
package pgsql

import "testing"

func TestEsTransicionValida_Reservado(t *testing.T) {
	cases := []struct {
		actual, nuevo string
		want          bool
	}{
		{"RESERVADO", "ACTIVO", true},
		{"RESERVADO", "CANCELADO", true},
		{"RESERVADO", "COMPLETADO", false},
		{"BORRADOR", "ACTIVO", false}, // BORRADOR ya no existe como estado valido
		{"ACTIVO", "COMPLETADO", true},
		{"COMPLETADO", "ACTIVO", false},
	}
	for _, c := range cases {
		if got := esTransicionValida(c.actual, c.nuevo); got != c.want {
			t.Errorf("esTransicionValida(%q,%q)=%v want %v", c.actual, c.nuevo, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Correr el test (falla)**

Run: `go test ./repositories/pgsql/ -run TestEsTransicionValida_Reservado -v`
Expected: FAIL (caso `RESERVADO` da false porque el switch aún tiene `BORRADOR`).

- [ ] **Step 3: Actualizar `esTransicionValida`**

En `repositories/pgsql/planes_repo.go`, reemplazar el `case "BORRADOR":` por:
```go
	case "RESERVADO":
		return nuevo == "ACTIVO" || nuevo == "CANCELADO"
```

- [ ] **Step 4: Correr el test (pasa)**

Run: `go test ./repositories/pgsql/ -run TestEsTransicionValida_Reservado -v`
Expected: PASS.

- [ ] **Step 5: Rename de la constante y estado inicial en el service**

En `services/planes_service.go`:
- Renombrar la constante:
```go
	EstadoPlanReservado  = "RESERVADO"
```
(reemplaza la línea `EstadoPlanBorrador = "BORRADOR"`).
- En `CrearPlan`, reemplazar `Estado: EstadoPlanBorrador,` por el estado inicial derivado del pago. Justo antes del `s.repo.CreatePlan(...)`, añadir:
```go
	estadoInicial := EstadoPlanReservado
	if input.PagoCodigo != nil && input.TipoPago == TipoPagoUnico && precioTotal > 0 {
		estadoInicial = EstadoPlanActivo
	}
```
y usar `Estado: estadoInicial,` en el `CrearPlanInput`.

- [ ] **Step 6: Buscar cualquier otra referencia a la constante vieja**

Run: `grep -rn "EstadoPlanBorrador" --include=*.go .`
Expected: sin resultados (si aparece alguno, renombrarlo a `EstadoPlanReservado`).

- [ ] **Step 7: Build + vet**

Run: `go build ./... && go vet ./services/... ./repositories/...`
Expected: sin errores.

- [ ] **Step 8: Commit**

```bash
git add services/planes_service.go repositories/pgsql/planes_repo.go repositories/pgsql/planes_repo_test.go
git commit -m "feat(planes): estado RESERVADO + venta directa crea ACTIVO cuando hay pago"
```

---

### Task 3: Backend — `CobrarPlan` (repo + service + interface)

**Files:**
- Modify: `repositories/planes_repository.go` (interface: método `CobrarPlan`)
- Modify: `repositories/pgsql/planes_repo.go` (impl `CobrarPlan`)
- Modify: `services/planes_service.go` (método `CobrarPlan`)

**Interfaces:**
- Consumes: `aplicarPagoUnicoTx(tx *sqlx.Tx, planID int, pagoCodigo string, monto float64) error` (ya existe), `esTransicionValida`, `recomputarCobranzaTx`.
- Produces:
  - Repo: `CobrarPlan(planID int, pagoCodigo string) error`
  - Service: `(s *PlanesService) CobrarPlan(planID int, pagoCodigo string) error`
  - Errores: `repository.ErrPlanTransicionInvalida` si el plan no está `RESERVADO`; `repository.ErrPlanNoEncontrado` si no existe.

- [ ] **Step 1: Agregar el método a la interface**

En `repositories/planes_repository.go`, dentro de `type PlanesRepository interface`, junto a los otros métodos:
```go
	CobrarPlan(planID int, pagoCodigo string) error
```

- [ ] **Step 2: Implementar `CobrarPlan` en el repo**

En `repositories/pgsql/planes_repo.go`, agregar:
```go
// CobrarPlan adjunta un pago existente a un plan RESERVADO, marca la cobranza
// PAGADO y activa el plan, todo en una transaccion. Rechaza planes que no esten
// en RESERVADO (evita doble cobro).
func (r *PlanesRepo) CobrarPlan(planID int, pagoCodigo string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var plan struct {
		Estado      string  `db:"estado"`
		PrecioTotal float64 `db:"precio_total"`
	}
	if err := tx.Get(&plan, `SELECT estado, precio_total FROM planes WHERE id = $1`, planID); err != nil {
		return fmt.Errorf("%w: plan con id %d", repository.ErrPlanNoEncontrado, planID)
	}
	if plan.Estado != "RESERVADO" {
		return fmt.Errorf("%w: el plan %d no esta RESERVADO (estado %s)", repository.ErrPlanTransicionInvalida, planID, plan.Estado)
	}

	if err := aplicarPagoUnicoTx(tx, planID, pagoCodigo, plan.PrecioTotal); err != nil {
		return err
	}

	estadoCobranza, err := recomputarCobranzaTx(tx, planID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE planes SET estado = 'ACTIVO', estado_cobranza = $1, actualizado_en = NOW()
		WHERE id = $2
	`, estadoCobranza, planID); err != nil {
		return fmt.Errorf("error al activar el plan cobrado: %w", err)
	}

	return tx.Commit()
}
```

- [ ] **Step 3: Implementar `CobrarPlan` en el service**

En `services/planes_service.go`, agregar:
```go
func (s *PlanesService) CobrarPlan(planID int, pagoCodigo string) error {
	if planID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	if strings.TrimSpace(pagoCodigo) == "" {
		return fmt.Errorf("pago_codigo es requerido: %w", ErrPlanInvalido)
	}
	return traducirErrorRepositorioPlan(s.repo.CobrarPlan(planID, strings.TrimSpace(pagoCodigo)))
}
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: sin errores (la interface, impl y service compilan; el mock/otros impls de la interface —si hay— deben tener el método; si aparece error de interface no satisfecha en otro repo, agregar el método allí también).

- [ ] **Step 5: Verificar el flujo real contra DB (rollback, no persiste)**

Usar un plan RESERVADO real. Ejemplo con plan 11 (ajustar id):
```bash
export PGPASSWORD=$(grep -E '^DB_PASSWORD=' .env | cut -d= -f2- | tr -d '\r"'"'"' ')
psql -h localhost -p 5432 -U postgres -d atrevida_db -v ON_ERROR_STOP=1 <<'SQL'
BEGIN;
UPDATE planes SET estado='RESERVADO' WHERE id=11;
-- simula que existe un pago y su aplicacion:
SELECT estado, precio_total FROM planes WHERE id=11;
ROLLBACK;
SQL
```
Expected: la query devuelve estado `RESERVADO` y un `precio_total` numérico (confirma que `CobrarPlan` tendrá datos válidos). El test funcional completo (con pago real) se hace en Task 4 vía HTTP.

- [ ] **Step 6: Commit**

```bash
git add repositories/planes_repository.go repositories/pgsql/planes_repo.go services/planes_service.go
git commit -m "feat(planes): CobrarPlan adjunta pago a plan RESERVADO y lo activa"
```

---

### Task 4: Backend — handler + ruta `POST /bd/planes/{id}/cobrar` + swagger

**Files:**
- Modify: `handlers/planes_handler.go` (request struct + handler `CobrarPlan`)
- Modify: `router/router.go` (ruta)
- Modify: `docs/*` (regenerado)

**Interfaces:**
- Consumes: `h.PlanesPG.CobrarPlan(planID, pagoCodigo)`, helpers `requiredPositiveParam`, `responderErrorPlan`, `utils.Respond`, `messageResponse`.
- Produces: endpoint HTTP `POST /bd/planes/:id/cobrar` body `{ "pago_codigo": string }`.

- [ ] **Step 1: Agregar el request struct y el handler con swagger**

En `handlers/planes_handler.go`, agregar cerca de los otros request structs:
```go
type cobrarPlanRequest struct {
	PagoCodigo string `json:"pago_codigo" example:"PAGO-000123"`
}
```
y el handler:
```go
// CobrarPlan godoc
// @Summary Cobrar un plan reservado
// @Description Adjunta un pago existente (por codigo) a un plan en estado RESERVADO, marca la cobranza como PAGADO y activa el plan. Falla si el plan no esta RESERVADO. Requiere token Bearer con rol gerencia o admin_sys.
// @Tags Planes BD
// @Accept json
// @Produce json
// @Param id path int true "ID del plan"
// @Param request body cobrarPlanRequest true "Codigo del pago a aplicar"
// @Success 200 {object} utils.APIResponse{data=messageResponse}
// @Failure 400 {object} utils.APIResponse
// @Failure 403 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Router /bd/planes/{id}/cobrar [post]
func (h *Container) CobrarPlan(c *gin.Context) {
	id, err := requiredPositiveParam(c, "id")
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var req cobrarPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "body invalido")
		return
	}

	if err := h.PlanesPG.CobrarPlan(id, req.PagoCodigo); err != nil {
		responderErrorPlan(c, err)
		return
	}

	utils.Respond(c, http.StatusOK, messageResponse{Mensaje: "plan cobrado y activado correctamente"})
}
```

- [ ] **Step 2: Registrar la ruta**

En `router/router.go`, junto a las otras rutas de planes:
```go
		bd.POST("/planes/:id/cobrar", h.AuthRequired, h.GerenciaRequired, h.CobrarPlan)
```

- [ ] **Step 3: Regenerar swagger y build**

Run:
```bash
go generate ./... && go build ./...
```
Expected: sin errores; `docs/` regenerado con el nuevo endpoint.

- [ ] **Step 4: Prueba HTTP end-to-end**

Reiniciar el backend con el código nuevo (`Ctrl+C` en la terminal del server, luego `go run main.go`). Con un token de rol `gerencia`, un plan `RESERVADO` (id) y un `codigo_pago` de un pago existente:
```bash
TOKEN=... ; PLAN_ID=... ; PAGO=...
curl -s -X POST localhost:8080/bd/planes/$PLAN_ID/cobrar \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"pago_codigo\":\"$PAGO\"}" | jq
# repetir la MISMA llamada -> debe fallar (plan ya ACTIVO, transicion invalida)
```
Expected: primera llamada 200 y `data.mensaje`; el plan queda `estado=ACTIVO`, `estado_cobranza=PAGADO`. Segunda llamada: error 400/409 por transición inválida.

- [ ] **Step 5: Commit**

```bash
git add handlers/planes_handler.go router/router.go docs/
git commit -m "feat(planes): endpoint POST /bd/planes/{id}/cobrar"
```

---

### Task 5: Frontend — API `cobrarPlan` y helper de reserva

**Files:**
- Modify: `AtrevidaFront/lib/api/planes.ts`

**Interfaces:**
- Consumes: `apiClient`, `ApiResponse`, `crearPlan` (ya existe, `CrearPlanData` no lleva `pago_codigo` cuando se reserva).
- Produces:
  - `cobrarPlan(id: number, pago_codigo: string): Promise<ApiResponse<{ mensaje: string }>>`
  - Reservar = `crearPlan(data)` **sin** `pago_codigo` (no requiere función nueva).

- [ ] **Step 1: Agregar `cobrarPlan`**

En `lib/api/planes.ts`, junto a las otras funciones:
```ts
/** POST /bd/planes/{id}/cobrar — adjunta un pago existente y activa el plan reservado. */
export async function cobrarPlan(id: number, pago_codigo: string): Promise<ApiResponse<{ mensaje: string }>> {
  return apiClient.post<ApiResponse<{ mensaje: string }>>(`/bd/planes/${id}/cobrar`, { pago_codigo });
}
```

- [ ] **Step 2: Lint**

Run: `cd AtrevidaFront && npm run lint`
Expected: 0 errores (los 3 warnings preexistentes de `clientes/page.tsx` son ajenos).

- [ ] **Step 3: Commit**

```bash
git add lib/api/planes.ts
git commit -m "feat(planes): cobrarPlan API para cobrar reservas"
```

---

### Task 6: Frontend — página "Paquetes de clientes": rename, filtro/orden, reservar y cobrar

**Files:**
- Modify: `AtrevidaFront/app/atrevida-gestion/paquetes-activos/page.tsx`
- Create: `AtrevidaFront/app/atrevida-gestion/paquetes-activos/ReservarPlanModal.tsx`
- Create: `AtrevidaFront/app/atrevida-gestion/paquetes-activos/CobrarPlanModal.tsx`

**Interfaces:**
- Consumes: `crearPlan`, `cobrarPlan` (Task 5), `getLocalesDB`, `getServiciosDB`/combos, `crearPagoDB` (de `lib/api/pagos.ts`), `getClientesDB` (de `lib/api/clientes.ts`), `CustomSelect`, `FormModal`, `RowActionsMenu`.
- Produces: página con título "Paquetes de clientes", filtro por todos los estados (default "Todos"), orden RESERVADO/ACTIVO primero, botón "Reservar paquete", acción "Cobrar" en filas RESERVADO.

- [ ] **Step 1: Rename del título de la página**

En `page.tsx`, en el `<PageHeader ... title="Paquetes Activos" accentWord="Paquetes" .../>` cambiar a:
```tsx
            title="Paquetes de clientes"
            accentWord="Paquetes"
            subtitle="Paquetes reservados y adquiridos por los clientes y su avance"
```

- [ ] **Step 2: Filtro de estado con todos los estados, default "Todos"**

Reemplazar el estado `soloActivos` por un `filtroEstado` string. Cambiar:
```tsx
  const [soloActivos, setSoloActivos] = useState(true);
```
por:
```tsx
  const [filtroEstado, setFiltroEstado] = useState(''); // '' = Todos
```
En `fetchPlanes`, cambiar `estado: soloActivos ? 'ACTIVO' : undefined` por `estado: filtroEstado || undefined`.
Reemplazar el `CustomSelect` del filtro estado por:
```tsx
                    <CustomSelect
                      id="filtro-estado"
                      ariaLabelledBy="lbl-estado"
                      value={filtroEstado}
                      onChange={setFiltroEstado}
                      options={[
                        { value: '', label: 'Todos' },
                        { value: 'RESERVADO', label: 'Reservado' },
                        { value: 'ACTIVO', label: 'Activo' },
                        { value: 'COMPLETADO', label: 'Completado' },
                        { value: 'CANCELADO', label: 'Cancelado' },
                      ]}
                    />
```

- [ ] **Step 3: Orden RESERVADO/ACTIVO primero, luego por fecha**

Antes de pasar `data={planes}` al `DataTable`, derivar una lista ordenada:
```tsx
  const planesOrdenados = useMemo(() => {
    const rank = (e: string) => (e === 'RESERVADO' ? 0 : e === 'ACTIVO' ? 1 : 2);
    return [...planes].sort((a, b) => {
      const r = rank(a.estado) - rank(b.estado);
      if (r !== 0) return r;
      return (b.creado_en ?? '').localeCompare(a.creado_en ?? '');
    });
  }, [planes]);
```
y usar `data={planesOrdenados}`.

- [ ] **Step 4: Acción "Cobrar" en filas RESERVADO**

En la columna `acciones`, agregar al principio de la construcción de `actions`:
```tsx
        if (row.estado === 'RESERVADO') {
          actions.push(
            { label: 'Cobrar', icon: <CheckCircle size={12} strokeWidth={2} />, onClick: () => setPlanACobrar(row) },
            { label: 'Activar', icon: <PlayCircle size={12} strokeWidth={2} />, onClick: () => handleCambiarEstado(row, 'ACTIVO') },
            { label: 'Cancelar', icon: <XCircle size={12} strokeWidth={2} />, onClick: () => handleCambiarEstado(row, 'CANCELADO'), variant: 'danger' },
          );
        } else if (row.estado === 'ACTIVO') {
```
(quitar el `if (row.estado === 'BORRADOR')` viejo; RESERVADO lo reemplaza). Agregar el estado `const [planACobrar, setPlanACobrar] = useState<PlanRow | null>(null);`.

- [ ] **Step 5: Crear `CobrarPlanModal`**

`app/atrevida-gestion/paquetes-activos/CobrarPlanModal.tsx`:
```tsx
'use client';

import { useState } from 'react';
import { FormModal } from '@/components/AdminConfig';
import { CustomSelect } from '@/components/Custom/CustomSelectAdmin';
import { toast } from '@/components/Shared/Toast';
import { crearPagoDB } from '@/lib/api/pagos';
import { cobrarPlan, type PlanItem } from '@/lib/api/planes';

interface Props {
  plan: PlanItem;
  localId: number;
  onClose: () => void;
  onCobrado: () => void;
}

export default function CobrarPlanModal({ plan, localId, onClose, onCobrado }: Props) {
  const [tipoPago, setTipoPago] = useState('efectivo');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    setSaving(true);
    try {
      const pago = await crearPagoDB({
        local_id: localId,
        cliente_id: plan.cliente_id,
        tipo_pago: tipoPago,
        detalle: [{ servicio_texto: plan.combo_nombre_texto ?? 'Paquete', cantidad: 1, precio_unitario: plan.precio_total }],
        estado: 'PAGADO',
        activo: true,
      });
      const codigo = (pago as { data?: { codigo_pago?: string } })?.data?.codigo_pago;
      if (!codigo) throw new Error('sin codigo de pago');
      await cobrarPlan(plan.id, codigo);
      toast.success('Paquete cobrado y activado');
      onCobrado();
      onClose();
    } catch {
      toast.error('No se pudo cobrar el paquete.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormModal isOpen onClose={onClose} title="Cobrar paquete" onSubmit={handleSubmit} loading={saving} submitLabel="Cobrar">
      <p>Cliente: {plan.cliente}</p>
      <p>Total: {plan.precio_total} Bs</p>
      <CustomSelect
        value={tipoPago}
        onChange={setTipoPago}
        options={[{ value: 'efectivo', label: 'Efectivo' }, { value: 'qr', label: 'QR' }]}
      />
    </FormModal>
  );
}
```
**Nota:** verificar la firma real de `crearPagoDB` y su tipo de `detalle` en `lib/api/pagos.ts` y ajustar los campos (`servicio_texto`/`cantidad`/`precio_unitario`) a los nombres exactos que use ese módulo antes de dar por cerrado el step.

- [ ] **Step 6: Crear `ReservarPlanModal`**

`app/atrevida-gestion/paquetes-activos/ReservarPlanModal.tsx` — modal con selección de cliente, local y combo que llama `crearPlan({ combo_id, cliente_id, local_id, tipo_pago: 'UNICO' })` **sin** `pago_codigo`:
```tsx
'use client';

import { useEffect, useState } from 'react';
import { FormModal } from '@/components/AdminConfig';
import { CustomSelect } from '@/components/Custom/CustomSelectAdmin';
import { toast } from '@/components/Shared/Toast';
import { crearPlan } from '@/lib/api/planes';
import { getCombosDB } from '@/lib/api/servicios';
import { getClientesDB } from '@/lib/api/clientes';

interface LocalOpt { id: number; nombre: string; }
interface Props { locales: LocalOpt[]; onClose: () => void; onReservado: () => void; }

export default function ReservarPlanModal({ locales, onClose, onReservado }: Props) {
  const [clienteId, setClienteId] = useState('');
  const [localId, setLocalId] = useState('');
  const [comboId, setComboId] = useState('');
  const [clientes, setClientes] = useState<{ id: number; nombre: string }[]>([]);
  const [combos, setCombos] = useState<{ id: number; nombre: string }[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getClientesDB().then((r) => setClientes(((r as { data?: { clientes?: { id: number; nombre: string }[] } })?.data?.clientes) ?? [])).catch(() => setClientes([]));
    getCombosDB().then((r) => setCombos(((r as { data?: { combos?: { id: number; nombre: string }[] } })?.data?.combos) ?? [])).catch(() => setCombos([]));
  }, []);

  const handleSubmit = async () => {
    if (!clienteId || !localId || !comboId) { toast.error('Completa cliente, local y combo.'); return; }
    setSaving(true);
    try {
      await crearPlan({ combo_id: Number(comboId), cliente_id: Number(clienteId), local_id: Number(localId), tipo_pago: 'UNICO' });
      toast.success('Paquete reservado');
      onReservado();
      onClose();
    } catch {
      toast.error('No se pudo reservar el paquete.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormModal isOpen onClose={onClose} title="Reservar paquete" onSubmit={handleSubmit} loading={saving} submitLabel="Reservar">
      <CustomSelect value={clienteId} onChange={setClienteId} placeholder="Cliente…" options={clientes.map((c) => ({ value: String(c.id), label: c.nombre }))} />
      <CustomSelect value={localId} onChange={setLocalId} placeholder="Local…" options={locales.map((l) => ({ value: String(l.id), label: l.nombre }))} />
      <CustomSelect value={comboId} onChange={setComboId} placeholder="Combo…" options={combos.map((c) => ({ value: String(c.id), label: c.nombre }))} />
    </FormModal>
  );
}
```
**Nota:** verificar los nombres reales de `getClientesDB` y `getCombosDB` y sus shapes de respuesta en `lib/api/clientes.ts` / `lib/api/servicios.ts`, y ajustar. Si `getCombosDB` requiere un `local` param, pasar el local seleccionado.

- [ ] **Step 7: Cablear los modales y el botón en `page.tsx`**

- Importar ambos modales y añadir estado `const [reservarOpen, setReservarOpen] = useState(false);`.
- En el `PageHeader`, pasar un action button (según la API de `PageHeader`, p.ej. `actionLabel="Reservar paquete"` `onAction={() => setReservarOpen(true)}`; verificar props reales de `PageHeader`).
- Al final del JSX, renderizar condicionalmente:
```tsx
      {reservarOpen && (
        <ReservarPlanModal locales={locales} onClose={() => setReservarOpen(false)} onReservado={fetchPlanes} />
      )}
      {planACobrar && (
        <CobrarPlanModal plan={planACobrar} localId={planACobrar.local_id ?? locales[0]?.id ?? 0} onClose={() => setPlanACobrar(null)} onCobrado={fetchPlanes} />
      )}
```
**Nota:** `PlanItem` no expone `local_id` numérico hoy (tiene `local_nombre_texto`). Si el cobro necesita `local_id`, resolverlo desde `locales` por nombre o extender `PlanItem`/el backend para incluir `local_id`. Confirmar y ajustar en este step.

- [ ] **Step 8: Lint**

Run: `cd AtrevidaFront && npm run lint`
Expected: 0 errores.

- [ ] **Step 9: Prueba manual**

Con dev + backend arriba: abrir "Paquetes de clientes" → "Reservar paquete" → crear una reserva (aparece como Reservado). En esa fila → "Cobrar" → elegir tipo → confirmar → pasa a Activo. Verificar filtro por cada estado y el orden (Reservado/Activo arriba).

- [ ] **Step 10: Commit**

```bash
git add app/atrevida-gestion/paquetes-activos/
git commit -m "feat(paquetes): reservar y cobrar desde Paquetes de clientes + filtro/orden por estado"
```

---

### Task 7: Frontend — caja: venta directa sin 2da llamada + modo "cobrar reserva"

**Files:**
- Modify: `AtrevidaFront/app/atrevida-gestion/caja/page.tsx`

**Interfaces:**
- Consumes: `crearPlan` (ahora activa solo cuando hay pago), `cobrarPlan`, `getPlanesDB` (filtro `estado=RESERVADO&cliente`).
- Produces: caja ya no llama `cambiarEstadoPlan` tras el pago; nuevo flujo opcional para cobrar una reserva existente.

- [ ] **Step 1: Quitar la 2da llamada en venta directa**

En `caja/page.tsx`, en el bloque que hoy hace `crearPlan(...)` seguido de `if (nuevoId) await cambiarEstadoPlan(nuevoId, 'ACTIVO')`, eliminar la línea de `cambiarEstadoPlan` (el backend ya crea el plan ACTIVO cuando viene `pago_codigo`). Quitar el import de `cambiarEstadoPlan` si queda sin uso.

- [ ] **Step 2: Verificar en el flujo real de venta**

Con backend nuevo arriba: registrar una venta de paquete en caja → el plan debe nacer `ACTIVO` directamente (verificar en "Paquetes de clientes" que no queda `RESERVADO`).

- [ ] **Step 3: Modo "cobrar reserva" en caja**

Agregar en caja una sección/toggle "Cobrar reserva": buscar cliente → `getPlanesDB({ cliente, estado: 'RESERVADO' })` → listar sus planes RESERVADO en un `CustomSelect` → al confirmar el pago, en vez de `crearPlan`, crear el pago (`crearPagoDB`) y llamar `cobrarPlan(planSeleccionado.id, codigo_pago)`.
**Nota:** revisar la estructura actual de `caja/page.tsx` (cómo arma `detalle`, cómo crea el pago) para insertar el modo con el mínimo de duplicación; reutilizar el mismo `crearPagoDB` que ya usa la venta.

- [ ] **Step 4: Lint**

Run: `cd AtrevidaFront && npm run lint`
Expected: 0 errores.

- [ ] **Step 5: Prueba manual del modo cobrar-reserva**

Reservar un paquete (Task 6) → en caja, modo "cobrar reserva" → buscar el cliente → elegir la reserva → cobrar → el plan pasa a `ACTIVO` y el pago queda registrado. Confirmar que NO se creó un plan duplicado (un solo plan para ese cliente/combo).

- [ ] **Step 6: Commit**

```bash
git add app/atrevida-gestion/caja/page.tsx
git commit -m "feat(caja): venta directa atomica + cobrar reserva existente"
```

---

## Notas de verificación global

- Backend: `go build ./...`, `go test ./repositories/pgsql/ -run TestEsTransicionValida_Reservado`, `go generate ./...` (docs en sync).
- Migración: probada en `atrevida_migtest` fresca (0→43) y en `atrevida_db` real.
- Frontend: `npm run lint` (0 errores) tras cada task.
- End-to-end: reservar → cobrar (desde fila y desde caja) → 1 solo plan, estado `ACTIVO`, cobranza `PAGADO`; doble cobro rechazado; venta directa nace `ACTIVO`.
- Front y back se despliegan juntos (cambio de contrato: nuevo endpoint + estado renombrado). El deploy debe correr la migración 000043.
