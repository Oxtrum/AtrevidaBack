package pgsql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regla de dinero (PLAN_CUENTA_CLIENTE_Y_AJUSTES): ningun registro de dinero se borra ni se edita; un error se
// corrige anulando con motivo y creando registros nuevos. Las filas de pago_aplicaciones (que parte de un pago
// cubre que cargo) no son la excepcion.

func TestAnularCobrosMarcaLasFilasYNoLasBorra(t *testing.T) {
	for nombre, query := range map[string]string{
		"por cargo": sqlAnularCobrosDelCargo,
		"por pago":  sqlAnularCobrosDelPago,
	} {
		if !strings.HasPrefix(strings.TrimSpace(query), "UPDATE pago_aplicaciones") {
			t.Fatalf("%s: debe ser un UPDATE, no un borrado: %s", nombre, query)
		}
		for _, fragmento := range []string{"anulado = TRUE", "motivo_anulacion = $2", "anulado_en = NOW()", "anulado = FALSE"} {
			if !strings.Contains(query, fragmento) {
				t.Fatalf("%s: falta %q: %s", nombre, fragmento, query)
			}
		}
	}
}

func TestLasSumasDeCobrosIgnoranLosAnulados(t *testing.T) {
	for nombre, query := range map[string]string{
		"cargos con lo pagado":  sqlListarCargos,
		"abonos con lo cubierto": sqlListarAbonos,
		"pendiente por cargo":    sqlPendientesPorCargo,
		"cubierto de un pago":    sqlCubiertoDeUnPago,
	} {
		if !strings.Contains(query, "pago_aplicaciones") {
			t.Fatalf("%s: la consulta debe leer pago_aplicaciones: %s", nombre, query)
		}
		if !strings.Contains(query, "a.anulado = FALSE") && !strings.Contains(query, "AND anulado = FALSE") {
			t.Fatalf("%s: debe ignorar los cobros anulados: %s", nombre, query)
		}
	}
}

func TestCadaCobroEsUnRegistroNuevoSinSumarSobreOtro(t *testing.T) {
	if strings.Contains(sqlInsertarCobro, "ON CONFLICT") || strings.Contains(sqlInsertarCobro, "DO UPDATE") {
		t.Fatalf("un cobro nuevo no puede editar uno existente: %s", sqlInsertarCobro)
	}
	if !strings.HasPrefix(strings.TrimSpace(sqlInsertarCobro), "INSERT INTO pago_aplicaciones") {
		t.Fatalf("debe insertar un registro nuevo: %s", sqlInsertarCobro)
	}
}

func TestNingunRepositorioBorraCobros(t *testing.T) {
	archivos, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		contenido, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToUpper(string(contenido)), "DELETE FROM PAGO_APLICACIONES") {
			t.Fatalf("%s borra filas de pago_aplicaciones; hay que anularlas", archivo)
		}
	}
}
