package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCredencialesObligatorias = errors.New("username y password son obligatorios")
	ErrUsuarioNoEncontrado      = errors.New("usuario no encontrado")
	ErrPasswordIncorrecta       = errors.New("contrasena incorrecta")
	ErrUsuarioYaExiste          = errors.New("usuario ya existe")
	ErrRolObligatorio           = errors.New("rol_codigo es obligatorio")
	ErrRolNoEncontrado          = errors.New("rol no encontrado")
	ErrNoAutorizado             = errors.New("Usuario no autorizado")
	ErrNoModificarPropioEstado  = errors.New("no puedes modificar tu propio estado")
	ErrUltimoAdminSysActivo     = errors.New("no puedes desactivar al unico usuario admin_sys activo")
)

type AuthService struct {
	repo        repository.AuthRepository
	tokenSecret []byte
	tokenTTL    time.Duration
}

func NewAuthService(repo repository.AuthRepository, tokenSecret string, tokenTTL time.Duration) *AuthService {
	if tokenSecret == "" {
		tokenSecret = "atrevida-local-dev-secret"
	}
	if tokenTTL <= 0 {
		tokenTTL = time.Hour
	}

	return &AuthService{
		repo:        repo,
		tokenSecret: []byte(tokenSecret),
		tokenTTL:    tokenTTL,
	}
}

type RegistrarUsuarioInput struct {
	Username  string
	Password  string
	RolCodigo string
}

func (s *AuthService) RegistrarUsuario(input RegistrarUsuarioInput) (int, error) {
	username := strings.TrimSpace(input.Username)
	password := input.Password
	rolCodigo := strings.TrimSpace(input.RolCodigo)

	if username == "" || strings.TrimSpace(password) == "" {
		return 0, ErrCredencialesObligatorias
	}
	if rolCodigo == "" {
		return 0, ErrRolObligatorio
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, errors.New("no se pudo encriptar la password")
	}

	id, err := s.repo.CreateUsuario(username, string(hash), rolCodigo)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "ya existe") {
			return 0, ErrUsuarioYaExiste
		}
		if strings.Contains(strings.ToLower(err.Error()), "rol no encontrado") {
			return 0, ErrRolNoEncontrado
		}
		return 0, err
	}

	return id, nil
}

func (s *AuthService) GetUsuarios() ([]models.UsuarioResumenPG, error) {
	return s.repo.GetUsuarios()
}

type CambiarPasswordInput struct {
	TokenUserID int
	Password    string
}

func (s *AuthService) CambiarPassword(input CambiarPasswordInput) error {
	if input.TokenUserID <= 0 {
		return errors.New("token invalido")
	}
	if strings.TrimSpace(input.Password) == "" {
		return ErrCredencialesObligatorias
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("no se pudo encriptar la password")
	}

	err = s.repo.UpdatePassword(input.TokenUserID, string(hash))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "usuario no encontrado") {
			return ErrUsuarioNoEncontrado
		}
		return err
	}

	return nil
}

type ActualizarUsuarioActivoInput struct {
	Username      string
	TokenUsername string
	Activo        bool
}

func (s *AuthService) ActualizarUsuarioActivo(input ActualizarUsuarioActivoInput) error {
	username := strings.TrimSpace(input.Username)
	tokenUsername := strings.TrimSpace(input.TokenUsername)

	if username == "" {
		return errors.New("username es requerido")
	}
	if tokenUsername == "" {
		return errors.New("token invalido")
	}
	if strings.EqualFold(username, tokenUsername) {
		return ErrNoModificarPropioEstado
	}

	err := s.repo.UpdateActivo(username, input.Activo)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "usuario no encontrado") {
			return ErrUsuarioNoEncontrado
		}
		if strings.Contains(strings.ToLower(err.Error()), "ya existe") {
			return ErrUsuarioYaExiste
		}
		if strings.Contains(strings.ToLower(err.Error()), "unico usuario admin_sys activo") {
			return ErrUltimoAdminSysActivo
		}
		return err
	}

	return nil
}

type LoginInput struct {
	Username string
	Password string
}

type LoginResult struct {
	Token     string
	Username  string
	RolCodigo string
	ExpiresIn int
}

func (s *AuthService) Login(input LoginInput) (*LoginResult, error) {
	username := strings.TrimSpace(input.Username)
	password := input.Password

	if username == "" || strings.TrimSpace(password) == "" {
		return nil, ErrCredencialesObligatorias
	}

	usuario, err := s.repo.GetUsuarioByUsername(username)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "usuario no encontrado") {
			return nil, ErrUsuarioNoEncontrado
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(password)); err != nil {
		return nil, ErrPasswordIncorrecta
	}

	token, err := s.generarToken(usuario.ID, usuario.Username, usuario.RolCodigo)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:     token,
		Username:  usuario.Username,
		RolCodigo: usuario.RolCodigo,
		ExpiresIn: int(s.tokenTTL.Seconds()),
	}, nil
}

type TokenData struct {
	UserID    int
	Username  string
	RolCodigo string
}

func (s *AuthService) ValidarToken(token string) (*TokenData, error) {
	claims, err := s.parsearToken(token)
	if err != nil {
		return nil, err
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token expirado")
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil || userID <= 0 {
		return nil, errors.New("token invalido")
	}

	return &TokenData{
		UserID:    userID,
		Username:  claims.Username,
		RolCodigo: claims.RolCodigo,
	}, nil
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenClaims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	RolCodigo string `json:"rol_codigo"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func (s *AuthService) generarToken(usuarioID int, username, rolCodigo string) (string, error) {
	now := time.Now()
	header := tokenHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}
	claims := tokenClaims{
		Subject:   fmt.Sprintf("%d", usuarioID),
		Username:  username,
		RolCodigo: rolCodigo,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(s.tokenTTL).Unix(),
	}

	headerPart, err := encodeTokenPart(header)
	if err != nil {
		return "", errors.New("no se pudo generar el token")
	}
	claimsPart, err := encodeTokenPart(claims)
	if err != nil {
		return "", errors.New("no se pudo generar el token")
	}

	signingInput := headerPart + "." + claimsPart
	signature := s.firmar(signingInput)

	return signingInput + "." + signature, nil
}

func (s *AuthService) parsearToken(token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("token invalido")
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := s.firmar(signingInput)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return nil, errors.New("token invalido")
	}

	var header tokenHeader
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("token invalido")
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.New("token invalido")
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return nil, errors.New("token invalido")
	}

	var claims tokenClaims
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("token invalido")
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, errors.New("token invalido")
	}

	return &claims, nil
}

func (s *AuthService) firmar(signingInput string) string {
	mac := hmac.New(sha256.New, s.tokenSecret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeTokenPart(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
