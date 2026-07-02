package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

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

func (r *AuthRepo) CreateUsuario(username, passwordHash, rolCodigo string, localID *int) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("no se pudo crear el usuario")
	}
	defer tx.Rollback()

	var rol struct {
		ID     int    `db:"id"`
		Codigo string `db:"codigo"`
	}
	err = tx.Get(&rol, `
		SELECT id, codigo
		FROM roles
		WHERE LOWER(codigo) = LOWER($1)
	`, rolCodigo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("rol no encontrado")
		}
		return 0, fmt.Errorf("no se pudo crear el usuario")
	}

	var localIDFinal *int
	var nombreLocal *string
	if !esRolAdminSys(rol.Codigo) {
		if localID == nil || *localID <= 0 {
			return 0, fmt.Errorf("local_id es obligatorio para usuarios no admin")
		}

		var nombre string
		err = tx.Get(&nombre, `
			SELECT nombre
			FROM locales
			WHERE id = $1
		`, *localID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("local no encontrado")
			}
			return 0, fmt.Errorf("no se pudo crear el usuario")
		}

		localIDValue := *localID
		localIDFinal = &localIDValue
		nombreLocal = &nombre
	}

	var usuarioID int

	err = tx.QueryRowx(`
		INSERT INTO usuarios (username, password, activo, rol_id, local_id, nombre_local)
		VALUES ($1, $2, TRUE, $3, $4, $5)
		RETURNING id
	`, username, passwordHash, rol.ID, localIDFinal, nombreLocal).Scan(&usuarioID)
	if err != nil {
		if esUniqueUsuariosError(err) {
			return 0, fmt.Errorf("usuario ya existe")
		}
		return 0, fmt.Errorf("no se pudo crear el usuario")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("no se pudo crear el usuario")
	}

	return usuarioID, nil
}

func (r *AuthRepo) GetUsuarioByUsername(username string) (*models.UsuarioPG, error) {
	var usuario models.UsuarioPG

	err := r.db.Get(&usuario, `
		SELECT u.id,
		       u.username,
		       u.password,
		       u.activo,
		       u.fecha_registro,
		       u.rol_id,
		       r.codigo AS rol_codigo,
		       r.nombre_rol AS rol_nombre,
		       u.local_id,
		       u.nombre_local
		FROM usuarios u
		JOIN roles r ON r.id = u.rol_id
		WHERE LOWER(u.username) = LOWER($1)
		  AND u.activo = TRUE
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
		SELECT u.username,
		       u.activo,
		       u.fecha_registro,
		       r.codigo AS rol_codigo,
		       r.nombre_rol AS rol_nombre,
		       u.local_id,
		       u.nombre_local
		FROM usuarios u
		JOIN roles r ON r.id = u.rol_id
		ORDER BY u.fecha_registro DESC, u.username
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
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el estado del usuario")
	}
	defer tx.Rollback()

	var usuario struct {
		ID        int    `db:"id"`
		Activo    bool   `db:"activo"`
		RolCodigo string `db:"rol_codigo"`
	}
	err = tx.Get(&usuario, `
		SELECT u.id,
		       u.activo,
		       r.codigo AS rol_codigo
		FROM usuarios u
		JOIN roles r ON r.id = u.rol_id
		WHERE LOWER(u.username) = LOWER($1)
	`, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("usuario no encontrado")
		}
		return fmt.Errorf("no se pudo actualizar el estado del usuario")
	}

	if usuario.Activo && !activo && strings.EqualFold(usuario.RolCodigo, "admin_sys") {
		var adminsActivos int
		err = tx.Get(&adminsActivos, `
			SELECT COUNT(*)
			FROM usuarios u
			JOIN roles r ON r.id = u.rol_id
			WHERE u.activo = TRUE
			  AND LOWER(r.codigo) = LOWER('admin_sys')
		`)
		if err != nil {
			return fmt.Errorf("no se pudo actualizar el estado del usuario")
		}
		if adminsActivos <= 1 {
			return fmt.Errorf("no puedes desactivar al unico usuario admin_sys activo")
		}
	}

	res, err := tx.Exec(`
		UPDATE usuarios
		SET activo = $1
		WHERE id = $2
	`, activo, usuario.ID)
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

	return tx.Commit()
}

func esUniqueUsuariosError(err error) bool {
	var pqErr *pgconn.PgError
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func esRolAdminSys(rolCodigo string) bool {
	return strings.EqualFold(strings.TrimSpace(rolCodigo), "admin_sys")
}
