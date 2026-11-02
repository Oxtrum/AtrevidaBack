package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const layoutFechaHecho = "2006-01-02"

// resolverFechaHechoConfigurada aplica ResolverFechaHecho con la zona horaria y el limite de dias de la configuracion.
// Devuelve nil si no se pidio fecha: la base pone el momento actual. Sin `src` usa los defaults; `ahora` nil usa time.Now.
func resolverFechaHechoConfigurada(src ConfigSource, ahora func() time.Time, raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	cfg := configPorDefecto()
	if src != nil {
		snapshot, err := src.Snapshot()
		if err != nil {
			return nil, err
		}
		cfg = snapshot
	}
	if ahora == nil {
		ahora = time.Now
	}
	fecha, err := ResolverFechaHecho(raw, ahora(), cfg.Ubicacion(), cfg.DiasMaximoFechaPasada)
	if err != nil {
		return nil, err
	}
	return &fecha, nil
}

// ResolverFechaHecho convierte la fecha del hecho pedida (YYYY-MM-DD, opcional) en el instante que se guarda.
//
// La fecha del hecho es cuando ocurrio el movimiento (la que digita el usuario, p. ej. al pasar de noche lo anotado
// en papel); la de registro la pone el servidor. Sin fecha o con la fecha de hoy se usa `ahora`, para conservar la hora
// real; una fecha pasada se guarda a las 12:00 de la zona horaria configurada. `maxDiasAtras` limita cuanto se puede
// retroceder; 0 significa sin limite.
//
// Los mensajes de error contienen "formato" o "no puede" porque los handlers mapean a 400 por texto.
func ResolverFechaHecho(raw string, ahora time.Time, loc *time.Location, maxDiasAtras int) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ahora, nil
	}
	dia, err := time.ParseInLocation(layoutFechaHecho, raw, loc)
	if err != nil {
		return time.Time{}, errors.New("fecha_hecho con formato invalido, use YYYY-MM-DD")
	}
	local := ahora.In(loc)
	hoy := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	switch {
	case dia.After(hoy):
		return time.Time{}, errors.New("fecha_hecho no puede ser futura")
	case dia.Equal(hoy):
		return ahora, nil
	}
	if maxDiasAtras > 0 && dia.Before(hoy.AddDate(0, 0, -maxDiasAtras)) {
		return time.Time{}, fmt.Errorf("fecha_hecho no puede ser anterior a %d dias", maxDiasAtras)
	}
	return time.Date(dia.Year(), dia.Month(), dia.Day(), 12, 0, 0, 0, loc), nil
}
