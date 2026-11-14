package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"atrevida-agenda-api/cuenta"
	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

// cuentaStore es lo que CuentaService necesita de la persistencia; lo cumple *pgsqlrepo.CuentaRepo.
type cuentaStore interface {
	ClienteExiste(clienteID int) (bool, error)
	CrearCargo(in pgsqlrepo.CargoInput) (int, error)
	CrearCargos(in []pgsqlrepo.CargoInput) ([]int, error)
	AnularCargo(id int, motivo string, actor repository.ConsumoActor) error
	ListarCargos(clienteID int, incluirAnulados bool) ([]models.CargoCliente, error)
	ListarAbonos(clienteID int) ([]models.AbonoCliente, error)
	TotalesCuenta(clienteID int) (abonos, cargos float64, err error)
	CrearAbono(in pgsqlrepo.AbonoInput) (string, error)
	AplicarAbono(codigoPago string, clienteID int, aplicaciones []cuenta.Reparto) error
	Cobrar(in pgsqlrepo.CobroInput) (pgsqlrepo.CobroResultado, error)
	ListarDeudores(f pgsqlrepo.FiltroDeudores) ([]models.DeudorCliente, error)
	ContarDeudores(f pgsqlrepo.FiltroDeudores) (int, error)
}

// CuentaService maneja la cuenta del cliente: cargos, abonos y aplicaciones.
type CuentaService struct {
	store cuentaStore
	// Config es opcional: sin el, la fecha del hecho usa los defaults (UTC-4 y sin limite de dias).
	Config ConfigSource
	// Ahora es el reloj; solo se cambia en pruebas.
	Ahora func() time.Time
}

func NewCuentaService(store cuentaStore) *CuentaService {
	return &CuentaService{store: store, Ahora: time.Now}
}

// Tipos de cargo permitidos.
var tiposCargo = map[string]bool{"PAQUETE": true, "SERVICIO": true, "INSUMO": true, "OTRO": true}

const maxConceptoCargo = 300

// Obtener arma la cuenta de un cliente: saldo, cargos con su estado y abonos.
func (s *CuentaService) Obtener(clienteID int, incluirAnulados bool) (*models.CuentaCliente, error) {
	if clienteID < 1 {
		return nil, errors.New("cliente_id es requerido")
	}
	existe, err := s.store.ClienteExiste(clienteID)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, errors.New("cliente no encontrado")
	}
	cargos, err := s.store.ListarCargos(clienteID, incluirAnulados)
	if err != nil {
		return nil, err
	}
	abonos, err := s.store.ListarAbonos(clienteID)
	if err != nil {
		return nil, err
	}
	totalAbonos, totalCargos, err := s.store.TotalesCuenta(clienteID)
	if err != nil {
		return nil, err
	}
	// Lo derivado (estado del cargo, lo que sobra de cada abono) se calcula aqui, en un solo lugar.
	for i := range cargos {
		cargos[i].Completar()
	}
	var disponible, porPagar float64
	for i := range abonos {
		abonos[i].SinAplicar = cuenta.PendienteDe(abonos[i].Monto, abonos[i].Aplicado)
		disponible += abonos[i].SinAplicar
	}
	for _, c := range cargos {
		if !c.Anulado {
			porPagar += c.Pendiente
		}
	}
	return &models.CuentaCliente{
		ClienteID:   clienteID,
		Saldo:       cuenta.PosicionCliente(disponible, porPagar),
		TotalCargos: totalCargos,
		TotalAbonos: totalAbonos,
		Cargos:      cargos,
		Abonos:      abonos,
	}, nil
}

// CrearCargoInput es el pedido de un cargo nuevo.
type CrearCargoInput struct {
	ClienteID int
	LocalID   *int
	Tipo      string
	Concepto  string
	// Monto puede ser 0: un consumo que se registra pero no se cobra.
	Monto float64
	// FechaHecho es cuando ocurrio lo que se cobra (YYYY-MM-DD, opcional; sin ella es ahora). No puede ser futura.
	FechaHecho  string
	ReservaID   *int
	PlanID      *int
	CreadoPorID *int
	CreadoPor   string
	// ItemID y Cantidad: insumo que se cobra (solo tipo INSUMO). Exige LocalID: el stock baja en ese local.
	ItemID   *int
	Cantidad float64
}

func redondearMonto2(v float64) float64 { return math.Round(v*100) / 100 }

// CrearCargo valida y registra un cargo para un cliente.
func (s *CuentaService) CrearCargo(in CrearCargoInput) (int, error) {
	cargo, err := s.prepararCargo(in)
	if err != nil {
		return 0, err
	}
	return s.store.CrearCargo(cargo)
}

// CrearCargos valida y registra varios cargos de un cliente en una sola transaccion (p. ej. lo que se deja a cuenta
// desde la caja): si uno falla no queda ninguno.
func (s *CuentaService) CrearCargos(clienteID int, entradas []CrearCargoInput) ([]int, error) {
	if len(entradas) == 0 {
		return nil, errors.New("cargos es requerido: envia al menos uno")
	}
	cargos := make([]pgsqlrepo.CargoInput, 0, len(entradas))
	for _, in := range entradas {
		in.ClienteID = clienteID
		cargo, err := s.prepararCargo(in)
		if err != nil {
			return nil, err
		}
		cargos = append(cargos, cargo)
	}
	return s.store.CrearCargos(cargos)
}

// prepararCargo valida un cargo y lo deja listo para guardar.
func (s *CuentaService) prepararCargo(in CrearCargoInput) (pgsqlrepo.CargoInput, error) {
	var vacio pgsqlrepo.CargoInput
	if in.ClienteID < 1 {
		return vacio, errors.New("cliente_id es requerido")
	}
	tipo := strings.ToUpper(strings.TrimSpace(in.Tipo))
	if !tiposCargo[tipo] {
		return vacio, errors.New("tipo invalido, valores permitidos: PAQUETE, SERVICIO, INSUMO, OTRO")
	}
	concepto := strings.TrimSpace(in.Concepto)
	if concepto == "" {
		return vacio, errors.New("concepto es requerido")
	}
	if len([]rune(concepto)) > maxConceptoCargo {
		return vacio, fmt.Errorf("concepto no puede superar %d caracteres", maxConceptoCargo)
	}
	if math.IsNaN(in.Monto) || math.IsInf(in.Monto, 0) || in.Monto < 0 {
		return vacio, errors.New("monto no puede ser negativo")
	}
	fechaHecho, err := resolverFechaHechoConfigurada(s.Config, s.Ahora, in.FechaHecho)
	if err != nil {
		return vacio, err
	}
	cargo := pgsqlrepo.CargoInput{
		ClienteID: in.ClienteID, LocalID: in.LocalID, Tipo: tipo, Concepto: concepto,
		Monto: redondearMonto2(in.Monto), FechaHecho: fechaHecho, ReservaID: in.ReservaID, PlanID: in.PlanID,
		CreadoPorID: in.CreadoPorID, CreadoPor: strings.TrimSpace(in.CreadoPor),
	}
	if in.ItemID != nil {
		if err := validarCargoDeInsumo(tipo, in); err != nil {
			return vacio, err
		}
		cantidad := math.Round(in.Cantidad*1000) / 1000
		cargo.ItemID, cargo.Cantidad = in.ItemID, &cantidad
		if cargo.PermitirStockNegativo, err = s.permiteStockNegativo(); err != nil {
			return vacio, err
		}
	}
	return cargo, nil
}

// validarCargoDeInsumo: el insumo solo va en cargos de tipo INSUMO, con local (el stock es por local) y cantidad.
func validarCargoDeInsumo(tipo string, in CrearCargoInput) error {
	if *in.ItemID < 1 {
		return errors.New("item_id invalido")
	}
	if tipo != "INSUMO" {
		return errors.New("item_id solo aplica a cargos de insumo")
	}
	if in.LocalID == nil {
		return errors.New("local_id es requerido en un cargo de insumo: el stock baja en ese local")
	}
	if math.IsNaN(in.Cantidad) || math.IsInf(in.Cantidad, 0) || in.Cantidad <= 0 {
		return errors.New("cantidad debe ser mayor a 0")
	}
	return nil
}

// permiteStockNegativo lee inventario.permitir_stock_negativo; sin configuracion usa el valor por defecto.
func (s *CuentaService) permiteStockNegativo() (bool, error) {
	cfg := configPorDefecto()
	if s.Config != nil {
		snapshot, err := s.Config.Snapshot()
		if err != nil {
			return false, err
		}
		cfg = snapshot
	}
	return cfg.PermitirStockNegativo, nil
}

// AnularCargo anula un cargo con motivo; lo que tenia aplicado vuelve a ser saldo a favor.
// quien anula queda en el kardex si el cargo cobraba un insumo y se devuelve su stock.
func (s *CuentaService) AnularCargo(id int, motivo string, quien CajeroAuditoriaInput) error {
	if id < 1 {
		return errors.New("id debe ser un entero positivo")
	}
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return errors.New("motivo es requerido")
	}
	return s.store.AnularCargo(id, motivo, repository.ConsumoActor{UsuarioID: quien.ID, UsuarioNombre: strings.TrimSpace(quien.Nombre)})
}

// CrearAbonoInput es el pedido de un abono (pago a cuenta) con su reparto.
type CrearAbonoInput struct {
	ClienteID   int
	LocalID     int
	LocalNombre string
	Monto       float64
	TipoPago    string
	// FechaHecho es cuando se entrego el dinero (YYYY-MM-DD, opcional; sin ella es ahora). No puede ser futura.
	FechaHecho string
	// Aplicaciones reparte el abono entre cargos; lo que no se aplica queda como saldo a favor.
	Aplicaciones []cuenta.Reparto
	Cajero       CajeroAuditoriaInput
}

// CrearAbono valida y registra un pago a cuenta. Devuelve el codigo del pago.
func (s *CuentaService) CrearAbono(in CrearAbonoInput) (string, error) {
	if in.ClienteID < 1 {
		return "", errors.New("cliente_id es requerido")
	}
	if in.LocalID < 1 || strings.TrimSpace(in.LocalNombre) == "" {
		return "", errors.New("local_id es requerido")
	}
	if math.IsNaN(in.Monto) || math.IsInf(in.Monto, 0) || in.Monto <= 0 {
		return "", errors.New("monto debe ser mayor a 0")
	}
	tipoPago, err := NormalizarTipoPago(in.TipoPago)
	if err != nil {
		return "", err
	}
	cajero, err := normalizarCajeroAuditoria(in.Cajero)
	if err != nil {
		return "", err
	}
	fechaHecho, err := resolverFechaHechoConfigurada(s.Config, s.Ahora, in.FechaHecho)
	if err != nil {
		return "", err
	}
	return s.store.CrearAbono(pgsqlrepo.AbonoInput{
		ClienteID: in.ClienteID, LocalID: in.LocalID, LocalNombre: strings.TrimSpace(in.LocalNombre),
		Monto: redondearMonto2(in.Monto), TipoPago: tipoPago, FechaHecho: fechaHecho,
		Aplicaciones: in.Aplicaciones,
		CajeroID:     cajero.ID, CajeroNombre: cajero.Nombre, CajeroUser: cajero.Username,
	})
}

// CobrarInput es el pedido de un cobro de caja: los cargos que el cajero marco, si se usa el saldo disponible, el
// metodo del dinero nuevo y una recarga opcional.
type CobrarInput struct {
	ClienteID   int
	LocalID     int
	LocalNombre string
	// CargoIDs son los cargos que se pagan completos; puede ir vacio si solo se recarga saldo.
	CargoIDs []int
	// UsarSaldo descuenta primero el saldo disponible; false si el cliente prefiere guardarlo.
	UsarSaldo bool
	// Recarga es dinero extra que queda como saldo disponible.
	Recarga float64
	// TipoPago es el metodo del dinero nuevo; solo se exige si entra dinero nuevo.
	TipoPago string
	// FechaHecho es cuando ocurrio el cobro (YYYY-MM-DD, opcional; sin ella es ahora). No puede ser futura.
	FechaHecho string
	Cajero     CajeroAuditoriaInput
}

// Cobrar valida y registra un cobro de caja. Las reglas de dinero (que el saldo alcance, que cada cargo se cubra
// completo) las comprueba el repositorio dentro de la transaccion, con los cargos y el saldo bloqueados.
func (s *CuentaService) Cobrar(in CobrarInput) (pgsqlrepo.CobroResultado, error) {
	if in.ClienteID < 1 {
		return pgsqlrepo.CobroResultado{}, errors.New("cliente_id es requerido")
	}
	if in.LocalID < 1 || strings.TrimSpace(in.LocalNombre) == "" {
		return pgsqlrepo.CobroResultado{}, errors.New("local_id es requerido")
	}
	if math.IsNaN(in.Recarga) || math.IsInf(in.Recarga, 0) || in.Recarga < 0 {
		return pgsqlrepo.CobroResultado{}, errors.New("recarga no puede ser negativa")
	}
	recarga := redondearMonto2(in.Recarga)
	if len(in.CargoIDs) == 0 && recarga == 0 {
		return pgsqlrepo.CobroResultado{}, errors.New("cargo_ids o recarga es requerido")
	}
	vistos := make(map[int]bool, len(in.CargoIDs))
	for _, id := range in.CargoIDs {
		if id < 1 {
			return pgsqlrepo.CobroResultado{}, errors.New("cargo_ids invalido: cada id debe ser un entero positivo")
		}
		if vistos[id] {
			return pgsqlrepo.CobroResultado{}, fmt.Errorf("cargo_ids invalido: el cargo %d esta repetido", id)
		}
		vistos[id] = true
	}
	tipoPago, err := normalizarTipoPagoOpcional(in.TipoPago)
	if err != nil {
		return pgsqlrepo.CobroResultado{}, err
	}
	cajero, err := normalizarCajeroAuditoria(in.Cajero)
	if err != nil {
		return pgsqlrepo.CobroResultado{}, err
	}
	fechaHecho, err := resolverFechaHechoConfigurada(s.Config, s.Ahora, in.FechaHecho)
	if err != nil {
		return pgsqlrepo.CobroResultado{}, err
	}
	return s.store.Cobrar(pgsqlrepo.CobroInput{
		ClienteID: in.ClienteID, LocalID: in.LocalID, LocalNombre: strings.TrimSpace(in.LocalNombre),
		CargoIDs: in.CargoIDs, UsarSaldo: in.UsarSaldo, Recarga: recarga, TipoPago: tipoPago, FechaHecho: fechaHecho,
		CajeroID: cajero.ID, CajeroNombre: cajero.Nombre, CajeroUser: cajero.Username,
	})
}

// AplicarAbono reparte entre cargos lo que un abono existente tiene sin aplicar.
func (s *CuentaService) AplicarAbono(codigoPago string, clienteID int, aplicaciones []cuenta.Reparto) error {
	if strings.TrimSpace(codigoPago) == "" {
		return errors.New("codigo_pago es requerido")
	}
	if clienteID < 1 {
		return errors.New("cliente_id es requerido")
	}
	if len(aplicaciones) == 0 {
		return errors.New("aplicaciones es requerido")
	}
	return s.store.AplicarAbono(strings.TrimSpace(codigoPago), clienteID, aplicaciones)
}

// FiltroDeudoresInput filtra y pagina el listado de clientes con deuda.
type FiltroDeudoresInput struct {
	Busqueda string
	// OrdenarPor es una clave de pgsql.DeudoresOrdenables; vacio = mayor deuda primero.
	OrdenarPor string
	OrdenDesc  bool
	// PageLimit es el tamano de pagina mas una fila extra (0 = sin paginacion).
	PageLimit int
	Offset    int
}

func (f FiltroDeudoresInput) aRepo() (pgsqlrepo.FiltroDeudores, error) {
	if f.OrdenarPor != "" {
		if _, ok := pgsqlrepo.DeudoresOrdenables[f.OrdenarPor]; !ok {
			return pgsqlrepo.FiltroDeudores{}, errors.New("ordenar_por invalido")
		}
	}
	if f.Offset < 0 {
		return pgsqlrepo.FiltroDeudores{}, errors.New("offset invalido")
	}
	return pgsqlrepo.FiltroDeudores{
		Busqueda: strings.TrimSpace(f.Busqueda), OrdenarPor: f.OrdenarPor, OrdenDesc: f.OrdenDesc,
		Limit: f.PageLimit, Offset: f.Offset,
	}, nil
}

// ListarDeudores devuelve los clientes que deben, con la mayor deuda primero salvo que se pida otro orden.
func (s *CuentaService) ListarDeudores(f FiltroDeudoresInput) ([]models.DeudorCliente, error) {
	filtro, err := f.aRepo()
	if err != nil {
		return nil, err
	}
	return s.store.ListarDeudores(filtro)
}

// ContarDeudores cuenta los clientes con deuda que cumplen la busqueda.
func (s *CuentaService) ContarDeudores(f FiltroDeudoresInput) (int, error) {
	filtro, err := f.aRepo()
	if err != nil {
		return 0, err
	}
	return s.store.ContarDeudores(filtro)
}
