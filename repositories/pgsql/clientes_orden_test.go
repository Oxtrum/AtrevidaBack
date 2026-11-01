package pgsql

import (
	"sort"
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
)

func TestClienteOrderClauseSinOrdenConservaApellidoNombre(t *testing.T) {
	if got := clienteOrderClause(repository.FiltroClientes{}); got != "apellido, nombre, id" {
		t.Fatalf("orden por defecto inesperado: %s", got)
	}
}

func TestClienteOrderClauseOrdenaPorColumnaPermitida(t *testing.T) {
	got := clienteOrderClause(repository.FiltroClientes{OrdenarPor: "apellido", OrdenDesc: true})
	if !strings.HasPrefix(got, "TRANSLATE(LOWER(apellido)") || !strings.HasSuffix(got, "DESC NULLS LAST, id DESC") {
		t.Fatalf("orden por apellido inesperado: %s", got)
	}
}

func TestClienteOrderClauseCiVacioQuedaAlFinal(t *testing.T) {
	got := clienteOrderClause(repository.FiltroClientes{OrdenarPor: "ci"})
	if !strings.Contains(got, "NULLIF(ci, '')") {
		t.Fatalf("el CI vacio debe tratarse como ausente: %s", got)
	}
}

func TestClientesOrdenablesExponeLasColumnasDeLaTabla(t *testing.T) {
	want := []string{"apellido", "ci", "id", "nit", "nombre", "numero_telefono"}
	got := make([]string, 0, len(ClientesOrdenables))
	for key := range ClientesOrdenables {
		got = append(got, key)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("columnas ordenables = %v, esperadas %v", got, want)
	}
}

func TestBuildClientesQuerySinOrdenUsaCursorPorValor(t *testing.T) {
	query, args := buildClientesQuery(repository.FiltroClientes{
		PageLimit: 51, CursorSet: true, CursorApellido: "Lopez", CursorNombre: "Maria", CursorID: 3,
	})
	if !strings.Contains(query, "(apellido, nombre, id) > (") || strings.Contains(query, "OFFSET") {
		t.Fatalf("sin orden debe paginar por valor: %s", query)
	}
	if !strings.Contains(query, "ORDER BY apellido, nombre, id") || len(args) != 4 {
		t.Fatalf("query/args inesperados: %s %#v", query, args)
	}
}

func TestBuildClientesQueryConOrdenPaginaPorDesplazamiento(t *testing.T) {
	query, args := buildClientesQuery(repository.FiltroClientes{
		OrdenarPor: "nombre", PageLimit: 51, Offset: 50, CursorSet: true, CursorID: 3,
	})
	if strings.Contains(query, "(apellido, nombre, id) >") {
		t.Fatalf("con orden no se usa el cursor por valor: %s", query)
	}
	if !strings.Contains(query, "LIMIT $1 OFFSET $2") || len(args) != 2 || args[0] != 51 || args[1] != 50 {
		t.Fatalf("limit/offset inesperados: %s %#v", query, args)
	}
}

func TestBuildClientesQueryPrimeraPaginaOrdenadaNoLlevaOffset(t *testing.T) {
	query, args := buildClientesQuery(repository.FiltroClientes{OrdenarPor: "id", OrdenDesc: true, PageLimit: 51})
	if strings.Contains(query, "OFFSET") || len(args) != 1 {
		t.Fatalf("la primera pagina no lleva OFFSET: %s %#v", query, args)
	}
}
