package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagoPrecioPendienteYCeroExplicito(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, caso := range []struct {
		nombre string
		precio any
		omitir bool
		valido bool
	}{
		{"omitido", nil, true, false}, {"null", nil, false, false},
		{"vacio", "", false, false}, {"cero", 0, false, true}, {"decimal", 70.25, false, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			item := map[string]any{"servicio_id": nil, "servicio": "Variable", "cantidad": 1, "subtotal": 0}
			if !caso.omitir {
				item["precio_unitario"] = caso.precio
			}
			body, err := json.Marshal(map[string]any{"local_id": 1, "local_nombre": "SAN MARTIN", "cliente_id": nil, "cliente_nombre": "Prueba", "descuento": 0, "tipo_pago": "qr", "estado": "PAGADO", "activo": true, "detalle": []any{item}})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/bd/pagos", strings.NewReader(string(body)))
			req, ok := parseCrearPagoRequest(c)
			if ok != caso.valido {
				t.Fatalf("valido = %v, esperado %v: %s", ok, caso.valido, w.Body.String())
			}
			if !ok && w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d", w.Code)
			}
			if ok && req.Detalle[0].PrecioUnitario == nil {
				t.Fatal("precio explicito perdido")
			}
		})
	}
}
