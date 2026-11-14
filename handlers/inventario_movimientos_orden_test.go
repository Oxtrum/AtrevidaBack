package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetInventarioMovimientosRechazaOrdenInvalidoAntesDeConsultar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Container{}
	for _, query := range []string{
		"ordenar_por=password",
		"ordenar_por=cantidad&sentido=arriba",
		"sentido=desc",
		"ordenar_por=" + url.QueryEscape("m.id; DROP TABLE inventario_movimientos"),
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/bd/inventario/movimientos?"+query, nil)
		h.GetInventarioMovimientos(c)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, body = %s", query, w.Code, w.Body.String())
		}
	}
}
