package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"atrevida-agenda-api/models"
)

// Vinculo entre reservas y clientes registrados (PLAN_CUENTA_CLIENTE_Y_AJUSTES D30).
// Nada se vincula por adivinanza: el telefono solo enlaza si coincide exacto con un unico cliente.

const (
	// Exacto: sin LIKE ni ultimos digitos, para no enlazar al cliente equivocado.
	sqlClientesPorTelefono = `SELECT id FROM clientes WHERE telefono_e164 = $1 ORDER BY id`

	sqlClienteExiste = `SELECT EXISTS (SELECT 1 FROM clientes WHERE id = $1)`

	sqlReservaParaVincular = `SELECT cliente_id, telefono_e164, COALESCE(cliente, '') AS cliente FROM reservas WHERE id = $1 FOR UPDATE`

	// Solo reservas sin cliente: una ya enlazada no se reasigna porque sus cargos pertenecen a ese cliente.
	sqlVincularReserva = `UPDATE reservas SET cliente_id = $2, actualizado_en = NOW()
		WHERE id = $1 AND cliente_id IS NULL`

	// Un telefono lo pueden compartir varias personas: ademas del telefono exacto se exige el mismo nombre escrito.
	sqlVincularMismoTelefono = `UPDATE reservas SET cliente_id = $2, actualizado_en = NOW()
		WHERE cliente_id IS NULL AND telefono_e164 = $1 AND LOWER(TRIM(cliente)) = $3`

	sqlContarPorVincular = `SELECT COUNT(*) FROM reservas
		WHERE cliente_id IS NULL AND activo = TRUE AND COALESCE(estado, '') <> 'RECHAZADO'`

	sqlReservasPorVincular = `
		SELECT id, fecha, hora_desde::text AS hora_desde, hora_hasta::text AS hora_hasta, local_nombre,
		       cliente, numero_telefono, telefono_e164, servicio_nombre, estado
		FROM reservas
		WHERE cliente_id IS NULL AND activo = TRUE AND COALESCE(estado, '') <> 'RECHAZADO'
		ORDER BY fecha DESC, id DESC
		LIMIT $1 OFFSET $2`

	sqlSugerenciasDeCliente = `
		SELECT * FROM (
			SELECT c.id, c.nombre, c.apellido, c.numero_telefono, c.telefono_e164,
			       COALESCE(c.ci, '') AS ci, COALESCE(c.nit, '') AS nit,
			       (LENGTH(r.digitos) >= 8
			        AND RIGHT(regexp_replace(COALESCE(c.telefono_e164, c.numero_telefono, ''), '\D', '', 'g'), 8) = RIGHT(r.digitos, 8)
			       ) AS por_telefono,
			       (` + "TRANSLATE(LOWER(c.nombre || ' ' || c.apellido), 'áéíóúüñ', 'aeiouun')" + `
			        = ` + "TRANSLATE(LOWER(BTRIM(r.cliente)), 'áéíóúüñ', 'aeiouun')" + `) AS por_nombre
			FROM clientes c
			CROSS JOIN (
				SELECT cliente, regexp_replace(COALESCE(telefono_e164, numero_telefono, ''), '\D', '', 'g') AS digitos
				FROM reservas WHERE id = $1
			) r
		) s
		WHERE por_telefono OR por_nombre
		ORDER BY por_telefono DESC, por_nombre DESC, apellido, nombre, id
		LIMIT 8`
)

type ReservasVinculoRepo struct {
	db *sqlx.DB
}

func NewReservasVinculoRepo(db *sqlx.DB) *ReservasVinculoRepo {
	return &ReservasVinculoRepo{db: db}
}

// IDsPorTelefono devuelve los clientes cuyo telefono E.164 coincide exacto.
func (r *ReservasVinculoRepo) IDsPorTelefono(e164 string) ([]int, error) {
	var ids []int
	if err := r.db.Select(&ids, sqlClientesPorTelefono, e164); err != nil {
		return nil, fmt.Errorf("no se pudo buscar al cliente por telefono: %w", err)
	}
	return ids, nil
}

// Existe indica si hay un cliente con ese id.
func (r *ReservasVinculoRepo) Existe(id int) (bool, error) {
	var existe bool
	if err := r.db.Get(&existe, sqlClienteExiste, id); err != nil {
		return false, fmt.Errorf("no se pudo verificar al cliente: %w", err)
	}
	return existe, nil
}

// PorVincular lista las reservas vigentes sin cliente, las mas recientes primero, y el total.
func (r *ReservasVinculoRepo) PorVincular(limit, offset int) ([]models.ReservaPorVincularPG, int, error) {
	var total int
	if err := r.db.Get(&total, sqlContarPorVincular); err != nil {
		return nil, 0, fmt.Errorf("no se pudo contar las reservas por vincular")
	}
	reservas := []models.ReservaPorVincularPG{}
	if err := r.db.Select(&reservas, sqlReservasPorVincular, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("no se pudieron obtener las reservas por vincular")
	}
	return reservas, total, nil
}

// Sugerencias propone clientes que podrian ser el dueño de la reserva.
func (r *ReservasVinculoRepo) Sugerencias(reservaID int) ([]models.ClienteSugeridoPG, error) {
	sugeridos := []models.ClienteSugeridoPG{}
	if err := r.db.Select(&sugeridos, sqlSugerenciasDeCliente, reservaID); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener las sugerencias de cliente")
	}
	return sugeridos, nil
}

// Vincular enlaza la reserva con el cliente. Con mismoTelefono tambien enlaza las demas reservas sin cliente
// que tengan exactamente ese telefono y el mismo nombre escrito (sin distinguir mayusculas ni espacios sobrantes). Devuelve cuantas reservas quedaron enlazadas.
func (r *ReservasVinculoRepo) Vincular(reservaID, clienteID int, mismoTelefono bool) (int, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var actual struct {
		ClienteID    *int    `db:"cliente_id"`
		TelefonoE164 *string `db:"telefono_e164"`
		Cliente      string  `db:"cliente"`
	}
	if err := tx.Get(&actual, sqlReservaParaVincular, reservaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("reserva no encontrada")
		}
		return 0, fmt.Errorf("no se pudo leer la reserva: %w", err)
	}
	if actual.ClienteID != nil {
		return 0, errors.New("la reserva ya tiene un cliente vinculado")
	}

	var existe bool
	if err := tx.Get(&existe, sqlClienteExiste, clienteID); err != nil {
		return 0, fmt.Errorf("no se pudo verificar al cliente: %w", err)
	}
	if !existe {
		return 0, errors.New("cliente no encontrado")
	}

	res, err := tx.Exec(sqlVincularReserva, reservaID, clienteID)
	if err != nil {
		return 0, fmt.Errorf("no se pudo vincular la reserva: %w", err)
	}
	vinculadas, _ := res.RowsAffected()

	if nombre := strings.ToLower(strings.TrimSpace(actual.Cliente)); mismoTelefono && actual.TelefonoE164 != nil && *actual.TelefonoE164 != "" && nombre != "" {
		res, err := tx.Exec(sqlVincularMismoTelefono, *actual.TelefonoE164, clienteID, nombre)
		if err != nil {
			return 0, fmt.Errorf("no se pudieron vincular las reservas con el mismo telefono: %w", err)
		}
		mas, _ := res.RowsAffected()
		vinculadas += mas
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(vinculadas), nil
}
