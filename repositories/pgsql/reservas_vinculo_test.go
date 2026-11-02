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
	if !strings.Contains(sqlVincularReserva, "cliente_id IS NULL") {
		t.Fatalf("la primera vinculacion solo toca reservas sin cliente: %s", sqlVincularReserva)
	}
}

// Equivocarse de cliente se corrige mientras no haya cargos: los cargos pertenecen al cliente y no se mueven.
func TestCambiarClienteSoloSinCargosActivos(t *testing.T) {
	for _, parte := range []string{"cargos_cliente", "reserva_id = $1", "anulado = FALSE"} {
		if !strings.Contains(sqlCargosActivosDeReserva, parte) {
			t.Fatalf("falta %q en: %s", parte, sqlCargosActivosDeReserva)
		}
	}
	if !strings.Contains(sqlCambiarClienteReserva, "SET cliente_id = $2") || strings.Contains(sqlCambiarClienteReserva, "IS NULL") {
		t.Fatalf("cambiar de cliente reasigna una reserva ya vinculada: %s", sqlCambiarClienteReserva)
	}
}

// Vincular enlaza una sola reserva: ya no existe el vinculo masivo por telefono, que mezclaba a las personas que
// comparten un numero.
func TestVincularNoEnlazaOtrasReservasPorTelefono(t *testing.T) {
	if strings.Contains(sqlActualizarTelefonoCliente, "UPDATE reservas") {
		t.Fatalf("actualizar el telefono solo toca al cliente: %s", sqlActualizarTelefonoCliente)
	}
	for _, parte := range []string{"UPDATE clientes", "numero_telefono = $2", "telefono_e164 = $3", "WHERE id = $1"} {
		if !strings.Contains(sqlActualizarTelefonoCliente, parte) {
			t.Fatalf("falta %q en: %s", parte, sqlActualizarTelefonoCliente)
		}
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
