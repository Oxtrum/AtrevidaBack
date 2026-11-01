package cuenta

import (
	"fmt"
	"math"
	"sort"
	"time"
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

// Etiquetas del saldo de un cliente.
const (
	SaldoAFavor = "A_FAVOR"
	SaldoDeuda  = "DEUDA"
	SaldoAlDia  = "AL_DIA"
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

// Saldo es la posicion de un cliente: positivo = saldo a favor, negativo = deuda.
type Saldo struct {
	// Monto con signo: positivo a favor, negativo deuda.
	Monto float64 `json:"monto" example:"-500"`
	// Absoluto es el valor sin signo, para mostrarlo junto a la etiqueta.
	Absoluto float64 `json:"absoluto" example:"500"`
	// Etiqueta: A_FAVOR, DEUDA o AL_DIA.
	Etiqueta string `json:"etiqueta" example:"DEUDA" enums:"A_FAVOR,DEUDA,AL_DIA"`
}

// SaldoCliente calcula el saldo como total de abonos menos total de cargos. Nunca se guarda: se calcula (RN-C5).
func SaldoCliente(totalAbonos, totalCargos float64) Saldo {
	diferencia := centavos(totalAbonos) - centavos(totalCargos)
	switch {
	case diferencia > 0:
		return Saldo{Monto: desdeCentavos(diferencia), Absoluto: desdeCentavos(diferencia), Etiqueta: SaldoAFavor}
	case diferencia < 0:
		return Saldo{Monto: desdeCentavos(diferencia), Absoluto: desdeCentavos(-diferencia), Etiqueta: SaldoDeuda}
	default:
		return Saldo{Etiqueta: SaldoAlDia}
	}
}

// CargoPorCobrar es un cargo con algo pendiente, candidato a recibir parte de un abono.
type CargoPorCobrar struct {
	ID        int
	Pendiente float64
	// Fecha es la fecha del hecho del cargo; ordena el reparto del mas antiguo al mas nuevo.
	Fecha time.Time
}

// Reparto es cuanto de un abono se aplica a un cargo.
type Reparto struct {
	CargoID int     `json:"cargo_id" example:"12"`
	Monto   float64 `json:"monto" example:"400"`
}

// SugerirReparto propone repartir `monto` del cargo mas antiguo al mas nuevo (a igual fecha, por id). Devuelve
// el reparto y lo que sobra, que queda como saldo a favor. Es solo una sugerencia: el cajero la confirma o la cambia.
func SugerirReparto(cargos []CargoPorCobrar, monto float64) ([]Reparto, float64) {
	restante := centavos(monto)
	if restante <= 0 {
		return nil, 0
	}
	ordenados := make([]CargoPorCobrar, len(cargos))
	copy(ordenados, cargos)
	sort.SliceStable(ordenados, func(i, j int) bool {
		if !ordenados[i].Fecha.Equal(ordenados[j].Fecha) {
			return ordenados[i].Fecha.Before(ordenados[j].Fecha)
		}
		return ordenados[i].ID < ordenados[j].ID
	})
	var reparto []Reparto
	for _, c := range ordenados {
		pendiente := centavos(c.Pendiente)
		if pendiente <= 0 {
			continue
		}
		if restante <= 0 {
			break
		}
		aplicar := pendiente
		if restante < aplicar {
			aplicar = restante
		}
		reparto = append(reparto, Reparto{CargoID: c.ID, Monto: desdeCentavos(aplicar)})
		restante -= aplicar
	}
	return reparto, desdeCentavos(restante)
}

// ValidarAplicaciones comprueba un reparto contra las reglas de integridad:
// RN-C1 un abono no aplica mas de su monto disponible (monto menos lo ya aplicado) y
// RN-C2 un cargo no recibe mas de lo que tiene pendiente.
// Aplicar menos que el monto del abono es valido: lo que sobra queda como saldo a favor.
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
