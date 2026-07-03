package services

import (
	"errors"
	"testing"
	"time"

	"atrevida-agenda-api/models"

	"golang.org/x/crypto/bcrypt"
)

type fakeAuthRepo struct {
	usuario      *models.UsuarioPG
	createErr    error
	createCalled bool
}

func (f *fakeAuthRepo) CreateUsuario(username, passwordHash, rolCodigo string, localID *int) (int, error) {
	f.createCalled = true
	if f.createErr != nil {
		return 0, f.createErr
	}
	return 7, nil
}

func (f *fakeAuthRepo) GetUsuarioByUsername(username string) (*models.UsuarioPG, error) {
	if f.usuario == nil {
		return nil, errors.New("usuario no encontrado")
	}
	return f.usuario, nil
}

func (f *fakeAuthRepo) GetUsuarios() ([]models.UsuarioResumenPG, error) {
	return nil, nil
}

func (f *fakeAuthRepo) UpdatePassword(id int, passwordHash string) error {
	return nil
}

func (f *fakeAuthRepo) UpdateActivo(username string, activo bool) error {
	return nil
}

func TestAuthServiceLoginIncluyeLocalEnRespuestaYToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Secreto123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}

	localID := 1
	nombreLocal := "SAN MARTIN"
	service := NewAuthService(&fakeAuthRepo{
		usuario: &models.UsuarioPG{
			ID:          5,
			Username:    "operador",
			Password:    string(hash),
			RolCodigo:   "gerencia",
			LocalID:     &localID,
			NombreLocal: &nombreLocal,
		},
	}, "test-secret", time.Hour)

	result, err := service.Login(LoginInput{
		Username: "operador",
		Password: "Secreto123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.LocalID == nil || *result.LocalID != localID {
		t.Fatalf("Login() LocalID = %v, want %d", result.LocalID, localID)
	}
	if result.NombreLocal == nil || *result.NombreLocal != nombreLocal {
		t.Fatalf("Login() NombreLocal = %v, want %q", result.NombreLocal, nombreLocal)
	}

	tokenData, err := service.ValidarToken(result.Token)
	if err != nil {
		t.Fatalf("ValidarToken() error = %v", err)
	}
	if tokenData.LocalID == nil || *tokenData.LocalID != localID {
		t.Fatalf("ValidarToken() LocalID = %v, want %d", tokenData.LocalID, localID)
	}
	if tokenData.NombreLocal == nil || *tokenData.NombreLocal != nombreLocal {
		t.Fatalf("ValidarToken() NombreLocal = %v, want %q", tokenData.NombreLocal, nombreLocal)
	}
}

func TestAuthServiceRegistrarUsuarioValidaLocalInvalido(t *testing.T) {
	repo := &fakeAuthRepo{}
	service := NewAuthService(repo, "test-secret", time.Hour)
	localID := 0

	_, err := service.RegistrarUsuario(RegistrarUsuarioInput{
		Username:  "operador",
		Password:  "Secreto123",
		RolCodigo: "gerencia",
		LocalID:   &localID,
	})
	if !errors.Is(err, ErrLocalInvalido) {
		t.Fatalf("RegistrarUsuario() error = %v, want %v", err, ErrLocalInvalido)
	}
	if repo.createCalled {
		t.Fatal("RegistrarUsuario() called repo with invalid local_id")
	}
}

func TestAuthServiceRegistrarUsuarioMapeaErroresDeLocal(t *testing.T) {
	tests := []struct {
		name      string
		createErr error
		localID   *int
		want      error
	}{
		{
			name:      "local obligatorio",
			createErr: errors.New("local_id es obligatorio para usuarios no admin"),
			localID:   nil,
			want:      ErrLocalObligatorio,
		},
		{
			name:      "local no encontrado",
			createErr: errors.New("local no encontrado"),
			localID:   intPtr(99),
			want:      ErrLocalNoEncontrado,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAuthService(&fakeAuthRepo{createErr: tt.createErr}, "test-secret", time.Hour)

			_, err := service.RegistrarUsuario(RegistrarUsuarioInput{
				Username:  "operador",
				Password:  "Secreto123",
				RolCodigo: "gerencia",
				LocalID:   tt.localID,
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("RegistrarUsuario() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}
