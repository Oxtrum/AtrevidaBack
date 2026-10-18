package services

import (
	"fmt"
	"strings"

	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	pgsqlrepo "atrevida-agenda-api/repositories/pgsql"

	"github.com/jmoiron/sqlx"
)

// InventarioConsumo conecta reservas y sesiones de plan con el inventario: evalua
// disponibilidad de insumos, descuenta al completar y revierte al eliminar.
// Todas las operaciones sobre stock corren dentro de la transaccion de la reserva.
type InventarioConsumo struct {
	repo   *pgsqlrepo.InventarioRepo
	config *ConfiguracionService
}

func NewInventarioConsumo(repo *pgsqlrepo.InventarioRepo, config *ConfiguracionService) *InventarioConsumo {
	return &InventarioConsumo{repo: repo, config: config}
}

// DescripcionAdvertencias arma el texto "stock insuficiente: ..." que el handler mapea a 409.
func DescripcionAdvertencias(adv []models.AdvertenciaInventario) string {
	partes := make([]string, 0, len(adv))
	for _, a := range adv {
		partes = append(partes, fmt.Sprintf("%s requiere %v %s y hay %v", a.ItemNombre, a.Requerido, strings.ToLower(a.Unidad), a.Disponible))
	}
	return "stock insuficiente: " + strings.Join(partes, "; ")
}

// EvaluarAntesDeCrear estima la disponibilidad para una reserva aun no guardada.
func (c *InventarioConsumo) EvaluarAntesDeCrear(localID int, nombresServicio []string) ([]models.AdvertenciaInventario, error) {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return nil, err
	}
	return c.repo.EvaluarDisponibilidad(nombresServicio, localID, 0, cfg.ConsiderarComprometido)
}

// ValidarPoliticaAlCrear aplica la politica BLOQUEAR solo a usuarios autenticados: el
// formulario publico siempre registra (RN-13).
func (c *InventarioConsumo) ValidarPoliticaAlCrear(autenticado bool, localID int, nombresServicio []string) error {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return err
	}
	if !autenticado || cfg.PoliticaSinStock != "BLOQUEAR" {
		return nil
	}
	warns, err := c.repo.EvaluarDisponibilidad(nombresServicio, localID, 0, cfg.ConsiderarComprometido)
	if err != nil {
		return err
	}
	if len(warns) > 0 {
		return fmt.Errorf("%s", DescripcionAdvertencias(warns))
	}
	return nil
}

// ValidarPoliticaAlAgendar bloquea el cambio a AGENDADO si la politica es BLOQUEAR y no hay stock.
func (c *InventarioConsumo) ValidarPoliticaAlAgendar(reservaID int) error {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return err
	}
	if cfg.PoliticaSinStock != "BLOQUEAR" {
		return nil
	}
	warns, err := c.repo.AdvertenciasDeReserva(reservaID, cfg.ConsiderarComprometido)
	if err != nil {
		return err
	}
	if len(warns) > 0 {
		return fmt.Errorf("%s", DescripcionAdvertencias(warns))
	}
	return nil
}

// Recalcular actualiza alerta_inventario de la reserva y devuelve las advertencias.
func (c *InventarioConsumo) Recalcular(reservaID int) ([]models.AdvertenciaInventario, error) {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return nil, err
	}
	return c.repo.RecalcularAlerta(reservaID, cfg.ConsiderarComprometido)
}

// DespuesDeCrear recalcula la alerta y, si la politica descuenta al agendar y la
// reserva nace AGENDADA, descuenta de inmediato.
func (c *InventarioConsumo) DespuesDeCrear(reservaID int, estado string, actor repository.ConsumoActor) ([]models.AdvertenciaInventario, error) {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return nil, err
	}
	if estado == "AGENDADO" && cfg.DescontarEnEstado == "AGENDADO" {
		if err := c.consumirEnTx(reservaID, actor, cfg); err != nil {
			return nil, err
		}
	}
	return c.repo.RecalcularAlerta(reservaID, cfg.ConsiderarComprometido)
}

func (c *InventarioConsumo) consumirEnTx(reservaID int, actor repository.ConsumoActor, cfg ConfigTrastienda) error {
	tx, err := c.repo.DB().Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := pgsqlrepo.ConsumirPorReservaTx(tx, reservaID, actor, cfg.PermitirStockNegativo); err != nil {
		return err
	}
	return tx.Commit()
}

// HookCambioEstado devuelve la accion que se ejecuta dentro de la transaccion del cambio de estado
// cuando el nuevo estado es el configurado para descontar (por defecto COMPLETADO). nil = nada que hacer.
func (c *InventarioConsumo) HookCambioEstado(nuevoEstado string, reservaID int, actor repository.ConsumoActor) (func(*sqlx.Tx) error, error) {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return nil, err
	}
	if nuevoEstado != cfg.DescontarEnEstado {
		return nil, nil
	}
	return func(tx *sqlx.Tx) error {
		return pgsqlrepo.ConsumirPorReservaTx(tx, reservaID, actor, cfg.PermitirStockNegativo)
	}, nil
}

// HookEliminar devuelve la accion que revierte el stock al eliminar una reserva con salidas (RN-16).
func (c *InventarioConsumo) HookEliminar(reservaID int, actor repository.ConsumoActor) func(*sqlx.Tx) error {
	return func(tx *sqlx.Tx) error {
		return pgsqlrepo.RevertirPorReservaTx(tx, reservaID, actor)
	}
}

// ConsumirSesionPlan descuenta insumos de una sesion de plan consumida sin reserva ligada (RN-18).
func (c *InventarioConsumo) ConsumirSesionPlan(
	tx *sqlx.Tx, servicioID, localID, planMovimientoID int, actor repository.ConsumoActor,
) error {
	cfg, err := c.config.Snapshot()
	if err != nil {
		return err
	}
	return pgsqlrepo.ConsumirPorPlanMovimientoTx(tx, servicioID, localID, planMovimientoID, actor, cfg.PermitirStockNegativo)
}
