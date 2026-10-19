package models

import "time"

// ─── Inventario ───────────────────────────────────────────────────────────────

// CategoriaInventario agrupa items de inventario.
type CategoriaInventario struct {
	ID     int    `db:"id" json:"id" example:"1"`
	Nombre string `db:"nombre" json:"nombre" example:"Cremas y geles"`
	Activo bool   `db:"activo" json:"activo" example:"true"`
}

// StockLocal es la cantidad de un item en un local.
type StockLocal struct {
	LocalID     int     `db:"local_id" json:"local_id" example:"1"`
	LocalNombre string  `db:"local_nombre" json:"local_nombre" example:"PASEO ARANJUEZ"`
	Cantidad    float64 `db:"cantidad" json:"cantidad" example:"120.5"`
}

// InventarioItem es un producto/insumo del catalogo con su stock.
type InventarioItem struct {
	ID              int      `db:"id" json:"id" example:"1"`
	Codigo          string   `db:"codigo" json:"codigo" example:"INV-000001"`
	Nombre          string   `db:"nombre" json:"nombre" example:"Gel conductor"`
	Descripcion     *string  `db:"descripcion" json:"descripcion,omitempty"`
	CategoriaID     *int     `db:"categoria_id" json:"categoria_id,omitempty"`
	CategoriaNombre *string  `db:"categoria_nombre" json:"categoria_nombre,omitempty"`
	UnidadMedida    string   `db:"unidad_medida" json:"unidad_medida" example:"ML"`
	CostoReferencia *float64 `db:"costo_referencia" json:"costo_referencia,omitempty" example:"0.04"`
	StockMinimo     float64  `db:"stock_minimo" json:"stock_minimo" example:"500"`
	// StockActual es el stock del local consultado, o la suma de todos los locales.
	StockActual float64 `db:"stock_actual" json:"stock_actual" example:"350"`
	// BajoStock es true cuando StockActual <= StockMinimo (RN-05).
	BajoStock bool `db:"-" json:"bajo_stock" example:"false"`
	// ImagenKey es la clave del objeto en el storage (nunca se expone al cliente).
	ImagenKey *string `db:"imagen_key" json:"-"`
	// ImagenURL se calcula en el servicio a partir de ImagenKey.
	ImagenURL     *string      `db:"-" json:"imagen_url,omitempty"`
	Activo        bool         `db:"activo" json:"activo" example:"true"`
	StockPorLocal []StockLocal `db:"-" json:"stock_por_local,omitempty"`
	// LocalID/LocalNombre solo se llenan en las alertas: el stock se evalua por local.
	LocalID     *int    `db:"local_id" json:"local_id,omitempty"`
	LocalNombre *string `db:"local_nombre" json:"local_nombre,omitempty"`
}

// MovimientoInventario es una fila del kardex.
type MovimientoInventario struct {
	ID              int       `db:"id" json:"id" example:"10"`
	ItemID          int       `db:"item_id" json:"item_id" example:"1"`
	ItemNombre      string    `db:"item_nombre" json:"item_nombre" example:"Gel conductor"`
	LocalID         int       `db:"local_id" json:"local_id" example:"1"`
	LocalNombre     string    `db:"local_nombre" json:"local_nombre" example:"PASEO ARANJUEZ"`
	Tipo            string    `db:"tipo" json:"tipo" example:"ENTRADA_COMPRA"`
	Cantidad        float64   `db:"cantidad" json:"cantidad" example:"100"`
	StockResultante float64   `db:"stock_resultante" json:"stock_resultante" example:"450"`
	CostoUnitario   *float64  `db:"costo_unitario" json:"costo_unitario,omitempty"`
	ReservaID       *int      `db:"reserva_id" json:"reserva_id,omitempty"`
	CompraID        *int      `db:"compra_id" json:"compra_id,omitempty"`
	Motivo          *string   `db:"motivo" json:"motivo,omitempty"`
	UsuarioNombre   *string   `db:"usuario_nombre" json:"usuario_nombre,omitempty"`
	CreadoEn        time.Time `db:"creado_en" json:"creado_en"`
}

// ServicioInsumo es la cantidad de un item que consume un servicio por sesion.
type ServicioInsumo struct {
	ItemID            int     `db:"item_id" json:"item_id" example:"1"`
	ItemNombre        string  `db:"item_nombre" json:"item_nombre,omitempty" example:"Gel conductor"`
	UnidadMedida      string  `db:"unidad_medida" json:"unidad_medida,omitempty" example:"ML"`
	CantidadPorSesion float64 `db:"cantidad_por_sesion" json:"cantidad_por_sesion" example:"30"`
}

// AdvertenciaInventario describe un insumo sin stock suficiente (RN-12).
type AdvertenciaInventario struct {
	ItemID     int     `json:"item_id" example:"1"`
	ItemNombre string  `json:"item_nombre" example:"Gel conductor"`
	Requerido  float64 `json:"requerido" example:"30"`
	Disponible float64 `json:"disponible" example:"10"`
	Unidad     string  `json:"unidad" example:"ML"`
}

// AlertaReservaInventario es una reserva futura con falta de insumos.
type AlertaReservaInventario struct {
	ReservaID    int                     `json:"reserva_id" example:"55"`
	Fecha        string                  `json:"fecha" example:"2026-10-20"`
	HoraDesde    string                  `json:"hora_desde" example:"09:00"`
	Cliente      string                  `json:"cliente" example:"Maria Lopez"`
	Servicio     string                  `json:"servicio" example:"Masaje reductor"`
	Local        string                  `json:"local" example:"PASEO ARANJUEZ"`
	Advertencias []AdvertenciaInventario `json:"advertencias"`
}

// AlertasInventario agrupa lo que hay que reponer o revisar.
type AlertasInventario struct {
	ItemsBajoStock    []InventarioItem          `json:"items_bajo_stock"`
	ReservasConAlerta []AlertaReservaInventario `json:"reservas_con_alerta"`
	// Reposicion suma lo que necesitan todas las reservas agendadas futuras y lo compara con el stock.
	Reposicion []ReposicionInsumo `json:"reposicion"`
}

// ReposicionInsumo es la demanda total de un insumo en un local frente a su stock:
// responde cuanto hay que reponer para cumplir todas las reservas agendadas.
type ReposicionInsumo struct {
	ItemID      int     `db:"item_id" json:"item_id"`
	ItemNombre  string  `db:"item_nombre" json:"item_nombre" example:"Gel conductor"`
	Unidad      string  `db:"unidad_medida" json:"unidad" example:"ML"`
	LocalID     int     `db:"local_id" json:"local_id"`
	LocalNombre string  `db:"local_nombre" json:"local_nombre"`
	Stock       float64 `db:"stock" json:"stock" example:"30"`
	// Comprometido es la suma de insumos de las reservas AGENDADAS desde hoy.
	Comprometido float64 `db:"comprometido" json:"comprometido" example:"90"`
	// Faltante = Comprometido - Stock (cuanto comprar para cubrir todas las reservas).
	Faltante float64 `db:"-" json:"faltante" example:"60"`
	Reservas int     `db:"reservas" json:"reservas" example:"3"`
	// Proxima es la fecha de la primera reserva que lo consume (YYYY-MM-DD).
	Proxima string `db:"proxima" json:"proxima" example:"2026-10-19"`
}

// ─── Compras ──────────────────────────────────────────────────────────────────

// CompraDetalle es una linea de compra.
type CompraDetalle struct {
	ItemID        int     `db:"item_id" json:"item_id" example:"1"`
	ItemNombre    string  `db:"item_nombre" json:"item_nombre" example:"Gel conductor"`
	UnidadMedida  string  `db:"unidad_medida" json:"unidad_medida" example:"ML"`
	Cantidad      float64 `db:"cantidad" json:"cantidad" example:"1000"`
	CostoUnitario float64 `db:"costo_unitario" json:"costo_unitario" example:"0.04"`
	Subtotal      float64 `db:"subtotal" json:"subtotal" example:"40"`
}

// Compra es la cabecera de una compra de inventario.
type Compra struct {
	ID                         int             `db:"id" json:"id" example:"1"`
	Codigo                     string          `db:"codigo" json:"codigo" example:"COMP-000001"`
	LocalID                    int             `db:"local_id" json:"local_id" example:"1"`
	LocalNombre                string          `db:"local_nombre" json:"local_nombre" example:"PASEO ARANJUEZ"`
	Proveedor                  *string         `db:"proveedor" json:"proveedor,omitempty"`
	Fecha                      string          `db:"fecha" json:"fecha" example:"2026-10-18"`
	Total                      float64         `db:"total" json:"total" example:"40"`
	Estado                     string          `db:"estado" json:"estado" example:"REGISTRADA"`
	Notas                      *string         `db:"notas" json:"notas,omitempty"`
	NombreUsuario              *string         `db:"nombre_usuario" json:"nombre_usuario,omitempty"`
	Detalle                    []CompraDetalle `db:"-" json:"detalle,omitempty"`
	MovimientoFinancieroCodigo *string         `db:"movimiento_financiero_codigo" json:"movimiento_financiero_codigo,omitempty"`
}

// ─── Finanzas ─────────────────────────────────────────────────────────────────

// CategoriaFinanciera clasifica ingresos y egresos.
type CategoriaFinanciera struct {
	ID         int    `db:"id" json:"id" example:"1"`
	Nombre     string `db:"nombre" json:"nombre" example:"Alquiler"`
	Naturaleza string `db:"naturaleza" json:"naturaleza" example:"EGRESO"`
	Activo     bool   `db:"activo" json:"activo" example:"true"`
}

// MovimientoFinanciero es un egreso (fijo/variable) u otro ingreso.
type MovimientoFinanciero struct {
	ID              int      `db:"id" json:"id" example:"1"`
	Codigo          string   `db:"codigo" json:"codigo" example:"MOV-000001"`
	Naturaleza      string   `db:"naturaleza" json:"naturaleza" example:"EGRESO"`
	Clase           *string  `db:"clase" json:"clase,omitempty" example:"VARIABLE"`
	CategoriaID     int      `db:"categoria_id" json:"categoria_id" example:"1"`
	CategoriaNombre string   `db:"categoria_nombre" json:"categoria_nombre" example:"Alquiler"`
	GastoFijoID     *int     `db:"gasto_fijo_id" json:"gasto_fijo_id,omitempty"`
	Periodo         *string  `db:"periodo" json:"periodo,omitempty" example:"2026-10-01"`
	CompraID        *int     `db:"compra_id" json:"compra_id,omitempty"`
	LocalID         *int     `db:"local_id" json:"local_id,omitempty"`
	LocalNombre     *string  `db:"local_nombre" json:"local_nombre,omitempty"`
	Fecha           string   `db:"fecha" json:"fecha" example:"2026-10-18"`
	Monto           float64  `db:"monto" json:"monto" example:"480"`
	Descripcion     *string  `db:"descripcion" json:"descripcion,omitempty"`
	Proveedor       *string  `db:"proveedor" json:"proveedor,omitempty"`
	MetodoPago      *string  `db:"metodo_pago" json:"metodo_pago,omitempty" example:"efectivo"`
	Estado          string   `db:"estado" json:"estado" example:"PAGADO"`
	MotivoAnulacion *string  `db:"motivo_anulacion" json:"motivo_anulacion,omitempty"`
	NombreUsuario   *string  `db:"nombre_usuario" json:"nombre_usuario,omitempty"`
}

// GastoFijo es la plantilla de un gasto recurrente mensual.
type GastoFijo struct {
	ID              int     `db:"id" json:"id" example:"1"`
	Nombre          string  `db:"nombre" json:"nombre" example:"Alquiler Paseo Aranjuez"`
	CategoriaID     int     `db:"categoria_id" json:"categoria_id" example:"1"`
	CategoriaNombre string  `db:"categoria_nombre" json:"categoria_nombre" example:"Alquiler"`
	Monto           float64 `db:"monto" json:"monto" example:"6500"`
	DiaCobro        int     `db:"dia_cobro" json:"dia_cobro" example:"5"`
	LocalID         *int    `db:"local_id" json:"local_id,omitempty"`
	LocalNombre     *string `db:"local_nombre" json:"local_nombre,omitempty"`
	MetodoPago      *string `db:"metodo_pago" json:"metodo_pago,omitempty" example:"transferencia"`
	Proveedor       *string `db:"proveedor" json:"proveedor,omitempty"`
	FechaInicio     string  `db:"fecha_inicio" json:"fecha_inicio" example:"2026-01-01"`
	FechaFin        *string `db:"fecha_fin" json:"fecha_fin,omitempty"`
	Activo          bool    `db:"activo" json:"activo" example:"true"`
	Notas           *string `db:"notas" json:"notas,omitempty"`
	// ProximoCobro se calcula en el servicio.
	ProximoCobro *string `db:"-" json:"proximo_cobro,omitempty" example:"2026-11-05"`
}

// GastoFijoProximo es un cobro por vencer para el dashboard.
type GastoFijoProximo struct {
	GastoFijoID   int     `json:"gasto_fijo_id" example:"1"`
	Nombre        string  `json:"nombre" example:"Alquiler Paseo Aranjuez"`
	Monto         float64 `json:"monto" example:"6500"`
	FechaCobro    string  `json:"fecha_cobro" example:"2026-11-05"`
	DiasRestantes int     `json:"dias_restantes" example:"3"`
	LocalNombre   *string `json:"local_nombre,omitempty"`
}

// GastoFijoPendiente es un egreso fijo ya vencido que aun no se generó (vista previa de "generar").
type GastoFijoPendiente struct {
	GastoFijoID int     `json:"gasto_fijo_id" example:"1"`
	Nombre      string  `json:"nombre" example:"Alquiler Paseo Aranjuez"`
	Categoria   string  `json:"categoria" example:"Alquiler"`
	Periodo     string  `json:"periodo" example:"2026-09"`
	FechaCobro  string  `json:"fecha_cobro" example:"2026-09-05"`
	Monto       float64 `json:"monto" example:"6500"`
	LocalNombre *string `json:"local_nombre,omitempty" example:"PASEO ARANJUEZ"`
}

// ─── Reporte financiero ───────────────────────────────────────────────────────

// ReporteFinancieroResumen agrega ingresos, egresos y utilidad de un periodo.
type ReporteFinancieroResumen struct {
	IngresosVentas    float64 `json:"ingresos_ventas" example:"84857"`
	IngresosOtros     float64 `json:"ingresos_otros" example:"1200"`
	IngresosTotal     float64 `json:"ingresos_total" example:"86057"`
	EgresosFijos      float64 `json:"egresos_fijos" example:"30000"`
	EgresosVariables  float64 `json:"egresos_variables" example:"12500"`
	EgresosTotal      float64 `json:"egresos_total" example:"42500"`
	EgresosPendientes float64 `json:"egresos_pendientes" example:"4000"`
	UtilidadNeta      float64 `json:"utilidad_neta" example:"43557"`
	MargenPct         float64 `json:"margen_pct" example:"50.6"`
}

// EgresoPorCategoria desglosa egresos pagados por categoria.
type EgresoPorCategoria struct {
	Categoria string  `db:"categoria" json:"categoria" example:"Sueldos y personal"`
	Clase     string  `db:"clase" json:"clase" example:"FIJO"`
	Total     float64 `db:"total" json:"total" example:"18000"`
}

// PuntoSerieFinanciera es un periodo de la serie ingresos vs egresos.
type PuntoSerieFinanciera struct {
	Periodo  string  `json:"periodo" example:"2026-10"`
	Ingresos float64 `json:"ingresos" example:"86057"`
	Egresos  float64 `json:"egresos" example:"42500"`
	Utilidad float64 `json:"utilidad" example:"43557"`
}

// DetalleLocalFinanciero es el resumen de un local.
type DetalleLocalFinanciero struct {
	Local   string                   `json:"local" example:"PASEO ARANJUEZ"`
	Resumen ReporteFinancieroResumen `json:"resumen"`
}

// ReporteInventarioResumen resume el estado del inventario en el reporte.
type ReporteInventarioResumen struct {
	ValorStock       float64 `json:"valor_stock" example:"5400"`
	ItemsBajoStock   int     `json:"items_bajo_stock" example:"3"`
	ConsumoServicios float64 `json:"consumo_servicios" example:"2100"`
}

// ReporteFinancieroFiltros devuelve los filtros aplicados.
type ReporteFinancieroFiltros struct {
	FechaDesde string `json:"fecha_desde" example:"2026-10-01"`
	FechaHasta string `json:"fecha_hasta" example:"2026-10-31"`
	Local      string `json:"local" example:"PASEO ARANJUEZ"`
}

// ReporteFinanciero es la respuesta completa de /bd/reportes/financiero.
type ReporteFinanciero struct {
	Filtros             ReporteFinancieroFiltros `json:"filtros"`
	Resumen             ReporteFinancieroResumen `json:"resumen"`
	EgresosPorCategoria []EgresoPorCategoria     `json:"egresos_por_categoria"`
	Serie               []PuntoSerieFinanciera   `json:"serie"`
	Inventario          ReporteInventarioResumen `json:"inventario"`
	GastosGenerales     float64                  `json:"gastos_generales" example:"9000"`
	DetallePorLocal     []DetalleLocalFinanciero `json:"detalle_por_local"`
}
