package services

import (
	"context"
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

func (r *reservasResumenRepo) GetReservasAgendadasNoNotificadas(context.Context, string, int) ([]models.ReservaNotificacionPG, error) {
	return nil, nil
}

type reservasResumenRepo struct {
	calls        []repository.FiltroReservasPG
	paymentCalls []repository.FiltroResumenPagosReservas
	pagosResumen repository.ResumenPagosReservas
	reserva      *models.ReservaPGCompleta
	localID      int
}

func (r *reservasResumenRepo) CountReservas(repository.FiltroReservasPG) (int, error) { return 0, nil }

func (r *reservasResumenRepo) GetReservas(f repository.FiltroReservasPG) ([]models.ReservaPGCompleta, error) {
	r.calls = append(r.calls, f)
	return nil, nil
}

func (r *reservasResumenRepo) GetReservaByID(id int) (*models.ReservaPGCompleta, error) {
	return r.reserva, nil
}

func (r *reservasResumenRepo) GetLocalIDByNombre(nombre string) (int, error) {
	if r.localID != 0 {
		return r.localID, nil
	}
	return 1, nil
}

func (r *reservasResumenRepo) GetCapacidades(localNombre string) ([]repository.CapacidadLocal, error) {
	return nil, nil
}

func (r *reservasResumenRepo) GetResumenPagosReservas(f repository.FiltroResumenPagosReservas) (repository.ResumenPagosReservas, error) {
	r.paymentCalls = append(r.paymentCalls, f)
	return r.pagosResumen, nil
}

func (r *reservasResumenRepo) CreateReserva(input repository.CreateReservaInput) (int, error) {
	return 0, nil
}

func (r *reservasResumenRepo) UpdateReserva(input repository.UpdateReservaInput) error {
	return nil
}

func (r *reservasResumenRepo) UpdateReservaEstado(input repository.UpdateReservaEstadoInput) error {
	return nil
}

func (r *reservasResumenRepo) UpdateReservaNotificado(id int, notificado bool) error {
	return nil
}

func (r *reservasResumenRepo) UpdateReservasNotificado(ids []int, notificado bool) (int, error) {
	return 0, nil
}

func (r *reservasResumenRepo) AnularReserva(id int) error {
	return nil
}

func TestGetResumenReservasDomingoUsaSabadoAnterior(t *testing.T) {
	repo := &reservasResumenRepo{}
	service := NewReservasPGService(repo, nil)
	fechaDomingo := time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)

	if _, err := service.GetResumenReservas(fechaDomingo, "", nil); err != nil {
		t.Fatalf("GetResumenReservas() error = %v", err)
	}
	if len(repo.calls) != 2 {
		t.Fatalf("GetReservas calls = %d, want 2", len(repo.calls))
	}

	assertDate(t, repo.calls[0].Fecha, "2026-05-23", "fecha del dia")
	assertDate(t, repo.calls[1].FechaDesde, "2026-05-18", "inicio de semana")
	assertDate(t, repo.calls[1].FechaHasta, "2026-05-23", "fin de semana")
	if len(repo.paymentCalls) != 1 {
		t.Fatalf("GetResumenPagosReservas calls = %d, want 1", len(repo.paymentCalls))
	}
	assertDateValue(t, repo.paymentCalls[0].Fecha, "2026-05-23", "fecha de pagos")
	assertDateValue(t, repo.paymentCalls[0].FechaDesde, "2026-05-18", "inicio pagos")
	assertDateValue(t, repo.paymentCalls[0].FechaHasta, "2026-05-23", "fin pagos")
}

func TestGetResumenReservasResuelveLocalNombreAID(t *testing.T) {
	repo := &reservasResumenRepo{localID: 2}
	service := NewReservasPGService(repo, nil)
	fecha := time.Date(2026, time.May, 23, 0, 0, 0, 0, time.UTC)

	if _, err := service.GetResumenReservas(fecha, "PASEO ARANJUEZ", nil); err != nil {
		t.Fatalf("GetResumenReservas() error = %v", err)
	}

	if len(repo.calls) != 2 {
		t.Fatalf("GetReservas calls = %d, want 2", len(repo.calls))
	}
	if repo.calls[0].LocalID == nil || *repo.calls[0].LocalID != 2 {
		t.Fatalf("dia LocalID = %v, want 2", repo.calls[0].LocalID)
	}
	if repo.calls[0].LocalNombre != "" {
		t.Fatalf("dia LocalNombre = %q, want empty", repo.calls[0].LocalNombre)
	}
	if repo.calls[1].LocalID == nil || *repo.calls[1].LocalID != 2 {
		t.Fatalf("semana LocalID = %v, want 2", repo.calls[1].LocalID)
	}
	if repo.calls[1].LocalNombre != "" {
		t.Fatalf("semana LocalNombre = %q, want empty", repo.calls[1].LocalNombre)
	}
	if len(repo.paymentCalls) != 1 {
		t.Fatalf("GetResumenPagosReservas calls = %d, want 1", len(repo.paymentCalls))
	}
	if repo.paymentCalls[0].LocalID == nil || *repo.paymentCalls[0].LocalID != 2 {
		t.Fatalf("pagos LocalID = %v, want 2", repo.paymentCalls[0].LocalID)
	}
	if repo.paymentCalls[0].LocalNombre != "" {
		t.Fatalf("pagos LocalNombre = %q, want empty", repo.paymentCalls[0].LocalNombre)
	}
}

func TestGetResumenReservasAplicaLocalID(t *testing.T) {
	repo := &reservasResumenRepo{}
	service := NewReservasPGService(repo, nil)
	fecha := time.Date(2026, time.May, 23, 0, 0, 0, 0, time.UTC)
	localID := 2

	if _, err := service.GetResumenReservas(fecha, "SAN MARTIN", &localID); err != nil {
		t.Fatalf("GetResumenReservas() error = %v", err)
	}

	if len(repo.calls) != 2 {
		t.Fatalf("GetReservas calls = %d, want 2", len(repo.calls))
	}
	if repo.calls[0].LocalID == nil || *repo.calls[0].LocalID != localID {
		t.Fatalf("dia LocalID = %v, want %d", repo.calls[0].LocalID, localID)
	}
	if repo.calls[1].LocalID == nil || *repo.calls[1].LocalID != localID {
		t.Fatalf("semana LocalID = %v, want %d", repo.calls[1].LocalID, localID)
	}
	if repo.calls[0].LocalNombre != "" {
		t.Fatalf("dia LocalNombre = %q, want empty", repo.calls[0].LocalNombre)
	}
	if repo.calls[1].LocalNombre != "" {
		t.Fatalf("semana LocalNombre = %q, want empty", repo.calls[1].LocalNombre)
	}
	if len(repo.paymentCalls) != 1 {
		t.Fatalf("GetResumenPagosReservas calls = %d, want 1", len(repo.paymentCalls))
	}
	if repo.paymentCalls[0].LocalID == nil || *repo.paymentCalls[0].LocalID != localID {
		t.Fatalf("pagos LocalID = %v, want %d", repo.paymentCalls[0].LocalID, localID)
	}
	if repo.paymentCalls[0].LocalNombre != "" {
		t.Fatalf("pagos LocalNombre = %q, want empty", repo.paymentCalls[0].LocalNombre)
	}
}

func TestGetResumenReservasIncluyeResumenPagos(t *testing.T) {
	repo := &reservasResumenRepo{
		pagosResumen: repository.ResumenPagosReservas{
			IngresosDia:       1250.50,
			IngresosSemana:    8450.75,
			IngresosLunes:     1000,
			IngresosMartes:    1200,
			IngresosMiercoles: 1300,
			IngresosJueves:    1500,
			IngresosViernes:   3450.75,
			CancelacionesDia:  6,
		},
	}
	service := NewReservasPGService(repo, nil)
	fecha := time.Date(2026, time.May, 22, 0, 0, 0, 0, time.UTC)

	resumen, err := service.GetResumenReservas(fecha, "", nil)
	if err != nil {
		t.Fatalf("GetResumenReservas() error = %v", err)
	}

	if resumen.IngresosDia != 1250.50 {
		t.Fatalf("IngresosDia = %v, want 1250.50", resumen.IngresosDia)
	}
	if resumen.IngresosSemana != 8450.75 {
		t.Fatalf("IngresosSemana = %v, want 8450.75", resumen.IngresosSemana)
	}
	if resumen.CancelacionesDia != 6 {
		t.Fatalf("CancelacionesDia = %d, want 6", resumen.CancelacionesDia)
	}
	if resumen.Ingresos.TotalIngresos != 8450.75 {
		t.Fatalf("Ingresos.TotalIngresos = %v, want 8450.75", resumen.Ingresos.TotalIngresos)
	}
	if resumen.Ingresos.Lunes != 1000 {
		t.Fatalf("Ingresos.Lunes = %v, want 1000", resumen.Ingresos.Lunes)
	}
	if resumen.Ingresos.Viernes != 3450.75 {
		t.Fatalf("Ingresos.Viernes = %v, want 3450.75", resumen.Ingresos.Viernes)
	}
}

func TestGetReservaByIDAplicaLocalID(t *testing.T) {
	localID := 1
	service := NewReservasPGService(&reservasResumenRepo{
		reserva: &models.ReservaPGCompleta{
			ReservaPG: models.ReservaPG{
				ID:          44,
				LocalID:     &localID,
				LocalNombre: "SAN MARTIN",
				TipoEspacio: "M",
				Fecha:       time.Date(2026, time.May, 23, 0, 0, 0, 0, time.UTC),
				HoraDesde:   "09:00",
				HoraHasta:   "10:00",
				Cliente:     "Maria Lopez",
			},
		},
	}, nil)

	scopeLocalID := 2
	if _, err := service.GetReservaByID(44, &scopeLocalID); err == nil {
		t.Fatal("GetReservaByID() error = nil, want reserva no encontrada")
	}

	scopeLocalID = localID
	if _, err := service.GetReservaByID(44, &scopeLocalID); err != nil {
		t.Fatalf("GetReservaByID() error = %v, want nil", err)
	}
}

func assertDate(t *testing.T, got *time.Time, want string, label string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %s", label, want)
	}
	assertDateValue(t, *got, want, label)
}

func assertDateValue(t *testing.T, got time.Time, want string, label string) {
	t.Helper()
	if got.Format("2006-01-02") != want {
		t.Fatalf("%s = %s, want %s", label, got.Format("2006-01-02"), want)
	}
}
