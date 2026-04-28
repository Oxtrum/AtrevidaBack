# 📋 Atrevida Fit API

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

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| GET | `/` | Estado de la API | No |

### 🧪 Debug - Sheets

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| GET | `/reservas/unfiltered` | Lista completa de reservas (formateadas) | No |
| GET | `/reservas/raw` | Datos crudos desde Google Sheets | No |
| GET | `/reservas/celda-raw` | Obtiene el contenido crudo de una celda específica y su parseo | No |

> Query params para `/reservas/celda-raw`: `local`, `semana`, `dia`, `hora`

---

### 📊 Reservas - Sheets

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| GET | `/reservas` | Obtiene reservas formateadas con filtros | No |
| POST | `/reservas` | Crea una nueva reserva en Google Sheets | No |
| PATCH | `/reservas` | Actualiza una reserva en Google Sheets | No |

---

### 🧾 Catálogo - Sheets

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| GET | `/servicios` | Lista de servicios disponibles | No |
| GET | `/combos` | Lista de combos disponibles | No |

---

### 🗄️ Base de Datos (PostgreSQL)

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| GET | `/bd/servicios` | Lista de servicios desde BD | No |
| GET | `/bd/combos` | Lista de combos desde BD | No |
| GET | `/bd/reservas/calendario` | Calendario de reservas (bloques de 30 min, libres y reservados) | No |
| POST | `/bd/reservas` | Crea una nueva reserva en BD | No |
| PATCH | `/bd/reservas` | Actualiza una reserva en BD | No |

---

### ⚙️ Administración

| Método | Ruta | Descripción | Auth |
|--------|------|------------|------|
| POST | `/admin/importar` | Importa catálogo (servicios/combos) a la BD | No |

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
