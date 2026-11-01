package models

import (
	"time"

	"atrevida-agenda-api/cuenta"
)

// CargoCliente es lo que un cliente debe: un paquete, un servicio, un insumo u otro concepto.
type CargoCliente struct {
	// ID del cargo.
	ID int `db:"id" json:"id" example:"12"`
	// ID del cliente que debe el cargo.
	ClienteID int `db:"cliente_id" json:"cliente_id" example:"7"`
	// Local donde ocurrio el hecho (opcional).
	LocalID *int `db:"local_id" json:"local_id,omitempty" example:"1"`
	// Tipo del cargo: PAQUETE, SERVICIO, INSUMO u OTRO.
	Tipo string `db:"tipo" json:"tipo" example:"PAQUETE" enums:"PAQUETE,SERVICIO,INSUMO,OTRO"`
	// Concepto legible del cargo.
	Concepto string `db:"concepto" json:"concepto" example:"Paquete Relax 4 sesiones"`
	// Monto del cargo; puede ser 0 para un consumo que no se cobra.
	Monto float64 `db:"monto" json:"monto" example:"1000"`
	// Cuanto de este cargo ya cubren los abonos aplicados.
	Pagado float64 `db:"pagado" json:"pagado" example:"400"`
	// Lo que falta por cubrir (monto menos pagado); se calcula.
	Pendiente float64 `db:"-" json:"pendiente" example:"600"`
	// Estado derivado: PENDIENTE, PARCIAL o PAGADO; se calcula.
	Estado string `db:"-" json:"estado" example:"PARCIAL" enums:"PENDIENTE,PARCIAL,PAGADO"`
	// Fecha del hecho: cuando ocurrio lo que se cobra.
	FechaHecho time.Time `db:"fecha_hecho" json:"fecha_hecho" example:"2026-10-25T12:00:00Z"`
	// Momento real en que se registro el cargo; lo pone el servidor.
	RegistradoEn time.Time `db:"registrado_en" json:"registrado_en" example:"2026-10-28T21:30:00Z"`
	// Reserva de la que nace el cargo (opcional).
	ReservaID *int `db:"reserva_id" json:"reserva_id,omitempty" example:"44"`
	// Plan del que nace el cargo (opcional).
	PlanID *int `db:"plan_id" json:"plan_id,omitempty" example:"21"`
	// Si el cargo fue anulado; un cargo anulado no cuenta en el saldo.
	Anulado bool `db:"anulado" json:"anulado" example:"false"`
	// Motivo de la anulacion.
	MotivoAnulacion *string `db:"motivo_anulacion" json:"motivo_anulacion,omitempty"`
}

// Completar calcula lo derivado: pendiente y estado.
func (c *CargoCliente) Completar() {
	c.Pendiente = cuenta.PendienteDe(c.Monto, c.Pagado)
	c.Estado = cuenta.EstadoCargo(c.Monto, c.Pagado)
}

// AbonoCliente es un pago a cuenta de un cliente, repartido (total o parcialmente) entre sus cargos.
type AbonoCliente struct {
	// Codigo publico del pago.
	CodigoPago string `db:"codigo_pago" json:"codigo_pago" example:"PAGO-000123"`
	// Monto entregado.
	Monto float64 `db:"monto" json:"monto" example:"500"`
	// Metodo de pago: efectivo, tarjeta, qr o transferencia.
	TipoPago string `db:"tipo_pago" json:"tipo_pago" example:"efectivo"`
	// Fecha del hecho: cuando se entrego el dinero.
	FechaHecho time.Time `db:"fecha_hecho" json:"fecha_hecho" example:"2026-10-25T12:00:00Z"`
	// Momento real en que se registro; lo pone el servidor.
	RegistradoEn time.Time `db:"registrado_en" json:"registrado_en" example:"2026-10-28T21:30:00Z"`
	// Cuanto del abono ya cubrio de cargos.
	Aplicado float64 `db:"aplicado" json:"aplicado" example:"400"`
	// Lo que queda sin usar de este abono (parte del saldo disponible); se calcula.
	SinAplicar float64 `db:"-" json:"sin_aplicar" example:"100"`
}

// CuentaCliente es la cuenta completa de un cliente: saldo, cargos y abonos.
type CuentaCliente struct {
	// ID del cliente.
	ClienteID int `json:"cliente_id" example:"7"`
	// Saldo: dinero disponible, lo que hay por pagar y la deuda real (por pagar menos disponible).
	Saldo cuenta.Saldo `json:"saldo"`
	// Total de cargos no anulados.
	TotalCargos float64 `json:"total_cargos" example:"1030"`
	// Total de abonos activos y pagados.
	TotalAbonos float64 `json:"total_abonos" example:"500"`
	// Cargos del cliente con su estado, del mas antiguo al mas nuevo.
	Cargos []CargoCliente `json:"cargos"`
	// Abonos del cliente, del mas reciente al mas antiguo.
	Abonos []AbonoCliente `json:"abonos"`
}

// DeudorCliente es un cliente con deuda: lo que debe, lo cargado y lo abonado.
type DeudorCliente struct {
	// ID del cliente.
	ClienteID int `db:"cliente_id" json:"cliente_id" example:"7"`
	// Nombre del cliente.
	Nombre string `db:"nombre" json:"nombre" example:"Maria"`
	// Apellido del cliente.
	Apellido string `db:"apellido" json:"apellido" example:"Lopez"`
	// Telefono del cliente.
	NumeroTelefono string `db:"numero_telefono" json:"numero_telefono" example:"70011223"`
	// Total de cargos no anulados.
	TotalCargos float64 `db:"total_cargos" json:"total_cargos" example:"1030"`
	// Total de abonos activos y pagados.
	TotalAbonos float64 `db:"total_abonos" json:"total_abonos" example:"500"`
	// Lo que debe: cargos menos abonos; siempre mayor a 0 en este listado.
	Deuda float64 `db:"deuda" json:"deuda" example:"530"`
}
