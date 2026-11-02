package services

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizarTipoPagoAceptaLosCuatroMetodos(t *testing.T) {
	casos := map[string]string{
		"efectivo":      "efectivo",
		"tarjeta":       "tarjeta",
		"qr":            "qr",
		"transferencia": "transferencia",
		"  QR ":         "qr",
		"Tarjeta":       "tarjeta",
		"TRANSFERENCIA": "transferencia",
	}
	for entrada, esperado := range casos {
		got, err := NormalizarTipoPago(entrada)
		if err != nil || got != esperado {
			t.Errorf("NormalizarTipoPago(%q) = %q, %v; esperado %q", entrada, got, err, esperado)
		}
	}
}

func TestNormalizarTipoPagoRechazaOtrosMetodos(t *testing.T) {
	for _, entrada := range []string{"", "   ", "cheque", "debito_automatico", "efectivo,qr"} {
		_, err := NormalizarTipoPago(entrada)
		if err == nil {
			t.Fatalf("NormalizarTipoPago(%q) debio fallar", entrada)
		}
		for _, metodo := range []string{"efectivo", "tarjeta", "QR", "transferencia"} {
			if !strings.Contains(err.Error(), metodo) {
				t.Fatalf("el mensaje debe listar %q: %v", metodo, err)
			}
		}
	}
}

func TestTiposPagoEsLaListaCerradaQueUsaLaValidacion(t *testing.T) {
	if got := strings.Join(TiposPago, ","); got != "efectivo,tarjeta,qr,transferencia" {
		t.Fatalf("TiposPago = %s", got)
	}
}

func TestLaMigracionDeTiposPagoCoincideConLaLista(t *testing.T) {
	sql, err := os.ReadFile("../migrations/000074_tipos_pago_cuatro_metodos.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, metodo := range TiposPago {
		if !strings.Contains(string(sql), "'"+metodo+"'") {
			t.Fatalf("la restriccion de la base no incluye %q", metodo)
		}
	}
}
