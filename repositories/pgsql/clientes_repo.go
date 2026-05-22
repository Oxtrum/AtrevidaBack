package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
	conditions := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if filtro.Nombre != "" {
		conditions = append(conditions, fmt.Sprintf("nombre ILIKE $%d", idx))
		args = append(args, "%"+filtro.Nombre+"%")
		idx++
	}
	if filtro.Apellido != "" {
		conditions = append(conditions, fmt.Sprintf("apellido ILIKE $%d", idx))
		args = append(args, "%"+filtro.Apellido+"%")
		idx++
	}
	if filtro.NumeroTelefono != "" {
		conditions = append(conditions, fmt.Sprintf("numero_telefono ILIKE $%d", idx))
		args = append(args, "%"+filtro.NumeroTelefono+"%")
		idx++
	}

	query := fmt.Sprintf(`
		SELECT id, nombre, apellido, numero_telefono
		FROM clientes
		WHERE %s
		ORDER BY apellido, nombre, id
	`, strings.Join(conditions, " AND "))

	var clientes []models.ClientePG
	if err := r.db.Select(&clientes, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los clientes")
	}

	return clientes, nil
}

func (r *ClientesRepo) GetClienteByID(id int) (*models.ClientePG, error) {
	var cliente models.ClientePG

	err := r.db.Get(&cliente, `
		SELECT id, nombre, apellido, numero_telefono
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

func (r *ClientesRepo) CreateCliente(nombre, apellido, numeroTelefono string) (int, error) {
	var clienteID int

	err := r.db.QueryRowx(`
		INSERT INTO clientes (nombre, apellido, numero_telefono)
		VALUES ($1, $2, $3)
		RETURNING id
	`, nombre, apellido, numeroTelefono).Scan(&clienteID)
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
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
