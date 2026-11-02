package pgsql

import "fmt"

// textoOrden normaliza una columna de texto para ordenarla sin distinguir mayusculas ni acentos.
func textoOrden(columna string) string {
	return fmt.Sprintf("TRANSLATE(LOWER(%s), 'áéíóúüñ', 'aeiouun')", columna)
}
