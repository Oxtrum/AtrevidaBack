package services

import (
	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"testing"
)

type catalogoCostoReserva struct {
	repository.ServiciosRepository
	variable  bool
	consultas int
}

func (r *catalogoCostoReserva) GetServicioByNombre(nombre string) (*models.ServicioItem, error) {
	r.consultas++
	return &models.ServicioItem{CostoVariable: r.variable}, nil
}

func TestModalidadCostoHistorica(t *testing.T) {
	nombre := "Masaje"
	fijo := false
	catalogo := &catalogoCostoReserva{variable: true}
	svc := &ReservasPGService{serviciosRepo: catalogo}
	actual := &models.ReservaPGCompleta{}
	actual.ServicioNombre = &nombre
	actual.CostoVariable = &fijo
	if _, cambiar := svc.modalidadCostoParaCambio(actual, &nombre, true); cambiar {
		t.Fatal("no debe reclasificar un servicio con modalidad historica")
	}
	if catalogo.consultas != 0 {
		t.Fatal("consulta innecesaria al catalogo")
	}
	actual.CostoVariable = nil
	if _, cambiar := svc.modalidadCostoParaCambio(actual, &nombre, false); cambiar {
		t.Fatal("reprogramar no debe clasificar reservas legacy")
	}
	variable, cambiar := svc.modalidadCostoParaCambio(actual, &nombre, true)
	if !cambiar || variable == nil || !*variable {
		t.Fatal("confirmar debe fijar la modalidad del catalogo")
	}
	nuevo := "Otro servicio"
	actual.CostoVariable = &fijo
	variable, cambiar = svc.modalidadCostoParaCambio(actual, &nuevo, false)
	if !cambiar || variable == nil || !*variable {
		t.Fatal("cambiar de servicio debe actualizar la modalidad")
	}
}
