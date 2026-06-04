package repository

import "atrevida-agenda-api/models"

type FiltroPagos struct {
	CodigoPago    string
	LocalID       *int
	LocalNombre   string
	ClienteID     *int
	ClienteNIT    string
	ClienteNombre string
	Estado        string
	Activo        *bool
}

type CrearPagoInput struct {
	LocalID       int
	LocalNombre   string
	ClienteID     *int
	ClienteNIT    string
	ClienteNombre string
	Subtotal      *float64
	Descuento     *float64
	TotalFinal    *float64
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
	ClienteNombre *string
	Subtotal      *float64
	Descuento     *float64
	TotalFinal    *float64
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

type PagosRepository interface {
	GetPagos(filtro FiltroPagos) ([]models.PagoPG, error)
	GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error)
	CreatePago(input CrearPagoInput) (string, error)
	UpdatePago(input ActualizarPagoInput) error
	DeletePago(codigoPago string) error
}
