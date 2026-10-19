package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

var metodosPago = map[string]bool{"efectivo": true, "qr": true, "transferencia": true, "debito_automatico": true}

type FinanzasService struct {
	repo   *pgsqlrepo.FinanzasRepo
	config *ConfiguracionService
	// AlCambiarStock es opcional: se invoca tras registrar o anular una compra, que mueven stock.
	AlCambiarStock func(localID *int)
}

func NewFinanzasService(repo *pgsqlrepo.FinanzasRepo, config *ConfiguracionService) *FinanzasService {
	return &FinanzasService{repo: repo, config: config}
}

func validarMetodoPago(m *string) (*string, error) {
	if m == nil || strings.TrimSpace(*m) == "" {
		return nil, nil
	}
	v := strings.ToLower(strings.TrimSpace(*m))
	if !metodosPago[v] {
		return nil, errors.New("metodo_pago invalido, valores permitidos: efectivo, qr, transferencia, debito_automatico")
	}
	return &v, nil
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// ─── Categorias ───────────────────────────────────────────────────────────────

func (s *FinanzasService) ListarCategorias() ([]models.CategoriaFinanciera, error) {
	return s.repo.ListCategorias()
}

func (s *FinanzasService) CrearCategoria(nombre, naturaleza string) (int, error) {
	nombre = strings.TrimSpace(nombre)
	naturaleza = strings.ToUpper(strings.TrimSpace(naturaleza))
	if nombre == "" || len(nombre) > 100 {
		return 0, errors.New("nombre es requerido y no puede superar 100 caracteres")
	}
	if !naturalezaCategoriaValida(naturaleza) {
		return 0, errors.New("naturaleza invalida, valores permitidos: INGRESO, EGRESO, AMBOS")
	}
	return s.repo.CreateCategoria(nombre, naturaleza)
}

// naturalezaCategoriaValida: una categoria puede ser de INGRESO, de EGRESO o de AMBOS.
func naturalezaCategoriaValida(n string) bool {
	return n == "INGRESO" || n == "EGRESO" || n == "AMBOS"
}

func (s *FinanzasService) ActualizarCategoria(id int, nombre *string, activo *bool, naturaleza *string) error {
	if nombre == nil && activo == nil && naturaleza == nil {
		return errors.New("no se enviaron campos para actualizar")
	}
	if nombre != nil {
		n := strings.TrimSpace(*nombre)
		if n == "" || len(n) > 100 {
			return errors.New("nombre es requerido y no puede superar 100 caracteres")
		}
		nombre = &n
	}
	if naturaleza != nil {
		n := strings.ToUpper(strings.TrimSpace(*naturaleza))
		if !naturalezaCategoriaValida(n) {
			return errors.New("naturaleza invalida, valores permitidos: INGRESO, EGRESO, AMBOS")
		}
		naturaleza = &n
	}
	return s.repo.UpdateCategoria(id, nombre, activo, naturaleza)
}

// ─── Movimientos ──────────────────────────────────────────────────────────────

func (s *FinanzasService) ListarMovimientos(f repository.FiltroMovimientosFinancieros) ([]models.MovimientoFinanciero, error) {
	if f.Naturaleza != "" && f.Naturaleza != "INGRESO" && f.Naturaleza != "EGRESO" {
		return nil, errors.New("naturaleza invalida, valores permitidos: INGRESO, EGRESO")
	}
	if f.Clase != "" && f.Clase != "FIJO" && f.Clase != "VARIABLE" {
		return nil, errors.New("clase invalida, valores permitidos: FIJO, VARIABLE")
	}
	if f.Estado != "" && f.Estado != "PENDIENTE" && f.Estado != "PAGADO" && f.Estado != "ANULADO" {
		return nil, errors.New("estado invalido, valores permitidos: PENDIENTE, PAGADO, ANULADO")
	}
	// Con sistema multi-mes, el listado por defecto incluye todo; el front manda el rango del mes.
	return s.repo.ListMovimientos(f)
}

type MovimientoFinancieroInput struct {
	Naturaleza    string
	Clase         string
	CategoriaID   int
	LocalID       *int
	Fecha         string
	Monto         float64
	Descripcion   *string
	Proveedor     *string
	MetodoPago    *string
	Estado        string
	UsuarioID     *int
	UsuarioNombre string
}

// categoriaCompatible valida que la categoria exista, este activa y sirva para la naturaleza (las de tipo AMBOS sirven para ambas).
func (s *FinanzasService) categoriaCompatible(categoriaID int, naturaleza string) error {
	cat, err := s.repo.GetCategoria(categoriaID)
	if err != nil {
		return err
	}
	if !cat.Activo {
		return errors.New("la categoria esta inactiva")
	}
	if cat.Naturaleza != "AMBOS" && cat.Naturaleza != naturaleza {
		return fmt.Errorf("la categoria %q es de tipo %s y no puede usarse para un %s", cat.Nombre, cat.Naturaleza, naturaleza)
	}
	return nil
}

// RegistrarMovimiento crea un egreso variable u otro ingreso (RN-21/RN-22). Los egresos FIJO solo nacen
// de una plantilla; las compras de inventario nacen de POST /compras.
func (s *FinanzasService) RegistrarMovimiento(in MovimientoFinancieroInput) (*models.MovimientoFinanciero, error) {
	naturaleza := strings.ToUpper(strings.TrimSpace(in.Naturaleza))
	if naturaleza != "INGRESO" && naturaleza != "EGRESO" {
		return nil, errors.New("naturaleza invalida, valores permitidos: INGRESO, EGRESO")
	}
	var clase *string
	if naturaleza == "EGRESO" {
		c := strings.ToUpper(strings.TrimSpace(in.Clase))
		if c == "" {
			c = "VARIABLE"
		}
		if c != "VARIABLE" {
			return nil, errors.New("solo se pueden registrar manualmente egresos VARIABLE; los fijos se generan desde un gasto fijo")
		}
		clase = &c
	}
	if in.Monto <= 0 {
		return nil, errors.New("monto debe ser mayor a 0")
	}
	fecha, err := time.Parse("2006-01-02", strings.TrimSpace(in.Fecha))
	if err != nil {
		return nil, errors.New("fecha invalida, use formato YYYY-MM-DD")
	}
	estado := strings.ToUpper(strings.TrimSpace(in.Estado))
	if estado == "" {
		estado = "PAGADO"
	}
	if estado != "PAGADO" && estado != "PENDIENTE" {
		return nil, errors.New("estado invalido al registrar, valores permitidos: PAGADO, PENDIENTE")
	}
	metodo, err := validarMetodoPago(in.MetodoPago)
	if err != nil {
		return nil, err
	}
	if err := s.categoriaCompatible(in.CategoriaID, naturaleza); err != nil {
		return nil, err
	}
	id, err := s.repo.CreateMovimiento(pgsqlrepo.MovimientoInput{
		Naturaleza: naturaleza, Clase: clase, CategoriaID: in.CategoriaID, LocalID: in.LocalID, Fecha: fecha,
		Monto: redondearMonto(in.Monto), Descripcion: trimPtr(in.Descripcion), Proveedor: trimPtr(in.Proveedor),
		MetodoPago: metodo, Estado: estado, UsuarioID: in.UsuarioID, UsuarioNombre: in.UsuarioNombre,
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetMovimiento(id)
}

// MovimientoPatch son los campos editables; los *Set indican si el campo vino (incluso null).
type MovimientoPatch struct {
	CategoriaID    *int
	LocalID        *int
	LocalSet       bool
	Fecha          *string
	Monto          *float64
	Descripcion    *string
	DescripcionSet bool
	Proveedor      *string
	ProveedorSet   bool
	MetodoPago     *string
	MetodoSet      bool
	Estado         *string
}

// EditarMovimiento corrige un movimiento no anulado. Editar un egreso FIJO generado afecta solo ese mes (RN-34).
func (s *FinanzasService) EditarMovimiento(id int, p MovimientoPatch) error {
	actual, err := s.repo.GetMovimiento(id)
	if err != nil {
		return err
	}
	if actual.Estado == "ANULADO" {
		return errors.New("un movimiento anulado no puede editarse")
	}
	if actual.CompraID != nil {
		return errors.New("el egreso pertenece a una compra de inventario: para corregirlo anula la compra")
	}
	patch := pgsqlrepo.MovimientoPatch{
		CategoriaID: p.CategoriaID, LocalID: p.LocalID, LocalSet: p.LocalSet,
		DescripcionSet: p.DescripcionSet, Descripcion: trimPtr(p.Descripcion),
		ProveedorSet: p.ProveedorSet, Proveedor: trimPtr(p.Proveedor),
	}
	if p.Monto != nil {
		if *p.Monto <= 0 {
			return errors.New("monto debe ser mayor a 0")
		}
		m := redondearMonto(*p.Monto)
		patch.Monto = &m
	}
	if p.Fecha != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*p.Fecha))
		if err != nil {
			return errors.New("fecha invalida, use formato YYYY-MM-DD")
		}
		patch.Fecha = &t
	}
	if p.MetodoSet {
		m, err := validarMetodoPago(p.MetodoPago)
		if err != nil {
			return err
		}
		patch.MetodoSet, patch.MetodoPago = true, m
	}
	if p.Estado != nil {
		e := strings.ToUpper(strings.TrimSpace(*p.Estado))
		if e != "PAGADO" && e != "PENDIENTE" {
			return errors.New("estado invalido, valores permitidos: PAGADO, PENDIENTE (para anular usa PATCH /estado)")
		}
		patch.Estado = &e
	}
	if p.CategoriaID != nil {
		if err := s.categoriaCompatible(*p.CategoriaID, actual.Naturaleza); err != nil {
			return err
		}
	}
	return s.repo.UpdateMovimiento(id, patch)
}

// CambiarEstado marca PAGADO o ANULADO (la anulacion exige motivo, RN-23).
func (s *FinanzasService) CambiarEstado(id int, estado string, motivo *string) error {
	estado = strings.ToUpper(strings.TrimSpace(estado))
	if estado != "PAGADO" && estado != "ANULADO" {
		return errors.New("estado invalido, valores permitidos: PAGADO, ANULADO")
	}
	motivo = trimPtr(motivo)
	if estado == "ANULADO" {
		if motivo == nil {
			return errors.New("motivo es requerido para anular")
		}
		actual, err := s.repo.GetMovimiento(id)
		if err != nil {
			return err
		}
		if actual.CompraID != nil {
			return errors.New("el egreso pertenece a una compra de inventario: anula la compra desde Inventario")
		}
	}
	return s.repo.CambiarEstadoMovimiento(id, estado, motivo)
}

// ─── Compras ──────────────────────────────────────────────────────────────────

type CompraLineaInput struct {
	ItemID        int
	Cantidad      float64
	CostoUnitario float64
}

type CompraInput struct {
	LocalID       int
	Proveedor     *string
	Fecha         string
	Notas         *string
	MetodoPago    *string
	Detalle       []CompraLineaInput
	UsuarioID     *int
	UsuarioNombre string
}

func (s *FinanzasService) RegistrarCompra(in CompraInput) (*models.Compra, error) {
	if in.LocalID < 1 {
		return nil, errors.New("local_id es requerido")
	}
	fecha, err := time.Parse("2006-01-02", strings.TrimSpace(in.Fecha))
	if err != nil {
		return nil, errors.New("fecha invalida, use formato YYYY-MM-DD")
	}
	if len(in.Detalle) == 0 {
		return nil, errors.New("detalle es requerido: agrega al menos un item")
	}
	metodo, err := validarMetodoPago(in.MetodoPago)
	if err != nil {
		return nil, err
	}
	lineas := make([]pgsqlrepo.CompraLinea, 0, len(in.Detalle))
	vistos := map[int]bool{}
	for _, l := range in.Detalle {
		if l.ItemID < 1 {
			return nil, errors.New("item_id invalido en el detalle")
		}
		if l.Cantidad <= 0 {
			return nil, errors.New("cantidad debe ser mayor a 0 en cada linea")
		}
		if l.CostoUnitario < 0 {
			return nil, errors.New("costo_unitario no puede ser negativo")
		}
		if vistos[l.ItemID] {
			return nil, fmt.Errorf("el item %d esta repetido en la compra", l.ItemID)
		}
		vistos[l.ItemID] = true
		lineas = append(lineas, pgsqlrepo.CompraLinea{ItemID: l.ItemID, Cantidad: l.Cantidad, CostoUnitario: l.CostoUnitario})
	}
	id, err := s.repo.CrearCompra(pgsqlrepo.CompraInput{
		LocalID: in.LocalID, Proveedor: trimPtr(in.Proveedor), Fecha: fecha, Notas: trimPtr(in.Notas),
		MetodoPago: metodo, Lineas: lineas, UsuarioID: in.UsuarioID, UsuarioNombre: in.UsuarioNombre,
	})
	if err != nil {
		return nil, err
	}
	if s.AlCambiarStock != nil {
		s.AlCambiarStock(&in.LocalID)
	}
	return s.repo.GetCompra(id)
}

func (s *FinanzasService) ListarCompras(localID *int) ([]models.Compra, error) {
	return s.repo.ListCompras(localID)
}

func (s *FinanzasService) ObtenerCompra(id int) (*models.Compra, error) {
	return s.repo.GetCompra(id)
}

func (s *FinanzasService) AnularCompra(id int, motivo string, actor repository.ConsumoActor) error {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return errors.New("motivo es requerido para anular la compra")
	}
	if err := s.repo.AnularCompra(id, motivo, actor); err != nil {
		return err
	}
	if s.AlCambiarStock != nil {
		s.AlCambiarStock(nil)
	}
	return nil
}

func redondearMonto(v float64) float64 { return float64(int64(v*100+0.5*sign(v))) / 100 }

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// FlujoCaja es el libro unificado de ventas, cobros y egresos con totales y comparativa por local.
type FlujoCaja struct {
	Desde       string
	Hasta       string
	Movimientos []models.FlujoMovimiento
	Truncado    bool
	Totales     models.FlujoTotales
	PorLocal    []models.FlujoLocal
}

var (
	flujoOrigenes = map[string]bool{"VENTA": true, "COMPRA": true, "GASTO_FIJO": true, "GASTO_VARIABLE": true, "OTRO_INGRESO": true}
	flujoEstados  = map[string]bool{"PAGADO": true, "PENDIENTE": true, "ANULADO": true}
)

// ObtenerFlujo arma el flujo de caja del rango. Sin fechas usa el mes actual (zona horaria configurada).
func (s *FinanzasService) ObtenerFlujo(desde, hasta *time.Time, localID *int, origen, estado, q string, limit int) (*FlujoCaja, error) {
	origen = strings.ToUpper(strings.TrimSpace(origen))
	estado = strings.ToUpper(strings.TrimSpace(estado))
	if origen != "" && !flujoOrigenes[origen] {
		return nil, errors.New("origen invalido, use VENTA, COMPRA, GASTO_FIJO, GASTO_VARIABLE u OTRO_INGRESO")
	}
	if estado != "" && !flujoEstados[estado] {
		return nil, errors.New("estado invalido, use PAGADO, PENDIENTE o ANULADO")
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	d := time.Date(hoy.Year(), hoy.Month(), 1, 0, 0, 0, 0, time.UTC)
	h := d.AddDate(0, 1, -1)
	if desde != nil {
		d = *desde
	}
	if hasta != nil {
		h = *hasta
	}
	if h.Before(d) {
		return nil, errors.New("fecha_hasta no puede ser anterior a fecha_desde")
	}
	f := pgsqlrepo.FiltroFlujo{Desde: d, Hasta: h, LocalID: localID, Origen: origen, Estado: estado, Q: q, Limit: limit}
	movs, truncado, err := s.repo.ListFlujo(f)
	if err != nil {
		return nil, err
	}
	totales, err := s.repo.TotalesFlujo(f)
	if err != nil {
		return nil, err
	}
	porLocal, err := s.repo.FlujoPorLocal(f)
	if err != nil {
		return nil, err
	}
	return &FlujoCaja{
		Desde: d.Format("2006-01-02"), Hasta: h.Format("2006-01-02"),
		Movimientos: movs, Truncado: truncado, Totales: totales, PorLocal: porLocal,
	}, nil
}
