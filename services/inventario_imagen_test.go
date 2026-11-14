package services

import (
	"strings"
	"testing"
)

func TestNuevaClaveImagenItem(t *testing.T) {
	a, err := nuevaClaveImagenItem(7)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := nuevaClaveImagenItem(7)
	if a == b {
		t.Fatalf("las claves deben ser distintas en cada subida: %s", a)
	}
	if !strings.HasPrefix(a, "inventario/7/") || !strings.HasSuffix(a, ".webp") {
		t.Fatalf("clave con formato inesperado: %s", a)
	}
	if err := validarClaveImagenItem(7, a); err != nil {
		t.Fatalf("la clave emitida debe validar: %v", err)
	}
}

func TestValidarClaveImagenItem(t *testing.T) {
	casos := []struct {
		nombre string
		id     int
		key    string
		ok     bool
	}{
		{"valida", 1, "inventario/1/abc123.webp", true},
		{"otro item", 1, "inventario/2/abc123.webp", false},
		{"prefijo parcial", 1, "inventario/10/abc123.webp", false},
		{"sin extension webp", 1, "inventario/1/abc123.png", false},
		{"sin nombre", 1, "inventario/1/.webp", false},
		{"subcarpeta", 1, "inventario/1/a/b.webp", false},
		{"path traversal", 1, "inventario/1/../2/x.webp", false},
		{"doble extension", 1, "inventario/1/x.png.webp", false},
		{"vacia", 1, "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := validarClaveImagenItem(c.id, c.key)
			if (err == nil) != c.ok {
				t.Fatalf("validarClaveImagenItem(%d, %q) err=%v, ok esperado=%v", c.id, c.key, err, c.ok)
			}
		})
	}
}
