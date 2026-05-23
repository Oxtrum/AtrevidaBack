package handlers

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
)

// ─── Filtros ───

type servicioFiltrosResponse struct {
	// Filtro por nombre del servicio
	Nombre string `json:"nombre" example:"depilacion"`
	// Filtro por categoria del servicio
	Categoria string `json:"categoria" example:"Corporal"`
	// Filtro por local
	Local string `json:"local" example:"ARANJUEZ"`
	// Filtro por numero exacto de sesiones
	Sesiones int `json:"sesiones" example:"6"`
	// Filtro por evaluacion requerida
	RequiereEvaluacion *bool `json:"requiere_evaluacion" example:"true"`
}

type comboFiltrosResponse struct {
	// Filtro por nombre del combo
	Nombre string `json:"nombre" example:"relax"`
	// Filtro por categoria del combo
	Categoria string `json:"categoria" example:"Corporal"`
	// Filtro por local
	Local string `json:"local" example:"ARANJUEZ"`
	// Filtro por numero exacto de sesiones
	Sesiones int `json:"sesiones" example:"4"`
}

type reservaFiltrosResponse struct {
	// Filtro por local
	Local string `json:"local" example:"SAN MARTIN"`
	// Filtro por semana
	Semana string `json:"semana" example:"2026-05-25"`
	// Filtro por dia
	Dia string `json:"dia" example:"lunes"`
	// Filtro por tipo de reserva
	Tipo string `json:"tipo" example:"mesa"`
	// Filtro por nombre del cliente
	Cliente string `json:"cliente" example:"Maria"`
	// Filtro por estado reservado
	Reservados bool `json:"reservados" example:"true"`
}

type reservaPGFiltrosResponse struct {
	// Filtro por local
	Local string `json:"local" example:"SAN MARTIN"`
	// Filtro por fecha exacta
	Fecha string `json:"fecha" example:"2026-05-23"`
	// Filtro por fecha desde
	FechaDesde string `json:"fecha_desde" example:"2026-05-19"`
	// Filtro por fecha hasta
	FechaHasta string `json:"fecha_hasta" example:"2026-05-24"`
	// Filtro por tipo de reserva
	Tipo string `json:"tipo" example:"mesa"`
	// Filtro por nombre del cliente
	Cliente string `json:"cliente" example:"Maria Lopez"`
	// Filtro por numero de telefono
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
	// Filtro por servicio solicitado
	ServicioSolicitado string `json:"servicio_solicitado" example:"depilacion"`
	// Filtro por servicio confirmado
	ServicioConfirmado string `json:"servicio_confirmado" example:"depilacion laser piernas"`
	// Filtro por estado
	Estado string `json:"estado" example:"AGENDADO"`
	// Filtro por estado reservado
	Reservados *bool `json:"reservados" example:"true"`
}

type clienteFiltrosResponse struct {
	// Filtro por nombre
	Nombre string `json:"nombre" example:"Maria"`
	// Filtro por apellido
	Apellido string `json:"apellido" example:"Lopez"`
	// Filtro por numero de telefono
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
}

type horarioFiltrosResponse struct {
	// ID del local consultado
	LocalID int `json:"local_id" example:"3"`
	// Dia de la semana filtrado (1=lunes, 7=domingo)
	DiaSemana *int `json:"dia_semana" example:"1"`
}

// ─── List Responses ───

type categoriaListResponse struct {
	// Cantidad total de categorias
	Total int `json:"total" example:"5"`
	// Lista de categorias
	Categorias []models.CategoriaPG `json:"categorias"`
}

type clienteListResponse struct {
	// Cantidad total de clientes
	Total int `json:"total" example:"1"`
	// Filtros aplicados
	Filtros clienteFiltrosResponse `json:"filtros"`
	// Lista de clientes
	Clientes []models.ClientePG `json:"clientes"`
}

type servicioListResponse struct {
	// Cantidad total de servicios
	Total int `json:"total" example:"10"`
	// Filtros aplicados
	Filtros servicioFiltrosResponse `json:"filtros"`
	// Lista de servicios
	Servicios []models.ServicioItem `json:"servicios"`
}

type comboListResponse struct {
	// Cantidad total de combos
	Total int `json:"total" example:"3"`
	// Filtros aplicados
	Filtros comboFiltrosResponse `json:"filtros"`
	// Lista de combos
	Combos []models.ComboItem `json:"combos"`
}

type reservaListResponse struct {
	// Cantidad de locales con reservas
	TotalLocales int `json:"total_locales" example:"2"`
	// Filtros aplicados
	Filtros reservaFiltrosResponse `json:"filtros"`
	// Reservas agrupadas por local y semana
	Reservas []models.LocalReservas `json:"reservas"`
}

type reservaCalendarioResponse struct {
	// Cantidad de locales con reservas
	TotalLocales int `json:"total_locales" example:"2"`
	// Filtros aplicados
	Filtros reservaPGFiltrosResponse `json:"filtros"`
	// Reservas agrupadas por local
	Reservas []models.LocalReservas `json:"reservas"`
}

type reservaSimpleListResponse struct {
	// Cantidad total de reservas
	Total int `json:"total" example:"15"`
	// Lista plana de reservas
	Reservas []services.ReservaSimple `json:"reservas"`
}

type localListResponse struct {
	// Cantidad total de locales
	Total int `json:"total" example:"2"`
	// Lista de locales con espacios y horarios
	Locales []models.LocalConEspacios `json:"locales"`
}

type horarioListResponse struct {
	// Cantidad total de horarios
	Total int `json:"total" example:"5"`
	// Filtros aplicados
	Filtros horarioFiltrosResponse `json:"filtros"`
	// Lista de horarios
	Horarios []models.LocalHorarioPG `json:"horarios"`
}

// ─── Single Item Responses ───

type clienteItemResponse struct {
	// Datos del cliente
	Cliente *models.ClientePG `json:"cliente"`
}

type reservaItemResponse struct {
	// Datos de la reserva
	Reserva *services.ReservaSimple `json:"reserva"`
}

type servicioItemResponse struct {
	// Datos del servicio
	Servicio *models.ServicioItem `json:"servicio"`
}

type localItemResponse struct {
	// Total de resultados (siempre 1)
	Total int `json:"total" example:"1"`
	// Datos del local con espacios y horarios
	Local *models.LocalConEspacios `json:"local"`
}

type horarioItemResponse struct {
	// Datos del horario
	Horario *models.LocalHorarioPG `json:"horario"`
}

// ─── Creation Responses ───

type idResponse struct {
	// ID del recurso creado
	ID int `json:"id" example:"42"`
}

type reservaCreatedResponse struct {
	// ID de la reserva creada
	ID int `json:"id" example:"44"`
	// Mensaje de confirmacion
	Mensaje string `json:"mensaje" example:"Reserva creada correctamente"`
}

type slotsResponse struct {
	// Slots que se reservaron exitosamente
	SlotsOk []string `json:"slots_ok" example:"09:00,10:00"`
	// Slots que no pudieron reservarse
	SlotsError []string `json:"slots_error" example:"11:00,12:00"`
}

// ─── Mutation Responses ───

type messageResponse struct {
	// Mensaje descriptivo del resultado
	Mensaje string `json:"mensaje" example:"operacion realizada correctamente"`
}

// ─── Import Response ───

type importResponse struct {
	// Cantidad de categorias importadas
	Categorias int `json:"categorias" example:"8"`
	// Cantidad de servicios importados
	Servicios int `json:"servicios" example:"25"`
	// Cantidad de relaciones servicio-local creadas
	ServicioLocales int `json:"servicio_locales" example:"30"`
	// Cantidad de combos importados
	Combos int `json:"combos" example:"5"`
	// Cantidad de relaciones combo-local creadas
	ComboLocales int `json:"combo_locales" example:"6"`
	// Cantidad de relaciones combo-servicio creadas
	ComboServicios int `json:"combo_servicios" example:"12"`
}
