package services

import (
	"errors"
	"math"
	"strings"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type ReporteFinancieroService struct {
	fin        *pgsqlrepo.FinanzasRepo
	inventario *pgsqlrepo.InventarioRepo
	gastos     *GastosFijosService
	config     *ConfiguracionService
}

func NewReporteFinancieroService(
	fin *pgsqlrepo.FinanzasRepo, inventario *pgsqlrepo.InventarioRepo, gastos *GastosFijosService, config *ConfiguracionService,
) *ReporteFinancieroService {
	return &ReporteFinancieroService{fin: fin, inventario: inventario, gastos: gastos, config: config}
}

func redondear1(v float64) float64 { return math.Round(v*10) / 10 }

func resumenDesde(ventas float64, a pgsqlrepo.AgregadoMovimientos) models.ReporteFinancieroResumen {
	ingresos := redondearMonto(ventas + a.IngresosOtros)
	egresos := redondearMonto(a.EgresosFijos + a.EgresosVariables)
	utilidad := redondearMonto(ingresos - egresos)
	margen := 0.0
	if ingresos > 0 {
		margen = redondear1(utilidad / ingresos * 100)
	}
	return models.ReporteFinancieroResumen{
		IngresosVentas: redondearMonto(ventas), IngresosOtros: redondearMonto(a.IngresosOtros), IngresosTotal: ingresos,
		EgresosFijos: redondearMonto(a.EgresosFijos), EgresosVariables: redondearMonto(a.EgresosVariables),
		EgresosTotal: egresos, EgresosPendientes: redondearMonto(a.EgresosPendientes),
		UtilidadNeta: utilidad, MargenPct: margen,
	}
}

// Generar arma el reporte de ingresos vs egresos (PLAN §5.6). Solo cuentan movimientos PAGADO; los
// pendientes se informan aparte y los anulados nunca suman. Las ventas usan la misma regla que
// /bd/pagos/resumen para que los totales coincidan.
func (s *ReporteFinancieroService) Generar(fechaDesde, fechaHasta, local, agrupar string) (*models.ReporteFinanciero, error) {
	desde, err := parseFecha(fechaDesde)
	if err != nil {
		return nil, errors.New("fecha_desde invalida, use formato YYYY-MM-DD")
	}
	hasta, err := parseFecha(fechaHasta)
	if err != nil {
		return nil, errors.New("fecha_hasta invalida, use formato YYYY-MM-DD")
	}
	if hasta.Before(desde) {
		return nil, errors.New("fecha_hasta no puede ser anterior a fecha_desde")
	}
	agrupar = strings.ToLower(strings.TrimSpace(agrupar))
	if agrupar != "" && agrupar != "dia" && agrupar != "mes" {
		return nil, errors.New("agrupar invalido, valores permitidos: dia, mes")
	}
	local = strings.TrimSpace(local)
	var localID *int
	if local != "" {
		if localID, err = s.fin.LocalIDPorNombre(local); err != nil {
			return nil, err
		}
	}

	// Catch-up perezoso: el reporte es correcto aunque el job diario no haya corrido.
	if _, err := s.gastos.GenerarHoy(); err != nil {
		return nil, err
	}

	ventas, err := s.fin.VentasPeriodo(desde, hasta, localID)
	if err != nil {
		return nil, err
	}
	agregado, err := s.fin.Agregado(desde, hasta, localID, false)
	if err != nil {
		return nil, err
	}
	generales, err := s.fin.Agregado(desde, hasta, nil, true)
	if err != nil {
		return nil, err
	}
	porCategoria, err := s.fin.EgresosPorCategoria(desde, hasta, localID)
	if err != nil {
		return nil, err
	}
	puntos, err := s.fin.Serie(desde, hasta, localID, agrupar == "dia")
	if err != nil {
		return nil, err
	}
	serie := make([]models.PuntoSerieFinanciera, 0, len(puntos))
	for _, p := range puntos {
		serie = append(serie, models.PuntoSerieFinanciera{
			Periodo: p.Periodo, Ingresos: redondearMonto(p.Ingresos), Egresos: redondearMonto(p.Egresos),
			Utilidad: redondearMonto(p.Ingresos - p.Egresos),
		})
	}

	valorStock, bajoStock, err := s.inventario.ValorStock(localID)
	if err != nil {
		return nil, err
	}
	consumo, err := s.inventario.ConsumoServicios(fechaDesde, fechaHasta, localID)
	if err != nil {
		return nil, err
	}

	reporte := &models.ReporteFinanciero{
		Filtros:             models.ReporteFinancieroFiltros{FechaDesde: fechaDesde, FechaHasta: fechaHasta, Local: local},
		Resumen:             resumenDesde(ventas, agregado),
		EgresosPorCategoria: porCategoria,
		Serie:               serie,
		Inventario: models.ReporteInventarioResumen{
			ValorStock: redondearMonto(valorStock), ItemsBajoStock: bajoStock, ConsumoServicios: redondearMonto(consumo),
		},
		// Gastos generales (sin local): se muestran aparte y no se prorratean entre locales.
		GastosGenerales: redondearMonto(generales.EgresosFijos + generales.EgresosVariables),
		DetallePorLocal: []models.DetalleLocalFinanciero{},
	}

	if localID == nil {
		locales, err := s.fin.LocalesActivos()
		if err != nil {
			return nil, err
		}
		for _, l := range locales {
			id := l.ID
			v, err := s.fin.VentasPeriodo(desde, hasta, &id)
			if err != nil {
				return nil, err
			}
			a, err := s.fin.Agregado(desde, hasta, &id, false)
			if err != nil {
				return nil, err
			}
			reporte.DetallePorLocal = append(reporte.DetallePorLocal, models.DetalleLocalFinanciero{
				Local: l.Nombre, Resumen: resumenDesde(v, a),
			})
		}
	}
	return reporte, nil
}
