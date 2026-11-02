package cuenta

import (
	"fmt"
	"math"
)

// Dominio de la cuenta del cliente (PLAN_CUENTA_CLIENTE_Y_AJUSTES §5.3): cargos (lo que debe), abonos (lo que
// entrega) y aplicaciones (que parte de un abono cubre que cargo). Es logica pura, sin base de datos.
// Todos los calculos van en centavos enteros para no arrastrar errores de decimales.

// Estados derivados de un cargo.
const (
	CargoPendiente = "PENDIENTE"
	CargoParcial   = "PARCIAL"
	CargoPagado    = "PAGADO"
)

func centavos(monto float64) int64 { return int64(math.Round(monto * 100)) }

func desdeCentavos(c int64) float64 { return float64(c) / 100 }

// EstadoCargo deriva el estado de un cargo de lo ya aplicado. Un cargo en 0 no deja nada pendiente.
func EstadoCargo(monto, pagado float64) string {
	m, p := centavos(monto), centavos(pagado)
	switch {
	case p >= m:
		return CargoPagado
	case p <= 0:
		return CargoPendiente
	default:
		return CargoParcial
	}
}

// Saldo es la posicion de un cliente, en dos numeros que no se mezclan: lo que tiene disponible y lo que le falta
// pagar. Nunca se guarda: se calcula (RN-C5).
type Saldo struct {
	// Disponible es el dinero que el cliente recargo y todavia no uso.
	Disponible float64 `json:"disponible" example:"150"`
	// PorPagar es la suma de lo que le falta pagar de sus cargos vigentes.
	PorPagar float64 `json:"por_pagar" example:"250"`
	// Deuda es lo que falta despues de usar el saldo disponible; nunca es negativa.
	Deuda float64 `json:"deuda" example:"100"`
}

// PosicionCliente arma el saldo a partir del dinero disponible y de lo que hay por pagar.
func PosicionCliente(disponible, porPagar float64) Saldo {
	d, p := centavos(disponible), centavos(porPagar)
	deuda := p - d
	if deuda < 0 {
		deuda = 0
	}
	return Saldo{Disponible: desdeCentavos(d), PorPagar: desdeCentavos(p), Deuda: desdeCentavos(deuda)}
}

// Reparto es cuanto de un abono se aplica a un cargo.
type Reparto struct {
	CargoID int     `json:"cargo_id" example:"12"`
	Monto   float64 `json:"monto" example:"400"`
}

// ValidarAplicaciones comprueba un reparto contra las reglas de integridad:
// RN-C1 un abono no aplica mas de su monto disponible (monto menos lo ya aplicado) y
// RN-C2 un cargo no recibe mas de lo que tiene pendiente.
// Aplicar menos que el monto del abono es valido: lo que sobra queda como saldo disponible.
// Los mensajes contienen "no puede" o "no encontrado" porque los handlers mapean a 4xx por texto.
func ValidarAplicaciones(montoAbono, yaAplicado float64, aplicaciones []Reparto, pendientes map[int]float64) error {
	disponible := centavos(montoAbono) - centavos(yaAplicado)
	var total int64
	vistos := make(map[int]bool, len(aplicaciones))
	for _, a := range aplicaciones {
		if centavos(a.Monto) <= 0 {
			return fmt.Errorf("el monto debe ser mayor a 0 (cargo %d)", a.CargoID)
		}
		if vistos[a.CargoID] {
			return fmt.Errorf("el cargo %d repetido en las aplicaciones", a.CargoID)
		}
		vistos[a.CargoID] = true
		pendiente, ok := pendientes[a.CargoID]
		if !ok {
			return fmt.Errorf("cargo %d no encontrado entre los pendientes del cliente", a.CargoID)
		}
		if centavos(a.Monto) > centavos(pendiente) {
			return fmt.Errorf("la aplicacion no puede superar lo pendiente del cargo %d (%.2f)", a.CargoID, pendiente)
		}
		total += centavos(a.Monto)
	}
	if total > disponible {
		return fmt.Errorf("lo aplicado no puede superar el monto disponible del pago (%.2f)", desdeCentavos(disponible))
	}
	return nil
}

// PendienteDe es lo que falta por cubrir de un cargo; nunca es negativo.
func PendienteDe(monto, pagado float64) float64 {
	pendiente := centavos(monto) - centavos(pagado)
	if pendiente < 0 {
		return 0
	}
	return desdeCentavos(pendiente)
}
