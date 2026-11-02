package cuenta

import "errors"

// Cobro de la pantalla de caja. El cajero marca los cargos que el cliente paga ahora y se pagan completos, sin
// pagos parciales. El saldo disponible se usa primero si el cliente quiere; lo que falta se paga con un metodo
// de pago y una recarga opcional queda como saldo disponible. Es logica pura, en centavos enteros.

// CargoACobrar es un cargo marcado para pagarse completo.
type CargoACobrar struct {
	ID        int
	Pendiente float64
}

// FuenteDinero es un pago (abono) con dinero sin usar con el que se puede cubrir cargos.
type FuenteDinero struct {
	PagoID     int
	Disponible float64
}

// Cobro dice cuanto de un pago cubre un cargo. Cada uno se guarda como un registro nuevo.
type Cobro struct {
	PagoID  int     `json:"pago_id"`
	CargoID int     `json:"cargo_id"`
	Monto   float64 `json:"monto"`
}

// PlanCobro es el resumen que ve el cajero antes de confirmar.
type PlanCobro struct {
	// Total de los cargos marcados.
	Total float64 `json:"total" example:"250"`
	// UsaSaldo es lo que se descuenta del saldo disponible del cliente.
	UsaSaldo float64 `json:"usa_saldo" example:"150"`
	// Falta es lo que el cliente paga ahora con un metodo de pago.
	Falta float64 `json:"falta" example:"100"`
	// NuevoPago es el dinero nuevo que entra: lo que falta mas la recarga. Si es 0 no se pide metodo de pago.
	NuevoPago float64 `json:"nuevo_pago" example:"100"`
}

// PlanearCobro calcula cuanto sale del saldo, cuanto falta pagar y cuanto dinero nuevo entra. Si el cliente
// prefiere guardar su saldo, usarSaldo es false y todo se paga con dinero nuevo.
func PlanearCobro(total, disponible float64, usarSaldo bool, recarga float64) PlanCobro {
	t, d, r := centavos(total), centavos(disponible), centavos(recarga)
	if t < 0 {
		t = 0
	}
	if d < 0 {
		d = 0
	}
	if r < 0 {
		r = 0
	}
	var usa int64
	if usarSaldo {
		usa = d
		if usa > t {
			usa = t
		}
	}
	falta := t - usa
	return PlanCobro{
		Total:     desdeCentavos(t),
		UsaSaldo:  desdeCentavos(usa),
		Falta:     desdeCentavos(falta),
		NuevoPago: desdeCentavos(falta + r),
	}
}

// ErrDineroInsuficiente indica que las fuentes no alcanzan a cubrir los cargos completos.
var ErrDineroInsuficiente = errors.New("el dinero disponible no alcanza para cubrir los cargos")

// CubrirCargos reparte el dinero de las fuentes, en el orden dado, sobre los cargos, en el orden dado, hasta
// dejar cada cargo completo. Una fuente puede cubrir varios cargos y un cargo puede salir de varias fuentes.
// Si el dinero no alcanza no devuelve nada: un cobro no queda a medias.
func CubrirCargos(cargos []CargoACobrar, fuentes []FuenteDinero) ([]Cobro, error) {
	restantes := make([]int64, len(fuentes))
	for i, f := range fuentes {
		restantes[i] = centavos(f.Disponible)
	}
	var cobros []Cobro
	fuente := 0
	for _, c := range cargos {
		falta := centavos(c.Pendiente)
		for falta > 0 {
			for fuente < len(fuentes) && restantes[fuente] <= 0 {
				fuente++
			}
			if fuente >= len(fuentes) {
				return nil, ErrDineroInsuficiente
			}
			aporte := restantes[fuente]
			if aporte > falta {
				aporte = falta
			}
			cobros = append(cobros, Cobro{PagoID: fuentes[fuente].PagoID, CargoID: c.ID, Monto: desdeCentavos(aporte)})
			restantes[fuente] -= aporte
			falta -= aporte
		}
	}
	return cobros, nil
}
