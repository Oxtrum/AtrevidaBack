package handlers

import (
	"atrevida-agenda-api/models"
	"atrevida-agenda-api/services"
)

// ─── Filtros ───

type servicioFiltrosResponse struct {
	// Filtro aplicado: nombre del servicio
	Nombre string `json:"nombre" example:"depilacion"`
	// Filtro aplicado: categoría del servicio
	Categoria string `json:"categoria" example:"Corporal"`
	// Filtro aplicado: nombre del local
	Local string `json:"local" example:"ARANJUEZ"`
	// Filtro aplicado: cantidad exacta de sesiones
	Sesiones int `json:"sesiones" example:"6"`
	// Filtro aplicado: si requiere evaluación (true/false)
	RequiereEvaluacion *bool `json:"requiere_evaluacion" example:"true"`
	// Filtro aplicado: solo servicios visibles para pacientes nuevos
	PacienteNuevo *bool `json:"paciente_nuevo" example:"true"`
}

type comboFiltrosResponse struct {
	// Filtro aplicado: nombre del combo
	Nombre string `json:"nombre" example:"relax"`
	// Filtro aplicado: categoría del combo
	Categoria string `json:"categoria" example:"Corporal"`
	// Filtro aplicado: nombre del local
	Local string `json:"local" example:"ARANJUEZ"`
	// Filtro aplicado: cantidad de sesiones
	Sesiones int `json:"sesiones" example:"4"`
}

type reservaFiltrosResponse struct {
	// Filtro aplicado: nombre del local
	Local string `json:"local" example:"SAN MARTIN"`
	// Filtro aplicado: semana en formato YYYY-MM-DD (inicio lunes)
	Semana string `json:"semana" example:"2026-05-25"`
	// Filtro aplicado: día de la semana
	Dia string `json:"dia" example:"lunes"`
	// Filtro aplicado: tipo de reserva (mesa/bicicleta)
	Tipo string `json:"tipo" example:"mesa"`
	// Filtro aplicado: nombre del cliente
	Cliente string `json:"cliente" example:"Maria"`
	// Filtro aplicado: solo reservas con estado reservado
	Reservados bool `json:"reservados" example:"true"`
}

type reservaPGFiltrosResponse struct {
	// Filtro aplicado: nombre del local
	Local string `json:"local" example:"SAN MARTIN"`
	// Filtro aplicado: fecha exacta YYYY-MM-DD
	Fecha string `json:"fecha" example:"2026-05-23"`
	// Filtro aplicado: fecha inicio de rango YYYY-MM-DD
	FechaDesde string `json:"fecha_desde" example:"2026-05-19"`
	// Filtro aplicado: fecha fin de rango YYYY-MM-DD
	FechaHasta string `json:"fecha_hasta" example:"2026-05-24"`
	// Filtro aplicado: tipo de reserva (mesa/bicicleta)
	Tipo string `json:"tipo" example:"mesa"`
	// Filtro aplicado: nombre del cliente
	Cliente string `json:"cliente" example:"Maria Lopez"`
	// Filtro aplicado: número de teléfono
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
	// Filtro aplicado: servicio solicitado por el cliente
	ServicioSolicitado string `json:"servicio_solicitado" example:"depilacion"`
	// Filtro aplicado: servicio confirmado final
	ServicioConfirmado string `json:"servicio_confirmado" example:"depilacion laser piernas"`
	// Filtro aplicado: estado de la reserva
	Estado string `json:"estado" example:"AGENDADO"`
	// Filtro aplicado: solo reservas con estado reservado
	Reservados *bool `json:"reservados" example:"true"`
}

type clienteFiltrosResponse struct {
	// Filtro aplicado: nombre del cliente
	Nombre string `json:"nombre" example:"Maria"`
	// Filtro aplicado: apellido del cliente
	Apellido string `json:"apellido" example:"Lopez"`
	// Filtro aplicado: número de teléfono del cliente
	NumeroTelefono string `json:"numero_telefono" example:"+59170011223"`
}

type horarioFiltrosResponse struct {
	// Filtro aplicado: ID del local
	LocalID int `json:"local_id" example:"3"`
	// Filtro aplicado: día de la semana (1=lunes, 7=domingo)
	DiaSemana *int `json:"dia_semana" example:"1"`
}

type pagoFiltrosResponse struct {
	// Filtro aplicado: codigo del pago
	CodigoPago string `json:"codigo_pago" example:"PAGO-000001"`
	// Filtro aplicado: ID del local
	LocalID *int `json:"local_id,omitempty" example:"1"`
	// Filtro aplicado: nombre del local
	LocalNombre string `json:"local_nombre" example:"SAN MARTIN"`
	// Filtro aplicado: ID del cliente
	ClienteID *int `json:"cliente_id,omitempty" example:"12"`
	// Filtro aplicado: NIT del cliente
	ClienteNIT string `json:"cliente_nit" example:"1234567"`
	// Filtro aplicado: nombre del cliente
	ClienteNombre string `json:"cliente_nombre" example:"Maria"`
	// Filtro aplicado: tipo de pago
	TipoPago string `json:"tipo_pago" example:"efectivo"`
	// Filtro aplicado: estado del pago
	Estado string `json:"estado" example:"PENDIENTE"`
	// Filtro aplicado: estado activo
	Activo *bool `json:"activo,omitempty" example:"true"`
}

// ─── List Responses ───

type categoriaListResponse struct {
	// Total de categorías encontradas
	Total int `json:"total" example:"5"`
	// Lista de categorías
	Categorias []models.CategoriaPG `json:"categorias"`
}

type clienteListResponse struct {
	// Total de clientes encontrados
	Total int `json:"total" example:"1"`
	// Filtros aplicados en la búsqueda
	Filtros clienteFiltrosResponse `json:"filtros"`
	// Lista de clientes
	Clientes []models.ClientePG `json:"clientes"`
}

type servicioListResponse struct {
	// Total de servicios encontrados
	Total int `json:"total" example:"10"`
	// Filtros aplicados en la búsqueda
	Filtros servicioFiltrosResponse `json:"filtros"`
	// Lista de servicios
	Servicios []models.ServicioItem `json:"servicios"`
}

type comboListResponse struct {
	// Total de combos encontrados
	Total int `json:"total" example:"3"`
	// Filtros aplicados en la búsqueda
	Filtros comboFiltrosResponse `json:"filtros"`
	// Lista de combos
	Combos []models.ComboItem `json:"combos"`
}

type comboServicioListResponse struct {
	// Total de servicios del combo
	Total int `json:"total" example:"3"`
	// ID del combo padre
	ComboID int `json:"combo_id" example:"12"`
	// Lista de servicios del combo
	Servicios []models.ComboServicioDetallePG `json:"servicios"`
}

type reservaListResponse struct {
	// Total de locales con reservas
	TotalLocales int `json:"total_locales" example:"2"`
	// Filtros aplicados en la búsqueda
	Filtros reservaFiltrosResponse `json:"filtros"`
	// Reservas agrupadas por local
	Reservas []models.LocalReservas `json:"reservas"`
}

type reservaCalendarioResponse struct {
	// Total de locales con reservas
	TotalLocales int `json:"total_locales" example:"2"`
	// Filtros aplicados en la búsqueda
	Filtros reservaPGFiltrosResponse `json:"filtros"`
	// Reservas agrupadas por local con detalle semanal
	Reservas []models.LocalReservas `json:"reservas"`
}

type reservaSimpleListResponse struct {
	// Total de reservas encontradas
	Total int `json:"total" example:"15"`
	// Lista de reservas en formato plano (sin agrupar)
	Reservas []services.ReservaSimple `json:"reservas"`
}

type reservaNotificacionListResponse struct {
	// Total de notificaciones pendientes
	Total int `json:"total" example:"3"`
	// Reservas agendadas pendientes de marcar como leidas
	Reservas []services.ReservaSimple `json:"reservas"`
}

type localListResponse struct {
	// Total de locales encontrados
	Total int `json:"total" example:"2"`
	// Lista de locales con espacios y horarios
	Locales []models.LocalConEspacios `json:"locales"`
}

type horarioListResponse struct {
	// Total de horarios encontrados
	Total int `json:"total" example:"5"`
	// Filtros aplicados en la búsqueda
	Filtros horarioFiltrosResponse `json:"filtros"`
	// Lista de horarios del local
	Horarios []models.LocalHorarioPG `json:"horarios"`
}

type pagoListResponse struct {
	// Total de pagos encontrados
	Total int `json:"total" example:"3"`
	// Filtros aplicados en la busqueda
	Filtros pagoFiltrosResponse `json:"filtros"`
	// Lista de pagos sin detalle
	Pagos []models.PagoPG `json:"pagos"`
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

type comboServicioItemResponse struct {
	// Datos del servicio de combo
	Servicio *models.ComboServicioDetallePG `json:"servicio"`
}

type localItemResponse struct {
	// Total de locales (siempre 1 para consulta por ID)
	Total int `json:"total" example:"1"`
	// Datos del local con espacios y horarios
	Local *models.LocalConEspacios `json:"local"`
}

type horarioItemResponse struct {
	// Datos del horario
	Horario *models.LocalHorarioPG `json:"horario"`
}

type pagoItemResponse struct {
	// Datos del pago con detalle
	Pago *models.PagoCompletoPG `json:"pago"`
}

// ─── Creation Responses ───

type idResponse struct {
	// ID del recurso creado
	ID int `json:"id" example:"42"`
}

type reservaCreatedResponse struct {
	// ID de la reserva creada
	ID int `json:"id" example:"44"`
	// Mensaje de confirmación
	Mensaje string `json:"mensaje" example:"Reserva creada correctamente"`
}

type pagoCreatedResponse struct {
	// Codigo publico del pago creado
	CodigoPago string `json:"codigo_pago" example:"PAGO-000001"`
	// Mensaje de confirmacion
	Mensaje string `json:"mensaje" example:"Pago creado correctamente"`
}

type slotsResponse struct {
	// Slots disponibles
	SlotsOk []string `json:"slots_ok" example:"09:00,10:00"`
	// Slots no disponibles
	SlotsError []string `json:"slots_error" example:"11:00,12:00"`
}

// ─── Mutation Responses ───

type messageResponse struct {
	// Mensaje descriptivo del resultado de la operación
	Mensaje string `json:"mensaje" example:"operacion realizada correctamente"`
}

type actualizadasResponse struct {
	// Cantidad de registros actualizados
	Actualizadas int `json:"actualizadas" example:"3"`
}

// ─── Import Response ───

type importResponse struct {
	// Cantidad de categorías importadas
	Categorias int `json:"categorias" example:"8"`
	// Cantidad de servicios importados
	Servicios int `json:"servicios" example:"25"`
	// Cantidad de relaciones servicio-local importadas
	ServicioLocales int `json:"servicio_locales" example:"30"`
	// Cantidad de combos importados
	Combos int `json:"combos" example:"5"`
	// Cantidad de relaciones combo-local importadas
	ComboLocales int `json:"combo_locales" example:"6"`
	// Cantidad de servicios de combo importados
	ComboServicios int `json:"combo_servicios" example:"12"`
}
