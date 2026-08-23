package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"atrevida-agenda-api/services"

	"github.com/gin-gonic/gin"
)

type reservasPaginationRepo struct {
	rows        []models.ReservaPGCompleta
	total       int
	filter      repository.FiltroReservasPG
	countFilter repository.FiltroReservasPG
	calls       int
}

func (r *reservasPaginationRepo) GetReservas(f repository.FiltroReservasPG) ([]models.ReservaPGCompleta, error) {
	r.filter = f
	r.calls++
	return r.rows, nil
}
func (r *reservasPaginationRepo) CountReservas(f repository.FiltroReservasPG) (int, error) {
	r.countFilter = f
	return r.total, nil
}
func (r *reservasPaginationRepo) GetReservasAgendadasNoNotificadas(context.Context, string, int) ([]models.ReservaNotificacionPG, error) {
	return nil, nil
}
func (r *reservasPaginationRepo) GetReservaByID(int) (*models.ReservaPGCompleta, error) {
	return nil, nil
}
func (r *reservasPaginationRepo) GetLocalIDByNombre(string) (int, error) { return 1, nil }
func (r *reservasPaginationRepo) GetCapacidades(string) ([]repository.CapacidadLocal, error) {
	return nil, nil
}
func (r *reservasPaginationRepo) GetResumenPagosReservas(repository.FiltroResumenPagosReservas) (repository.ResumenPagosReservas, error) {
	return repository.ResumenPagosReservas{}, nil
}
func (r *reservasPaginationRepo) CreateReserva(repository.CreateReservaInput) (int, error) {
	return 0, nil
}
func (r *reservasPaginationRepo) UpdateReserva(repository.UpdateReservaInput) error { return nil }
func (r *reservasPaginationRepo) UpdateReservaEstado(repository.UpdateReservaEstadoInput) error {
	return nil
}
func (r *reservasPaginationRepo) UpdateReservaNotificado(int, bool) error { return nil }
func (r *reservasPaginationRepo) UpdateReservasNotificado([]int, bool) (int, error) {
	return 0, nil
}
func (r *reservasPaginationRepo) AnularReserva(int) error { return nil }

func reservaRows(total int) []models.ReservaPGCompleta {
	rows := make([]models.ReservaPGCompleta, 0, total)
	estado := "AGENDADO"
	for i := 1; i <= total; i++ {
		rows = append(rows, models.ReservaPGCompleta{
			ReservaPG: models.ReservaPG{
				ID: i, LocalNombre: "PASEO ARANJUEZ", Fecha: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
				HoraDesde: "16:00:00", HoraHasta: "17:00:00", Estado: &estado, Activo: true,
			},
		})
	}
	return rows
}

func TestGetReservasSimpleAplicaVigenciaAntesDePaginaYTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(51), total: 51}
	handler := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?estado=AGENDADO&vigente_fecha=2026-08-12&vigente_hora=15:30&orden=cronologico&limit=50&include_total=true", nil)

	handler.GetReservasSimplePG(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if repo.filter.VigenteFecha == nil || repo.filter.VigenteHora != "15:30" || !repo.filter.OrdenCronologico || repo.filter.PageLimit != 51 {
		t.Fatalf("filtro listado inesperado: %#v", repo.filter)
	}
	if repo.countFilter.VigenteFecha == nil || repo.countFilter.VigenteHora != "15:30" || repo.countFilter.CursorSet {
		t.Fatalf("filtro conteo inesperado: %#v", repo.countFilter)
	}
	var response struct {
		Data struct {
			Total      int `json:"total"`
			Paginacion struct {
				HasMore        bool `json:"has_more"`
				TotalRegistros int  `json:"total_registros"`
				TotalPaginas   int  `json:"total_paginas"`
			} `json:"paginacion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 50 || !response.Data.Paginacion.HasMore || response.Data.Paginacion.TotalRegistros != 51 || response.Data.Paginacion.TotalPaginas != 2 {
		t.Fatalf("respuesta inesperada: %+v", response.Data)
	}
}

func TestGetReservasSimpleCursorIncluyeBusquedaYOrden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &reservasPaginationRepo{rows: reservaRows(3), total: 3}
	handler := &Container{ReservasPG: services.NewReservasPGService(repo, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?busqueda=Maria&orden=cronologico&limit=2", nil)
	handler.GetReservasSimplePG(c)

	var first struct {
		Data struct {
			Paginacion struct {
				NextCursor *string `json:"next_cursor"`
			} `json:"paginacion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil || first.Data.Paginacion.NextCursor == nil {
		t.Fatalf("cursor ausente: err=%v body=%s", err, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?busqueda=Maria&orden=cronologico&limit=2&cursor="+url.QueryEscape(*first.Data.Paginacion.NextCursor), nil)
	handler.GetReservasSimplePG(c)
	if w.Code != http.StatusOK || !repo.filter.CursorSet || repo.filter.Busqueda != "Maria" || !repo.filter.OrdenCronologico {
		t.Fatalf("status=%d filtro=%#v", w.Code, repo.filter)
	}

	previousCalls := repo.calls
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?busqueda=Otra&orden=cronologico&limit=2&cursor="+url.QueryEscape(*first.Data.Paginacion.NextCursor), nil)
	handler.GetReservasSimplePG(c)
	if w.Code != http.StatusBadRequest || repo.calls != previousCalls {
		t.Fatalf("cursor de otros filtros: status=%d calls=%d", w.Code, repo.calls)
	}
}

func TestGetReservasSimpleRechazaVigenciaIncompleta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Container{ReservasPG: services.NewReservasPGService(&reservasPaginationRepo{}, nil)}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bd/reservas?vigente_fecha=2026-08-12&limit=50", nil)
	handler.GetReservasSimplePG(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
