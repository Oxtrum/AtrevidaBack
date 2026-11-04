package services

import (
	"errors"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type AtencionesService struct {
	repo   *pgsqlrepo.AtencionesRepo
	config *ConfiguracionService
}

func NewAtencionesService(repo *pgsqlrepo.AtencionesRepo, config *ConfiguracionService) *AtencionesService {
	return &AtencionesService{repo: repo, config: config}
}

type RegistrarAtencionInput struct {
	ReservaID                     int
	LocalID                       *int
	EmpleadoID                    *int
	EmpleadoNombre, FechaAtencion string
	HoraInicio, HoraFin           *string
	Lugar                         string
	Observaciones                 *string
	ConfirmarStockInsuficiente    bool
	Actor                         repository.ConsumoActor
	Insumos                       []pgsqlrepo.AtencionInsumoInput
}

type ActualizarAtencionInput struct {
	AtencionID                    int
	LocalID                       *int
	EmpleadoID                    *int
	EmpleadoNombre, FechaAtencion string
	HoraInicio, HoraFin           *string
	Lugar                         string
	Observaciones                 *string
	ConfirmarStockInsuficiente    bool
	Actor                         repository.ConsumoActor
	Insumos                       []pgsqlrepo.AtencionInsumoInput
}

func (s *AtencionesService) Registrar(in RegistrarAtencionInput) (*models.Atencion, error) {
	if in.ReservaID <= 0 {
		return nil, errors.New("reserva es requerida")
	}
	in.EmpleadoNombre = strings.TrimSpace(in.EmpleadoNombre)
	in.Lugar = strings.TrimSpace(in.Lugar)
	if in.EmpleadoNombre == "" {
		return nil, errors.New("empleado_nombre es requerido")
	}
	if in.Lugar == "" {
		return nil, errors.New("lugar es requerido")
	}
	fecha, err := time.Parse("2006-01-02", in.FechaAtencion)
	if err != nil {
		return nil, errors.New("fecha_atencion invalida, use YYYY-MM-DD")
	}
	if err = validarHoras(in.HoraInicio, in.HoraFin); err != nil {
		return nil, err
	}
	vistos := map[int]bool{}
	for _, x := range in.Insumos {
		if x.ItemID <= 0 || x.Cantidad <= 0 {
			return nil, errors.New("insumo invalido")
		}
		if vistos[x.ItemID] {
			return nil, errors.New("un insumo no puede repetirse")
		}
		vistos[x.ItemID] = true
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	return s.repo.Registrar(pgsqlrepo.RegistrarAtencionInput{ReservaID: in.ReservaID, LocalID: in.LocalID, EmpleadoID: in.EmpleadoID, EmpleadoNombre: in.EmpleadoNombre, FechaAtencion: fecha, HoraInicio: limpiarHora(in.HoraInicio), HoraFin: limpiarHora(in.HoraFin), Lugar: in.Lugar, Observaciones: limpiarTexto(in.Observaciones), ConfirmarStockInsuficiente: in.ConfirmarStockInsuficiente, Actor: in.Actor, Insumos: in.Insumos, PoliticaSinStock: cfg.PoliticaSinStock})
}

func (s *AtencionesService) Actualizar(in ActualizarAtencionInput) (*models.Atencion, error) {
	if in.AtencionID <= 0 {
		return nil, errors.New("id de atencion invalido")
	}
	in.EmpleadoNombre = strings.TrimSpace(in.EmpleadoNombre)
	in.Lugar = strings.TrimSpace(in.Lugar)
	if in.EmpleadoNombre == "" || in.Lugar == "" {
		return nil, errors.New("empleado_nombre y lugar son requeridos")
	}
	fecha, err := time.Parse("2006-01-02", in.FechaAtencion)
	if err != nil {
		return nil, errors.New("fecha_atencion invalida, use YYYY-MM-DD")
	}
	if err = validarHoras(in.HoraInicio, in.HoraFin); err != nil {
		return nil, err
	}
	vistos := map[int]bool{}
	for _, x := range in.Insumos {
		if x.ItemID <= 0 || x.Cantidad <= 0 {
			return nil, errors.New("insumo invalido")
		}
		if vistos[x.ItemID] {
			return nil, errors.New("un insumo no puede repetirse")
		}
		vistos[x.ItemID] = true
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	return s.repo.Actualizar(pgsqlrepo.ActualizarAtencionInput{
		AtencionID: in.AtencionID, LocalID: in.LocalID, EmpleadoID: in.EmpleadoID,
		EmpleadoNombre: in.EmpleadoNombre, FechaAtencion: fecha, HoraInicio: limpiarHora(in.HoraInicio),
		HoraFin: limpiarHora(in.HoraFin), Lugar: in.Lugar, Observaciones: limpiarTexto(in.Observaciones),
		ConfirmarStockInsuficiente: in.ConfirmarStockInsuficiente, Actor: in.Actor, Insumos: in.Insumos,
		PoliticaSinStock: cfg.PoliticaSinStock,
	})
}
func (s *AtencionesService) SugerenciasPersonal(q string, localID *int) ([]models.PersonalSugerido, error) {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 2 {
		return []models.PersonalSugerido{}, nil
	}
	return s.repo.SugerenciasPersonal(q, localID)
}
func (s *AtencionesService) InsumosSugeridos(reservaID int, localID *int) ([]models.ServicioInsumo, error) {
	if reservaID <= 0 {
		return nil, errors.New("reserva_id invalido")
	}
	return s.repo.InsumosSugeridos(reservaID, localID)
}
func (s *AtencionesService) Listar(localID *int, limit, offset int) ([]models.Atencion, error) {
	return s.repo.Listar(localID, limit, offset)
}
func (s *AtencionesService) Contar(localID *int) (int, error) {
	return s.repo.Contar(localID)
}
func (s *AtencionesService) Obtener(id int) (*models.Atencion, error) {
	if id <= 0 {
		return nil, errors.New("id invalido")
	}
	return s.repo.Obtener(id)
}
func (s *AtencionesService) Desactivar(id int, localID *int, restituirStock bool, actor repository.ConsumoActor) error {
	if id <= 0 {
		return errors.New("id de atencion invalido")
	}
	return s.repo.Desactivar(id, localID, restituirStock, actor)
}
func limpiarHora(v *string) *string {
	if v == nil {
		return nil
	}
	x := strings.TrimSpace(*v)
	if x == "" {
		return nil
	}
	return &x
}
func limpiarTexto(v *string) *string {
	if v == nil {
		return nil
	}
	x := strings.TrimSpace(*v)
	if x == "" {
		return nil
	}
	return &x
}
func validarHoras(inicio, fin *string) error {
	a, b := limpiarHora(inicio), limpiarHora(fin)
	parse := func(v *string) (time.Time, error) {
		if v == nil {
			return time.Time{}, nil
		}
		return time.Parse("15:04", *v)
	}
	ai, e := parse(a)
	if e != nil {
		return errors.New("hora_inicio invalida, use HH:MM")
	}
	bf, e := parse(b)
	if e != nil {
		return errors.New("hora_fin invalida, use HH:MM")
	}
	if a != nil && b != nil && !bf.After(ai) {
		return errors.New("hora_fin debe ser posterior a hora_inicio")
	}
	return nil
}
