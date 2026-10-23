package services

import (
	"errors"
	"strings"
	"time"

	repository "atrevida-agenda-api/repositories"
)

// limiteVentasDetalle acota el detalle exportable; por encima se avisa con truncado.
const limiteVentasDetalle = 20000

type VentaDetalleLinea struct {
	// Codigo del pago al que pertenece la linea.
	CodigoPago string `json:"codigo_pago" example:"PAGO-000012"`
	// Fecha y hora del pago.
	Fecha time.Time `json:"fecha" example:"2026-10-14T15:30:00-04:00"`
	// Local donde se vendio.
	Local string `json:"local" example:"SAN MARTIN"`
	// Nombre del cliente.
	ClienteNombre string `json:"cliente_nombre" example:"Maria Rojas"`
	// NIT o CI del cliente.
	ClienteNIT string `json:"cliente_nit" example:"7654321"`
	// Cajero que registro el cobro; vacio si no se guardo.
	Cajero string `json:"cajero" example:"Ana Perez"`
	// Tipo de pago: qr, efectivo o sin_tipo.
	TipoPago string `json:"tipo_pago" example:"efectivo"`
	// Servicio cobrado.
	Servicio string `json:"servicio" example:"Limpieza facial"`
	// Cantidad vendida del servicio.
	Cantidad int `json:"cantidad" example:"2"`
	// Precio unitario del servicio.
	PrecioUnitario float64 `json:"precio_unitario" example:"150"`
	// Subtotal de la linea antes de descuentos.
	Subtotal float64 `json:"subtotal" example:"300"`
	// Parte del descuento del pago que corresponde a esta linea (proporcional a su subtotal).
	DescuentoProrrateado float64 `json:"descuento_prorrateado" example:"15"`
	// Total final del pago completo (se repite en cada linea del mismo pago).
	TotalPago float64 `json:"total_pago" example:"285"`
}

type VentasDetalleResponse struct {
	// Filtros aplicados.
	Filtros PagoResumenFiltros `json:"filtros"`
	// true si el periodo tiene mas lineas que el limite y se devolvieron solo las mas antiguas.
	Truncado bool `json:"truncado" example:"false"`
	// Lineas de venta, de la mas antigua a la mas reciente.
	Lineas []VentaDetalleLinea `json:"lineas"`
}

// GetVentasDetalle devuelve cada servicio vendido del periodo (pagos activos y PAGADOS), con sucursal, cliente, cajero,
// tipo de pago y descuento prorrateado. Es la base de los reportes por servicio, sucursal, dia y cajero.
func (s *PagosService) GetVentasDetalle(input ResumenPagosInput) (*VentasDetalleResponse, error) {
	fechaDesde, err := parseResumenPagoDate(input.FechaDesde, "fecha_desde")
	if err != nil {
		return nil, err
	}
	fechaHasta, err := parseResumenPagoDate(input.FechaHasta, "fecha_hasta")
	if err != nil {
		return nil, err
	}
	if fechaHasta.Before(fechaDesde) {
		return nil, errors.New("fecha_hasta no puede ser anterior a fecha_desde")
	}
	local := strings.TrimSpace(input.Local)

	filas, err := s.repo.ListVentasDetalle(repository.FiltroResumenPagos{
		Context: input.Context, FechaDesde: fechaDesde, FechaHasta: fechaHasta, Local: local,
	}, limiteVentasDetalle+1)
	if err != nil {
		return nil, err
	}
	truncado := len(filas) > limiteVentasDetalle
	if truncado {
		filas = filas[:limiteVentasDetalle]
	}
	lineas := make([]VentaDetalleLinea, len(filas))
	for i, f := range filas {
		lineas[i] = VentaDetalleLinea{
			CodigoPago: f.CodigoPago, Fecha: f.Fecha, Local: f.Local, ClienteNombre: f.ClienteNombre,
			ClienteNIT: f.ClienteNIT, Cajero: f.Cajero, TipoPago: f.TipoPago, Servicio: f.Servicio,
			Cantidad: f.Cantidad, PrecioUnitario: f.PrecioUnitario, Subtotal: f.Subtotal,
			DescuentoProrrateado: f.DescuentoProrrateado, TotalPago: f.TotalPago,
		}
	}
	return &VentasDetalleResponse{
		Filtros: PagoResumenFiltros{
			FechaDesde: fechaDesde.Format("2006-01-02"), FechaHasta: fechaHasta.Format("2006-01-02"), Local: local,
		},
		Truncado: truncado,
		Lineas:   lineas,
	}, nil
}
