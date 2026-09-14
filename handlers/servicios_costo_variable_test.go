package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type serviciosCostoTestRepo struct {
	repository.ServiciosRepository
	creado      repository.CrearServicioInput
	actualizado repository.ActualizarServicioInput
	errUpdate   error
}

func (r *serviciosCostoTestRepo) CreateServicio(input repository.CrearServicioInput) (int, error) {
	r.creado = input
	return 71, nil
}

func (r *serviciosCostoTestRepo) UpdateServicio(input repository.ActualizarServicioInput) error {
	r.actualizado = input
	return r.errUpdate
}

func TestCreateServicioCostoVariableYClienteAnterior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, variable := range []bool{false, true} {
		t.Run(map[bool]string{false: "request anterior", true: "variable sin costo"}[variable], func(t *testing.T) {
			repo := &serviciosCostoTestRepo{}
			h := &Container{ServiciosPG: services.NewServiciosPGService(repo)}
			router := gin.New()
			router.POST("/bd/servicios", h.CreateServicio)
			body := `{"nombre":"Servicio","categoria":"Manual"}`
			if variable {
				body = `{"nombre":"Servicio","categoria":"Manual","costo_variable":true}`
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/bd/servicios", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", w.Code, w.Body.String())
			}
			if repo.creado.CostoVariable != variable {
				t.Fatal("modalidad no coincide")
			}
			if variable && (repo.creado.Costo == nil || *repo.creado.Costo != 0) {
				t.Fatal("variable no guarda cero")
			}
			if !variable && repo.creado.Costo != nil {
				t.Fatal("request anterior sin costo cambio")
			}
		})
	}
}

func TestServicioCostoInvalidoResponde400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Container{ServiciosPG: services.NewServiciosPGService(&serviciosCostoTestRepo{})}
	router := gin.New()
	router.POST("/bd/servicios", h.CreateServicio)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bd/servicios", strings.NewReader(`{"nombre":"Servicio","categoria":"Manual","costo":-1}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchServicioModalidadFalseEsCampoYErroresSon400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, errUpdate := range []error{nil, repository.ErrCostoServicioInvalido} {
		repo := &serviciosCostoTestRepo{errUpdate: errUpdate}
		h := &Container{ServiciosPG: services.NewServiciosPGService(repo)}
		router := gin.New()
		router.PATCH("/bd/servicios/:id", h.UpdateServicio)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/bd/servicios/71", strings.NewReader(`{"costo_variable":false}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		want := http.StatusOK
		if errors.Is(errUpdate, repository.ErrCostoServicioInvalido) {
			want = http.StatusBadRequest
		}
		if w.Code != want {
			t.Fatalf("status = %d, want %d: %s", w.Code, want, w.Body.String())
		}
		if repo.actualizado.CostoVariable == nil || *repo.actualizado.CostoVariable {
			t.Fatal("false se confundio con campo omitido")
		}
		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
	}
}
