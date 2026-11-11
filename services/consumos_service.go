package services

import (
	"errors"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

// ConsumosInput son los filtros crudos del calendario y del reporte de consumos.
type ConsumosInput struct {
	LocalID         *int
	FechaDesde      *time.Time
	FechaHasta      *time.Time
	ItemID          *int
	Cliente         string
	Servicio        string
	Empleado        string
	EmpleadoID      *int
	EstadoDescuento string
	Q               string
	PageLimit       int
	Offset          int
}

// normalizarEstadoConsumo unifica el estado de deduccion que acepta la API.
func normalizarEstadoConsumo(raw string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "TODOS":
		return "", nil
	case "DESCONTADO":
		return "DESCONTADO", nil
	case "PENDIENTE", "PENDIENTE_STOCK":
		return "PENDIENTE", nil
	default:
		return "", errors.New("estado_descuento invalido, use DESCONTADO o PENDIENTE")
	}
}

// normalizarRangoConsumos completa las fechas faltantes con el otro extremo o
// con hoy y rechaza un rango invertido.
func normalizarRangoConsumos(desde, hasta *time.Time, hoy time.Time) (time.Time, time.Time, error) {
	inicio, fin := hoy, hoy
	if desde != nil {
		inicio = *desde
	}
	if hasta != nil {
		fin = *hasta
	}
	if desde != nil && hasta == nil {
		fin = inicio
	}
	if desde == nil && hasta != nil {
		inicio = fin
	}
	if fin.Before(inicio) {
		return time.Time{}, time.Time{}, errors.New("fecha_hasta debe ser mayor o igual a fecha_desde")
	}
	return inicio, fin, nil
}

// PrepararConsumos valida y normaliza los filtros antes de consultar la base.
func (s *InventarioService) PrepararConsumos(in ConsumosInput) (repository.FiltroConsumos, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return repository.FiltroConsumos{}, err
	}
	desde, hasta, err := normalizarRangoConsumos(in.FechaDesde, in.FechaHasta, cfg.Hoy())
	if err != nil {
		return repository.FiltroConsumos{}, err
	}
	estado, err := normalizarEstadoConsumo(in.EstadoDescuento)
	if err != nil {
		return repository.FiltroConsumos{}, err
	}
	if in.Offset < 0 {
		return repository.FiltroConsumos{}, errors.New("offset invalido")
	}
	return repository.FiltroConsumos{
		LocalID: in.LocalID, FechaDesde: desde, FechaHasta: hasta, ItemID: in.ItemID,
		Cliente: strings.TrimSpace(in.Cliente), Servicio: strings.TrimSpace(in.Servicio),
		Empleado: strings.TrimSpace(in.Empleado), EmpleadoID: in.EmpleadoID,
		EstadoDescuento: estado, Q: strings.TrimSpace(in.Q),
		PageLimit: in.PageLimit, Offset: in.Offset,
	}, nil
}

func (s *InventarioService) ListarConsumos(f repository.FiltroConsumos) ([]models.ConsumoEvento, error) {
	return s.repo.ListarConsumos(f)
}

func (s *InventarioService) ContarConsumos(f repository.FiltroConsumos) (int, error) {
	return s.repo.ContarConsumos(f)
}

func (s *InventarioService) ResumenConsumos(f repository.FiltroConsumos) ([]models.ConsumoResumen, error) {
	return s.repo.ResumenConsumos(f)
}

func (s *InventarioService) ResumenConsumosPorEmpleado(f repository.FiltroConsumos) ([]models.ConsumoEmpleadoResumen, error) {
	return s.repo.ResumenConsumosPorEmpleado(f)
}
