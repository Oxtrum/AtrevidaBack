package pgsql

import (
	"database/sql"
	"fmt"
)

func affectedRows(result sql.Result, operation string) (int64, error) {
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error al verificar filas afectadas en %s: %w", operation, err)
	}
	return count, nil
}
