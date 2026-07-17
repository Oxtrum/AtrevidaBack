package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type combosHandlerRepo struct {
	combos  []models.ComboCatalogoPG
	total   int
	filter  repository.FiltroCombos
	activo  *bool
	comboID int
}

func (r *combosHandlerRepo) ListCombos(filter repository.FiltroCombos) ([]models.ComboCatalogoPG, int, error) {
	r.filter = filter
	return r.combos, r.total, nil
}

func (r *combosHandlerRepo) GetComboByID(int, bool) (*models.ComboCatalogoPG, error) { return nil, nil }
func (r *combosHandlerRepo) CreateCombo(repository.CrearComboInput) (int, error)     { return 0, nil }
func (r *combosHandlerRepo) UpdateCombo(repository.ActualizarComboInput) error       { return nil }
func (r *combosHandlerRepo) SetComboActivo(id int, activo bool) error {
	r.comboID = id
	r.activo = &activo
	return nil
}
func (r *combosHandlerRepo) SetComboLocales(int, []int) error { return nil }
func (r *combosHandlerRepo) SetComboImagen(int, *string) error { return nil }
func (r *combosHandlerRepo) ReplaceComboServicios(int, []repository.ComboServicioCatalogoInput) error {
	return nil
}

func TestGetCombosPGListaCompletaYExponeFiltrosAplicados(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &combosHandlerRepo{combos: []models.ComboCatalogoPG{{ID: 7, Nombre: "Relax"}}, total: 1}
	handler := &Container{CombosPG: services.NewCombosService(repo)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/combos?nombre=rel&local_id=2", nil)

	handler.GetCombosPG(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if repo.filter.Nombre != "rel" || repo.filter.LocalID == nil || *repo.filter.LocalID != 2 {
		t.Fatalf("filtro = %#v, want nombre rel y local_id 2", repo.filter)
	}
}

func TestGetCombosPGRechazaLocalIDInvalido(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Container{CombosPG: services.NewCombosService(&combosHandlerRepo{})}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/combos?local_id=0", nil)

	handler.GetCombosPG(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteComboPGRealizaBorradoLogico(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &combosHandlerRepo{}
	handler := &Container{CombosPG: services.NewCombosService(repo)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/bd/combos/7", nil)

	handler.DeleteComboPG(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if repo.comboID != 7 || repo.activo == nil || *repo.activo {
		t.Fatalf("estado enviado = id:%d activo:%v, want id:7 activo:false", repo.comboID, repo.activo)
	}
}
