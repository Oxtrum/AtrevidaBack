package services

import (
	"errors"
	"sort"
	"strings"
	"time"

	repository "atrevida-agenda-api/repositories"
)

type ResumenPagosInput struct {
	FechaDesde string
	FechaHasta string
	Local      string
}

type PagoResumenFiltros struct {
	// Fecha inicial del reporte en formato YYYY-MM-DD.
	FechaDesde string `json:"fecha_desde" example:"2026-01-01"`
	// Fecha final del reporte en formato YYYY-MM-DD.
	FechaHasta string `json:"fecha_hasta" example:"2026-01-31"`
	// Local filtrado; vacio cuando el reporte es general.
	Local string `json:"local" example:"SAN MARTIN"`
}

type PagoResumenServicio struct {
	// Nombre del servicio cobrado.
	Servicio string `json:"servicio" example:"Limpieza facial"`
	// Cantidad total vendida del servicio.
	Cantidad int `json:"cantidad" example:"12"`
	// Monto total generado por el servicio.
	MontoTotal float64 `json:"monto_total" example:"1200"`
}

type PagoResumenTipoPago struct {
	// Tipo de pago utilizado.
	TipoPago string `json:"tipo_pago" example:"efectivo"`
	// Cantidad de pagos registrados con este tipo.
	CantidadPagos int `json:"cantidad_pagos" example:"8"`
	// Total final acumulado para este tipo de pago.
	Total float64 `json:"total" example:"2500"`
}

type PagoResumenReporte struct {
	// Tipo de reporte: general o Local: <nombre local>.
	TipoReporte string `json:"tipo_reporte" example:"general"`
	// Nombre del local del reporte; vacio para reporte general.
	Local string `json:"local" example:"SAN MARTIN"`
	// Total final vendido en el periodo.
	TotalPeriodo float64 `json:"total_periodo" example:"84857"`
	// Subtotal acumulado antes de descuentos.
	Subtotal float64 `json:"subtotal" example:"85000"`
	// Descuentos acumulados en el periodo.
	Descuentos float64 `json:"descuentos" example:"143"`
	// Cantidad de pagos considerados.
	CantidadPagos int `json:"cantidad_pagos" example:"32"`
	// Cantidad total de servicios vendidos segun el detalle de pagos.
	CantidadServiciosVendidos int `json:"cantidad_servicios_vendidos" example:"45"`
	// Promedio de venta por pago.
	TicketPromedio float64 `json:"ticket_promedio" example:"2651.78"`
	// Servicio con mayor cantidad vendida.
	ServicioMasComprado *PagoResumenServicio `json:"servicio_mas_comprado"`
	// Servicio con mayor monto generado.
	ServicioMasDineroGenera *PagoResumenServicio `json:"servicio_mas_dinero_genera"`
	// Totales agrupados por tipo de pago.
	VentasPorTipoPago []PagoResumenTipoPago `json:"ventas_por_tipo_pago"`
}

type PagoResumenResponse struct {
	// Filtros aplicados al reporte.
	Filtros PagoResumenFiltros `json:"filtros"`
	// Reporte general o reporte del local filtrado.
	Reporte PagoResumenReporte `json:"reporte"`
	// Reportes por local; vacio cuando se filtra por un local.
	DetalleReportes []PagoResumenReporte `json:"detalle_reportes"`
}

func (s *PagosService) GetResumenPagos(input ResumenPagosInput) (*PagoResumenResponse, error) {
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
	rows, err := s.repo.GetResumenPagos(repository.FiltroResumenPagos{
		FechaDesde: fechaDesde,
		FechaHasta: fechaHasta,
		Local:      local,
	})
	if err != nil {
		return nil, err
	}

	general := newPagoResumenCollector()
	locales := map[string]*pagoResumenCollector{}
	localFiltroReal := local

	for _, row := range rows {
		general.add(row)

		localNombre := strings.TrimSpace(row.LocalNombre)
		if localNombre == "" {
			localNombre = "Sin local"
		}
		if local != "" && localFiltroReal == local {
			localFiltroReal = localNombre
		}

		if _, ok := locales[localNombre]; !ok {
			locales[localNombre] = newPagoResumenCollector()
		}
		locales[localNombre].add(row)
	}

	tipoReporte := "general"
	reporteLocal := ""
	if local != "" {
		tipoReporte = "Local: " + localFiltroReal
		reporteLocal = localFiltroReal
	}

	detalles := []PagoResumenReporte{}
	if local == "" {
		nombresLocales := make([]string, 0, len(locales))
		for nombre := range locales {
			nombresLocales = append(nombresLocales, nombre)
		}
		sort.Strings(nombresLocales)

		for _, nombre := range nombresLocales {
			detalles = append(detalles, locales[nombre].toReport("Local: "+nombre, nombre))
		}
	}

	return &PagoResumenResponse{
		Filtros: PagoResumenFiltros{
			FechaDesde: fechaDesde.Format("2006-01-02"),
			FechaHasta: fechaHasta.Format("2006-01-02"),
			Local:      local,
		},
		Reporte:         general.toReport(tipoReporte, reporteLocal),
		DetalleReportes: detalles,
	}, nil
}

func parseResumenPagoDate(raw, field string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New(field + " es requerido")
	}

	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, errors.New(field + " debe tener formato YYYY-MM-DD")
	}

	return parsed, nil
}

type pagoResumenCollector struct {
	pagos             map[int]bool
	servicios         map[string]*pagoResumenServicioAggregate
	tipoPagos         map[string]*pagoResumenTipoPagoAggregate
	subtotal          float64
	descuentos        float64
	total             float64
	cantidad          int
	serviciosCantidad int
}

type pagoResumenServicioAggregate struct {
	servicio   string
	cantidad   int
	montoTotal float64
}

type pagoResumenTipoPagoAggregate struct {
	tipoPago string
	cantidad int
	total    float64
}

func newPagoResumenCollector() *pagoResumenCollector {
	return &pagoResumenCollector{
		pagos:     map[int]bool{},
		servicios: map[string]*pagoResumenServicioAggregate{},
		tipoPagos: map[string]*pagoResumenTipoPagoAggregate{},
	}
}

func (c *pagoResumenCollector) add(row repository.PagoResumenRow) {
	if !c.pagos[row.PagoID] {
		c.pagos[row.PagoID] = true
		c.subtotal += row.Subtotal
		c.descuentos += row.Descuento
		c.total += row.TotalFinal
		c.cantidad++

		tipoPago := strings.TrimSpace(row.TipoPago)
		if tipoPago == "" {
			tipoPago = "sin_tipo"
		}
		if _, ok := c.tipoPagos[tipoPago]; !ok {
			c.tipoPagos[tipoPago] = &pagoResumenTipoPagoAggregate{tipoPago: tipoPago}
		}
		c.tipoPagos[tipoPago].cantidad++
		c.tipoPagos[tipoPago].total += row.TotalFinal
	}

	if !row.Servicio.Valid {
		return
	}

	servicio := strings.TrimSpace(row.Servicio.String)
	if servicio == "" {
		return
	}

	cantidad := 0
	if row.Cantidad.Valid {
		cantidad = int(row.Cantidad.Int64)
	}
	monto := 0.0
	if row.DetalleSubtotal.Valid {
		monto = row.DetalleSubtotal.Float64
	}

	key := strings.ToUpper(servicio)
	if _, ok := c.servicios[key]; !ok {
		c.servicios[key] = &pagoResumenServicioAggregate{servicio: servicio}
	}
	c.servicios[key].cantidad += cantidad
	c.servicios[key].montoTotal += monto
	c.serviciosCantidad += cantidad
}

func (c *pagoResumenCollector) toReport(tipoReporte, local string) PagoResumenReporte {
	ticketPromedio := 0.0
	if c.cantidad > 0 {
		ticketPromedio = c.total / float64(c.cantidad)
	}

	return PagoResumenReporte{
		TipoReporte:               tipoReporte,
		Local:                     local,
		TotalPeriodo:              c.total,
		Subtotal:                  c.subtotal,
		Descuentos:                c.descuentos,
		CantidadPagos:             c.cantidad,
		CantidadServiciosVendidos: c.serviciosCantidad,
		TicketPromedio:            ticketPromedio,
		ServicioMasComprado:       c.topServicioPorCantidad(),
		ServicioMasDineroGenera:   c.topServicioPorMonto(),
		VentasPorTipoPago:         c.ventasPorTipoPago(),
	}
}

func (c *pagoResumenCollector) topServicioPorCantidad() *PagoResumenServicio {
	var best *pagoResumenServicioAggregate
	for _, current := range c.servicios {
		if best == nil ||
			current.cantidad > best.cantidad ||
			(current.cantidad == best.cantidad && current.montoTotal > best.montoTotal) ||
			(current.cantidad == best.cantidad && current.montoTotal == best.montoTotal && current.servicio < best.servicio) {
			best = current
		}
	}

	return pagoResumenServicioToResponse(best)
}

func (c *pagoResumenCollector) topServicioPorMonto() *PagoResumenServicio {
	var best *pagoResumenServicioAggregate
	for _, current := range c.servicios {
		if best == nil ||
			current.montoTotal > best.montoTotal ||
			(current.montoTotal == best.montoTotal && current.cantidad > best.cantidad) ||
			(current.montoTotal == best.montoTotal && current.cantidad == best.cantidad && current.servicio < best.servicio) {
			best = current
		}
	}

	return pagoResumenServicioToResponse(best)
}

func pagoResumenServicioToResponse(agg *pagoResumenServicioAggregate) *PagoResumenServicio {
	if agg == nil {
		return nil
	}

	return &PagoResumenServicio{
		Servicio:   agg.servicio,
		Cantidad:   agg.cantidad,
		MontoTotal: agg.montoTotal,
	}
}

func (c *pagoResumenCollector) ventasPorTipoPago() []PagoResumenTipoPago {
	tipos := make([]string, 0, len(c.tipoPagos))
	for tipo := range c.tipoPagos {
		tipos = append(tipos, tipo)
	}
	sort.Strings(tipos)

	resultado := make([]PagoResumenTipoPago, 0, len(tipos))
	for _, tipo := range tipos {
		agg := c.tipoPagos[tipo]
		resultado = append(resultado, PagoResumenTipoPago{
			TipoPago:      agg.tipoPago,
			CantidadPagos: agg.cantidad,
			Total:         agg.total,
		})
	}

	return resultado
}
