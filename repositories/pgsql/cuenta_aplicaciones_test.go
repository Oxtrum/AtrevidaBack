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
		"abonos con saldo":       sqlAbonosConSaldo,
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

func TestElSaldoDisponibleSeBloqueaParaQueDosCobrosNoGastenElMismoDinero(t *testing.T) {
	if !strings.Contains(sqlAbonosConSaldo, "FOR UPDATE OF p") {
		t.Fatalf("los abonos con saldo deben bloquearse dentro de la transaccion: %s", sqlAbonosConSaldo)
	}
	for _, fragmento := range []string{"es_abono = TRUE", "activo = TRUE", "estado = 'PAGADO'", "ORDER BY p.fecha_creacion, p.id"} {
		if !strings.Contains(sqlAbonosConSaldo, fragmento) {
			t.Fatalf("falta %q: %s", fragmento, sqlAbonosConSaldo)
		}
	}
}

func TestListarCargosTraeDeDondeNaceCadaCargo(t *testing.T) {
	for _, parte := range []string{"LEFT JOIN reservas", "LEFT JOIN planes", "reserva_fecha", "reserva_servicio", "plan_nombre"} {
		if !strings.Contains(sqlListarCargos, parte) {
			t.Fatalf("el listado de cargos debe traer su origen; falta %q: %s", parte, sqlListarCargos)
		}
	}
	if strings.Contains(sqlListarCargos, "INNER JOIN") || strings.Contains(sqlListarCargos, "\n\t\tJOIN ") {
		t.Fatal("un cargo manual no tiene reserva ni plan: los cruces deben ser LEFT JOIN")
	}
}
