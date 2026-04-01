# 📋 Nombre de tu API

Descripción breve: API REST desarrollada en Go con Gin para consultar datos de reservas desde Google Sheets.

---

## 🚀 Tecnologías

- [Go](https://golang.org/)
- [Gin](https://github.com/gin-gonic/gin) — HTTP framework
- [Google Sheets API](https://developers.google.com/sheets/api)
- [godotenv](https://github.com/joho/godotenv) — manejo de variables de entorno

---

## ⚙️ Configuración

1. Instalar dependencias
   ```bash
   go mod tidy
   ```

2. Configurar el archivo `.env` a partir del ejemplo

3. Correr la API
   ```bash
   go run main.go
   ```

---

## 🌐 Endpoints

| Método | Ruta                  | Descripción                                      | Auth |
|--------|-----------------------|--------------------------------------------------|------|
| GET    | `/`                   | Estado de la API                                 | No   |
| GET    | `/reservas/unfiltered`| Lista completa de reservas (formateadas)         | No   |
| GET    | `/reservas/raw`       | Datos crudos desde Google Sheets (sin formatear) | No   |

> La tabla se irá actualizando a medida que se agreguen nuevos endpoints.

---

## 📁 Estructura del proyecto
```
.
├── config/
│   └── config.go        # Carga de variables de entorno
├── models/
│   └── reserva.go       # Definición de structs
├── services/
│   └── sheets_service.go  # Lógica de negocio y conexión a Sheets
├── utils/
│   └── parser.go        # Utilidades de parseo
├── .env.example         # Plantilla de variables de entorno
├── .gitignore
├── credentials.json     # ⚠️ No se sube al repo (.gitignore)
├── go.mod
├── go.sum
└── main.go
```

---

## 🔐 Variables de entorno

| Variable            | Descripción                              | Ejemplo                  |
|---------------------|------------------------------------------|--------------------------|
| `SPREADSHEET_ID`    | ID del Google Spreadsheet a consultar    | `1BxiMVs0XRA...`|
| `SHEETS_DISPONIBLES`| Nombres de hojas separados por coma      | `local 1,local 2`        |

---

## 📌 Notas

- El rango de lectura por defecto es `!A1:G200`. Es un valor provisional a ajustarse a posteriori.
- Las credenciales de Google Sheets deben configurarse aparte (acceso interno).
