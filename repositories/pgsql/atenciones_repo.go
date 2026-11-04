package pgsql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type AtencionesRepo struct{ db *sqlx.DB }

func NewAtencionesRepo(db *sqlx.DB) *AtencionesRepo { return &AtencionesRepo{db: db} }

type RegistrarAtencionInput struct {
	ReservaID                  int
	LocalID                    *int
	EmpleadoID                 *int
	EmpleadoNombre             string
	FechaAtencion              time.Time
	HoraInicio, HoraFin        *string
	Lugar                      string
	Observaciones              *string
	ConfirmarStockInsuficiente bool
	Actor                      repository.ConsumoActor
	Insumos                    []AtencionInsumoInput
	PoliticaSinStock           string
}

type ActualizarAtencionInput struct {
	AtencionID                 int
	LocalID                    *int
	EmpleadoID                 *int
	EmpleadoNombre             string
	FechaAtencion              time.Time
	HoraInicio, HoraFin        *string
	Lugar                      string
	Observaciones              *string
	ConfirmarStockInsuficiente bool
	Actor                      repository.ConsumoActor
	Insumos                    []AtencionInsumoInput
	PoliticaSinStock           string
}

type AtencionInsumoInput struct {
	ItemID   int
	Cantidad float64
}

func normalizarNombresServicio(nombres []string) []string {
	normalizados := make([]string, 0, len(nombres))
	for _, nombre := range nombres {
		normalizados = append(normalizados, strings.ToLower(strings.TrimSpace(nombre)))
	}
	return normalizados
}

func (r *AtencionesRepo) SugerenciasPersonal(q string, localID *int) ([]models.PersonalSugerido, error) {
	args := []any{"%" + strings.TrimSpace(q) + "%"}
	where := "e.activo = TRUE AND CONCAT_WS(' ', e.nombre, e.apellido) ILIKE $1"
	if localID != nil {
		args = append(args, *localID)
		where += " AND (e.local_id IS NULL OR e.local_id = $2)"
	}
	out := []models.PersonalSugerido{}
	err := r.db.Select(&out, `SELECT e.id, CONCAT_WS(' ', e.nombre, e.apellido) AS nombre, e.cargo, e.local_id, l.nombre AS local_nombre
		FROM empleados e LEFT JOIN locales l ON l.id=e.local_id WHERE `+where+` ORDER BY e.apellido, e.nombre LIMIT 20`, args...)
	return out, err
}

func (r *AtencionesRepo) InsumosSugeridos(reservaID int, localID *int) ([]models.ServicioInsumo, error) {
	var nombres []string
	if err := r.db.Select(&nombres, `SELECT nombre FROM (SELECT servicio_nombre AS nombre FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2) UNION SELECT servicio_confirmado FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2) UNION SELECT servicio_solicitado FROM reservas WHERE id=$1 AND ($2::int IS NULL OR local_id=$2)) s WHERE NULLIF(BTRIM(nombre),'') IS NOT NULL`, reservaID, localID); err != nil {
		return nil, err
	}
	if len(nombres) == 0 {
		return []models.ServicioInsumo{}, nil
	}
	var servicioID int
	if err := r.db.Get(&servicioID, `SELECT id FROM servicios WHERE activo=TRUE AND LOWER(BTRIM(nombre))=ANY($1) ORDER BY id LIMIT 1`, pq.Array(normalizarNombresServicio(nombres))); err != nil {
		return []models.ServicioInsumo{}, nil
	}
	out := []models.ServicioInsumo{}
	err := r.db.Select(&out, `SELECT si.item_id,i.nombre AS item_nombre,i.unidad_medida,si.cantidad_por_sesion,COALESCE(st.cantidad,0) AS disponible FROM servicio_insumos si JOIN inventario_items i ON i.id=si.item_id JOIN reservas r ON r.id=$2 AND ($3::int IS NULL OR r.local_id=$3) LEFT JOIN inventario_stock st ON st.item_id=i.id AND st.local_id=r.local_id WHERE si.servicio_id=$1 ORDER BY i.nombre`, servicioID, reservaID, localID)
	return out, err
}

func (r *AtencionesRepo) Registrar(in RegistrarAtencionInput) (*models.Atencion, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var reserva struct {
		LocalID int    `db:"local_id"`
		Estado  string `db:"estado"`
	}
	err = tx.Get(&reserva, `SELECT local_id, COALESCE(estado,'PENDIENTE') estado FROM reservas WHERE id=$1 AND activo=TRUE FOR UPDATE`, in.ReservaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("reserva no encontrada")
	}
	if err != nil {
		return nil, err
	}
	if in.LocalID != nil && reserva.LocalID != *in.LocalID {
		return nil, errors.New("no tienes acceso a ese local")
	}
	if strings.ToUpper(strings.TrimSpace(reserva.Estado)) != "AGENDADO" {
		return nil, errors.New("solo se puede atender una reserva AGENDADA")
	}
	if in.EmpleadoID != nil {
		var n int
		err = tx.Get(&n, `SELECT COUNT(*) FROM empleados WHERE id=$1 AND activo=TRUE AND (local_id IS NULL OR local_id=$2)`, *in.EmpleadoID, reserva.LocalID)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("empleado no disponible para este local")
		}
	}
	var id int
	err = tx.QueryRowx(`INSERT INTO atenciones (reserva_id,empleado_id,empleado_nombre,fecha_atencion,hora_inicio,hora_fin,local_id,lugar,observaciones,registrado_por_id,registrado_por)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, in.ReservaID, in.EmpleadoID, in.EmpleadoNombre, in.FechaAtencion, in.HoraInicio, in.HoraFin, reserva.LocalID, in.Lugar, in.Observaciones, in.Actor.UsuarioID, nullStr(in.Actor.UsuarioNombre)).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("no se pudo registrar la atencion: %w", err)
	}
	type stockPendiente struct {
		itemID     int
		requerido  float64
		pendiente  float64
		disponible float64
	}
	pendientes := make([]stockPendiente, 0)
	stockPrevio := make([]int, 0)
	tipoReversion := repository.MovReversionServicio
	reservaID := in.ReservaID
	atencionID := id
	solicitados := make(map[int]bool, len(in.Insumos))
	for _, line := range in.Insumos {
		solicitados[line.ItemID] = true
	}
	var consumosPrevios []struct {
		ItemID   int     `db:"item_id"`
		Cantidad float64 `db:"cantidad"`
	}
	if err = tx.Select(&consumosPrevios, `SELECT item_id, SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END) AS cantidad FROM inventario_movimientos WHERE reserva_id=$1 AND tipo IN ('SALIDA_SERVICIO','REVERSION_SERVICIO') GROUP BY item_id HAVING SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END) > 0`, in.ReservaID); err != nil {
		return nil, fmt.Errorf("no se pudo revisar los descuentos previos: %w", err)
	}
	tipoRevertido := make([]struct {
		itemID   int
		cantidad float64
	}, 0)
	for _, consumo := range consumosPrevios {
		if solicitados[consumo.ItemID] {
			continue
		}
		if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: consumo.ItemID, LocalID: reserva.LocalID, Tipo: tipoReversion, Cantidad: consumo.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true, Motivo: fmt.Sprintf("Insumo no utilizado en atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
			return nil, fmt.Errorf("fallo_ajuste_inventario: %w", err)
		}
		tipoRevertido = append(tipoRevertido, struct {
			itemID   int
			cantidad float64
		}{itemID: consumo.ItemID, cantidad: consumo.Cantidad})
	}
	for _, line := range in.Insumos {
		var item struct {
			Nombre string `db:"nombre"`
			Unidad string `db:"unidad_medida"`
		}
		err = tx.Get(&item, `SELECT nombre, unidad_medida FROM inventario_items WHERE id=$1 AND activo=TRUE`, line.ItemID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("item de inventario no encontrado")
		}
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(`INSERT INTO inventario_stock(item_id,local_id,cantidad) VALUES($1,$2,0) ON CONFLICT(item_id,local_id) DO NOTHING`, line.ItemID, reserva.LocalID); err != nil {
			return nil, fmt.Errorf("no se pudo preparar el stock para la atencion: %w", err)
		}
		var disponible float64
		if err = tx.Get(&disponible, `SELECT cantidad FROM inventario_stock WHERE item_id=$1 AND local_id=$2 FOR UPDATE`, line.ItemID, reserva.LocalID); err != nil {
			return nil, fmt.Errorf("no se pudo consultar el stock para la atencion: %w", err)
		}
		var descontadoPrevio float64
		if err = tx.Get(&descontadoPrevio, `SELECT COALESCE(SUM(CASE WHEN tipo='SALIDA_SERVICIO' THEN cantidad ELSE -cantidad END),0) FROM inventario_movimientos WHERE reserva_id=$1 AND item_id=$2 AND tipo IN ('SALIDA_SERVICIO','REVERSION_SERVICIO')`, in.ReservaID, line.ItemID); err != nil {
			return nil, fmt.Errorf("no se pudo verificar el consumo previo del insumo: %w", err)
		}
		if descontadoPrevio < 0 {
			descontadoPrevio = 0
		}
		if descontadoPrevio > line.Cantidad {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: line.ItemID, LocalID: reserva.LocalID, Tipo: tipoReversion, Cantidad: descontadoPrevio - line.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true, Motivo: fmt.Sprintf("Ajuste de atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
				return nil, fmt.Errorf("fallo_ajuste_inventario: %w", err)
			}
			descontadoPrevio = line.Cantidad
		}
		faltante := line.Cantidad - descontadoPrevio
		if disponible < faltante {
			if strings.EqualFold(strings.TrimSpace(in.PoliticaSinStock), "BLOQUEAR") && !in.ConfirmarStockInsuficiente {
				return nil, fmt.Errorf("%w: stock disponible %v, se requieren %v adicionales del insumo %s", repository.ErrStockInsuficiente, disponible, faltante, item.Nombre)
			}
			if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,$2,$3,$4,$5,$6,'PENDIENTE_STOCK')`, id, line.ItemID, item.Nombre, item.Unidad, line.Cantidad, descontadoPrevio); err != nil {
				return nil, fmt.Errorf("no se pudo registrar el insumo pendiente: %w", err)
			}
			pendientes = append(pendientes, stockPendiente{itemID: line.ItemID, requerido: line.Cantidad, pendiente: faltante, disponible: disponible})
			continue
		}
		if faltante > 0 {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{ItemID: line.ItemID, LocalID: reserva.LocalID, Tipo: repository.MovSalidaServicio, Cantidad: faltante, ReservaID: &reservaID, AtencionID: &atencionID, UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: false, Motivo: fmt.Sprintf("Atencion #%d, reserva #%d", id, in.ReservaID)}); err != nil {
				return nil, fmt.Errorf("fallo_descuento_inventario: %w", err)
			}
		} else {
			stockPrevio = append(stockPrevio, line.ItemID)
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada) VALUES($1,$2,$3,$4,$5,$5)`, id, line.ItemID, item.Nombre, item.Unidad, line.Cantidad); err != nil {
			return nil, err
		}
	}
	if err = MarcarSesionPorReservaTx(tx, in.ReservaID); err != nil {
		return nil, fmt.Errorf("fallo_consumo_sesion: %w", err)
	}
	if _, err = tx.Exec(`UPDATE reservas SET estado='COMPLETADO', actualizado_en=NOW() WHERE id=$1`, in.ReservaID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	for _, pendiente := range pendientes {
		log.Printf("[atenciones] event=stock_pendiente reserva_id=%d atencion_id=%d local_id=%d item_id=%d requerido=%v pendiente=%v disponible=%v", in.ReservaID, id, reserva.LocalID, pendiente.itemID, pendiente.requerido, pendiente.pendiente, pendiente.disponible)
	}
	for _, itemID := range stockPrevio {
		log.Printf("[atenciones] event=stock_previamente_descontado reserva_id=%d atencion_id=%d local_id=%d item_id=%d", in.ReservaID, id, reserva.LocalID, itemID)
	}
	for _, item := range tipoRevertido {
		log.Printf("[atenciones] event=stock_revertido_no_usado reserva_id=%d atencion_id=%d local_id=%d item_id=%d cantidad=%v", in.ReservaID, id, reserva.LocalID, item.itemID, item.cantidad)
	}
	return r.Obtener(id)
}

func (r *AtencionesRepo) Obtener(id int) (*models.Atencion, error) {
	var a models.Atencion
	err := r.db.Get(&a, `SELECT id,reserva_id,empleado_id,empleado_nombre,fecha_atencion,hora_inicio::text hora_inicio,hora_fin::text hora_fin,local_id,lugar,observaciones,registrado_por_id,registrado_por,creado_en,activo FROM atenciones WHERE id=$1 AND activo=TRUE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("atencion no encontrada")
	}
	if err != nil {
		return nil, err
	}
	err = r.db.Select(&a.Insumos, `SELECT id,atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento FROM atencion_insumos WHERE atencion_id=$1 ORDER BY id`, id)
	return &a, err
}

func (r *AtencionesRepo) Listar(localID *int, limit, offset int) ([]models.Atencion, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	out := []models.Atencion{}
	args := []any{}
	where := "a.activo=TRUE"
	if localID != nil {
		args = append(args, *localID)
		where += " AND a.local_id=$1"
	}
	args = append(args, limit)
	args = append(args, offset)
	err := r.db.Select(&out, `SELECT a.id,a.reserva_id,COALESCE(NULLIF(r.servicio_confirmado,''),NULLIF(r.servicio_solicitado,''),NULLIF(r.servicio_nombre,''),'') AS servicio_reserva,a.empleado_id,a.empleado_nombre,a.fecha_atencion,TO_CHAR(a.hora_inicio,'HH24:MI') hora_inicio,TO_CHAR(a.hora_fin,'HH24:MI') hora_fin,a.local_id,a.lugar,a.observaciones,a.registrado_por_id,a.registrado_por,a.creado_en,a.activo,(SELECT COUNT(*) FROM atencion_insumos ai WHERE ai.atencion_id=a.id AND ai.estado_descuento='PENDIENTE_STOCK') AS insumos_pendientes,(SELECT string_agg(ai.item_nombre, ', ' ORDER BY ai.item_nombre) FROM atencion_insumos ai WHERE ai.atencion_id=a.id AND ai.estado_descuento='PENDIENTE_STOCK') AS insumos_pendientes_detalle FROM atenciones a JOIN reservas r ON r.id=a.reserva_id WHERE `+where+` ORDER BY a.fecha_atencion DESC,a.id DESC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	return out, err
}

func (r *AtencionesRepo) Contar(localID *int) (int, error) {
	var total int
	err := r.db.Get(&total, `SELECT COUNT(*) FROM atenciones WHERE activo=TRUE AND ($1::int IS NULL OR local_id=$1)`, localID)
	return total, err
}

func (r *AtencionesRepo) Desactivar(id int, localID *int, restituirStock bool, actor repository.ConsumoActor) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var atencion struct {
		ReservaID int `db:"reserva_id"`
		LocalID   int `db:"local_id"`
	}
	if err = tx.Get(&atencion, `SELECT reserva_id,local_id FROM atenciones WHERE id=$1 AND activo=TRUE FOR UPDATE`, id); errors.Is(err, sql.ErrNoRows) {
		return errors.New("atencion no encontrada")
	} else if err != nil {
		return err
	}
	if localID != nil && atencion.LocalID != *localID {
		return errors.New("no tienes acceso a ese local")
	}
	if restituirStock {
		var lineas []struct {
			ItemID   int     `db:"item_id"`
			Cantidad float64 `db:"cantidad_descontada"`
		}
		if err = tx.Select(&lineas, `SELECT item_id,cantidad_descontada FROM atencion_insumos WHERE atencion_id=$1 AND item_id IS NOT NULL AND cantidad_descontada>0`, id); err != nil {
			return fmt.Errorf("no se pudieron leer los descuentos de la atencion: %w", err)
		}
		reservaID, atencionID := atencion.ReservaID, id
		for _, linea := range lineas {
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
				ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
				Cantidad: linea.Cantidad, ReservaID: &reservaID, AtencionID: &atencionID,
				UsuarioID: actor.UsuarioID, UsuarioNombre: actor.UsuarioNombre, PermitirNegativo: true,
				Motivo: fmt.Sprintf("Restitucion al desactivar atencion #%d", id),
			}); err != nil {
				return fmt.Errorf("no se pudo restituir el stock: %w", err)
			}
		}
	}
	if _, err = tx.Exec(`UPDATE atenciones SET activo=FALSE WHERE id=$1`, id); err != nil {
		return fmt.Errorf("no se pudo desactivar la atencion: %w", err)
	}
	return tx.Commit()
}

func (r *AtencionesRepo) Actualizar(in ActualizarAtencionInput) (*models.Atencion, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var atencion struct {
		ReservaID int `db:"reserva_id"`
		LocalID   int `db:"local_id"`
	}
	if err = tx.Get(&atencion, `SELECT reserva_id,local_id FROM atenciones WHERE id=$1 AND activo=TRUE FOR UPDATE`, in.AtencionID); errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("atencion no encontrada")
	} else if err != nil {
		return nil, err
	}
	if in.LocalID != nil && atencion.LocalID != *in.LocalID {
		return nil, errors.New("no tienes acceso a ese local")
	}
	var pendientes int
	if err = tx.Get(&pendientes, `SELECT COUNT(*) FROM atencion_insumos WHERE atencion_id=$1 AND estado_descuento='PENDIENTE_STOCK'`, in.AtencionID); err != nil {
		return nil, err
	}
	if pendientes == 0 {
		return nil, errors.New("solo se pueden editar atenciones con deducciones pendientes")
	}
	if in.EmpleadoID != nil {
		var existe int
		if err = tx.Get(&existe, `SELECT COUNT(*) FROM empleados WHERE id=$1 AND activo=TRUE AND (local_id IS NULL OR local_id=$2)`, *in.EmpleadoID, atencion.LocalID); err != nil {
			return nil, err
		}
		if existe == 0 {
			return nil, errors.New("empleado no disponible para este local")
		}
	}
	type lineaActual struct {
		ItemID             sql.NullInt64 `db:"item_id"`
		ItemNombre         string        `db:"item_nombre"`
		UnidadMedida       string        `db:"unidad_medida"`
		CantidadUsada      float64       `db:"cantidad_usada"`
		CantidadDescontada float64       `db:"cantidad_descontada"`
		EstadoDescuento    string        `db:"estado_descuento"`
	}
	var anteriores []lineaActual
	if err = tx.Select(&anteriores, `SELECT item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento FROM atencion_insumos WHERE atencion_id=$1`, in.AtencionID); err != nil {
		return nil, err
	}
	porItem := make(map[int]float64, len(anteriores))
	for _, anterior := range anteriores {
		if anterior.ItemID.Valid {
			porItem[int(anterior.ItemID.Int64)] = anterior.CantidadDescontada
		}
	}
	solicitados := make(map[int]bool, len(in.Insumos))
	for _, linea := range in.Insumos {
		solicitados[linea.ItemID] = true
	}
	reservaID, atencionID := atencion.ReservaID, in.AtencionID
	for itemID, cantidadDescontada := range porItem {
		if solicitados[itemID] || cantidadDescontada <= 0 {
			continue
		}
		if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
			ItemID: itemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
			Cantidad: cantidadDescontada, ReservaID: &reservaID, AtencionID: &atencionID,
			UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true,
			Motivo: fmt.Sprintf("Insumo quitado al editar atencion #%d", in.AtencionID),
		}); err != nil {
			return nil, fmt.Errorf("no se pudo restituir el insumo quitado: %w", err)
		}
	}
	if _, err = tx.Exec(`UPDATE atenciones SET empleado_id=$1,empleado_nombre=$2,fecha_atencion=$3,hora_inicio=$4,hora_fin=$5,lugar=$6,observaciones=$7 WHERE id=$8`, in.EmpleadoID, in.EmpleadoNombre, in.FechaAtencion, in.HoraInicio, in.HoraFin, in.Lugar, in.Observaciones, in.AtencionID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM atencion_insumos WHERE atencion_id=$1`, in.AtencionID); err != nil {
		return nil, err
	}
	for _, anterior := range anteriores {
		if anterior.ItemID.Valid {
			continue
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,NULL,$2,$3,$4,$5,$6)`, in.AtencionID, anterior.ItemNombre, anterior.UnidadMedida, anterior.CantidadUsada, anterior.CantidadDescontada, anterior.EstadoDescuento); err != nil {
			return nil, err
		}
	}
	for _, linea := range in.Insumos {
		var item struct {
			Nombre string `db:"nombre"`
			Unidad string `db:"unidad_medida"`
		}
		if err = tx.Get(&item, `SELECT nombre,unidad_medida FROM inventario_items WHERE id=$1 AND activo=TRUE`, linea.ItemID); errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("item de inventario no encontrado")
		} else if err != nil {
			return nil, err
		}
		anterior := porItem[linea.ItemID]
		descontado := anterior
		if anterior > linea.Cantidad {
			restituir := anterior - linea.Cantidad
			if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
				ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovReversionServicio,
				Cantidad: restituir, ReservaID: &reservaID, AtencionID: &atencionID,
				UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre, PermitirNegativo: true,
				Motivo: fmt.Sprintf("Ajuste de insumo al editar atencion #%d", in.AtencionID),
			}); err != nil {
				return nil, fmt.Errorf("no se pudo ajustar el stock del insumo: %w", err)
			}
			descontado = linea.Cantidad
		} else if anterior < linea.Cantidad {
			if _, err = tx.Exec(`INSERT INTO inventario_stock(item_id,local_id,cantidad) VALUES($1,$2,0) ON CONFLICT(item_id,local_id) DO NOTHING`, linea.ItemID, atencion.LocalID); err != nil {
				return nil, err
			}
			var disponible float64
			if err = tx.Get(&disponible, `SELECT cantidad FROM inventario_stock WHERE item_id=$1 AND local_id=$2 FOR UPDATE`, linea.ItemID, atencion.LocalID); err != nil {
				return nil, err
			}
			adicional := linea.Cantidad - anterior
			if disponible < adicional {
				if strings.EqualFold(strings.TrimSpace(in.PoliticaSinStock), "BLOQUEAR") && !in.ConfirmarStockInsuficiente {
					return nil, fmt.Errorf("%w: stock disponible %v, se requieren %v adicionales del insumo %s", repository.ErrStockInsuficiente, disponible, adicional, item.Nombre)
				}
			} else {
				if _, err = AplicarMovimientoStock(tx, repository.MovimientoStockInput{
					ItemID: linea.ItemID, LocalID: atencion.LocalID, Tipo: repository.MovSalidaServicio,
					Cantidad: adicional, ReservaID: &reservaID, AtencionID: &atencionID,
					UsuarioID: in.Actor.UsuarioID, UsuarioNombre: in.Actor.UsuarioNombre,
					Motivo: fmt.Sprintf("Ajuste de consumo al editar atencion #%d", in.AtencionID),
				}); err != nil {
					return nil, err
				}
				descontado = linea.Cantidad
			}
		}
		estado := "DESCONTADO"
		if descontado < linea.Cantidad {
			estado = "PENDIENTE_STOCK"
		}
		if _, err = tx.Exec(`INSERT INTO atencion_insumos(atencion_id,item_id,item_nombre,unidad_medida,cantidad_usada,cantidad_descontada,estado_descuento) VALUES($1,$2,$3,$4,$5,$6,$7)`, in.AtencionID, linea.ItemID, item.Nombre, item.Unidad, linea.Cantidad, descontado, estado); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Obtener(in.AtencionID)
}
