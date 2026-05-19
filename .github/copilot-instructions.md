# Copilot Instructions

Cuando trabajes en este repositorio:

- Si cambias un endpoint, tambien cambia su documentacion Swagger.
- Si cambias `handler`, `router`, request structs o responses, revisa si debes actualizar anotaciones `swaggo`.
- Mantén la documentacion junto al handler, no en archivos separados.
- Usa anotaciones `@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure` y `@Router`.
- Reutiliza `utils.APIResponse` cuando documentes respuestas JSON del proyecto.
- Regenera Swagger con `go generate ./...` o `./scripts/update-swagger.ps1`.
- Los archivos generados en `docs/` forman parte del cambio y deben quedar actualizados.
- Verifica que `go build ./...` siga compilando.

URL esperada de la documentacion al correr la API:

`http://localhost:8080/swagger/index.html`
