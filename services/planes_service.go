package services

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

const (
	EstadoPlanReservado  = "RESERVADO"
	EstadoPlanActivo     = "ACTIVO"
	EstadoPlanCompletado = "COMPLETADO"
	EstadoPlanVencido    = "VENCIDO"
	EstadoPlanCancelado  = "CANCELADO"

	TipoPagoUnico  = "UNICO"
	TipoPagoCuotas = "CUOTAS"
)

var (
	ErrPlanInvalido           = errors.New("datos de plan invalidos")
	ErrPlanNoEncontrado       = errors.New("plan no encontrado")
	ErrPlanTransicionInvalida = errors.New("transicion de estado no permitida")
	ErrPlanOrigenInvalido     = errors.New("origen del plan invalido: combo_id y servicios son mutuamente excluyentes")
)

type FiltroPlanes struct {
	Cliente        string
	ClienteID      *int
	LocalID        *int
	Local          string
	Estado         string
	EstadoCobranza string
	FechaDesde     *time.Time
	FechaHasta     *time.Time
}

type PlanServicioInput struct {
	ServicioIDOrigen       *int
	NombreTexto         string
	TiempoTexto         *string
	PrecioUnitarioTexto *float64
	SesionesContratadas    int
	Orden                  int
	SesionNumero           int
}

type CrearPlanInput struct {
	ClienteID      int
	LocalID        int
	ComboID        *int
	Servicios      []PlanServicioInput
	FechaInicio    *time.Time
	FechaFin       *time.Time
	TipoPago       string
	CantidadCuotas int
	Descuento      float64
	Notas          *string
	CreadoPor      *int
	PagoCodigo     *string
}

type ActualizarPlanInput struct {
	ID          int
	Notas       *string
	FechaInicio *time.Time
	FechaFin    *time.Time
}

type CambiarEstadoInput struct {
	ID        int
	Estado    string
	UsuarioID *int
}

type PlanesService struct {
	repo       repository.PlanesRepository
	combosRepo repository.CombosRepository
}

func NewPlanesService(repo repository.PlanesRepository, combosRepo repository.CombosRepository) *PlanesService {
	return &PlanesService{repo: repo, combosRepo: combosRepo}
}

func (s *PlanesService) ListarPlanes(f FiltroPlanes) ([]models.PlanPG, error) {
	planes, err := s.repo.ListPlanes(repository.FiltroPlanes{
		Cliente:        strings.TrimSpace(f.Cliente),
		ClienteID:      f.ClienteID,
		LocalID:        f.LocalID,
		Local:          strings.TrimSpace(f.Local),
		Estado:         strings.TrimSpace(f.Estado),
		EstadoCobranza: strings.TrimSpace(f.EstadoCobranza),
		FechaDesde:     f.FechaDesde,
		FechaHasta:     f.FechaHasta,
	})
	return planes, traducirErrorRepositorioPlan(err)
}

func (s *PlanesService) ObtenerPlan(id int) (*models.PlanCompletoPG, error) {
	if id < 1 {
		return nil, fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	plan, err := s.repo.GetPlanByID(id)
	return plan, traducirErrorRepositorioPlan(err)
}

func (s *PlanesService) CrearPlan(input CrearPlanInput) (int, error) {
	if input.ClienteID < 1 {
		return 0, fmt.Errorf("cliente_id es requerido: %w", ErrPlanInvalido)
	}
	if input.LocalID < 1 {
		return 0, fmt.Errorf("local_id es requerido: %w", ErrPlanInvalido)
	}
	if input.ComboID != nil && len(input.Servicios) > 0 {
		return 0, fmt.Errorf("%w", ErrPlanOrigenInvalido)
	}
	if input.ComboID == nil && len(input.Servicios) == 0 {
		return 0, fmt.Errorf("debe proporcionar combo_id o servicios: %w", ErrPlanInvalido)
	}
	if input.TipoPago != TipoPagoUnico && input.TipoPago != TipoPagoCuotas {
		return 0, fmt.Errorf("tipo_pago debe ser UNICO o CUOTAS: %w", ErrPlanInvalido)
	}
	if input.Descuento < 0 {
		return 0, fmt.Errorf("descuento no puede ser negativo: %w", ErrPlanInvalido)
	}

	var (
		servicios           []repository.CrearPlanServicioInput
		comboIDOrigen       *int
		comboNombreTexto *string
		subtotal            float64
		moneda              = "BOB"
	)

	if input.ComboID != nil {
		combo, err := s.combosRepo.GetComboByID(*input.ComboID, false)
		if err != nil {
			return 0, fmt.Errorf("combo no encontrado o inactivo: %w", ErrPlanInvalido)
		}
		comboIDOrigen = &combo.ID
		comboNombreTexto = &combo.Nombre
		moneda = combo.Moneda
		for _, cs := range combo.Servicios {
			pu := cs.Costo
			if pu == nil {
				zero := 0.0
				pu = &zero
			}
			servicios = append(servicios, repository.CrearPlanServicioInput{
				ServicioIDOrigen:       cs.ServicioID,
				NombreTexto:         cs.ServicioNombre,
				TiempoTexto:         cs.Tiempo,
				PrecioUnitarioTexto: pu,
				SesionesContratadas:    cs.Sesiones,
				Orden:                  cs.Orden,
				SesionNumero:           cs.SesionNumero,
			})
			subtotal += *pu * float64(cs.Sesiones)
		}
	} else {
		ordenes := map[int]bool{}
		for _, s := range input.Servicios {
			if strings.TrimSpace(s.NombreTexto) == "" {
				return 0, fmt.Errorf("nombre_snapshot es requerido para cada servicio: %w", ErrPlanInvalido)
			}
			if s.SesionesContratadas < 1 {
				return 0, fmt.Errorf("sesiones_contratadas debe ser positivo: %w", ErrPlanInvalido)
			}
			if s.Orden < 0 || ordenes[s.Orden] {
				return 0, fmt.Errorf("orden debe ser no negativo y unico: %w", ErrPlanInvalido)
			}
			pu := s.PrecioUnitarioTexto
			if pu == nil {
				zero := 0.0
				pu = &zero
			}
			ordenes[s.Orden] = true
			sesionNumero := s.SesionNumero
			if sesionNumero < 1 {
				sesionNumero = 1
			}
			servicios = append(servicios, repository.CrearPlanServicioInput{
				ServicioIDOrigen:       s.ServicioIDOrigen,
				NombreTexto:         s.NombreTexto,
				TiempoTexto:         s.TiempoTexto,
				PrecioUnitarioTexto: pu,
				SesionesContratadas:    s.SesionesContratadas,
				Orden:                  s.Orden,
				SesionNumero:           sesionNumero,
			})
			subtotal += *pu * float64(s.SesionesContratadas)
		}
	}

	subtotal = redondearImportePlan(subtotal)
	if input.Descuento > subtotal {
		return 0, fmt.Errorf("descuento no puede superar el subtotal: %w", ErrPlanInvalido)
	}
	precioTotal := redondearImportePlan(subtotal - input.Descuento)

	var cuotas []repository.CrearPlanCuotaInput
	switch input.TipoPago {
	case TipoPagoUnico:
		cuotas = append(cuotas, repository.CrearPlanCuotaInput{Numero: 1, Monto: precioTotal})
	case TipoPagoCuotas:
		cantidad := input.CantidadCuotas
		if cantidad < 2 {
			cantidad = 2
		}
		montoBase := math.Floor(precioTotal*100/float64(cantidad)) / 100
		sobrante := redondearImportePlan(precioTotal - montoBase*float64(cantidad-1))
		for i := 1; i <= cantidad; i++ {
			monto := montoBase
			if i == cantidad {
				monto = sobrante
			}
			cuotas = append(cuotas, repository.CrearPlanCuotaInput{
				Numero: i,
				Monto:  redondearImportePlan(monto),
			})
		}
	}

	estadoInicial := EstadoPlanReservado
	if input.PagoCodigo != nil && input.TipoPago == TipoPagoUnico && precioTotal > 0 {
		estadoInicial = EstadoPlanActivo
	}

	id, err := s.repo.CreatePlan(repository.CrearPlanInput{
		ClienteID:           input.ClienteID,
		LocalID:             input.LocalID,
		ComboIDOrigen:       comboIDOrigen,
		ComboNombreTexto: comboNombreTexto,
		FechaInicio:         input.FechaInicio,
		FechaFin:            input.FechaFin,
		Estado:              estadoInicial,
		TipoPago:            input.TipoPago,
		Subtotal:            subtotal,
		Descuento:           input.Descuento,
		PrecioTotal:         precioTotal,
		Moneda:              moneda,
		Notas:               input.Notas,
		CreadoPor:           input.CreadoPor,
		Servicios:           servicios,
		Cuotas:              cuotas,
		PagoCodigo:          input.PagoCodigo,
	})
	return id, traducirErrorRepositorioPlan(err)
}

func (s *PlanesService) ActualizarPlan(input ActualizarPlanInput) error {
	if input.ID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	if input.Notas == nil && input.FechaInicio == nil && input.FechaFin == nil {
		return fmt.Errorf("debe especificarse al menos un campo a modificar: %w", ErrPlanInvalido)
	}
	if input.FechaInicio != nil && input.FechaFin != nil && input.FechaFin.Before(*input.FechaInicio) {
		return fmt.Errorf("fecha_fin no puede ser anterior a fecha_inicio: %w", ErrPlanInvalido)
	}
	return traducirErrorRepositorioPlan(s.repo.UpdatePlan(repository.ActualizarPlanInput{
		ID: input.ID, Notas: input.Notas,
		FechaInicio: input.FechaInicio, FechaFin: input.FechaFin,
	}))
}

func (s *PlanesService) CambiarEstado(input CambiarEstadoInput) error {
	if input.ID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	estado := strings.ToUpper(strings.TrimSpace(input.Estado))
	if estado != EstadoPlanActivo && estado != EstadoPlanCompletado && estado != EstadoPlanCancelado {
		return fmt.Errorf("estado debe ser ACTIVO, COMPLETADO o CANCELADO: %w", ErrPlanInvalido)
	}
	return traducirErrorRepositorioPlan(s.repo.UpdatePlanEstado(input.ID, estado, input.UsuarioID))
}

func (s *PlanesService) MarcarSesion(planID, numero int, realizado bool) error {
	if planID < 1 || numero < 1 {
		return fmt.Errorf("id y numero deben ser positivos: %w", ErrPlanInvalido)
	}
	n, err := s.repo.MarcarSesion(planID, numero, realizado)
	if err != nil {
		return traducirErrorRepositorioPlan(err)
	}
	if n == 0 {
		return fmt.Errorf("%w: sesion %d del plan %d", ErrPlanNoEncontrado, numero, planID)
	}
	return nil
}

func (s *PlanesService) CobrarPlan(planID int, pagoCodigo string) error {
	if planID < 1 {
		return fmt.Errorf("id debe ser un entero positivo: %w", ErrPlanInvalido)
	}
	if strings.TrimSpace(pagoCodigo) == "" {
		return fmt.Errorf("pago_codigo es requerido: %w", ErrPlanInvalido)
	}
	return traducirErrorRepositorioPlan(s.repo.CobrarPlan(planID, strings.TrimSpace(pagoCodigo)))
}

func traducirErrorRepositorioPlan(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrPlanNoEncontrado) {
		return fmt.Errorf("%w: %v", ErrPlanNoEncontrado, err)
	}
	if errors.Is(err, repository.ErrPlanDatosInvalidos) {
		return fmt.Errorf("%w: %v", ErrPlanInvalido, err)
	}
	if errors.Is(err, repository.ErrPlanTransicionInvalida) {
		return fmt.Errorf("%w: %v", ErrPlanTransicionInvalida, err)
	}
	if errors.Is(err, repository.ErrPlanEstadoBloqueado) {
		return fmt.Errorf("%w: %v", ErrPlanInvalido, err)
	}
	return err
}

func redondearImportePlan(v float64) float64 {
	return math.Round(v*100) / 100
}
