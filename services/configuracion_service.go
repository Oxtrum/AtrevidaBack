package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"
)

// Claves de configuracion_sistema (PLAN §3.5).
const (
	CfgPoliticaSinStock       = "inventario.politica_sin_stock"
	CfgDescontarEnEstado      = "inventario.descontar_en_estado"
	CfgPermitirStockNegativo  = "inventario.permitir_stock_negativo"
	CfgConsiderarComprometido = "inventario.considerar_comprometido"
	CfgEstadoGastoFijoGen     = "finanzas.estado_gasto_fijo_generado"
	CfgDiasAvisoGastoFijo     = "finanzas.dias_aviso_gasto_fijo"
	CfgZonaHoraria            = "finanzas.zona_horaria"
	CfgMoneda                 = "finanzas.moneda"
	CfgRRHHDiaPagoDefecto     = "rrhh.dia_pago_defecto"
	CfgRRHHAporteLaboralPct   = "rrhh.aporte_laboral_pct"
	CfgRRHHAportePatronalPct  = "rrhh.aporte_patronal_pct"
	CfgRRHHDiasMesFaltas      = "rrhh.dias_mes_faltas"
)

// ConfigTrastienda es la vista tipada de los parametros para uso interno.
type ConfigTrastienda struct {
	PoliticaSinStock       string
	DescontarEnEstado      string
	PermitirStockNegativo  bool
	ConsiderarComprometido bool
	EstadoGastoFijoGen     string
	DiasAvisoGastoFijo     int
	ZonaHoraria            string
	Moneda                 string
	DiaPagoDefecto         int
	AporteLaboralPct       float64
	AportePatronalPct      float64
	DiasMesFaltas          int
}

// Ubicacion devuelve la zona horaria configurada (America/La_Paz por defecto).
func (c ConfigTrastienda) Ubicacion() *time.Location {
	loc, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil {
		return time.FixedZone("America/La_Paz", -4*3600)
	}
	return loc
}

// Hoy devuelve la fecha actual (sin hora) en la zona horaria configurada.
func (c ConfigTrastienda) Hoy() time.Time {
	now := time.Now().In(c.Ubicacion())
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func configPorDefecto() ConfigTrastienda {
	return ConfigTrastienda{
		PoliticaSinStock: "ADVERTIR", DescontarEnEstado: "COMPLETADO", PermitirStockNegativo: true,
		ConsiderarComprometido: true, EstadoGastoFijoGen: "PENDIENTE", DiasAvisoGastoFijo: 3,
		ZonaHoraria: "America/La_Paz", Moneda: "Bs",
		DiaPagoDefecto: 1, AporteLaboralPct: 12.71, AportePatronalPct: 16.71, DiasMesFaltas: 30,
	}
}

type ConfiguracionService struct {
	repo *pgsqlrepo.ConfiguracionRepo
	// AlCambiarDisponibilidad es opcional: se invoca cuando cambia un parametro que altera que
	// reservas quedan en falta de stock (hoy: considerar_comprometido).
	AlCambiarDisponibilidad func()
}

func NewConfiguracionService(repo *pgsqlrepo.ConfiguracionRepo) *ConfiguracionService {
	return &ConfiguracionService{repo: repo}
}

// Snapshot lee los parametros; los ausentes o invalidos caen a su default.
func (s *ConfiguracionService) Snapshot() (ConfigTrastienda, error) {
	cfg := configPorDefecto()
	rows, err := s.repo.GetAll()
	if err != nil {
		return cfg, err
	}
	for _, row := range rows {
		switch row.Clave {
		case CfgPoliticaSinStock:
			cfg.PoliticaSinStock = row.Valor
		case CfgDescontarEnEstado:
			cfg.DescontarEnEstado = row.Valor
		case CfgPermitirStockNegativo:
			cfg.PermitirStockNegativo = row.Valor == "true"
		case CfgConsiderarComprometido:
			cfg.ConsiderarComprometido = row.Valor == "true"
		case CfgEstadoGastoFijoGen:
			cfg.EstadoGastoFijoGen = row.Valor
		case CfgDiasAvisoGastoFijo:
			if n, err := strconv.Atoi(row.Valor); err == nil {
				cfg.DiasAvisoGastoFijo = n
			}
		case CfgZonaHoraria:
			cfg.ZonaHoraria = row.Valor
		case CfgMoneda:
			cfg.Moneda = row.Valor
		case CfgRRHHDiaPagoDefecto:
			if n, err := strconv.Atoi(row.Valor); err == nil && n >= 1 && n <= 31 {
				cfg.DiaPagoDefecto = n
			}
		case CfgRRHHAporteLaboralPct:
			if f, err := strconv.ParseFloat(row.Valor, 64); err == nil && f >= 0 && f <= 100 {
				cfg.AporteLaboralPct = f
			}
		case CfgRRHHAportePatronalPct:
			if f, err := strconv.ParseFloat(row.Valor, 64); err == nil && f >= 0 && f <= 100 {
				cfg.AportePatronalPct = f
			}
		case CfgRRHHDiasMesFaltas:
			if n, err := strconv.Atoi(row.Valor); err == nil && n >= 1 && n <= 31 {
				cfg.DiasMesFaltas = n
			}
		}
	}
	return cfg, nil
}

// Get devuelve el mapa clave -> valor tipado (bool/int/string) para la API.
func (s *ConfiguracionService) Get() (map[string]interface{}, error) {
	cfg, err := s.Snapshot()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		CfgPoliticaSinStock:       cfg.PoliticaSinStock,
		CfgDescontarEnEstado:      cfg.DescontarEnEstado,
		CfgPermitirStockNegativo:  cfg.PermitirStockNegativo,
		CfgConsiderarComprometido: cfg.ConsiderarComprometido,
		CfgEstadoGastoFijoGen:     cfg.EstadoGastoFijoGen,
		CfgDiasAvisoGastoFijo:     cfg.DiasAvisoGastoFijo,
		CfgZonaHoraria:            cfg.ZonaHoraria,
		CfgMoneda:                 cfg.Moneda,
		CfgRRHHDiaPagoDefecto:     cfg.DiaPagoDefecto,
		CfgRRHHAporteLaboralPct:   cfg.AporteLaboralPct,
		CfgRRHHAportePatronalPct:  cfg.AportePatronalPct,
		CfgRRHHDiasMesFaltas:      cfg.DiasMesFaltas,
	}, nil
}

func valorEnum(clave string, v interface{}, permitidos ...string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s debe ser texto", clave)
	}
	s = strings.ToUpper(strings.TrimSpace(s))
	for _, p := range permitidos {
		if s == p {
			return s, nil
		}
	}
	return "", fmt.Errorf("%s invalido, valores permitidos: %s", clave, strings.Join(permitidos, ", "))
}

// Update valida y guarda solo las claves enviadas. Devuelve la configuracion resultante.
func (s *ConfiguracionService) Update(patch map[string]interface{}) (map[string]interface{}, error) {
	if len(patch) == 0 {
		return nil, errors.New("no se enviaron parametros para actualizar")
	}
	valores := map[string]string{}
	for clave, v := range patch {
		switch clave {
		case CfgPoliticaSinStock:
			val, err := valorEnum(clave, v, "ADVERTIR", "BLOQUEAR")
			if err != nil {
				return nil, err
			}
			valores[clave] = val
		case CfgDescontarEnEstado:
			val, err := valorEnum(clave, v, "COMPLETADO", "AGENDADO")
			if err != nil {
				return nil, err
			}
			valores[clave] = val
		case CfgEstadoGastoFijoGen:
			val, err := valorEnum(clave, v, "PENDIENTE", "PAGADO")
			if err != nil {
				return nil, err
			}
			valores[clave] = val
		case CfgPermitirStockNegativo, CfgConsiderarComprometido:
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("%s debe ser true o false", clave)
			}
			valores[clave] = strconv.FormatBool(b)
		case CfgDiasAvisoGastoFijo:
			f, ok := v.(float64) // JSON numbers decode as float64
			if !ok || f != float64(int(f)) || f < 0 || f > 31 {
				return nil, fmt.Errorf("%s debe ser un entero entre 0 y 31", clave)
			}
			valores[clave] = strconv.Itoa(int(f))
		case CfgRRHHDiaPagoDefecto, CfgRRHHDiasMesFaltas:
			f, ok := v.(float64)
			if !ok || f != float64(int(f)) || f < 1 || f > 31 {
				return nil, fmt.Errorf("%s debe ser un entero entre 1 y 31", clave)
			}
			valores[clave] = strconv.Itoa(int(f))
		case CfgRRHHAporteLaboralPct, CfgRRHHAportePatronalPct:
			f, ok := v.(float64)
			if !ok || f < 0 || f > 100 {
				return nil, fmt.Errorf("%s debe ser un porcentaje entre 0 y 100", clave)
			}
			valores[clave] = strconv.FormatFloat(f, 'f', -1, 64)
		case CfgZonaHoraria:
			z, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%s debe ser texto", clave)
			}
			if _, err := time.LoadLocation(strings.TrimSpace(z)); err != nil {
				return nil, fmt.Errorf("%s invalida: %q", clave, z)
			}
			valores[clave] = strings.TrimSpace(z)
		case CfgMoneda:
			m, ok := v.(string)
			m = strings.TrimSpace(m)
			if !ok || m == "" || len(m) > 5 {
				return nil, fmt.Errorf("%s debe ser un texto de 1 a 5 caracteres", clave)
			}
			valores[clave] = m
		default:
			return nil, fmt.Errorf("parametro desconocido: %s", clave)
		}
	}
	if err := s.repo.SetValores(valores); err != nil {
		return nil, err
	}
	if _, cambia := valores[CfgConsiderarComprometido]; cambia && s.AlCambiarDisponibilidad != nil {
		s.AlCambiarDisponibilidad()
	}
	return s.Get()
}
