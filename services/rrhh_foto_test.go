package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"atrevida-agenda-api/models"
)

type storageFalso struct {
	err error
}

func (f storageFalso) CrearURLSubida(path string) (SubidaFirmada, error) {
	return SubidaFirmada{UploadURL: "https://s3.test/" + path, Path: path}, nil
}
func (f storageFalso) Eliminar(string) error      { return nil }
func (f storageFalso) URLPublica(p string) string { return "https://s3.test/publico/" + p }
func (f storageFalso) URLLecturaFirmada(p string, _ time.Duration) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "https://s3.test/firmada/" + p + "?sig=1", nil
}

func TestClaveFotoEmpleado(t *testing.T) {
	key, err := nuevaClaveImagen(prefijoFotoEmpleado(3))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "personal/3/") || !strings.HasSuffix(key, ".webp") {
		t.Fatalf("clave con formato inesperado: %s", key)
	}
	if err := validarClaveImagen(prefijoFotoEmpleado(3), "foto_key", key); err != nil {
		t.Fatalf("la clave emitida debe validar: %v", err)
	}
	// Una clave de inventario o de otro empleado no sirve como foto de este empleado.
	for _, mala := range []string{"inventario/3/abc.webp", "personal/4/abc.webp", "personal/3/../4/abc.webp", "personal/3/abc.jpg"} {
		if err := validarClaveImagen(prefijoFotoEmpleado(3), "foto_key", mala); err == nil {
			t.Fatalf("la clave %q no debe validar", mala)
		}
	}
}

func TestCompletarFotoEmpleado(t *testing.T) {
	key := "personal/1/abc.webp"

	t.Run("con foto y storage firma la URL", func(t *testing.T) {
		s := &RRHHService{Storage: storageFalso{}}
		e := models.Empleado{ID: 1, FotoKey: &key}
		s.completarFoto(&e)
		if e.FotoURL == nil || !strings.Contains(*e.FotoURL, "firmada/personal/1/abc.webp") {
			t.Fatalf("foto_url inesperada: %v", e.FotoURL)
		}
	})

	t.Run("sin storage no hay URL", func(t *testing.T) {
		s := &RRHHService{}
		e := models.Empleado{ID: 1, FotoKey: &key}
		s.completarFoto(&e)
		if e.FotoURL != nil {
			t.Fatalf("no debe haber URL sin storage: %v", *e.FotoURL)
		}
	})

	t.Run("sin foto no hay URL", func(t *testing.T) {
		s := &RRHHService{Storage: storageFalso{}}
		e := models.Empleado{ID: 1}
		s.completarFoto(&e)
		if e.FotoURL != nil {
			t.Fatal("no debe haber URL sin foto")
		}
	})

	t.Run("si la firma falla se omite sin romper", func(t *testing.T) {
		s := &RRHHService{Storage: storageFalso{err: errors.New("boom")}}
		e := models.Empleado{ID: 1, FotoKey: &key}
		s.completarFoto(&e)
		if e.FotoURL != nil {
			t.Fatal("no debe haber URL si la firma falla")
		}
	})
}
