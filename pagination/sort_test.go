package pagination

import (
	"errors"
	"strings"
	"testing"
)

var columnasPrueba = Columns{
	"nombre": "LOWER(t.nombre)",
	"precio": "t.precio",
	"fecha":  "t.creado_en",
}

func TestParseSortSinParametrosNoOrdena(t *testing.T) {
	sort, err := ParseSort("", "", columnasPrueba)
	if err != nil || sort.Active() {
		t.Fatalf("ParseSort vacio = %+v, %v", sort, err)
	}
}

func TestParseSortAceptaColumnaPermitida(t *testing.T) {
	casos := []struct {
		clave, direccion string
		desc             bool
	}{
		{"nombre", "", false},
		{"nombre", "asc", false},
		{"precio", "desc", true},
		{" precio ", " DESC ", true},
	}
	for _, c := range casos {
		sort, err := ParseSort(c.clave, c.direccion, columnasPrueba)
		if err != nil || !sort.Active() || sort.Desc != c.desc || sort.Key != strings.TrimSpace(c.clave) {
			t.Fatalf("ParseSort(%q, %q) = %+v, %v", c.clave, c.direccion, sort, err)
		}
	}
}

func TestParseSortRechazaColumnaNoPermitida(t *testing.T) {
	for _, clave := range []string{"password", "t.nombre", "nombre; DROP TABLE pagos", "NOMBRE"} {
		_, err := ParseSort(clave, "", columnasPrueba)
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("ParseSort(%q) error = %v", clave, err)
		}
	}
}

func TestParseSortListaLasOpcionesEnElError(t *testing.T) {
	_, err := ParseSort("nada", "", columnasPrueba)
	if err == nil || !strings.Contains(err.Error(), "fecha, nombre, precio") {
		t.Fatalf("el error debe listar las columnas en orden alfabetico: %v", err)
	}
}

func TestParseSortRechazaDireccionInvalida(t *testing.T) {
	if _, err := ParseSort("nombre", "arriba", columnasPrueba); !errors.Is(err, ErrInvalid) {
		t.Fatalf("direccion invalida error = %v", err)
	}
}

func TestParseSortDireccionSinColumna(t *testing.T) {
	if _, err := ParseSort("", "desc", columnasPrueba); !errors.Is(err, ErrInvalid) {
		t.Fatalf("orden sin ordenar_por error = %v", err)
	}
}

func TestOrderBySinOrdenUsaElPorDefecto(t *testing.T) {
	got := Sort{}.OrderBy(columnasPrueba, "t.creado_en DESC, t.id DESC", "t.id")
	if got != "t.creado_en DESC, t.id DESC" {
		t.Fatalf("OrderBy sin orden = %q", got)
	}
}

func TestOrderByArmaExpresionDireccionNulosYDesempate(t *testing.T) {
	asc := Sort{Key: "nombre"}.OrderBy(columnasPrueba, "x", "t.id")
	if asc != "LOWER(t.nombre) ASC NULLS LAST, t.id ASC" {
		t.Fatalf("OrderBy asc = %q", asc)
	}
	desc := Sort{Key: "precio", Desc: true}.OrderBy(columnasPrueba, "x", "t.id")
	if desc != "t.precio DESC NULLS LAST, t.id DESC" {
		t.Fatalf("OrderBy desc = %q", desc)
	}
}

func TestOrderByDesempataPorVariasColumnasEnElMismoSentido(t *testing.T) {
	got := Sort{Key: "nombre", Desc: true}.OrderBy(columnasPrueba, "x", "t.fecha", "t.hora", "t.id")
	if got != "LOWER(t.nombre) DESC NULLS LAST, t.fecha DESC, t.hora DESC, t.id DESC" {
		t.Fatalf("OrderBy con varios desempates = %q", got)
	}
}

func TestOrderByConClaveDesconocidaCaeAlPorDefecto(t *testing.T) {
	got := Sort{Key: "inyeccion"}.OrderBy(columnasPrueba, "t.creado_en DESC", "t.id")
	if got != "t.creado_en DESC" {
		t.Fatalf("OrderBy con clave desconocida = %q", got)
	}
}

func TestOffsetCursorRoundTrip(t *testing.T) {
	request, _ := Parse("2", "")
	filtros := map[string]string{"ordenar_por": "nombre"}
	items := []testPosition{{"a", 1}, {"b", 2}, {"c", 3}}
	_, meta, err := Build(items, request, "test", filtros, func(testPosition) any { return OffsetPosition{Offset: 0 + request.Limit} })
	if err != nil || meta == nil || meta.NextCursor == nil {
		t.Fatalf("Build = %+v, %v", meta, err)
	}
	var despues OffsetPosition
	if err := Decode(*meta.NextCursor, "test", filtros, &despues); err != nil || despues.Offset != 2 {
		t.Fatalf("Decode = %+v, %v", despues, err)
	}
}

func TestCursorDeOtroOrdenSeRechaza(t *testing.T) {
	request, _ := Parse("2", "")
	items := []testPosition{{"a", 1}, {"b", 2}, {"c", 3}}
	_, meta, _ := Build(items, request, "test", map[string]string{"ordenar_por": "nombre"}, func(testPosition) any { return OffsetPosition{Offset: 2} })
	var despues OffsetPosition
	err := Decode(*meta.NextCursor, "test", map[string]string{"ordenar_por": "precio"}, &despues)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("un cursor de otro orden debe rechazarse: %v", err)
	}
}

func TestOffsetPositionValida(t *testing.T) {
	if (OffsetPosition{Offset: -1}).Valid() {
		t.Fatal("un offset negativo no es valido")
	}
	if !(OffsetPosition{Offset: 0}).Valid() || !(OffsetPosition{Offset: 40}).Valid() {
		t.Fatal("offset 0 y positivos son validos")
	}
}
