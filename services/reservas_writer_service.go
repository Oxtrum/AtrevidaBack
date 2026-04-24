package services

import (
	repository "atrevida-agenda-api/repositories"
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

// Writer service

type ReservasWriterService struct {
	repo repository.ReservasRepository
}

func NewReservasWriterService(repo repository.ReservasRepository) *ReservasWriterService {
	return &ReservasWriterService{repo: repo}
}

// POST: crear reserva

func (s *ReservasWriterService) CrearReserva(input CrearReservaInput) (ResultadoReserva, error) {
	resultado := ResultadoReserva{}

	slots, err := s.resolverRangoHoras(input.Local, input.Semana, input.Dia, input.HoraDesde, input.HoraHasta)
	if err != nil {
		return resultado, err
	}

	for _, hora := range slots {
		if err := s.escribirEnSlot(input.Local, input.Semana, input.Dia, hora, input.Tipo, input.Cliente, input.Servicio); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("%s: %s", hora, err.Error()))
		} else {
			resultado.Exitosos = append(resultado.Exitosos, hora)
		}
	}

	return resultado, nil
}

// escribirEnSlot lee la celda, valida tipo disponible, ocupa el primer espacio libre y escribe.
func (s *ReservasWriterService) escribirEnSlot(local, semana, dia, hora, tipo, cliente, servicio string) error {
	a1, err := s.repo.ResolverCoordenada(local, semana, dia, hora)
	if err != nil {
		return err
	}

	raw, err := s.repo.GetCeldaRaw(local, a1)
	if err != nil {
		return err
	}

	items := parseSlotCelda(raw)

	existentes := tiposExistentes(items)
	if !existentes[strings.ToUpper(tipo)] {
		keys := make([]string, 0, len(existentes))
		for k := range existentes {
			keys = append(keys, k)
		}
		return fmt.Errorf("el tipo '%s' no está disponible en este slot (disponibles: %s)",
			tipo, strings.Join(keys, ", "))
	}

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

	return s.repo.WriteCelda(local, a1, reconstruirCelda(items))
}

// UPDATE: modificar reserva existente

func (s *ReservasWriterService) ActualizarReserva(input ActualizarReservaInput) (ResultadoReserva, error) {
	resultado := ResultadoReserva{}

	horasOriginales, err := s.resolverRangoHoras(input.Local, input.Semana, input.Dia, input.Hora, "")
	if err != nil {
		return resultado, err
	}

	diaDest := input.Dia
	if input.NuevoDia != "" {
		diaDest = input.NuevoDia
	}

	horasDestino := horasOriginales
	if input.NuevaHoraDesde != "" {
		horasDestino, err = s.resolverRangoHoras(input.Local, input.Semana, diaDest, input.NuevaHoraDesde, input.NuevaHoraHasta)
		if err != nil {
			return resultado, err
		}
	}

	tipoFinal := input.Tipo
	if input.NuevoTipo != "" {
		tipoFinal = input.NuevoTipo
	}

	for _, hora := range horasOriginales {
		if err := s.borrarDeSlot(input.Local, input.Semana, input.Dia, hora, input.Tipo, input.Cliente); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("borrar %s: %s", hora, err.Error()))
		}
	}

	for _, hora := range horasDestino {
		if err := s.escribirEnSlot(input.Local, input.Semana, diaDest, hora, tipoFinal, input.Cliente, input.NuevoServicio); err != nil {
			resultado.Errores = append(resultado.Errores, fmt.Sprintf("escribir %s: %s", hora, err.Error()))
		} else {
			resultado.Exitosos = append(resultado.Exitosos, hora)
		}
	}

	return resultado, nil
}

// borrarDeSlot deja vacía la línea del cliente en el slot.
func (s *ReservasWriterService) borrarDeSlot(local, semana, dia, hora, tipo, cliente string) error {
	a1, err := s.repo.ResolverCoordenada(local, semana, dia, hora)
	if err != nil {
		return err
	}

	raw, err := s.repo.GetCeldaRaw(local, a1)
	if err != nil {
		return err
	}

	items := parseSlotCelda(raw)
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

	return s.repo.WriteCelda(local, a1, reconstruirCelda(items))
}

// Manejo rangos de hora

func (s *ReservasWriterService) resolverRangoHoras(local, semana, dia, horaDesde, horaHasta string) ([]string, error) {
	data := s.repo.GetSheetData(local)

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

	inicio := -1
	fin := -1
	hastaNorm := strings.TrimSpace(horaHasta)

	for i, h := range todasHoras {
		partes := strings.SplitN(h, " a ", 2)
		horaInicio := strings.TrimSpace(partes[0])
		horaFin := ""
		if len(partes) == 2 {
			horaFin = strings.TrimSpace(partes[1])
		}

		if inicio == -1 && strings.EqualFold(horaInicio, buscar) {
			inicio = i
		}

		if inicio != -1 {
			if strings.EqualFold(horaInicio, hastaNorm) {
				fin = i
				break
			}
			if strings.EqualFold(horaFin, hastaNorm) {
				fin = i + 1
				break
			}
		}
	}

	if inicio == -1 {
		return nil, fmt.Errorf("hora de inicio '%s' no encontrada", horaDesde)
	}
	if fin == -1 {
		return nil, fmt.Errorf("hora de fin '%s' no corresponde al inicio ni fin de ningún bloque", horaHasta)
	}
	if fin <= inicio {
		return nil, fmt.Errorf("hora_hasta debe ser posterior a hora_desde")
	}

	return todasHoras[inicio:fin], nil
}
