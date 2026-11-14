package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"atrevida-agenda-api/models"

	"github.com/gin-gonic/gin"
)

// pasaMiddleware ejecuta el middleware con un usuario del rol dado y devuelve el status final.
func pasaMiddleware(t *testing.T, rol string, middleware gin.HandlerFunc) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) { c.Set("auth_rol_codigo", rol) }, middleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w.Code
}

func TestPermisosPorRol(t *testing.T) {
	h := &Container{}
	tests := []struct {
		name       string
		middleware gin.HandlerFunc
		permitidos map[string]bool
	}{
		{
			name:       "AdminSysRequired: lo financiero es solo de admin_sys",
			middleware: h.AdminSysRequired,
			permitidos: map[string]bool{models.RolAdminSys: true, models.RolCoordinacion: false, models.RolGerencia: false},
		},
		{
			name:       "CoordinacionRequired: catalogo e inventario, sin gerencia",
			middleware: h.CoordinacionRequired,
			permitidos: map[string]bool{models.RolAdminSys: true, models.RolCoordinacion: true, models.RolGerencia: false},
		},
		{
			name:       "GerenciaRequired: cualquier rol operativo",
			middleware: h.GerenciaRequired,
			permitidos: map[string]bool{models.RolAdminSys: true, models.RolCoordinacion: true, models.RolGerencia: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for rol, permitido := range tt.permitidos {
				want := http.StatusForbidden
				if permitido {
					want = http.StatusOK
				}
				if got := pasaMiddleware(t, rol, tt.middleware); got != want {
					t.Errorf("rol %s: status = %d, want %d", rol, got, want)
				}
			}
			if got := pasaMiddleware(t, "rol_desconocido", tt.middleware); got != http.StatusForbidden {
				t.Errorf("rol desconocido: status = %d, want %d", got, http.StatusForbidden)
			}
		})
	}
}

// La coordinadora trabaja sobre todos los locales aunque su token no traiga local; gerencia sin local no pasa.
func TestAlcancePorLocalSegunRol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contexto := func(rol string, localID int) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("auth_rol_codigo", rol)
		if localID > 0 {
			c.Set("auth_local_id", localID)
			c.Set("auth_nombre_local", "PASEO ARANJUEZ")
		}
		return c
	}

	for _, rol := range []string{models.RolAdminSys, models.RolCoordinacion} {
		scope, ok := authenticatedLocalScopeFromToken(contexto(rol, 0))
		if !ok || scope.LocalID != nil {
			t.Errorf("rol %s: scope = %+v ok = %v, want sin local y ok", rol, scope, ok)
		}
	}

	if _, ok := authenticatedLocalScopeFromToken(contexto(models.RolGerencia, 0)); ok {
		t.Error("gerencia sin local: want rechazo")
	}
	scope, ok := authenticatedLocalScopeFromToken(contexto(models.RolGerencia, 3))
	if !ok || scope.LocalID == nil || *scope.LocalID != 3 {
		t.Errorf("gerencia con local 3: scope = %+v ok = %v", scope, ok)
	}
}
