package handlers

import (
	"errors"
	"testing"

	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin/binding"
)

func TestRegistrarUsuarioValidationMessage(t *testing.T) {
	tests := []struct {
		name    string
		request registrarUsuarioRequest
		want    string
	}{
		{
			name: "rol obligatorio",
			request: registrarUsuarioRequest{
				Username: "operador",
				Password: "Secreto123",
			},
			want: services.ErrRolObligatorio.Error(),
		},
		{
			name:    "credenciales obligatorias",
			request: registrarUsuarioRequest{},
			want:    services.ErrCredencialesObligatorias.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(tt.request)
			if got := registrarUsuarioValidationMessage(err); got != tt.want {
				t.Fatalf("registrarUsuarioValidationMessage() = %q, want %q", got, tt.want)
			}
		})
	}

	if got := registrarUsuarioValidationMessage(errors.New("json invalido")); got != "body JSON invalido" {
		t.Fatalf("registrarUsuarioValidationMessage() = %q, want %q", got, "body JSON invalido")
	}
}
