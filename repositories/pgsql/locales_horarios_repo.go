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

var _ repository.LocalesHorariosRepository = (*LocalesHorariosRepo)(nil)

type LocalesHorariosRepo struct {
	db *sqlx.DB
}

func NewLocalesHorariosRepo(db *sqlx.DB) *LocalesHorariosRepo {
	return &LocalesHorariosRepo{db: db}
}

func (r *LocalesHorariosRepo) GetHorarioByID(id int) (*models.LocalHorarioPG, error) {
	var horario models.LocalHorarioPG

	err := r.db.Get(&horario, `
		SELECT
			id,
			local_id,
			dia_semana,
			TO_CHAR(hora_desde, 'HH24:MI') AS hora_desde,
			TO_CHAR(hora_hasta, 'HH24:MI') AS hora_hasta,
			activo
		FROM locales_horarios
		WHERE id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("horario no encontrado")
		}
		return nil, fmt.Errorf("no se pudo obtener el horario")
	}

	return &horario, nil
}

func (r *LocalesHorariosRepo) GetHorariosByLocal(filtro repository.FiltroLocalHorarios) ([]models.LocalHorarioPG, error) {
	if err := r.validarLocalActivo(filtro.LocalID); err != nil {
		return nil, err
	}

	conditions := []string{"local_id = $1", "activo = TRUE"}
	args := []interface{}{filtro.LocalID}
	idx := 2

	if filtro.DiaSemana != nil {
		conditions = append(conditions, fmt.Sprintf("dia_semana = $%d", idx))
		args = append(args, *filtro.DiaSemana)
		idx++
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			local_id,
			dia_semana,
			TO_CHAR(hora_desde, 'HH24:MI') AS hora_desde,
			TO_CHAR(hora_hasta, 'HH24:MI') AS hora_hasta,
			activo
		FROM locales_horarios
		WHERE %s
		ORDER BY dia_semana, hora_desde, id
	`, strings.Join(conditions, " AND "))

	var horarios []models.LocalHorarioPG
	if err := r.db.Select(&horarios, query, args...); err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los horarios del local")
	}

	return horarios, nil
}

func (r *LocalesHorariosRepo) CreateHorario(input repository.CrearLocalHorarioInput) (int, error) {
	if err := r.validarLocalActivo(input.LocalID); err != nil {
		return 0, err
	}

	if !horarioLocalValido(input.HoraDesde, input.HoraHasta) {
		return 0, fmt.Errorf("hora_hasta debe ser posterior a hora_desde")
	}

	if err := r.validarSolapamiento(input.LocalID, input.DiaSemana, input.HoraDesde, input.HoraHasta, nil); err != nil {
		return 0, err
	}

	var id int
	err := r.db.QueryRowx(`
		INSERT INTO locales_horarios (local_id, dia_semana, hora_desde, hora_hasta, activo)
		VALUES ($1, $2, $3::time, $4::time, TRUE)
		RETURNING id
	`, input.LocalID, input.DiaSemana, input.HoraDesde, input.HoraHasta).Scan(&id)
	if err != nil {
		if esUniqueHorarioError(err) {
			return 0, fmt.Errorf("ya existe un horario igual para ese local y dia")
		}
		return 0, fmt.Errorf("no se pudo crear el horario")
	}

	return id, nil
}

func (r *LocalesHorariosRepo) UpdateHorario(input repository.ActualizarLocalHorarioInput) error {
	var actual struct {
		LocalID   int    `db:"local_id"`
		DiaSemana int    `db:"dia_semana"`
		HoraDesde string `db:"hora_desde"`
		HoraHasta string `db:"hora_hasta"`
	}

	err := r.db.Get(&actual, `
		SELECT
			local_id,
			dia_semana,
			TO_CHAR(hora_desde, 'HH24:MI') AS hora_desde,
			TO_CHAR(hora_hasta, 'HH24:MI') AS hora_hasta
		FROM locales_horarios
		WHERE id = $1 AND activo = TRUE
	`, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("horario no encontrado")
		}
		return fmt.Errorf("no se pudo actualizar el horario")
	}

	diaSemanaFinal := actual.DiaSemana
	if input.DiaSemana != nil {
		diaSemanaFinal = *input.DiaSemana
	}

	horaDesdeFinal := actual.HoraDesde
	if input.HoraDesde != nil {
		horaDesdeFinal = *input.HoraDesde
	}

	horaHastaFinal := actual.HoraHasta
	if input.HoraHasta != nil {
		horaHastaFinal = *input.HoraHasta
	}

	if !horarioLocalValido(horaDesdeFinal, horaHastaFinal) {
		return fmt.Errorf("hora_hasta debe ser posterior a hora_desde")
	}

	if err := r.validarSolapamiento(actual.LocalID, diaSemanaFinal, horaDesdeFinal, horaHastaFinal, &input.ID); err != nil {
		return err
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if input.DiaSemana != nil {
		sets = append(sets, fmt.Sprintf("dia_semana = $%d", idx))
		args = append(args, *input.DiaSemana)
		idx++
	}
	if input.HoraDesde != nil {
		sets = append(sets, fmt.Sprintf("hora_desde = $%d::time", idx))
		args = append(args, *input.HoraDesde)
		idx++
	}
	if input.HoraHasta != nil {
		sets = append(sets, fmt.Sprintf("hora_hasta = $%d::time", idx))
		args = append(args, *input.HoraHasta)
		idx++
	}

	if len(sets) == 0 {
		return fmt.Errorf("debe enviar al menos un campo para actualizar")
	}

	args = append(args, input.ID)
	query := fmt.Sprintf(
		"UPDATE locales_horarios SET %s WHERE id = $%d AND activo = TRUE",
		strings.Join(sets, ", "), idx,
	)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		if esUniqueHorarioError(err) {
			return fmt.Errorf("ya existe un horario igual para ese local y dia")
		}
		return fmt.Errorf("no se pudo actualizar el horario")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el horario")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("horario no encontrado")
	}

	return nil
}

func (r *LocalesHorariosRepo) DeleteHorario(id int) error {
	res, err := r.db.Exec(
		`UPDATE locales_horarios SET activo = FALSE WHERE id = $1 AND activo = TRUE`,
		id,
	)
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el horario")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("no se pudo eliminar el horario")
	}
	if rowsAffected == 0 {
		return fmt.Errorf("horario no encontrado")
	}

	return nil
}

func (r *LocalesHorariosRepo) validarLocalActivo(localID int) error {
	var existe bool

	err := r.db.QueryRowx(
		`SELECT EXISTS(SELECT 1 FROM locales WHERE id = $1 AND activo = TRUE)`,
		localID,
	).Scan(&existe)
	if err != nil {
		return fmt.Errorf("no se pudo validar el local")
	}
	if !existe {
		return fmt.Errorf("local no encontrado")
	}

	return nil
}

func (r *LocalesHorariosRepo) validarSolapamiento(localID, diaSemana int, horaDesde, horaHasta string, excludeID *int) error {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM locales_horarios
			WHERE local_id = $1
			  AND dia_semana = $2
			  AND activo = TRUE
			  AND hora_desde < $4::time
			  AND hora_hasta > $3::time
	`
	args := []interface{}{localID, diaSemana, horaDesde, horaHasta}

	if excludeID != nil {
		query += ` AND id <> $5`
		args = append(args, *excludeID)
	}

	query += `)`

	var existe bool
	if err := r.db.QueryRowx(query, args...).Scan(&existe); err != nil {
		return fmt.Errorf("no se pudo validar el horario")
	}
	if existe {
		return fmt.Errorf("ya hay un horario que cubre esas horas para el mismo local el mismo dia")
	}

	return nil
}

func esUniqueHorarioError(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func horarioLocalValido(horaDesde, horaHasta string) bool {
	return horaDesde < horaHasta
}
