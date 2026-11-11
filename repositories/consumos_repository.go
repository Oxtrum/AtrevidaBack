package repository

import "time"

// FiltroConsumos consulta el consumo declarado en atenciones activas. Las
// fechas son inclusivas y se normalizan en el service (por defecto, hoy).
type FiltroConsumos struct {
	LocalID    *int
	FechaDesde time.Time
	FechaHasta time.Time
	ItemID     *int
	Cliente    string
	Servicio   string
	Empleado   string
	EmpleadoID *int
	// EstadoDescuento vacio = todos; DESCONTADO = solo lineas descontadas;
	// PENDIENTE = lineas con deduccion pendiente (PENDIENTE_STOCK o NO_CATALOGADO).
	EstadoDescuento string
	Q               string
	// PageLimit > 0 activa la paginacion por desplazamiento del calendario.
	PageLimit int
	Offset    int
}
