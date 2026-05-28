package repository

import "atrevida-agenda-api/models"

type AuthRepository interface {
	CreateUsuario(username, passwordHash string) (int, error)
	GetUsuarioByUsername(username string) (*models.UsuarioPG, error)
}
