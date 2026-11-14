package pgsql

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ConfiguracionRow es una fila clave/valor tipada de configuracion_sistema.
type ConfiguracionRow struct {
	Clave string `db:"clave"`
	Valor string `db:"valor"`
	Tipo  string `db:"tipo"`
}

type ConfiguracionRepo struct {
	db *sqlx.DB
}

func NewConfiguracionRepo(db *sqlx.DB) *ConfiguracionRepo {
	return &ConfiguracionRepo{db: db}
}

// GetAll devuelve todos los parametros.
func (r *ConfiguracionRepo) GetAll() ([]ConfiguracionRow, error) {
	var rows []ConfiguracionRow
	if err := r.db.Select(&rows, `SELECT clave, valor, tipo FROM configuracion_sistema ORDER BY clave`); err != nil {
		return nil, fmt.Errorf("error al leer configuracion: %w", err)
	}
	return rows, nil
}

// SetValores actualiza varios parametros en una sola transaccion.
func (r *ConfiguracionRepo) SetValores(valores map[string]string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for clave, valor := range valores {
		res, err := tx.Exec(
			`UPDATE configuracion_sistema SET valor = $1, fecha_modificacion = NOW() WHERE clave = $2`,
			valor, clave,
		)
		if err != nil {
			return fmt.Errorf("error al actualizar parametro %s: %w", clave, err)
		}
		n, err := affectedRows(res, "actualizar configuracion")
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("parametro %s no encontrado", clave)
		}
	}
	return tx.Commit()
}
