package services

import (
	"errors"
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type PagosService struct {
	repo repository.PagosRepository
}

func NewPagosService(repo repository.PagosRepository) *PagosService {
	return &PagosService{repo: repo}
}

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

func (s *PagosService) GetPagos(filtro FiltroPagos) ([]models.PagoPG, error) {
	estado, err := normalizarEstadoPagoOpcional(filtro.Estado)
	if err != nil {
		return nil, err
	}

	return s.repo.GetPagos(repository.FiltroPagos{
		CodigoPago:    strings.TrimSpace(filtro.CodigoPago),
		LocalID:       filtro.LocalID,
		LocalNombre:   strings.TrimSpace(filtro.LocalNombre),
		ClienteID:     filtro.ClienteID,
		ClienteNIT:    strings.TrimSpace(filtro.ClienteNIT),
		ClienteNombre: strings.TrimSpace(filtro.ClienteNombre),
		Estado:        estado,
		Activo:        filtro.Activo,
	})
}

func (s *PagosService) GetPagoByCodigo(codigoPago string) (*models.PagoCompletoPG, error) {
	codigoPago = strings.TrimSpace(codigoPago)
	if codigoPago == "" {
		return nil, errors.New("codigo_pago es requerido")
	}

	return s.repo.GetPagoByCodigo(codigoPago)
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

func (s *PagosService) CreatePago(input CrearPagoInput) (string, error) {
	estado, err := NormalizarEstadoPago(input.Estado)
	if err != nil {
		return "", err
	}

	if len(input.Detalle) == 0 {
		return "", errors.New("detalle es requerido")
	}

	detalle := make([]repository.CrearDetallePagoInput, 0, len(input.Detalle))
	sumaDetalle := 0.0
	for _, d := range input.Detalle {
		if err := validarDetallePago(d.ServicioID, d.Servicio, d.PrecioUnitario, d.Cantidad, d.Subtotal); err != nil {
			return "", err
		}
		sumaDetalle += d.Subtotal
		detalle = append(detalle, repository.CrearDetallePagoInput{
			ServicioID:     d.ServicioID,
			Servicio:       strings.TrimSpace(d.Servicio),
			PrecioUnitario: d.PrecioUnitario,
			Cantidad:       d.Cantidad,
			Subtotal:       d.Subtotal,
		})
	}

	subtotal, descuento, totalFinal, err := calcularTotalesPago(input.Subtotal, input.Descuento, input.TotalFinal, sumaDetalle)
	if err != nil {
		return "", err
	}

	if err := validarPagoBase(input.LocalID, input.LocalNombre, input.ClienteID, input.ClienteNIT, input.ClienteNombre, &subtotal, &descuento, &totalFinal); err != nil {
		return "", err
	}

	return s.repo.CreatePago(repository.CrearPagoInput{
		LocalID:       input.LocalID,
		LocalNombre:   strings.TrimSpace(input.LocalNombre),
		ClienteID:     input.ClienteID,
		ClienteNIT:    strings.TrimSpace(input.ClienteNIT),
		ClienteNombre: strings.TrimSpace(input.ClienteNombre),
		Subtotal:      &subtotal,
		Descuento:     &descuento,
		TotalFinal:    &totalFinal,
		Estado:        estado,
		Activo:        input.Activo,
		Detalle:       detalle,
	})
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
}

type ActualizarDetallePagoInput struct {
	ID             *int
	ServicioID     *int
	Servicio       string
	PrecioUnitario float64
	Cantidad       int
	Subtotal       float64
}

func (s *PagosService) UpdatePago(input ActualizarPagoInput) error {
	codigoPago := strings.TrimSpace(input.CodigoPago)
	if codigoPago == "" {
		return errors.New("codigo_pago es requerido")
	}

	localNombre := trimStringPtr(input.LocalNombre)
	clienteNIT := trimStringPtr(input.ClienteNIT)
	clienteNombre := trimStringPtr(input.ClienteNombre)

	if input.LocalID != nil && *input.LocalID <= 0 {
		return errors.New("local_id invalido")
	}
	if localNombre != nil && *localNombre == "" {
		return errors.New("local_nombre no puede estar vacio")
	}
	if input.ClienteID != nil && *input.ClienteID <= 0 {
		return errors.New("cliente_id invalido")
	}
	if clienteNIT != nil && *clienteNIT == "" {
		return errors.New("cliente_nit no puede estar vacio")
	}
	if clienteNombre != nil && *clienteNombre == "" {
		return errors.New("cliente_nombre no puede estar vacio")
	}
	if input.Subtotal != nil && *input.Subtotal < 0 {
		return errors.New("subtotal no puede ser negativo")
	}
	if input.Descuento != nil && *input.Descuento < 0 {
		return errors.New("descuento no puede ser negativo")
	}
	if input.TotalFinal != nil && *input.TotalFinal < 0 {
		return errors.New("total_final no puede ser negativo")
	}

	var detalle *[]repository.ActualizarDetallePagoInput
	if input.Detalle != nil {
		procesado := make([]repository.ActualizarDetallePagoInput, 0, len(*input.Detalle))
		ids := map[int]bool{}
		for _, d := range *input.Detalle {
			if d.ID != nil {
				if *d.ID <= 0 {
					return errors.New("id de detalle invalido")
				}
				if ids[*d.ID] {
					return errors.New("id de detalle duplicado")
				}
				ids[*d.ID] = true
				procesado = append(procesado, repository.ActualizarDetallePagoInput{ID: d.ID})
				continue
			}

			if err := validarDetallePago(d.ServicioID, d.Servicio, d.PrecioUnitario, d.Cantidad, d.Subtotal); err != nil {
				return err
			}
			procesado = append(procesado, repository.ActualizarDetallePagoInput{
				ServicioID:     d.ServicioID,
				Servicio:       strings.TrimSpace(d.Servicio),
				PrecioUnitario: d.PrecioUnitario,
				Cantidad:       d.Cantidad,
				Subtotal:       d.Subtotal,
			})
		}
		detalle = &procesado
	}

	var estado *string
	if input.Estado != nil {
		normalizado, err := NormalizarEstadoPago(*input.Estado)
		if err != nil {
			return err
		}
		estado = &normalizado
	}

	return s.repo.UpdatePago(repository.ActualizarPagoInput{
		CodigoPago:    codigoPago,
		LocalID:       input.LocalID,
		LocalNombre:   localNombre,
		ClienteID:     input.ClienteID,
		ClienteIDSet:  input.ClienteIDSet,
		ClienteNIT:    clienteNIT,
		ClienteNombre: clienteNombre,
		Subtotal:      input.Subtotal,
		Descuento:     input.Descuento,
		TotalFinal:    input.TotalFinal,
		Estado:        estado,
		Activo:        input.Activo,
		Detalle:       detalle,

		RecalcularSubtotal:   input.Detalle != nil && input.Subtotal == nil,
		RecalcularTotalFinal: input.Detalle != nil && input.TotalFinal == nil,
	})
}

func (s *PagosService) DeletePago(codigoPago string) error {
	codigoPago = strings.TrimSpace(codigoPago)
	if codigoPago == "" {
		return errors.New("codigo_pago es requerido")
	}

	return s.repo.DeletePago(codigoPago)
}

func NormalizarEstadoPago(raw string) (string, error) {
	estado := strings.ToUpper(strings.TrimSpace(raw))
	switch estado {
	case "PAGADO", "BORRADOR", "PENDIENTE":
		return estado, nil
	default:
		return "", fmt.Errorf("estado invalido, valores permitidos: PAGADO, BORRADOR, PENDIENTE")
	}
}

func normalizarEstadoPagoOpcional(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return NormalizarEstadoPago(raw)
}

func validarPagoBase(localID int, localNombre string, clienteID *int, clienteNIT, clienteNombre string, subtotal, descuento, totalFinal *float64) error {
	if localID <= 0 {
		return errors.New("local_id invalido")
	}
	if strings.TrimSpace(localNombre) == "" {
		return errors.New("local_nombre es requerido")
	}
	if clienteID != nil && *clienteID <= 0 {
		return errors.New("cliente_id invalido")
	}
	if strings.TrimSpace(clienteNIT) == "" {
		return errors.New("cliente_nit es requerido")
	}
	if strings.TrimSpace(clienteNombre) == "" {
		return errors.New("cliente_nombre es requerido")
	}
	if subtotal == nil {
		return errors.New("subtotal es requerido")
	}
	if descuento == nil {
		return errors.New("descuento es requerido")
	}
	if totalFinal == nil {
		return errors.New("total_final es requerido")
	}
	if *subtotal < 0 {
		return errors.New("subtotal no puede ser negativo")
	}
	if *descuento < 0 {
		return errors.New("descuento no puede ser negativo")
	}
	if *totalFinal < 0 {
		return errors.New("total_final no puede ser negativo")
	}
	if *descuento > *subtotal {
		return errors.New("descuento no puede ser mayor al subtotal")
	}

	return nil
}

func calcularTotalesPago(subtotal, descuento, totalFinal *float64, sumaDetalle float64) (float64, float64, float64, error) {
	subtotalCalculado := sumaDetalle
	if subtotal != nil {
		subtotalCalculado = *subtotal
	}

	descuentoCalculado := 0.0
	if descuento != nil {
		descuentoCalculado = *descuento
	}

	totalCalculado := subtotalCalculado - descuentoCalculado
	if totalFinal != nil {
		totalCalculado = *totalFinal
	}

	if descuentoCalculado > subtotalCalculado {
		return 0, 0, 0, errors.New("descuento no puede ser mayor al subtotal")
	}
	if totalCalculado < 0 {
		return 0, 0, 0, errors.New("total_final no puede ser negativo")
	}

	return subtotalCalculado, descuentoCalculado, totalCalculado, nil
}

func validarDetallePago(servicioID *int, servicio string, precioUnitario float64, cantidad int, subtotal float64) error {
	if servicioID != nil && *servicioID <= 0 {
		return errors.New("servicio_id invalido")
	}
	if strings.TrimSpace(servicio) == "" {
		return errors.New("servicio es requerido en cada detalle")
	}
	if precioUnitario < 0 {
		return errors.New("precio_unitario no puede ser negativo")
	}
	if cantidad <= 0 {
		return errors.New("cantidad debe ser mayor a cero")
	}
	if subtotal < 0 {
		return errors.New("subtotal del detalle no puede ser negativo")
	}

	return nil
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
