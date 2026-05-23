package handlers

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
)

// ─── Filtros ───

type servicioFiltrosResponse struct {
	Nombre             string `json:"nombre"`
	Categoria          string `json:"categoria"`
	Local              string `json:"local"`
	Sesiones           int    `json:"sesiones"`
	RequiereEvaluacion *bool  `json:"requiere_evaluacion"`
}

type comboFiltrosResponse struct {
	Nombre    string `json:"nombre"`
	Categoria string `json:"categoria"`
	Local     string `json:"local"`
	Sesiones  int    `json:"sesiones"`
}

type reservaFiltrosResponse struct {
	Local      string `json:"local"`
	Semana     string `json:"semana"`
	Dia        string `json:"dia"`
	Tipo       string `json:"tipo"`
	Cliente    string `json:"cliente"`
	Reservados bool   `json:"reservados"`
}

type reservaPGFiltrosResponse struct {
	Local              string `json:"local"`
	Fecha              string `json:"fecha"`
	FechaDesde         string `json:"fecha_desde"`
	FechaHasta         string `json:"fecha_hasta"`
	Tipo               string `json:"tipo"`
	Cliente            string `json:"cliente"`
	NumeroTelefono     string `json:"numero_telefono"`
	ServicioSolicitado string `json:"servicio_solicitado"`
	ServicioConfirmado string `json:"servicio_confirmado"`
	Estado             string `json:"estado"`
	Reservados         *bool  `json:"reservados"`
}

type clienteFiltrosResponse struct {
	Nombre         string `json:"nombre"`
	Apellido       string `json:"apellido"`
	NumeroTelefono string `json:"numero_telefono"`
}

type horarioFiltrosResponse struct {
	LocalID   int  `json:"local_id"`
	DiaSemana *int `json:"dia_semana"`
}

// ─── List Responses ───

type categoriaListResponse struct {
	Total      int                `json:"total"`
	Categorias []models.CategoriaPG `json:"categorias"`
}

type clienteListResponse struct {
	Total    int                   `json:"total"`
	Filtros  clienteFiltrosResponse `json:"filtros"`
	Clientes []models.ClientePG    `json:"clientes"`
}

type servicioListResponse struct {
	Total     int                     `json:"total"`
	Filtros   servicioFiltrosResponse `json:"filtros"`
	Servicios []models.ServicioItem   `json:"servicios"`
}

type comboListResponse struct {
	Total   int                  `json:"total"`
	Filtros comboFiltrosResponse `json:"filtros"`
	Combos  []models.ComboItem   `json:"combos"`
}

type reservaListResponse struct {
	TotalLocales int                   `json:"total_locales"`
	Filtros      reservaFiltrosResponse `json:"filtros"`
	Reservas     []models.LocalReservas `json:"reservas"`
}

type reservaCalendarioResponse struct {
	TotalLocales int                     `json:"total_locales"`
	Filtros      reservaPGFiltrosResponse `json:"filtros"`
	Reservas     []models.LocalReservas   `json:"reservas"`
}

type reservaSimpleListResponse struct {
	Total    int                   `json:"total"`
	Reservas []services.ReservaSimple `json:"reservas"`
}

type localListResponse struct {
	Total   int                     `json:"total"`
	Locales []models.LocalConEspacios `json:"locales"`
}

type horarioListResponse struct {
	Total    int                    `json:"total"`
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

type localItemResponse struct {
	Total int                     `json:"total"`
	Local *models.LocalConEspacios `json:"local"`
}

type horarioItemResponse struct {
	Horario *models.LocalHorarioPG `json:"horario"`
}

// ─── Creation Responses ───

type idResponse struct {
	ID int `json:"id"`
}

type reservaCreatedResponse struct {
	ID      int    `json:"id"`
	Mensaje string `json:"mensaje"`
}

type slotsResponse struct {
	SlotsOk    []string `json:"slots_ok"`
	SlotsError []string `json:"slots_error"`
}

// ─── Mutation Responses ───

type messageResponse struct {
	Mensaje string `json:"mensaje"`
}

// ─── Import Response ───

type importResponse struct {
	Categorias      int `json:"categorias"`
	Servicios       int `json:"servicios"`
	ServicioLocales int `json:"servicio_locales"`
	Combos          int `json:"combos"`
	ComboLocales    int `json:"combo_locales"`
	ComboServicios  int `json:"combo_servicios"`
}
