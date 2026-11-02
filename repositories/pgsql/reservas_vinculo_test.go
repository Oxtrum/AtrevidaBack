package pgsql

import (
	"strings"
	"testing"
)

func TestClientesPorTelefonoCompararExactoSinAproximar(t *testing.T) {
	if !strings.Contains(sqlClientesPorTelefono, "telefono_e164 = $1") {
		t.Fatalf("debe comparar el E.164 exacto: %s", sqlClientesPorTelefono)
	}
	for _, aproximado := range []string{"LIKE", "RIGHT(", "ILIKE"} {
		if strings.Contains(sqlClientesPorTelefono, aproximado) {
			t.Fatalf("el vinculo automatico no puede usar coincidencias aproximadas (%s)", aproximado)
		}
	}
}

func TestVincularSoloTocaReservasSinCliente(t *testing.T) {
	for nombre, sql := range map[string]string{
		"una reserva":    sqlVincularReserva,
		"mismo telefono": sqlVincularMismoTelefono,
	} {
		if !strings.Contains(sql, "cliente_id IS NULL") {
			t.Fatalf("%s: no puede reasignar una reserva que ya tiene cliente: %s", nombre, sql)
		}
	}
	if !strings.Contains(sqlVincularMismoTelefono, "telefono_e164 = $1") {
		t.Fatalf("el vinculo masivo debe usar el telefono exacto: %s", sqlVincularMismoTelefono)
	}
}

func TestPorVincularListaSoloReservasVigentesSinCliente(t *testing.T) {
	for _, parte := range []string{"cliente_id IS NULL", "activo = TRUE", "RECHAZADO"} {
		if !strings.Contains(sqlReservasPorVincular, parte) {
			t.Fatalf("falta %q en: %s", parte, sqlReservasPorVincular)
		}
	}
}

func TestSugerenciasPonenPrimeroLaCoincidenciaDeTelefono(t *testing.T) {
	if !strings.Contains(sqlSugerenciasDeCliente, "ORDER BY") || !strings.Contains(sqlSugerenciasDeCliente, "por_telefono DESC") {
		t.Fatalf("las coincidencias por telefono van primero: %s", sqlSugerenciasDeCliente)
	}
}
