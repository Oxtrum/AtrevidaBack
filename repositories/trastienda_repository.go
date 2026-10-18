package repository

import (
	"errors"
	"time"
)

// Errores compartidos del modulo de trastienda (inventario + finanzas). El
// handler los mapea por substring del mensaje (ver AGENTS.md).
var (
	ErrItemNoEncontrado   = errors.New("item de inventario no encontrado")
	ErrStockInsuficiente  = errors.New("stock insuficiente")
	ErrCompraNoEncontrada = errors.New("compra no encontrada")
	ErrMovimientoFinNoEnc = errors.New("movimiento financiero no encontrado")
	ErrGastoFijoNoEnc     = errors.New("gasto fijo no encontrado")
	ErrCategoriaNoEnc     = errors.New("categoria no encontrada")
)

// Tipos de movimiento del kardex (RN-03).
const (
	MovEntradaCompra      = "ENTRADA_COMPRA"
	MovSalidaServicio     = "SALIDA_SERVICIO"
	MovReversionServicio  = "REVERSION_SERVICIO"
	MovAjustePositivo     = "AJUSTE_POSITIVO"
	MovAjusteNegativo     = "AJUSTE_NEGATIVO"
	ReservaEstadoAgendado = "AGENDADO"
)

// MovimientoStockInput describe un cambio de stock. Siempre viaja con su fila de
// kardex y dentro de una transaccion (RN-02).
type MovimientoStockInput struct {
	ItemID           int
	LocalID          int
	Tipo             string
	Cantidad         float64 // siempre > 0; el signo lo da el tipo
	CostoUnitario    *float64
	ReservaID        *int
	PlanMovimientoID *int
	CompraID         *int
	Motivo           string
	UsuarioID        *int
	UsuarioNombre    string
	// PermitirNegativo deja el stock bajo cero en lugar de devolver ErrStockInsuficiente.
	PermitirNegativo bool
}

// ConsumoActor identifica quien dispara un descuento/reversion de stock.
type ConsumoActor struct {
	UsuarioID     *int
	UsuarioNombre string
}

// FiltroItemsInventario filtra el listado de items.
type FiltroItemsInventario struct {
	Q           string
	CategoriaID *int
	LocalID     *int
	BajoStock   bool
	Activo      *bool
	Limit       int
	// PorLocal (sin LocalID) devuelve una fila por item y local, sin sumar el stock entre locales.
	PorLocal bool
}

// FiltroMovimientosInventario filtra el kardex.
type FiltroMovimientosInventario struct {
	ItemID     *int
	LocalID    *int
	Tipo       string
	FechaDesde *time.Time
	FechaHasta *time.Time
	// Q busca por item, local o motivo.
	Q     string
	Limit int
	// Paginacion por cursor (creado_en DESC, id DESC): PageLimit > 0 la activa y
	// pide una fila extra para saber si hay pagina siguiente.
	PageLimit  int
	CursorSet  bool
	CursorTime time.Time
	CursorID   int
}

// FiltroMovimientosFinancieros filtra egresos e ingresos manuales.
type FiltroMovimientosFinancieros struct {
	Naturaleza  string
	Clase       string
	CategoriaID *int
	Estado      string
	LocalID     *int
	FechaDesde  *time.Time
	FechaHasta  *time.Time
}
