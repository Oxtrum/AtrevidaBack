package services

import (
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestGetVentasDetalleValidaFechas(t *testing.T) {
	svc := NewPagosService(&fakePagosRepo{})
	casos := map[string]ResumenPagosInput{
		"sin fecha_desde":    {FechaHasta: "2026-10-31"},
		"sin fecha_hasta":    {FechaDesde: "2026-10-01"},
		"formato invalido":   {FechaDesde: "01/10/2026", FechaHasta: "2026-10-31"},
		"rango al reves":     {FechaDesde: "2026-10-31", FechaHasta: "2026-10-01"},
		"hasta antes desde ": {FechaDesde: "2026-10-05", FechaHasta: "2026-10-04"},
	}
	for nombre, in := range casos {
		if _, err := svc.GetVentasDetalle(in); err == nil {
			t.Errorf("%s: debia rechazarse", nombre)
		}
	}
}

func TestGetVentasDetalleMapeaYAvisaTruncado(t *testing.T) {
	fecha := time.Date(2026, 10, 14, 15, 30, 0, 0, time.UTC)
	repo := &fakePagosRepo{ventas: []repository.VentaDetalleRow{{
		CodigoPago: "PAGO-000001", Fecha: fecha, Local: "SAN MARTIN", ClienteNombre: "Maria", Cajero: "Ana",
		TipoPago: "efectivo", Servicio: "Limpieza facial", Cantidad: 2, PrecioUnitario: 150, Subtotal: 300,
		DescuentoProrrateado: 15, TotalPago: 285,
	}}}
	svc := NewPagosService(repo)

	res, err := svc.GetVentasDetalle(ResumenPagosInput{FechaDesde: "2026-10-01", FechaHasta: "2026-10-31", Local: "  SAN MARTIN "})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Lineas) != 1 || res.Lineas[0].Servicio != "Limpieza facial" || res.Lineas[0].DescuentoProrrateado != 15 {
		t.Fatalf("mapeo inesperado: %+v", res.Lineas)
	}
	if res.Truncado || res.Filtros.Local != "SAN MARTIN" {
		t.Fatalf("filtros o truncado inesperados: %+v", res)
	}
	if repo.ventasLimite != limiteVentasDetalle+1 {
		t.Fatalf("debe pedir una linea de mas para detectar el truncado: %d", repo.ventasLimite)
	}

	// Si el repositorio devuelve una linea mas que el limite, se corta y se avisa.
	repo.ventas = make([]repository.VentaDetalleRow, limiteVentasDetalle+1)
	res, err = svc.GetVentasDetalle(ResumenPagosInput{FechaDesde: "2026-10-01", FechaHasta: "2026-10-31"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Truncado || len(res.Lineas) != limiteVentasDetalle {
		t.Fatalf("esperaba truncado con %d lineas, hay %d (truncado=%v)", limiteVentasDetalle, len(res.Lineas), res.Truncado)
	}
}
