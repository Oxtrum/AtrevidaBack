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

var _ repository.ClientesRepository = (*ClientesRepo)(nil)

type ClientesRepo struct {
	db *sqlx.DB
}

func NewClientesRepo(db *sqlx.DB) *ClientesRepo {
	return &ClientesRepo{db: db}
}

func (r *ClientesRepo) GetClientes(filtro repository.FiltroClientes) ([]models.ClientePG, error) {
	conditions, args := clienteFilterConditions(filtro)
	idx := len(args) + 1
	if filtro.CursorSet {
		conditions = append(conditions, fmt.Sprintf("(apellido, nombre, id) > ($%d, $%d, $%d)", idx, idx+1, idx+2))
		args = append(args, filtro.CursorApellido, filtro.CursorNombre, filtro.CursorID)
		idx += 3
	}
	limitClause := ""
	if filtro.PageLimit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filtro.PageLimit)
	}

	query := fmt.Sprintf(`
		SELECT id, nombre, apellido, numero_telefono, telefono_e164,
		       COALESCE(ci, '') AS ci, COALESCE(nit, '') AS nit
		FROM clientes
		WHERE %s
		ORDER BY apellido, nombre, id%s
	`, strings.Join(conditions, " AND "), limitClause)

	var clientes []models.ClientePG
	if err := r.db.SelectContext(queryContext(filtro.Context), &clientes, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los clientes")
	}
	if clientes == nil {
		clientes = []models.ClientePG{}
	}

	return clientes, nil
}

func (r *ClientesRepo) CountClientes(filtro repository.FiltroClientes) (int, error) {
	conditions, args := clienteFilterConditions(filtro)
	query := fmt.Sprintf("SELECT COUNT(*) FROM clientes WHERE %s", strings.Join(conditions, " AND "))
	var total int
	if err := r.db.GetContext(queryContext(filtro.Context), &total, query, args...); err != nil {
		return 0, fmt.Errorf("no se pudo contar los clientes")
	}
	return total, nil
}

func clienteFilterConditions(filtro repository.FiltroClientes) ([]string, []interface{}) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	addLike := func(column, value string) {
		if value == "" {
			return
		}
		conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", column, len(args)+1))
		args = append(args, "%"+value+"%")
	}
	addLike("nombre", filtro.Nombre)
	addLike("apellido", filtro.Apellido)
	if filtro.NumeroTelefono != "" {
		digitos := soloDigitosCliente(filtro.NumeroTelefono)
		if digitos != "" {
			ultimos := digitos
			if len(ultimos) > 8 {
				ultimos = ultimos[len(ultimos)-8:]
			}
			idx := len(args) + 1
			// `telefono_e164` es la fuente canónica. La comparación por últimos
			// dígitos mantiene la búsqueda con el formato boliviano histórico.
			conditions = append(conditions, fmt.Sprintf(`(
				regexp_replace(COALESCE(telefono_e164, ''), '\D', '', 'g') = $%d
				OR RIGHT(regexp_replace(COALESCE(telefono_e164, ''), '\D', '', 'g'), 8) = $%d
				OR regexp_replace(numero_telefono, '\D', '', 'g') = $%d
				OR RIGHT(regexp_replace(numero_telefono, '\D', '', 'g'), 8) = $%d
			)`, idx, idx+1, idx+2, idx+3))
			args = append(args, digitos, ultimos, digitos, ultimos)
		}
	}
	if filtro.Busqueda != "" {
		idx := len(args) + 1
		conditions = append(conditions, fmt.Sprintf("(nombre ILIKE $%d OR apellido ILIKE $%d OR (nombre || ' ' || apellido) ILIKE $%d OR numero_telefono ILIKE $%d OR COALESCE(telefono_e164, '') ILIKE $%d)", idx, idx, idx, idx, idx))
		args = append(args, "%"+filtro.Busqueda+"%")
	}
	return conditions, args
}

func soloDigitosCliente(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (r *ClientesRepo) GetClienteByID(id int) (*models.ClientePG, error) {
	var cliente models.ClientePG

	err := r.db.Get(&cliente, `
		SELECT id, nombre, apellido, numero_telefono, telefono_e164,
		       COALESCE(ci, '') AS ci, COALESCE(nit, '') AS nit
		FROM clientes
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("cliente no encontrado")
		}
		return nil, fmt.Errorf("no se pudo obtener el cliente")
	}

	return &cliente, nil
}

func (r *ClientesRepo) CreateCliente(input repository.CrearClienteInput) (int, error) {
	var clienteID int

	// NULLIF: una cadena vacia se guarda como NULL, no como ''. Asi la columna
	// distingue "no registrado" de "registrado en blanco".
	err := r.db.QueryRowx(`
		INSERT INTO clientes (nombre, apellido, numero_telefono, telefono_e164, ci, nit)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''))
		RETURNING id
	`, input.Nombre, input.Apellido, input.NumeroTelefono, input.TelefonoE164, input.CI, input.NIT).Scan(&clienteID)
	if err != nil {
		if esUniqueClientesError(err) {
			return 0, fmt.Errorf("ya existe un cliente con ese nombre, apellido y numero de telefono")
		}
		return 0, fmt.Errorf("no se pudo crear el cliente")
	}

	return clienteID, nil
}

func (r *ClientesRepo) UpdateCliente(input repository.ActualizarClienteInput) error {
	sets := []string{}
	args := []interface{}{}
	idx := 1

	if input.Nombre != nil {
		sets = append(sets, fmt.Sprintf("nombre = $%d", idx))
		args = append(args, *input.Nombre)
		idx++
	}
	if input.Apellido != nil {
		sets = append(sets, fmt.Sprintf("apellido = $%d", idx))
		args = append(args, *input.Apellido)
		idx++
	}
	if input.NumeroTelefono != nil {
		sets = append(sets, fmt.Sprintf("numero_telefono = $%d", idx))
		args = append(args, *input.NumeroTelefono)
		idx++
	}
	if input.TelefonoE164Set {
		sets = append(sets, fmt.Sprintf("telefono_e164 = $%d", idx))
		args = append(args, input.TelefonoE164)
		idx++
	}
	if input.CI != nil {
		sets = append(sets, fmt.Sprintf("ci = NULLIF($%d, '')", idx))
		args = append(args, *input.CI)
		idx++
	}
	if input.NIT != nil {
		sets = append(sets, fmt.Sprintf("nit = NULLIF($%d, '')", idx))
		args = append(args, *input.NIT)
		idx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("debe especificarse al menos un campo a modificar")
	}

	args = append(args, input.ID)
	query := fmt.Sprintf(
		"UPDATE clientes SET %s WHERE id = $%d",
		strings.Join(sets, ", "), idx,
	)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		if esUniqueClientesError(err) {
			return fmt.Errorf("ya existe un cliente con ese nombre, apellido y numero de telefono")
		}
		return fmt.Errorf("no se pudo actualizar el cliente")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el cliente")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("cliente no encontrado")
	}

	return nil
}

func (r *ClientesRepo) DeleteCliente(id int) error {
	res, err := r.db.Exec(`DELETE FROM clientes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el cliente")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el cliente")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("cliente no encontrado")
	}

	return nil
}

func esUniqueClientesError(err error) bool {
	var pqErr *pgconn.PgError
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
