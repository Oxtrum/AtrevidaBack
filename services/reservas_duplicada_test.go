package services

import (
	"errors"
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type reservasDuplicadasRepo struct {
	reservaCostoUpdateRepo
	existentes []models.ReservaPGCompleta
	filtros    []repository.FiltroReservasPG
	creadas    int
}

func (r *reservasDuplicadasRepo) GetReservas(f repository.FiltroReservasPG) ([]models.ReservaPGCompleta, error) {
	r.filtros = append(r.filtros, f)
	return r.existentes, nil
}

func (r *reservasDuplicadasRepo) CreateReserva(input repository.CreateReservaInput) (int, error) {
	r.creadas++
	return 7, nil
}

func reservaExistente(estado, desde, hasta, servicio string) models.ReservaPGCompleta {
	rv := models.ReservaPGCompleta{}
	rv.ID = 3
	rv.LocalNombre = "SAN MARTIN"
	rv.Fecha = time.Date(2099, 1, 5, 0, 0, 0, 0, time.UTC)
	rv.HoraDesde = desde
	rv.HoraHasta = hasta
	rv.Estado = &estado
	rv.ServicioNombre = &servicio
	rv.Activo = true
	return rv
}

func inputReservaDia() CrearReservaPGInput {
	return CrearReservaPGInput{
		Local:       "SAN MARTIN",
		Fecha:       "2099-01-05",
		HoraDesde:   "08:00",
		HoraHasta:   "09:00",
		Tipo:        "M",
		Cliente:     "Diego",
		Telefono:    "70011223",
		Autenticado: true,
	}
}

func TestCrearReservaAvisaSiElClienteYaTieneReservaEseDia(t *testing.T) {
	repo := &reservasDuplicadasRepo{existentes: []models.ReservaPGCompleta{
		reservaExistente("AGENDADO", "10:00:00", "11:00:00", "Limpieza facial"),
	}}
	svc := &ReservasPGService{repo: repo}

	_, err := svc.CrearReservaConAvisos(inputReservaDia())

	var dup *ReservaDuplicadaError
	if !errors.As(err, &dup) {
		t.Fatalf("error = %v, want ReservaDuplicadaError", err)
	}
	if len(dup.Reservas) != 1 || dup.Reservas[0].HoraDesde != "10:00" || dup.Reservas[0].HoraHasta != "11:00" {
		t.Fatalf("reservas del aviso = %+v, want una de 10:00 a 11:00", dup.Reservas)
	}
	if repo.creadas != 0 {
		t.Fatalf("CreateReserva calls = %d, want 0 sin confirmar", repo.creadas)
	}
	f := repo.filtros[0]
	if f.NumeroTelefono == "" || f.Fecha == nil || !f.SoloActivas {
		t.Fatalf("filtro = %+v, want telefono, fecha y solo activas", f)
	}
}

func TestCrearReservaConfirmadaOmiteElAvisoDeDuplicado(t *testing.T) {
	repo := &reservasDuplicadasRepo{existentes: []models.ReservaPGCompleta{
		reservaExistente("PENDIENTE", "10:00:00", "11:00:00", "Limpieza facial"),
	}}
	svc := &ReservasPGService{repo: repo}
	in := inputReservaDia()
	in.ConfirmarDuplicado = true

	res, err := svc.CrearReservaConAvisos(in)

	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if res.ID != 7 || repo.creadas != 1 {
		t.Fatalf("id = %d, creadas = %d, want 7 y 1", res.ID, repo.creadas)
	}
}

func TestCrearReservaIgnoraReservasRechazadas(t *testing.T) {
	repo := &reservasDuplicadasRepo{existentes: []models.ReservaPGCompleta{
		reservaExistente("RECHAZADO", "10:00:00", "11:00:00", "Limpieza facial"),
	}}
	svc := &ReservasPGService{repo: repo}

	if _, err := svc.CrearReservaConAvisos(inputReservaDia()); err != nil {
		t.Fatalf("error = %v, want nil: una reserva rechazada no cuenta", err)
	}
	if repo.creadas != 1 {
		t.Fatalf("CreateReserva calls = %d, want 1", repo.creadas)
	}
}

func TestCrearReservaPublicaNoConsultaDuplicados(t *testing.T) {
	repo := &reservasDuplicadasRepo{existentes: []models.ReservaPGCompleta{
		reservaExistente("AGENDADO", "10:00:00", "11:00:00", "Limpieza facial"),
	}}
	svc := &ReservasPGService{repo: repo}
	in := inputReservaDia()
	in.Autenticado = false

	if _, err := svc.CrearReservaConAvisos(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if len(repo.filtros) != 0 {
		t.Fatalf("GetReservas calls = %d, want 0: el formulario publico no debe filtrar datos de otros", len(repo.filtros))
	}
}

func reservaAReprogramar() *models.ReservaPGCompleta {
	estado, tel := "AGENDADO", "70011223"
	cur := &models.ReservaPGCompleta{}
	cur.ID = 1
	cur.Estado = &estado
	cur.NumeroTelefono = &tel
	cur.LocalNombre = "SAN MARTIN"
	cur.Fecha = time.Date(2099, 1, 5, 0, 0, 0, 0, time.UTC)
	cur.HoraDesde = "10:00"
	cur.HoraHasta = "11:00"
	return cur
}

func reprogramarA(fecha string) ActualizarReservaPGInput {
	return ActualizarReservaPGInput{Id: 1, Local: "SAN MARTIN", NuevaFecha: fecha}
}

func repoReprogramacion(existentes ...models.ReservaPGCompleta) *reservasDuplicadasRepo {
	repo := &reservasDuplicadasRepo{existentes: existentes}
	repo.reserva = reservaAReprogramar()
	return repo
}

func TestReprogramarAvisaSiElDiaDestinoYaTieneOtraReservaDelCliente(t *testing.T) {
	otra := reservaExistente("AGENDADO", "13:00:00", "14:00:00", "Limpieza facial")
	otra.ID = 9
	repo := repoReprogramacion(otra)
	svc := &ReservasPGService{repo: repo}

	err := svc.ActualizarReserva(reprogramarA("2099-01-06"))

	var dup *ReservaDuplicadaError
	if !errors.As(err, &dup) || len(dup.Reservas) != 1 || dup.Reservas[0].ID != 9 {
		t.Fatalf("error = %v, want ReservaDuplicadaError con la reserva 9", err)
	}
	if repo.update.Id != 0 {
		t.Fatal("UpdateReserva no debe ejecutarse sin confirmar")
	}
	f := repo.filtros[0]
	if f.LocalNombre != "" || f.LocalID != nil {
		t.Fatalf("filtro = %+v, want sin local: una reserva en otro local tambien cuenta", f)
	}
}

func TestReprogramarNoCuentaLaPropiaReserva(t *testing.T) {
	propia := reservaExistente("AGENDADO", "10:00:00", "11:00:00", "Limpieza facial")
	propia.ID = 1
	repo := repoReprogramacion(propia)
	svc := &ReservasPGService{repo: repo}

	if err := svc.ActualizarReserva(reprogramarA("2099-01-06")); err != nil {
		t.Fatalf("error = %v, want nil: la reserva movida no es un duplicado de si misma", err)
	}
	if repo.update.Id != 1 {
		t.Fatal("UpdateReserva no se ejecuto")
	}
}

func TestReprogramarConfirmadoOmiteElAviso(t *testing.T) {
	otra := reservaExistente("PENDIENTE", "13:00:00", "14:00:00", "Limpieza facial")
	otra.ID = 9
	repo := repoReprogramacion(otra)
	svc := &ReservasPGService{repo: repo}
	in := reprogramarA("2099-01-06")
	in.ConfirmarDuplicado = true

	if err := svc.ActualizarReserva(in); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if repo.update.Id != 1 {
		t.Fatal("UpdateReserva no se ejecuto")
	}
}

func TestCambiarSoloLaHoraNoConsultaDuplicados(t *testing.T) {
	otra := reservaExistente("AGENDADO", "13:00:00", "14:00:00", "Limpieza facial")
	otra.ID = 9
	repo := repoReprogramacion(otra)
	svc := &ReservasPGService{repo: repo}

	err := svc.ActualizarReserva(ActualizarReservaPGInput{Id: 1, Local: "SAN MARTIN", NuevaHoraDesde: "09:00", NuevaHoraHasta: "10:00"})

	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if len(repo.filtros) != 0 {
		t.Fatalf("GetReservas calls = %d, want 0 sin cambio de fecha", len(repo.filtros))
	}
}

func TestCrearReservaAvisaTambienSiLaOtraReservaEsDeOtroLocal(t *testing.T) {
	otra := reservaExistente("AGENDADO", "10:00:00", "11:00:00", "Limpieza facial")
	otra.LocalNombre = "PASEO ARANJUEZ"
	repo := &reservasDuplicadasRepo{existentes: []models.ReservaPGCompleta{otra}}
	svc := &ReservasPGService{repo: repo}

	_, err := svc.CrearReservaConAvisos(inputReservaDia())

	var dup *ReservaDuplicadaError
	if !errors.As(err, &dup) || dup.Reservas[0].Local != "PASEO ARANJUEZ" {
		t.Fatalf("error = %v, want aviso con la reserva de PASEO ARANJUEZ", err)
	}
	if f := repo.filtros[0]; f.LocalNombre != "" || f.LocalID != nil {
		t.Fatalf("filtro = %+v, want sin local", f)
	}
}
