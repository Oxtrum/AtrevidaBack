package services

import (
	"context"
	"strings"
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type fakeLocalesRepo struct {
	localID   int
	espacios  []repository.TipoEspacioInput
	llamadas  int
	errGuarda error
}

func (r *fakeLocalesRepo) GetAllLocales(context.Context) ([]models.LocalConEspacios, error) {
	return nil, nil
}
func (r *fakeLocalesRepo) GetLocalById(int) (*models.LocalConEspacios, error) { return nil, nil }
func (r *fakeLocalesRepo) CreateLocal(string, []repository.TipoEspacioInput) (int, error) {
	return 0, nil
}
func (r *fakeLocalesRepo) UpdateLocal(int, *string, *bool) error { return nil }
func (r *fakeLocalesRepo) DeleteLocal(int) error                  { return nil }
func (r *fakeLocalesRepo) SetEspacios(localID int, espacios []repository.TipoEspacioInput) error {
	r.llamadas++
	r.localID = localID
	r.espacios = espacios
	return r.errGuarda
}

func TestActualizarEspaciosNormalizaTipoYPermiteCero(t *testing.T) {
	repo := &fakeLocalesRepo{}
	svc := NewLocalesService(repo)

	err := svc.UpdateEspacios(3, []EspacioInput{
		{TipoEspacio: " m ", CantidadEspacios: 0},
		{TipoEspacio: "B", CantidadEspacios: 5},
	})

	if err != nil {
		t.Fatalf("error = %v, want nil: 0 es valido (mantenimiento)", err)
	}
	if repo.localID != 3 || len(repo.espacios) != 2 {
		t.Fatalf("repo recibio local %d y %+v", repo.localID, repo.espacios)
	}
	if repo.espacios[0].TipoEspacio != "M" || repo.espacios[0].CantidadEspacios != 0 {
		t.Fatalf("espacios[0] = %+v, want M con 0", repo.espacios[0])
	}
}

func TestActualizarEspaciosRechazaDatosInvalidos(t *testing.T) {
	casos := map[string]struct {
		in   []EspacioInput
		want string
	}{
		"cantidad negativa": {[]EspacioInput{{TipoEspacio: "M", CantidadEspacios: -1}}, "negativa"},
		"tipo desconocido":  {[]EspacioInput{{TipoEspacio: "X", CantidadEspacios: 1}}, "tipo_espacio"},
		"tipo repetido": {[]EspacioInput{
			{TipoEspacio: "M", CantidadEspacios: 1}, {TipoEspacio: "m", CantidadEspacios: 2},
		}, "repetido"},
		"sin espacios": {nil, "al menos"},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			repo := &fakeLocalesRepo{}
			err := NewLocalesService(repo).UpdateEspacios(3, c.in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want que contenga %q", err, c.want)
			}
			if repo.llamadas != 0 {
				t.Fatal("no debe escribir con datos invalidos")
			}
		})
	}
}
