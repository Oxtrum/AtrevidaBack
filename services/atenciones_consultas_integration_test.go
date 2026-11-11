package services

// Pruebas de integracion para las consultas de Jornada, Historial y Consumos.
// Reutilizan la base descartable configurada para trastienda_integration_test.go:
//
//   TRASTIENDA_TEST_DB=atrevida_migtest go test ./services -run AtencionesConsultas -v

import (
	"testing"
	"time"

	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

func insertarAtencionConsulta(t *testing.T, e *trastiendaEnv, localID int, localNombre string, fecha time.Time, cliente, servicio, personal string, activa bool) (reservaID, atencionID int) {
	t.Helper()
	if err := e.db.Get(&reservaID, `
		INSERT INTO reservas (local_id, local_nombre, tipo_espacio, fecha, hora_desde, hora_hasta, cliente, estado, servicio_nombre, servicio_confirmado)
		VALUES ($1, $2, 'M', $3, '10:00', '11:00', $4, 'COMPLETADO', $5, $5)
		RETURNING id`, localID, localNombre, fecha, cliente, servicio); err != nil {
		t.Fatalf("crear reserva de consulta: %v", err)
	}
	if err := e.db.Get(&atencionID, `
		INSERT INTO atenciones (reserva_id, empleado_nombre, fecha_atencion, hora_inicio, hora_fin, local_id, lugar, activo)
		VALUES ($1, $2, $3, '10:00', '11:00', $4, 'Cabina QA', $5)
		RETURNING id`, reservaID, personal, fecha, localID, activa); err != nil {
		t.Fatalf("crear atencion de consulta: %v", err)
	}
	return reservaID, atencionID
}

func agregarConsumoConsulta(t *testing.T, e *trastiendaEnv, atencionID, itemID int, usada, descontada float64, estado string) {
	t.Helper()
	if _, err := e.db.Exec(`
		INSERT INTO atencion_insumos (atencion_id, item_id, item_nombre, unidad_medida, cantidad_usada, cantidad_descontada, estado_descuento)
		SELECT $1, i.id, i.nombre, i.unidad_medida, $3, $4, $5
		FROM inventario_items i WHERE i.id = $2`, atencionID, itemID, usada, descontada, estado); err != nil {
		t.Fatalf("agregar consumo de consulta: %v", err)
	}
}

func idsConsumos(eventos []int) map[int]bool {
	out := make(map[int]bool, len(eventos))
	for _, id := range eventos {
		out[id] = true
	}
	return out
}

func TestAtencionesConsultasConsumosRespetaLocalFiltrosYActivas(t *testing.T) {
	e := setupTrastienda(t)
	repo := pgsqlrepo.NewInventarioRepo(e.db)
	fecha := time.Date(2031, time.February, 4, 0, 0, 0, 0, time.UTC)
	itemPendiente := e.nuevoItem("Pendiente", 0)
	itemDescontado := e.nuevoItem("Descontado", 0)

	var otroLocalID int
	otroLocalNombre := "LOCAL-CONSULTA-" + e.sufijo
	mustExec(t, e.db, `INSERT INTO locales (nombre) VALUES ($1) RETURNING id`, otroLocalNombre)
	if err := e.db.Get(&otroLocalID, `SELECT id FROM locales WHERE nombre = $1`, otroLocalNombre); err != nil {
		t.Fatal(err)
	}

	_, sinInsumos := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente sin insumos", "Servicio QA", "Ana QA", true)
	_, pendiente := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente pendiente", "Servicio QA", "Ana QA", true)
	_, descontada := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente descontado", "Servicio QA", "Beto QA", true)
	_, desactivada := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente desactivado", "Servicio QA", "Oculta QA", false)
	_, otroLocal := insertarAtencionConsulta(t, e, otroLocalID, otroLocalNombre, fecha, "Cliente otro local", "Servicio QA", "Carla QA", true)
	agregarConsumoConsulta(t, e, pendiente, itemPendiente, 3, 0, "PENDIENTE_STOCK")
	agregarConsumoConsulta(t, e, descontada, itemDescontado, 2, 2, "DESCONTADO")
	agregarConsumoConsulta(t, e, desactivada, itemPendiente, 1, 0, "PENDIENTE_STOCK")
	agregarConsumoConsulta(t, e, otroLocal, itemPendiente, 1, 0, "PENDIENTE_STOCK")

	filtroLocal := repository.FiltroConsumos{LocalID: &e.localID, FechaDesde: fecha, FechaHasta: fecha}
	eventos, err := repo.ListarConsumos(filtroLocal)
	if err != nil {
		t.Fatal(err)
	}
	ids := idsConsumos(func() []int {
		out := make([]int, 0, len(eventos))
		for _, evento := range eventos {
			out = append(out, evento.AtencionID)
		}
		return out
	}())
	if len(eventos) != 3 || !ids[sinInsumos] || !ids[pendiente] || !ids[descontada] || ids[desactivada] || ids[otroLocal] {
		t.Fatalf("eventos por local/activo inesperados: %+v", ids)
	}
	for _, evento := range eventos {
		if evento.AtencionID == sinInsumos && len(evento.Insumos) != 0 {
			t.Fatalf("la atencion sin insumos no debe inventar lineas: %+v", evento.Insumos)
		}
	}

	filtroItem := filtroLocal
	filtroItem.ItemID = &itemPendiente
	eventos, err = repo.ListarConsumos(filtroItem)
	if err != nil || len(eventos) != 1 || eventos[0].AtencionID != pendiente {
		t.Fatalf("filtro por insumo = %+v, %v; se esperaba solo atencion %d", eventos, err, pendiente)
	}

	filtroPendiente := filtroLocal
	filtroPendiente.EstadoDescuento = "PENDIENTE"
	eventos, err = repo.ListarConsumos(filtroPendiente)
	if err != nil || len(eventos) != 1 || eventos[0].AtencionID != pendiente {
		t.Fatalf("filtro pendiente = %+v, %v; se esperaba solo atencion %d", eventos, err, pendiente)
	}
}

func TestAtencionesConsultasAgrupaPersonalSinDuplicarAtenciones(t *testing.T) {
	e := setupTrastienda(t)
	repo := pgsqlrepo.NewInventarioRepo(e.db)
	fecha := time.Date(2031, time.February, 5, 0, 0, 0, 0, time.UTC)
	uno := e.nuevoItem("Uno", 0)
	dos := e.nuevoItem("Dos", 0)

	_, anaConDosLineas := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente A", "Servicio QA", "Ana QA", true)
	_, _ = insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente B", "Servicio QA", "Ana QA", true)
	_, beto := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente C", "Servicio QA", "Beto QA", true)
	agregarConsumoConsulta(t, e, anaConDosLineas, uno, 1, 1, "DESCONTADO")
	agregarConsumoConsulta(t, e, anaConDosLineas, dos, 1, 1, "DESCONTADO")
	agregarConsumoConsulta(t, e, beto, uno, 1, 1, "DESCONTADO")

	resumen, err := repo.ResumenConsumosPorEmpleado(repository.FiltroConsumos{LocalID: &e.localID, FechaDesde: fecha, FechaHasta: fecha})
	if err != nil {
		t.Fatal(err)
	}
	porNombre := make(map[string]struct{ atenciones, lineas int })
	for _, fila := range resumen {
		porNombre[fila.EmpleadoNombre] = struct{ atenciones, lineas int }{fila.Atenciones, fila.InsumosRegistrados}
	}
	if got := porNombre["Ana QA"]; got.atenciones != 2 || got.lineas != 2 {
		t.Fatalf("Ana QA = %+v; se esperaban 2 atenciones y 2 lineas", got)
	}
	if got := porNombre["Beto QA"]; got.atenciones != 1 || got.lineas != 1 {
		t.Fatalf("Beto QA = %+v; se esperaban 1 atencion y 1 linea", got)
	}
}

func TestAtencionesConsultasCalendarioPersonalRespetaAlcanceYActivas(t *testing.T) {
	e := setupTrastienda(t)
	repo := pgsqlrepo.NewAtencionesRepo(e.db)
	fecha := time.Date(2031, time.February, 6, 0, 0, 0, 0, time.UTC)

	_, ana := insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente de Ana", "Masaje QA", "Ana QA", true)
	_, _ = insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente oculto", "Masaje QA", "Oculta QA", false)
	var otroLocalID int
	otroLocalNombre := "LOCAL-PERSONAL-" + e.sufijo
	mustExec(t, e.db, `INSERT INTO locales (nombre) VALUES ($1) RETURNING id`, otroLocalNombre)
	if err := e.db.Get(&otroLocalID, `SELECT id FROM locales WHERE nombre = $1`, otroLocalNombre); err != nil {
		t.Fatal(err)
	}
	_, otro := insertarAtencionConsulta(t, e, otroLocalID, otroLocalNombre, fecha, "Cliente de otro local", "Masaje QA", "Beto QA", true)

	filtro := repository.FiltroCalendarioPersonal{LocalID: &e.localID, FechaDesde: fecha, FechaHasta: fecha}
	eventos, err := repo.CalendarioPersonal(filtro)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 1 || eventos[0].AtencionID != ana || eventos[0].EmpleadoNombre != "Ana QA" {
		t.Fatalf("calendario de personal inesperado: %+v", eventos)
	}
	if total, err := repo.ContarCalendarioPersonal(filtro); err != nil || total != 1 {
		t.Fatalf("total calendario = %d, %v; se esperaba 1", total, err)
	}

	filtro.Empleado = "Beto"
	eventos, err = repo.CalendarioPersonal(filtro)
	if err != nil || len(eventos) != 0 {
		t.Fatalf("el filtro de personal no debe cruzar locales: %+v, %v", eventos, err)
	}
	if otro == ana {
		t.Fatal("las atenciones de prueba deben ser distintas")
	}
}

func TestAtencionesConsultasHistorialIncluyeCancelacionesYFiltraInasistencias(t *testing.T) {
	e := setupTrastienda(t)
	repo := pgsqlrepo.NewAtencionesRepo(e.db)
	fecha := time.Now().UTC().Truncate(24 * time.Hour)
	servicio := e.nuevoServicio("Historial cancelacion", nil)
	cancelada := e.nuevaReserva(servicio, "AGENDADO", 0)
	if _, err := e.reservas.ActualizarEstadoReservaConAvisos(ActualizarEstadoReservaInput{
		Id: cancelada, Estado: "CANCELADO", MotivoCancelacionTipo: "INASISTENCIA",
		Actor: repository.ConsumoActor{UsuarioNombre: "QA historial"},
	}); err != nil {
		t.Fatalf("cancelar reserva: %v", err)
	}
	_, _ = insertarAtencionConsulta(t, e, e.localID, e.localName, fecha, "Cliente atendido", servicio, "Personal QA", true)

	filas, err := repo.Listar(repository.FiltroAtenciones{LocalID: &e.localID, FechaDesde: &fecha, FechaHasta: &fecha, Resultado: "CANCELADA"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filas) != 1 || filas[0].ReservaID != cancelada || filas[0].Resultado != "CANCELADA" || filas[0].MotivoCancelacionTipo == nil || *filas[0].MotivoCancelacionTipo != "INASISTENCIA" || filas[0].ID >= 0 {
		t.Fatalf("cancelacion en historial invalida: %+v", filas)
	}

	filas, err = repo.Listar(repository.FiltroAtenciones{LocalID: &e.localID, FechaDesde: &fecha, FechaHasta: &fecha, Resultado: "INASISTENCIA"})
	if err != nil || len(filas) != 1 || filas[0].ReservaID != cancelada {
		t.Fatalf("filtro de inasistencia invalido: %+v, %v", filas, err)
	}
	filas, err = repo.Listar(repository.FiltroAtenciones{LocalID: &e.localID, FechaDesde: &fecha, FechaHasta: &fecha, Resultado: "ATENDIDA"})
	if err != nil || len(filas) != 1 || filas[0].Resultado != "ATENDIDA" {
		t.Fatalf("filtro de atendida invalido: %+v, %v", filas, err)
	}
}
