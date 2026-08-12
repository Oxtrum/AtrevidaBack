package repository

import (
	"context"
	"database/sql"
	"time"

	"atrevida-agenda-api/models"
)

type FiltroPagos struct {
	Context       context.Context
	CodigoPago    string
	LocalID       *int
	LocalNombre   string
	ClienteID     *int
	ClienteNIT    string
	ClienteNombre string
	TipoPago      string
	Estado        string
	Activo        *bool

	IDCajero                   *int
	NombreCajero               string
	UsernameCajero             string
	IDCajeroModificacion       *int
	NombreCajeroModificacion   string
	UsernameCajeroModificacion string
	PageLimit                  int
	CursorSet                  bool
	CursorFecha                time.Time
	CursorID                   int
}

type CrearPagoInput struct {
	LocalID       int
	LocalNombre   string
	ClienteID     *int
	ClienteNIT    *string
	ClienteNombre string
	Subtotal      *float64
	Descuento     *float64
	TotalFinal    *float64
	TipoPago      string
	Estado        string
	Activo        bool
	Cajero        CajeroAuditoriaInput
	Detalle       []CrearDetallePagoInput
}

type CajeroAuditoriaInput struct {
	ID       *int
	Nombre   string
	Username *string
}

type CrearDetallePagoInput struct {
	ServicioID     *int
	Servicio       string
	PrecioUnitario float64
	Cantidad       int
	Subtotal       float64
}

type ActualizarPagoInput struct {
	CodigoPago    string
	LocalID       *int
	LocalNombre   *string
	ClienteID     *int
	ClienteIDSet  bool
	ClienteNIT    *string
	ClienteNITSet bool
	ClienteNombre *string
	Subtotal      *float64
	Descuento     *float64
	TotalFinal    *float64
	TipoPago      *string
	Estado        *string
	Activo        *bool
	Cajero        CajeroAuditoriaInput
	Detalle       *[]ActualizarDetallePagoInput

	RecalcularSubtotal   bool
	RecalcularTotalFinal bool
}

type ActualizarDetallePagoInput struct {
	ID             *int
	ServicioID     *int
	Servicio       string
	PrecioUnitario float64
	Cantidad       int
	Subtotal       float64
}

type FiltroResumenPagos struct {
	Context    context.Context
	FechaDesde time.Time
	FechaHasta time.Time
	Local      string
}

type PagoResumenTotalRow struct {
	LocalNombre               string  `db:"local_nombre"`
	EsGeneral                 bool    `db:"es_general"`
	Subtotal                  float64 `db:"subtotal"`
	Descuento                 float64 `db:"descuento"`
	TotalFinal                float64 `db:"total_final"`
	CantidadPagos             int     `db:"cantidad_pagos"`
	CantidadServiciosVendidos int     `db:"cantidad_servicios_vendidos"`
}

type PagoResumenTipoRow struct {
	LocalNombre   string  `db:"local_nombre"`
	EsGeneral     bool    `db:"es_general"`
	TipoPago      string  `db:"tipo_pago"`
	CantidadPagos int     `db:"cantidad_pagos"`
	Total         float64 `db:"total"`
}

type PagoResumenServicioRow struct {
	LocalNombre string  `db:"local_nombre"`
	EsGeneral   bool    `db:"es_general"`
	Servicio    string  `db:"servicio"`
	Cantidad    int     `db:"cantidad"`
	MontoTotal  float64 `db:"monto_total"`
}

type PagoResumenAgregado struct {
	Totales   []PagoResumenTotalRow
	Tipos     []PagoResumenTipoRow
	Servicios []PagoResumenServicioRow
}

type PagoResumenRow struct {
	PagoID          int             `db:"pago_id"`
	LocalNombre     string          `db:"local_nombre"`
	TipoPago        string          `db:"tipo_pago"`
	Subtotal        float64         `db:"subtotal"`
	Descuento       float64         `db:"descuento"`
	TotalFinal      float64         `db:"total_final"`
	Servicio        sql.NullString  `db:"servicio"`
	Cantidad        sql.NullInt64   `db:"cantidad"`
	DetalleSubtotal sql.NullFloat64 `db:"detalle_subtotal"`
}

type PagosRepository interface {
	GetPagos(filtro FiltroPagos) ([]models.PagoPG, error)
	CountPagos(filtro FiltroPagos) (int, error)
	GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error)
	CreatePago(input CrearPagoInput) (string, error)
	UpdatePago(input ActualizarPagoInput) error
	DeletePago(codigoPago string) error
	GetResumenPagos(filtro FiltroResumenPagos) (PagoResumenAgregado, error)
}
