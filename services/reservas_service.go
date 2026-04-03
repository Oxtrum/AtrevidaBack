package services

import (
	"strings"

	"atrevida-agenda-api/models"
)

type FiltroReservas struct {
	Local      string
	Semana     string
	Dia        string
	Tipo       string
	Cliente    string
	Reservados bool
}

func GetReservasFiltradas(f FiltroReservas) ([]models.LocalReservas, error) {
	todos := GetAllReservas()

	locales := filterLocales(todos, f.Local)
	if len(locales) == 0 && f.Local != "" {
		return []models.LocalReservas{}, nil
	}

	for i := range locales {
		locales[i].Semanas = filterSemanas(locales[i].Semanas, f.Semana)
	}

	if f.Dia != "" {
		for i := range locales {
			for j := range locales[i].Semanas {
				locales[i].Semanas[j].Reservas = filterDia(
					locales[i].Semanas[j].Reservas, f.Dia,
				)
			}
		}
	}

	hayFiltroItems := f.Tipo != "" || f.Cliente != "" || f.Reservados
	if hayFiltroItems {
		for i := range locales {
			for j := range locales[i].Semanas {
				locales[i].Semanas[j].Reservas = filterItems(
					locales[i].Semanas[j].Reservas,
					f.Tipo,
					f.Cliente,
					f.Reservados,
				)
			}
		}
	}

	return clearVacios(locales), nil
}

// ── Filtros internos ──────────────────────────────────────────────────────────

func filterLocales(todos []models.LocalReservas, param string) []models.LocalReservas {
	if param == "" {
		return todos
	}
	var out []models.LocalReservas
	for _, l := range todos {
		if strings.EqualFold(l.Local, param) {
			out = append(out, l)
		}
	}
	return out
}

func filterSemanas(semanas []models.Semana, param string) []models.Semana {
	if param == "" {
		return semanas
	}
	var out []models.Semana
	for _, s := range semanas {
		if strings.Contains(strings.ToLower(s.Titulo), strings.ToLower(param)) {
			out = append(out, s)
		}
	}
	return out
}

func filterDia(slots []models.ReservaSlot, param string) []models.ReservaSlot {
	out := make([]models.ReservaSlot, 0, len(slots))
	for _, slot := range slots {
		reducido := models.ReservaSlot{
			Hora: slot.Hora,
			Dias: make(map[string][]models.ReservaItem),
		}
		for dia, items := range slot.Dias {
			if strings.EqualFold(dia, param) {
				reducido.Dias[dia] = items
			}
		}
		out = append(out, reducido)
	}
	return out
}

func filterItems(
	slots []models.ReservaSlot,
	paramTipo string,
	paramCliente string,
	soloReservados bool,
) []models.ReservaSlot {
	out := make([]models.ReservaSlot, 0, len(slots))
	for _, slot := range slots {
		filtrado := models.ReservaSlot{
			Hora: slot.Hora,
			Dias: make(map[string][]models.ReservaItem),
		}
		for dia, items := range slot.Dias {
			aprobados := applyFiltrosItem(items, paramTipo, paramCliente, soloReservados)
			if len(aprobados) > 0 {
				filtrado.Dias[dia] = aprobados
			}
		}
		out = append(out, filtrado)
	}
	return out
}

func applyFiltrosItem(
	items []models.ReservaItem,
	paramTipo string,
	paramCliente string,
	soloReservados bool,
) []models.ReservaItem {
	var out []models.ReservaItem
	for _, item := range items {
		if soloReservados && strings.TrimSpace(item.Cliente) == "" {
			continue
		}
		if paramTipo != "" && !strings.EqualFold(item.Tipo, paramTipo) {
			continue
		}
		if paramCliente != "" &&
			!strings.Contains(strings.ToLower(item.Cliente), strings.ToLower(paramCliente)) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func clearVacios(locales []models.LocalReservas) []models.LocalReservas {
	var out []models.LocalReservas
	for _, l := range locales {
		var semanas []models.Semana
		for _, s := range l.Semanas {
			var slots []models.ReservaSlot
			for _, slot := range s.Reservas {
				if len(slot.Dias) > 0 {
					slots = append(slots, slot)
				}
			}
			if len(slots) > 0 {
				s.Reservas = slots
				semanas = append(semanas, s)
			}
		}
		if len(semanas) > 0 {
			l.Semanas = semanas
			out = append(out, l)
		}
	}
	return out
}
