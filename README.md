# Atrevida Fit API

API REST desarrollada en Go con Gin para gestionar reservas, catalogo y datos operativos de Atrevida Fit.

## Stack

- Go
- Gin
- PostgreSQL
- Google Sheets API
- Swagger (`swaggo`)

## Puesta en marcha

1. Instala dependencias:

```bash
go mod tidy
```

2. Crea tu `.env` a partir de `.env.example`.

3. Levanta la API:

```bash
go run main.go
```

La API queda en `http://localhost:8080`.

## Documentacion Swagger

- UI Swagger: `http://localhost:8080/swagger/index.html`
- Archivos generados: [`docs/`](/c:/Projects/Atrevida/AtrevidaBack/docs)
- Generacion manual:

```bash
go generate ./...
```

Tambien puedes regenerar solo Swagger con:

```powershell
./scripts/update-swagger.ps1
```

## Actualizacion automatica en commit y push

El repo incluye hooks versionados en `.githooks/`:

- `pre-commit`: regenera `docs/` y la agrega al commit.
- `pre-push`: regenera `docs/` y bloquea el push si hay cambios sin commitear en `docs/`.

Para activarlos en tu clon:

```powershell
./scripts/install-git-hooks.ps1
```

Eso configura:

```bash
git config core.hooksPath .githooks
```

## Reglas para IAs y agentes

El repo incluye instrucciones para asistentes de codigo:

- [`AGENTS.md`](/c:/Projects/Atrevida/AtrevidaBack/AGENTS.md): reglas generales del proyecto para agentes que leen instrucciones del repo.
- [`.github/copilot-instructions.md`](/c:/Projects/Atrevida/AtrevidaBack/.github/copilot-instructions.md): instrucciones especificas para GitHub Copilot.

La regla importante es simple: si una IA cambia la API, tambien debe actualizar Swagger y regenerar `docs/`.

## Endpoints principales

### Base

| Metodo | Ruta | Descripcion |
|---|---|---|
| GET | `/` | Estado de la API |
| GET | `/swagger/*any` | UI y spec de Swagger |

### Reservas Sheets

| Metodo | Ruta | Descripcion |
|---|---|---|
| GET | `/reservas` | Lista reservas con filtros |
| POST | `/reservas` | Crea una reserva en Google Sheets |
| PATCH | `/reservas` | Actualiza una reserva en Google Sheets |

### Catalogo

| Metodo | Ruta | Descripcion |
|---|---|---|
| GET | `/servicios` | Lista servicios |
| GET | `/combos` | Lista combos |

### Base de datos

| Metodo | Ruta | Descripcion |
|---|---|---|
| GET | `/bd/servicios` | Lista servicios desde BD |
| GET | `/bd/servicios/:id` | Obtiene un servicio por ID |
| GET | `/bd/combos` | Lista combos desde BD |
| GET | `/bd/locales` | Lista locales |
| GET | `/bd/locales/:id` | Obtiene un local por ID |
| GET | `/bd/reservas` | Lista simple de reservas |
| GET | `/bd/reservas/:id` | Obtiene una reserva por ID |
| GET | `/bd/reservas/calendario` | Calendario de reservas |
| POST | `/bd/reservas` | Crea una reserva en BD |
| PATCH | `/bd/reservas` | Actualiza una reserva en BD |

### Administracion

| Metodo | Ruta | Descripcion |
|---|---|---|
| POST | `/admin/importar` | Importa catalogo a la BD |

## Variables de entorno

| Variable | Descripcion |
|---|---|
| `SPREADSHEET_ID` | ID del spreadsheet de Google Sheets |
| `SHEETS_DISPONIBLES` | Nombres de hojas separados por coma |

## Notas

- `docs/` es codigo generado y debe mantenerse versionado.
- Si agregas o cambias anotaciones `@...`, regenera Swagger antes de revisar el resultado en `/swagger/index.html`.
