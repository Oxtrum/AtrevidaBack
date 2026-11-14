package pgsql

import (
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	"atrevida-agenda-api/pagination"
)

// DeudoresOrdenables lista las columnas por las que se puede ordenar el listado de clientes con deuda (ordenar_por).
var DeudoresOrdenables = pagination.Columns{
	"deuda":           "(COALESCE(ca.total, 0) - COALESCE(ab.total, 0))",
	"total_cargos":    "COALESCE(ca.total, 0)",
	"total_abonos":    "COALESCE(ab.total, 0)",
	"nombre":          textoOrden("c.nombre"),
	"apellido":        textoOrden("c.apellido"),
	"numero_telefono": "c.numero_telefono",
}

// FiltroDeudores filtra y pagina el listado de clientes con deuda. Pagina siempre por desplazamiento: el orden
// depende de totales calculados, que no admiten un cursor por valor.
type FiltroDeudores struct {
	Busqueda   string
	OrdenarPor string
	OrdenDesc  bool
	// Limit es el tamano de pagina (con una fila extra para saber si hay otra pagina); 0 = sin limite.
	Limit  int
	Offset int
}

const deudoresPorDefecto = "(COALESCE(ca.total, 0) - COALESCE(ab.total, 0)) DESC, c.id"

// deudoresFrom arma el FROM y el WHERE comunes al listado y a su conteo. Solo cuentan los abonos activos y pagados
// y los cargos no anulados, igual que el saldo de la cuenta de un cliente.
func deudoresFrom(f FiltroDeudores) (string, []interface{}) {
	where := "COALESCE(ca.total, 0) - COALESCE(ab.total, 0) > 0"
	args := []interface{}{}
	if busqueda := strings.TrimSpace(f.Busqueda); busqueda != "" {
		args = append(args, "%"+busqueda+"%")
		where += fmt.Sprintf(" AND (c.nombre ILIKE $%d OR c.apellido ILIKE $%d OR (c.nombre || ' ' || c.apellido) ILIKE $%d OR c.numero_telefono ILIKE $%d)", 1, 1, 1, 1)
	}
	from := fmt.Sprintf(`
		FROM clientes c
		LEFT JOIN (
			SELECT cliente_id, SUM(total_final) AS total FROM pagos
			WHERE es_abono = TRUE AND activo = TRUE AND estado = 'PAGADO' GROUP BY cliente_id
		) ab ON ab.cliente_id = c.id
		LEFT JOIN (
			SELECT cliente_id, SUM(monto) AS total FROM cargos_cliente
			WHERE anulado = FALSE GROUP BY cliente_id
		) ca ON ca.cliente_id = c.id
		WHERE %s`, where)
	return from, args
}

func buildDeudoresQuery(f FiltroDeudores) (string, []interface{}) {
	from, args := deudoresFrom(f)
	orden := pagination.Sort{Key: f.OrdenarPor, Desc: f.OrdenDesc}.OrderBy(DeudoresOrdenables, deudoresPorDefecto, "c.id")
	page := ""
	if f.Limit > 0 {
		args = append(args, f.Limit)
		page = fmt.Sprintf(" LIMIT $%d", len(args))
		if f.Offset > 0 {
			args = append(args, f.Offset)
			page += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	}
	query := fmt.Sprintf(`
		SELECT c.id AS cliente_id, c.nombre, c.apellido, c.numero_telefono,
		       COALESCE(ca.total, 0) AS total_cargos, COALESCE(ab.total, 0) AS total_abonos,
		       COALESCE(ca.total, 0) - COALESCE(ab.total, 0) AS deuda
		%s
		ORDER BY %s%s`, from, orden, page)
	return query, args
}

// ListarDeudores devuelve los clientes que deben, con la mayor deuda primero salvo que se pida otro orden.
func (r *CuentaRepo) ListarDeudores(f FiltroDeudores) ([]models.DeudorCliente, error) {
	query, args := buildDeudoresQuery(f)
	deudores := []models.DeudorCliente{}
	if err := r.db.Select(&deudores, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron listar los clientes con deuda: %w", err)
	}
	return deudores, nil
}

// ContarDeudores cuenta los clientes con deuda que cumplen la busqueda.
func (r *CuentaRepo) ContarDeudores(f FiltroDeudores) (int, error) {
	from, args := deudoresFrom(f)
	var total int
	if err := r.db.Get(&total, "SELECT COUNT(*) "+from, args...); err != nil {
		return 0, fmt.Errorf("no se pudieron contar los clientes con deuda: %w", err)
	}
	return total, nil
}
