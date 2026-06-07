package repository

import (
	"database/sql"
	"time"

	"atrevida-agenda-api/models"
)

type FiltroPagos struct {
	CodigoPago    string
	LocalID       *int
	LocalNombre   string
	ClienteID     *int
	ClienteNIT    string
	ClienteNombre string
	TipoPago      string
	Estado        string
	Activo        *bool
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
	Detalle       []CrearDetallePagoInput
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
	FechaDesde time.Time
	FechaHasta time.Time
	Local      string
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
	GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error)
	CreatePago(input CrearPagoInput) (string, error)
	UpdatePago(input ActualizarPagoInput) error
	DeletePago(codigoPago string) error
	GetResumenPagos(filtro FiltroResumenPagos) ([]PagoResumenRow, error)
}
