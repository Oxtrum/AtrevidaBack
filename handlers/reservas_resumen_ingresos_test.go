package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"atrevida-agenda-api/services"
)

func resumenConIngresos() services.ResumenReservas {
	return services.ResumenReservas{
		ReservasAgendadasDia:    4,
		ServiciosCompletadosDia: 3,
		IngresosDia:             250,
		IngresosSemana:          900,
		Ingresos:                services.ResumenIngresosSemana{TotalIngresos: 900, Lunes: 650, Martes: 250},
	}
}

func TestResumenReservasMuestraIngresosAlAdministrador(t *testing.T) {
	resp := buildReservaResumenResponse(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), resumenConIngresos(), true)
	if resp.IngresosHoy == nil || *resp.IngresosHoy != 250 {
		t.Fatalf("ingresos_hoy = %v, quiero 250", resp.IngresosHoy)
	}
	if resp.IngresosSemana == nil || *resp.IngresosSemana != 900 {
		t.Fatalf("ingresos_semana = %v, quiero 900", resp.IngresosSemana)
	}
	if resp.Ingresos == nil || resp.Ingresos.TotalIngresos != 900 {
		t.Fatalf("ingresos = %+v, quiero el desglose", resp.Ingresos)
	}
}

func TestResumenReservasOcultaIngresosAQuienNoEsAdministrador(t *testing.T) {
	resp := buildReservaResumenResponse(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), resumenConIngresos(), false)
	crudo, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]interface{}
	if err := json.Unmarshal(crudo, &campos); err != nil {
		t.Fatal(err)
	}
	for _, clave := range []string{"ingresos_hoy", "ingresos_semana", "ingresos"} {
		if _, hay := campos[clave]; hay {
			t.Fatalf("la respuesta trae %s y no debe: %s", clave, crudo)
		}
	}
	if resp.ReservasAgendadasDia != 4 || resp.ServiciosCompletadosDia != 3 {
		t.Fatalf("los conteos operativos se perdieron: %+v", resp)
	}
}
