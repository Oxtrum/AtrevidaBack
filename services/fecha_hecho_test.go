package services

import (
	"strings"
	"testing"
	"time"
)

var laPaz = time.FixedZone("America/La_Paz", -4*3600)

// ahoraPrueba es el 28 de octubre de 2026, 21:30 hora de La Paz (01:30 UTC del 29).
var ahoraPrueba = time.Date(2026, 10, 29, 1, 30, 0, 0, time.UTC)

func TestResolverFechaHechoVaciaUsaElMomentoActual(t *testing.T) {
	got, err := ResolverFechaHecho("", ahoraPrueba, laPaz, 0)
	if err != nil || !got.Equal(ahoraPrueba) {
		t.Fatalf("got = %v, %v", got, err)
	}
	got, err = ResolverFechaHecho("   ", ahoraPrueba, laPaz, 0)
	if err != nil || !got.Equal(ahoraPrueba) {
		t.Fatalf("espacios = %v, %v", got, err)
	}
}

func TestResolverFechaHechoDeHoyUsaElMomentoActual(t *testing.T) {
	// En La Paz todavia es 28; 01:30 UTC del 29 no debe contarse como manana.
	got, err := ResolverFechaHecho("2026-10-28", ahoraPrueba, laPaz, 0)
	if err != nil || !got.Equal(ahoraPrueba) {
		t.Fatalf("got = %v, %v", got, err)
	}
}

func TestResolverFechaHechoPasadaUsaMediodiaLocal(t *testing.T) {
	got, err := ResolverFechaHecho("2026-10-25", ahoraPrueba, laPaz, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 25, 12, 0, 0, 0, laPaz)
	if !got.Equal(want) {
		t.Fatalf("got = %v, esperado %v", got, want)
	}
}

func TestResolverFechaHechoFuturaSeRechaza(t *testing.T) {
	_, err := ResolverFechaHecho("2026-10-29", ahoraPrueba, laPaz, 0)
	if err == nil || !strings.Contains(err.Error(), "no puede ser futura") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolverFechaHechoConFormatoInvalidoSeRechaza(t *testing.T) {
	for _, raw := range []string{"28/10/2026", "2026-13-01", "ayer", "2026-10-28T10:00:00Z"} {
		_, err := ResolverFechaHecho(raw, ahoraPrueba, laPaz, 0)
		if err == nil || !strings.Contains(err.Error(), "formato") {
			t.Fatalf("%q: error = %v", raw, err)
		}
	}
}

func TestResolverFechaHechoRespetaElMaximoDeDiasAtras(t *testing.T) {
	// Hoy es 28; con maximo 3 se acepta hasta el 25 y se rechaza el 24.
	if _, err := ResolverFechaHecho("2026-10-25", ahoraPrueba, laPaz, 3); err != nil {
		t.Fatalf("el limite exacto debe aceptarse: %v", err)
	}
	_, err := ResolverFechaHecho("2026-10-24", ahoraPrueba, laPaz, 3)
	if err == nil || !strings.Contains(err.Error(), "no puede ser anterior a 3 dias") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolverFechaHechoCeroDiasNoPoneLimite(t *testing.T) {
	if _, err := ResolverFechaHecho("2020-01-01", ahoraPrueba, laPaz, 0); err != nil {
		t.Fatalf("0 significa sin limite: %v", err)
	}
}
