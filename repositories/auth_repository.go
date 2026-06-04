package repository

import "atrevida-agenda-api/models"

type AuthRepository interface {
	CreateUsuario(username, passwordHash, rolCodigo string) (int, error)
	GetUsuarioByUsername(username string) (*models.UsuarioPG, error)
	GetUsuarios() ([]models.UsuarioResumenPG, error)
	UpdatePassword(id int, passwordHash string) error
	UpdateActivo(username string, activo bool) error
}
