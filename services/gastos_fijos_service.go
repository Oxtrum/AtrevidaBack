package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"atrevida-agenda-api/models"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

type GastosFijosService struct {
	repo   *pgsqlrepo.FinanzasRepo
	config *ConfiguracionService
	fin    *FinanzasService
}

func NewGastosFijosService(repo *pgsqlrepo.FinanzasRepo, config *ConfiguracionService, fin *FinanzasService) *GastosFijosService {
	return &GastosFijosService{repo: repo, config: config, fin: fin}
}

func primerDiaMes(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func ultimoDiaMes(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// fechaCobro es el dia de cobro dentro del mes del periodo; si el mes no tiene ese dia (31 en febrero)
// se usa el ultimo dia del mes (RN-31).
func fechaCobro(periodo time.Time, diaCobro int) time.Time {
	dia := diaCobro
	if ult := ultimoDiaMes(periodo); dia > ult {
		dia = ult
	}
	return time.Date(periodo.Year(), periodo.Month(), dia, 0, 0, 0, 0, time.UTC)
}

func parseFecha(valor string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(valor))
}

// proximoCobro devuelve la primera fecha de cobro >= hoy dentro de la vigencia, o nil.
func proximoCobro(g models.GastoFijo, hoy time.Time) *time.Time {
	if !g.Activo {
		return nil
	}
	inicio, err := parseFecha(g.FechaInicio)
	if err != nil {
		return nil
	}
	var fin *time.Time
	if g.FechaFin != nil {
		if f, err := parseFecha(*g.FechaFin); err == nil {
			fin = &f
		}
	}
	for i := 0; i < 24; i++ {
		periodo := primerDiaMes(hoy).AddDate(0, i, 0)
		cobro := fechaCobro(periodo, g.DiaCobro)
		if cobro.Before(hoy) || cobro.Before(inicio) {
			continue
		}
		if fin != nil && cobro.After(*fin) {
			return nil
		}
		return &cobro
	}
	return nil
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

func (s *GastosFijosService) Listar() ([]models.GastoFijo, error) {
	gastos, err := s.repo.ListGastosFijos(false)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	for i := range gastos {
		if p := proximoCobro(gastos[i], hoy); p != nil {
			v := p.Format("2006-01-02")
			gastos[i].ProximoCobro = &v
		}
	}
	return gastos, nil
}

type GastoFijoInput struct {
	Nombre      string
	CategoriaID int
	Monto       float64
	DiaCobro    int
	LocalID     *int
	MetodoPago  *string
	Proveedor   *string
	FechaInicio string
	FechaFin    *string
	Activo      *bool
	Notas       *string
}

func (s *GastosFijosService) validar(in GastoFijoInput) (pgsqlrepo.GastoFijoInput, error) {
	out := pgsqlrepo.GastoFijoInput{}
	nombre := strings.TrimSpace(in.Nombre)
	if nombre == "" || len(nombre) > 200 {
		return out, errors.New("nombre es requerido y no puede superar 200 caracteres")
	}
	if in.Monto <= 0 {
		return out, errors.New("monto debe ser mayor a 0")
	}
	if in.DiaCobro < 1 || in.DiaCobro > 31 {
		return out, errors.New("dia_cobro debe estar entre 1 y 31")
	}
	inicio, err := parseFecha(in.FechaInicio)
	if err != nil {
		return out, errors.New("fecha_inicio invalida, use formato YYYY-MM-DD")
	}
	var fin *time.Time
	if in.FechaFin != nil && strings.TrimSpace(*in.FechaFin) != "" {
		f, err := parseFecha(*in.FechaFin)
		if err != nil {
			return out, errors.New("fecha_fin invalida, use formato YYYY-MM-DD")
		}
		if f.Before(inicio) {
			return out, errors.New("fecha_fin no puede ser anterior a fecha_inicio")
		}
		fin = &f
	}
	metodo, err := validarMetodoPago(in.MetodoPago)
	if err != nil {
		return out, err
	}
	if err := s.fin.categoriaCompatible(in.CategoriaID, "EGRESO"); err != nil {
		return out, err
	}
	activo := true
	if in.Activo != nil {
		activo = *in.Activo
	}
	return pgsqlrepo.GastoFijoInput{
		Nombre: nombre, CategoriaID: in.CategoriaID, Monto: redondearMonto(in.Monto), DiaCobro: in.DiaCobro,
		LocalID: in.LocalID, MetodoPago: metodo, Proveedor: trimPtr(in.Proveedor), FechaInicio: inicio,
		FechaFin: fin, Activo: activo, Notas: trimPtr(in.Notas),
	}, nil
}

func (s *GastosFijosService) Crear(in GastoFijoInput) (*models.GastoFijo, error) {
	val, err := s.validar(in)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.CreateGastoFijo(val)
	if err != nil {
		return nil, err
	}
	return s.repo.GetGastoFijo(id)
}

// GastoFijoPatch: los campos nullable usan *Set para distinguir "no vino" de "vino null".
type GastoFijoPatch struct {
	Nombre       *string
	CategoriaID  *int
	Monto        *float64
	DiaCobro     *int
	LocalID      *int
	LocalSet     bool
	MetodoPago   *string
	MetodoSet    bool
	Proveedor    *string
	ProveedorSet bool
	FechaInicio  *string
	FechaFin     *string
	FechaFinSet  bool
	Activo       *bool
	Notas        *string
	NotasSet     bool
}

// Actualizar edita la plantilla. Solo afecta periodos futuros aun no generados (RN-34).
func (s *GastosFijosService) Actualizar(id int, p GastoFijoPatch) error {
	actual, err := s.repo.GetGastoFijo(id)
	if err != nil {
		return err
	}
	in := GastoFijoInput{
		Nombre: actual.Nombre, CategoriaID: actual.CategoriaID, Monto: actual.Monto, DiaCobro: actual.DiaCobro,
		LocalID: actual.LocalID, MetodoPago: actual.MetodoPago, Proveedor: actual.Proveedor,
		FechaInicio: actual.FechaInicio, FechaFin: actual.FechaFin, Activo: &actual.Activo, Notas: actual.Notas,
	}
	if p.Nombre != nil {
		in.Nombre = *p.Nombre
	}
	if p.CategoriaID != nil {
		in.CategoriaID = *p.CategoriaID
	}
	if p.Monto != nil {
		in.Monto = *p.Monto
	}
	if p.DiaCobro != nil {
		in.DiaCobro = *p.DiaCobro
	}
	if p.LocalSet {
		in.LocalID = p.LocalID
	}
	if p.MetodoSet {
		in.MetodoPago = p.MetodoPago
	}
	if p.ProveedorSet {
		in.Proveedor = p.Proveedor
	}
	if p.FechaInicio != nil {
		in.FechaInicio = *p.FechaInicio
	}
	if p.FechaFinSet {
		in.FechaFin = p.FechaFin
	}
	if p.Activo != nil {
		in.Activo = p.Activo
	}
	if p.NotasSet {
		in.Notas = p.Notas
	}
	val, err := s.validar(in)
	if err != nil {
		return err
	}
	return s.repo.UpdateGastoFijo(id, val)
}

// Desactivar da de baja la plantilla sin perder el historial generado.
func (s *GastosFijosService) Desactivar(id int) error {
	return s.repo.DesactivarGastoFijo(id)
}

// ─── Generacion mensual (RN-30..RN-35) ────────────────────────────────────────

// GenerarHasta crea los egresos FIJO que ya vencieron hasta `hoy`, incluyendo meses atrasados
// (catch-up). Es idempotente: el indice unico (gasto_fijo_id, periodo) impide duplicados (RN-32).
func (s *GastosFijosService) GenerarHasta(hoy time.Time) (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	gastos, err := s.repo.ListGastosFijos(true)
	if err != nil {
		return 0, err
	}
	generados := 0
	for _, g := range gastos {
		inicio, err := parseFecha(g.FechaInicio)
		if err != nil {
			continue
		}
		var fin *time.Time
		if g.FechaFin != nil {
			if f, err := parseFecha(*g.FechaFin); err == nil {
				fin = &f
			}
		}
		desde := primerDiaMes(inicio)
		if ultimo, err := s.repo.UltimoPeriodoGenerado(g.ID); err != nil {
			return generados, err
		} else if ultimo != nil {
			if siguiente := primerDiaMes(*ultimo).AddDate(0, 1, 0); siguiente.After(desde) {
				desde = siguiente
			}
		}
		for periodo := desde; !periodo.After(hoy); periodo = periodo.AddDate(0, 1, 0) {
			cobro := fechaCobro(periodo, g.DiaCobro)
			if cobro.Before(inicio) {
				continue // el primer mes puede cobrarse antes de que la plantilla entre en vigencia
			}
			if cobro.After(hoy) || (fin != nil && cobro.After(*fin)) {
				break
			}
			creado, err := s.repo.GenerarMovimientoGastoFijo(g.ID, periodo, cobro, cfg.EstadoGastoFijoGen)
			if err != nil {
				return generados, fmt.Errorf("gasto fijo %q: %w", g.Nombre, err)
			}
			if creado {
				generados++
			}
		}
	}
	return generados, nil
}

// GenerarHoy ejecuta GenerarHasta con la fecha actual en la zona horaria configurada.
func (s *GastosFijosService) GenerarHoy() (int, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return 0, err
	}
	return s.GenerarHasta(cfg.Hoy())
}

// Proximos lista los cobros que vencen dentro de dias_aviso_gasto_fijo (dashboard). Hace catch-up
// antes de calcular para que el aviso sea correcto aunque el job diario haya fallado.
func (s *GastosFijosService) Proximos() ([]models.GastoFijoProximo, error) {
	cfg, err := s.config.Snapshot()
	if err != nil {
		return nil, err
	}
	hoy := cfg.Hoy()
	if _, err := s.GenerarHasta(hoy); err != nil {
		return nil, err
	}
	gastos, err := s.repo.ListGastosFijos(true)
	if err != nil {
		return nil, err
	}
	out := []models.GastoFijoProximo{}
	for _, g := range gastos {
		p := proximoCobro(g, hoy)
		if p == nil {
			continue
		}
		dias := int(p.Sub(hoy).Hours() / 24)
		if dias > cfg.DiasAvisoGastoFijo {
			continue
		}
		out = append(out, models.GastoFijoProximo{
			GastoFijoID: g.ID, Nombre: g.Nombre, Monto: g.Monto, FechaCobro: p.Format("2006-01-02"),
			DiasRestantes: dias, LocalNombre: g.LocalNombre,
		})
	}
	return out, nil
}
