package services

import (
	"fmt"
	"strings"
)

// Structs de entrada

type CrearReservaInput struct {
	Local     string
	Semana    string
	Dia       string
	HoraDesde string
	HoraHasta string // opcional: si se omite, solo se reserva HoraDesde
	Tipo      string // "M" o "B"
	Cliente   string
	Servicio  string // texto plano, opcional
}

type ActualizarReservaInput struct {
	// Identificador de la reserva existente
	Local   string
	Semana  string
	Dia     string
	Hora    string // slot exacto ("9:00 a 9:30") o inicio ("9:00")
	Tipo    string // "M" o "B"
	Cliente string

	// Campos modificables (vacío = no cambiar)
	NuevoDia       string
	NuevaHoraDesde string
	NuevaHoraHasta string
	NuevoTipo      string
	NuevoServicio  string
}

type ResultadoReserva struct {
	Exitosos []string
	Errores  []string
}

// POST: crear reserva

func CrearReserva(input CrearReservaInput) (ResultadoReserva, error) {
	resultado := ResultadoReserva{}

	slots, err := resolverRangoHoras(input.Local, input.Semana, input.Dia, input.HoraDesde, input.HoraHasta)
	if err != nil {
		return resultado, err
	}

	for _, hora := range slots {
		if err := escribirEnSlot(input.Local, input.Semana, input.Dia, hora, input.Tipo, input.Cliente, input.Servicio); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("%s: %s", hora, err.Error()))
		} else {
			resultado.Exitosos = append(resultado.Exitosos, hora)
		}
	}

	return resultado, nil
}

// escribirEnSlot lee la celda, valida tipo disponible, ocupa el primer espacio libre y escribe.
func escribirEnSlot(local, semana, dia, hora, tipo, cliente, servicio string) error {
	a1, err := ResolverCoordenada(local, semana, dia, hora)
	if err != nil {
		return err
	}

	raw, err := GetCeldaRaw(local, a1)
	if err != nil {
		return err
	}

	// usar el parser existente
	items := ParseSlotCelda(raw)

	// validar que el tipo existe en este slot
	existentes := TiposExistentes(items)
	if !existentes[strings.ToUpper(tipo)] {
		keys := make([]string, 0, len(existentes))
		for k := range existentes {
			keys = append(keys, k)
		}
		return fmt.Errorf("el tipo '%s' no está disponible en este slot (disponibles: %s)",
			tipo, strings.Join(keys, ", "))
	}

	// ocupar el primer espacio libre del tipo pedido
	tipoNombre := letraToTipoNombre(tipo)
	encontrado := false
	for i, item := range items {
		if item.Tipo == tipoNombre && item.Cliente == "" {
			items[i].Cliente = cliente
			items[i].Servicio = servicio
			encontrado = true
			break
		}
	}

	if !encontrado {
		return fmt.Errorf("no hay espacios libres de tipo '%s' en este slot", tipo)
	}

	return WriteCelda(local, a1, ReconstruirCelda(items))
}

// UPDATE: modificar reserva existente 

func ActualizarReserva(input ActualizarReservaInput) (ResultadoReserva, error) {
	resultado := ResultadoReserva{}

	horasOriginales, err := resolverRangoHoras(input.Local, input.Semana, input.Dia, input.Hora, "")
	if err != nil {
		return resultado, err
	}

	diaDest := input.Dia
	if input.NuevoDia != "" {
		diaDest = input.NuevoDia
	}

	horasDestino := horasOriginales
	if input.NuevaHoraDesde != "" {
		horasDestino, err = resolverRangoHoras(input.Local, input.Semana, diaDest, input.NuevaHoraDesde, input.NuevaHoraHasta)
		if err != nil {
			return resultado, err
		}
	}

	tipoFinal := input.Tipo
	if input.NuevoTipo != "" {
		tipoFinal = input.NuevoTipo
	}

	for _, hora := range horasOriginales {
		if err := borrarDeSlot(input.Local, input.Semana, input.Dia, hora, input.Tipo, input.Cliente); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("borrar %s: %s", hora, err.Error()))
		}
	}

	for _, hora := range horasDestino {
		if err := escribirEnSlot(input.Local, input.Semana, diaDest, hora, tipoFinal, input.Cliente, input.NuevoServicio); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("escribir %s: %s", hora, err.Error()))
		} else {
			resultado.Exitosos = append(resultado.Exitosos, hora)
		}
	}

	return resultado, nil
}

// borrarDeSlot deja vacía la línea del cliente en el slot.
func borrarDeSlot(local, semana, dia, hora, tipo, cliente string) error {
	a1, err := ResolverCoordenada(local, semana, dia, hora)
	if err != nil {
		return err
	}

	raw, err := GetCeldaRaw(local, a1)
	if err != nil {
		return err
	}

	items := ParseSlotCelda(raw)
	tipoNombre := letraToTipoNombre(tipo)

	encontrado := false
	for i, item := range items {
		if item.Tipo == tipoNombre &&
			strings.EqualFold(strings.TrimSpace(item.Cliente), strings.TrimSpace(cliente)) {
			items[i].Cliente = ""
			items[i].Servicio = ""
			encontrado = true
			break
		}
	}

	if !encontrado {
		return fmt.Errorf("no se encontró '%s' de tipo '%s' en el slot %s", cliente, tipo, hora)
	}

	return WriteCelda(local, a1, ReconstruirCelda(items))
}

// Resolución de rangos de hora

func resolverRangoHoras(local, semana, dia, horaDesde, horaHasta string) ([]string, error) {
	data := GetSheetData(local)

	todasHoras := []string{}
	enSemana := false

	for _, fila := range data {
		if len(fila) == 0 {
			continue
		}
		val, ok := fila[0].(string)
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)

		if strings.Contains(strings.ToUpper(val), "SEMANA") {
			enSemana = strings.Contains(strings.ToUpper(val), strings.ToUpper(semana))
			continue
		}
		if !enSemana || val == "HORAS" || val == "" {
			continue
		}
		todasHoras = append(todasHoras, val)
	}

	if len(todasHoras) == 0 {
		return nil, fmt.Errorf("no se encontraron horas para la semana '%s' en '%s'", semana, local)
	}

	buscar := strings.TrimSpace(horaDesde)

	// sin horaHasta: retornar el slot que empiece con horaDesde
	if horaHasta == "" {
		for _, h := range todasHoras {
			if strings.EqualFold(strings.TrimSpace(h), buscar) {
				return []string{h}, nil
			}
			horaInicio := strings.TrimSpace(strings.SplitN(h, " a ", 2)[0])
			if strings.EqualFold(horaInicio, buscar) {
				return []string{h}, nil
			}
		}
		return nil, fmt.Errorf("hora '%s' no encontrada en la semana '%s'", horaDesde, semana)
	}

	// con horaHasta: retornar todos los slots desde horaDesde hasta (sin incluir) horaHasta
	inicio := -1
	fin := -1

	for i, h := range todasHoras {
		horaInicio := strings.TrimSpace(strings.SplitN(h, " a ", 2)[0])
		if inicio == -1 && strings.EqualFold(horaInicio, buscar) {
			inicio = i
		}
		if strings.EqualFold(horaInicio, strings.TrimSpace(horaHasta)) {
			fin = i
			break
		}
	}

	if inicio == -1 {
		return nil, fmt.Errorf("hora de inicio '%s' no encontrada", horaDesde)
	}
	if fin == -1 {
		return nil, fmt.Errorf("hora de fin '%s' no encontrada", horaHasta)
	}
	if fin <= inicio {
		return nil, fmt.Errorf("hora_hasta debe ser posterior a hora_desde")
	}

	return todasHoras[inicio:fin], nil
}

// Utilidades

// letraToTipoNombre convierte "M" → "mesa", "B" → "bicicleta"
// para comparar contra los valores que devuelve ParseCelda.
func letraToTipoNombre(letra string) string {
	switch strings.ToUpper(letra) {
	case "M":
		return "mesa"
	case "B":
		return "bicicleta"
	}
	return ""
}
