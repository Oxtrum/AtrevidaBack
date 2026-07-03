package services

import (
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
)

type reservasResumenRepo struct {
	calls   []repository.FiltroReservasPG
	reserva *models.ReservaPGCompleta
	localID int
}

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
	if got.Format("2006-01-02") != want {
		t.Fatalf("%s = %s, want %s", label, got.Format("2006-01-02"), want)
	}
}
