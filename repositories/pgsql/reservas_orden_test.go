package pgsql

import (
	"sort"
	"strings"
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
)

func TestReservaOrderClauseConOrdenPedidoDesempataPorFechaHoraEId(t *testing.T) {
	got := reservaOrderClause(repository.FiltroReservasPG{OrdenarPor: "cliente", OrdenCronologico: true})
	if !strings.HasPrefix(got, "TRANSLATE(LOWER(r.cliente)") || !strings.HasSuffix(got, "ASC NULLS LAST, r.fecha ASC, r.hora_desde ASC, r.id ASC") {
		t.Fatalf("el orden pedido manda sobre el cronologico: %s", got)
	}
}

func TestReservaOrderClauseDescendenteInvierteTambienLosDesempates(t *testing.T) {
	got := reservaOrderClause(repository.FiltroReservasPG{OrdenarPor: "fecha", OrdenDesc: true})
	if got != "r.fecha DESC NULLS LAST, r.fecha DESC, r.hora_desde DESC, r.id DESC" {
		t.Fatalf("orden por fecha descendente inesperado: %s", got)
	}
}

func TestReservaOrderClauseServicioUsaLoMismoQueMuestraLaTabla(t *testing.T) {
	got := reservaOrderClause(repository.FiltroReservasPG{OrdenarPor: "servicio"})
	if !strings.Contains(got, "COALESCE(r.servicio_confirmado, r.servicio_nombre, r.servicio_solicitado)") {
		t.Fatalf("el servicio ordena por el texto visible: %s", got)
	}
}

func TestReservasOrdenablesExponeLasColumnasDeLaTabla(t *testing.T) {
	want := []string{"cliente", "estado", "fecha", "hora_desde", "local", "servicio", "tipo"}
	got := make([]string, 0, len(ReservasOrdenables))
	for key := range ReservasOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildReservasQuerySinOrdenConservaElCursorPorValor(t *testing.T) {
	query, args := buildReservasQuery(repository.FiltroReservasPG{
		PageLimit: 51, CursorSet: true, CursorLocal: "SAN MARTIN", CursorFecha: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		CursorHora: "16:00", CursorID: 7,
	})
	if !strings.Contains(query, "(r.local_nombre, r.fecha, r.hora_desde, r.id) > (") || strings.Contains(query, "OFFSET") || len(args) != 5 {
		t.Fatalf("sin orden pedido se conserva el cursor por valor: %s %#v", query, args)
	}
}

func TestBuildReservasQueryConOrdenPaginaPorDesplazamiento(t *testing.T) {
	query, args := buildReservasQuery(repository.FiltroReservasPG{
		OrdenarPor: "local", PageLimit: 51, Offset: 50, CursorSet: true, CursorID: 7,
	})
	if strings.Contains(query, "(r.local_nombre, r.fecha, r.hora_desde, r.id) >") {
		t.Fatalf("con orden no se usa el cursor por valor: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[0] != 51 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
}

func TestBuildReservasQueryPrimeraPaginaOrdenadaNoLlevaOffset(t *testing.T) {
	query, args := buildReservasQuery(repository.FiltroReservasPG{OrdenarPor: "estado", PageLimit: 51})
	if strings.Contains(query, "OFFSET") || len(args) != 1 {
		t.Fatalf("la primera pagina no lleva OFFSET: %s %#v", query, args)
	}
}
