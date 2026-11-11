package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolverLocalIDNoPermiteCambiarElLocalDelToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("auth_rol_codigo", "cajero")
	c.Set("auth_local_id", 7)
	c.Set("auth_nombre_local", "Local 7")
	pedido := 8

	if _, ok := resolverLocalID(c, &pedido); ok {
		t.Fatal("un usuario asignado a un local no debe poder consultar otro")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; se esperaba %d", w.Code, http.StatusForbidden)
	}
}

func TestResolverLocalIDFijaElLocalDelTokenAunqueNoSeEnvieFiltro(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("auth_rol_codigo", "cajero")
	c.Set("auth_local_id", 7)
	c.Set("auth_nombre_local", "Local 7")

	local, ok := resolverLocalID(c, nil)
	if !ok || local == nil || *local != 7 {
		t.Fatalf("alcance resuelto = %v, %v; se esperaba local 7", local, ok)
	}
}
