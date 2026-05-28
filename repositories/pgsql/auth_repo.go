package pgsql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

var _ repository.AuthRepository = (*AuthRepo)(nil)

type AuthRepo struct {
	db *sqlx.DB
}

func NewAuthRepo(db *sqlx.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

func (r *AuthRepo) CreateUsuario(username, passwordHash string) (int, error) {
	var usuarioID int

	err := r.db.QueryRowx(`
		INSERT INTO usuarios (username, password, activo)
		VALUES ($1, $2, TRUE)
		RETURNING id
	`, username, passwordHash).Scan(&usuarioID)
	if err != nil {
		if esUniqueUsuariosError(err) {
			return 0, fmt.Errorf("usuario ya existe")
		}
		return 0, fmt.Errorf("no se pudo crear el usuario")
	}

	return usuarioID, nil
}

func (r *AuthRepo) GetUsuarioByUsername(username string) (*models.UsuarioPG, error) {
	var usuario models.UsuarioPG

	err := r.db.Get(&usuario, `
		SELECT id, username, password, activo
		FROM usuarios
		WHERE LOWER(username) = LOWER($1)
		  AND activo = TRUE
	`, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado")
		}
		return nil, fmt.Errorf("no se pudo obtener el usuario")
	}

	return &usuario, nil
}

func esUniqueUsuariosError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
