package pgsql

import (
	"fmt"
	"strings"
)

func batchValuesPlaceholders(rows, columns int) string {
	groups := make([]string, rows)
	parameter := 1
	for row := 0; row < rows; row++ {
		values := make([]string, columns)
		for column := 0; column < columns; column++ {
			values[column] = fmt.Sprintf("$%d", parameter)
			parameter++
		}
		groups[row] = "(" + strings.Join(values, ",") + ")"
	}
	return strings.Join(groups, ",")
}
