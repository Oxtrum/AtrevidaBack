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
		SELECT id, username, password, activo, fecha_registro
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

func (r *AuthRepo) GetUsuarios() ([]models.UsuarioResumenPG, error) {
	var usuarios []models.UsuarioResumenPG

	err := r.db.Select(&usuarios, `
		SELECT username, activo, fecha_registro
		FROM usuarios
		ORDER BY fecha_registro DESC, username
	`)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los usuarios")
	}

	return usuarios, nil
}

func (r *AuthRepo) UpdatePassword(id int, passwordHash string) error {
	res, err := r.db.Exec(`
		UPDATE usuarios
		SET password = $1
		WHERE id = $2
		  AND activo = TRUE
	`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("no se pudo actualizar la contrasena")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar la contrasena")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("usuario no encontrado")
	}

	return nil
}

func (r *AuthRepo) UpdateActivo(username string, activo bool) error {
	res, err := r.db.Exec(`
		UPDATE usuarios
		SET activo = $1
		WHERE LOWER(username) = LOWER($2)
	`, activo, username)
	if err != nil {
		if esUniqueUsuariosError(err) {
			return fmt.Errorf("usuario ya existe")
		}
		return fmt.Errorf("no se pudo actualizar el estado del usuario")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el estado del usuario")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("usuario no encontrado")
	}

	return nil
}

func esUniqueUsuariosError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
