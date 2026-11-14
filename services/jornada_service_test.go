package services

import (
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func filaJornada(estado string, atencionID *int, fecha time.Time, horaHasta string) repository.JornadaRow {
	return repository.JornadaRow{
		EstadoReserva: estado,
		AtencionID:    atencionID,
		Fecha:         fecha,
		HoraHasta:     horaHasta,
	}
}

func TestDerivarEstadoJornadaProgramadaAntesDelFin(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 9, 59, 0, 0, loc)
	got := derivarEstadoJornada(filaJornada("AGENDADO", nil, fecha, "10:00"), ahora, loc)
	if got != "PROGRAMADA" {
		t.Fatalf("got = %q, esperado PROGRAMADA", got)
	}
}

func TestDerivarEstadoJornadaPendienteCierreAlTerminar(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 10, 0, 0, 0, loc)
	got := derivarEstadoJornada(filaJornada("AGENDADO", nil, fecha, "10:00"), ahora, loc)
	if got != "PENDIENTE_CIERRE" {
		t.Fatalf("got = %q, esperado PENDIENTE_CIERRE", got)
	}
}

func TestDerivarEstadoJornadaAtendidaGanaSobreLaReserva(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 9, 0, 0, 0, loc)
	atencionID := 55
	got := derivarEstadoJornada(filaJornada("AGENDADO", &atencionID, fecha, "10:00"), ahora, loc)
	if got != "ATENDIDA" {
		t.Fatalf("got = %q, esperado ATENDIDA", got)
	}
}

func TestDerivarEstadoJornadaExcluyeEstadosNoOperativos(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 9, 0, 0, 0, loc)
	for _, estado := range []string{"PENDIENTE", "RECHAZADO", "COMPLETADO", ""} {
		if got := derivarEstadoJornada(filaJornada(estado, nil, fecha, "10:00"), ahora, loc); got != "" {
			t.Fatalf("%q: got = %q, esperado vacio", estado, got)
		}
	}
}

func TestDerivarEstadoJornadaCancelada(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 9, 0, 0, 0, loc)
	if got := derivarEstadoJornada(filaJornada("CANCELADO", nil, fecha, "10:00"), ahora, loc); got != "CANCELADA" {
		t.Fatalf("got = %q, esperado CANCELADA", got)
	}
}

func TestDerivarEstadoJornadaUsaLaFechaDeLaReserva(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	ahora := time.Date(2026, 11, 9, 9, 0, 0, 0, loc)
	pasada := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	futura := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	if got := derivarEstadoJornada(filaJornada("AGENDADO", nil, pasada, "10:00"), ahora, loc); got != "PENDIENTE_CIERRE" {
		t.Fatalf("pasada: got = %q", got)
	}
	if got := derivarEstadoJornada(filaJornada("AGENDADO", nil, futura, "10:00"), ahora, loc); got != "PROGRAMADA" {
		t.Fatalf("futura: got = %q", got)
	}
}

func TestJornadaFinalizadaSinHoraValidaNoCierra(t *testing.T) {
	loc := time.FixedZone("America/La_Paz", -4*3600)
	fecha := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 11, 9, 23, 0, 0, 0, loc)
	for _, hora := range []string{"", "  ", "25:00", "abc"} {
		if jornadaFinalizada(filaJornada("AGENDADO", nil, fecha, hora), ahora, loc) {
			t.Fatalf("%q: no debia cerrar la jornada", hora)
		}
	}
}
