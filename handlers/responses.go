package handlers

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
)

// ─── Filtros ───

type servicioFiltrosResponse struct {
	Nombre             string `json:"nombre" example:"depilacion"`
	Categoria          string `json:"categoria" example:"Corporal"`
	Local              string `json:"local" example:"ARANJUEZ"`
	Sesiones           int    `json:"sesiones" example:"6"`
	RequiereEvaluacion *bool  `json:"requiere_evaluacion" example:"true"`
}

type comboFiltrosResponse struct {
	Nombre    string `json:"nombre" example:"relax"`
	Categoria string `json:"categoria" example:"Corporal"`
	Local     string `json:"local" example:"ARANJUEZ"`
	Sesiones  int    `json:"sesiones" example:"4"`
}

type reservaFiltrosResponse struct {
	Local      string `json:"local" example:"SAN MARTIN"`
	Semana     string `json:"semana" example:"2026-05-25"`
	Dia        string `json:"dia" example:"lunes"`
	Tipo       string `json:"tipo" example:"mesa"`
	Cliente    string `json:"cliente" example:"Maria"`
	Reservados bool   `json:"reservados" example:"true"`
}

type reservaPGFiltrosResponse struct {
	Local              string `json:"local" example:"SAN MARTIN"`
	Fecha              string `json:"fecha" example:"2026-05-23"`
	FechaDesde         string `json:"fecha_desde" example:"2026-05-19"`
	FechaHasta         string `json:"fecha_hasta" example:"2026-05-24"`
	Tipo               string `json:"tipo" example:"mesa"`
	Cliente            string `json:"cliente" example:"Maria Lopez"`
	NumeroTelefono     string `json:"numero_telefono" example:"+59170011223"`
	ServicioSolicitado string `json:"servicio_solicitado" example:"depilacion"`
	ServicioConfirmado string `json:"servicio_confirmado" example:"depilacion laser piernas"`
	Estado             string `json:"estado" example:"AGENDADO"`
	Reservados         *bool  `json:"reservados" example:"true"`
}

type clienteFiltrosResponse struct {
	Nombre         string `json:"nombre" example:"Maria"`
	Apellido       string `json:"apellido" example:"Lopez"`
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
}

type horarioFiltrosResponse struct {
	LocalID   int   `json:"local_id" example:"3"`
	DiaSemana *int  `json:"dia_semana" example:"1"`
}

// ─── List Responses ───

type categoriaListResponse struct {
	Total      int                 `json:"total" example:"5"`
	Categorias []models.CategoriaPG `json:"categorias"`
}

type clienteListResponse struct {
	Total    int                   `json:"total" example:"1"`
	Filtros  clienteFiltrosResponse `json:"filtros"`
	Clientes []models.ClientePG    `json:"clientes"`
}

type servicioListResponse struct {
	Total     int                      `json:"total" example:"10"`
	Filtros   servicioFiltrosResponse  `json:"filtros"`
	Servicios []models.ServicioItem    `json:"servicios"`
}

type comboListResponse struct {
	Total   int                 `json:"total" example:"3"`
	Filtros comboFiltrosResponse `json:"filtros"`
	Combos  []models.ComboItem   `json:"combos"`
}

type comboServicioListResponse struct {
	Total     int                              `json:"total" example:"3"`
	ComboID   int                              `json:"combo_id" example:"12"`
	Servicios []models.ComboServicioDetallePG  `json:"servicios"`
}

type reservaListResponse struct {
	TotalLocales int                       `json:"total_locales" example:"2"`
	Filtros      reservaFiltrosResponse    `json:"filtros"`
	Reservas     []models.LocalReservas    `json:"reservas"`
}

type reservaCalendarioResponse struct {
	TotalLocales int                         `json:"total_locales" example:"2"`
	Filtros      reservaPGFiltrosResponse    `json:"filtros"`
	Reservas     []models.LocalReservas      `json:"reservas"`
}

type reservaSimpleListResponse struct {
	Total    int                     `json:"total" example:"15"`
	Reservas []services.ReservaSimple `json:"reservas"`
}

type localListResponse struct {
	Total   int                      `json:"total" example:"2"`
	Locales []models.LocalConEspacios `json:"locales"`
}

type horarioListResponse struct {
	Total    int                     `json:"total" example:"5"`
	Filtros  horarioFiltrosResponse  `json:"filtros"`
	Horarios []models.LocalHorarioPG `json:"horarios"`
}

// ─── Single Item Responses ───

type clienteItemResponse struct {
	Cliente *models.ClientePG `json:"cliente"`
}

type reservaItemResponse struct {
	Reserva *services.ReservaSimple `json:"reserva"`
}

type servicioItemResponse struct {
	Servicio *models.ServicioItem `json:"servicio"`
}

type comboServicioItemResponse struct {
	Servicio *models.ComboServicioDetallePG `json:"servicio"`
}

type localItemResponse struct {
	Total int                    `json:"total" example:"1"`
	Local *models.LocalConEspacios `json:"local"`
}

type horarioItemResponse struct {
	Horario *models.LocalHorarioPG `json:"horario"`
}

// ─── Creation Responses ───

type idResponse struct {
	ID int `json:"id" example:"42"`
}

type reservaCreatedResponse struct {
	ID      int    `json:"id" example:"44"`
	Mensaje string `json:"mensaje" example:"Reserva creada correctamente"`
}

type slotsResponse struct {
	SlotsOk    []string `json:"slots_ok" example:"09:00,10:00"`
	SlotsError []string `json:"slots_error" example:"11:00,12:00"`
}

// ─── Mutation Responses ───

type messageResponse struct {
	Mensaje string `json:"mensaje" example:"operacion realizada correctamente"`
}

// ─── Import Response ───

type importResponse struct {
	Categorias      int `json:"categorias" example:"8"`
	Servicios       int `json:"servicios" example:"25"`
	ServicioLocales int `json:"servicio_locales" example:"30"`
	Combos          int `json:"combos" example:"5"`
	ComboLocales    int `json:"combo_locales" example:"6"`
	ComboServicios  int `json:"combo_servicios" example:"12"`
}
