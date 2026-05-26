# AGENTS.md

Este proyecto usa Swagger con `swaggo`. Cualquier agente o IA que modifique endpoints, requests o responses debe seguir estas reglas.

## Regla principal

Si cambias cualquiera de estos puntos, tambien debes actualizar la documentacion Swagger:

- rutas nuevas o eliminadas
- parametros `query`, `path` o `body`
- codigos de respuesta
- nombres o estructuras de request/response
- tags, summary o descripcion del endpoint

## Como documentar

1. Agrega o actualiza anotaciones `swaggo` directamente sobre el handler correspondiente.
2. Usa el estilo ya presente en `handlers/`.
3. Para endpoints HTTP, incluye como minimo:
   - `@Summary`
   - `@Description`
   - `@Tags`
   - `@Accept` cuando aplique
   - `@Produce json`
   - `@Param` para path/query/body
   - `@Success`
   - `@Failure`
   - `@Router`
4. Reutiliza `utils.APIResponse` como wrapper de respuesta cuando corresponda.

## Generacion obligatoria

Despues de tocar documentacion o endpoints, ejecuta:

```bash
go generate ./...
```

o en PowerShell:

```powershell
./scripts/update-swagger.ps1
```

Eso debe regenerar:

- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

## Verificacion obligatoria

Antes de terminar un cambio, valida:

```bash
go build ./...
```

Si la API puede correr localmente, la UI debe quedar disponible en:

```text
http://localhost:8080/swagger/index.html
```

## Hooks del repo

Este repo tiene hooks versionados en `.githooks/`:

- `pre-commit`: regenera Swagger y agrega `docs/`
- `pre-push`: bloquea el push si `docs/` cambio y no esta commiteado

Si el clon aun no los usa, instala:

```powershell
./scripts/install-git-hooks.ps1
```

## Checklist de documentacion

Cuando se te pida documentar APIs, revisa CADA endpoint modificado contra esta lista:

- [ ] `@Summary` describe la accion en pocas palabras
- [ ] `@Description` explica comportamiento, reglas de negocio y casos borde
- [ ] `@Tags` agrupa correctamente (ej: `Reservas BD`, `Reservas Sheets`, `Combos BD`, etc.)
- [ ] `@Accept json` presente en handlers que reciben body
- [ ] `@Produce json` presente (excepto Swagger UI)
- [ ] `@Param` cubre TODOS los parametros: path, query, body
- [ ] Cada campo del struct request/response tiene comentario `//` con descripcion
- [ ] `@Success` coincide con el codigo real del handler (200, 201, 207...)
- [ ] `@Failure` cubre TODOS los codigos que el handler puede retornar (400, 404, 409, 500...)
- [ ] `@Failure` incluye texto descriptivo entre comillas: `"Error de validacion: ..."`
- [ ] `@Router` coincide exactamente con la ruta registrada en `router.go`
- [ ] Response types en `{object}` existen en el codigo
- [ ] `example` tags presentes en campos de request/response

## Criterio de cierre

No des por terminado un cambio de API si:

- Swagger no fue actualizado
- `docs/` no fue regenerado
- `go build ./...` no pasa
- La checklist de documentacion no fue verificada

