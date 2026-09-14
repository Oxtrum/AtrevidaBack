package services

import (
	repository "atrevida-agenda-api/repositories"
	"testing"
)

type planesCostoSnapshotRepo struct {
	repository.PlanesRepository
	input repository.CrearPlanInput
}

func (r *planesCostoSnapshotRepo) CreatePlan(input repository.CrearPlanInput) (int, error) {
	r.input = input
	return 1, nil
}

func TestPlanMantieneSnapshotsSinConsultarCatalogo(t *testing.T) {
	for _, modalidad := range []string{TipoPagoUnico, TipoPagoCuotas} {
		t.Run(modalidad, func(t *testing.T) {
			cero, pactado := 0.0, 80.0
			id := 1
			repo := &planesCostoSnapshotRepo{}
			svc := NewPlanesService(repo, nil)
			_, err := svc.CrearPlan(CrearPlanInput{
				ClienteID: 1, LocalID: 1, TipoPago: modalidad, CantidadCuotas: 3,
				Servicios: []PlanServicioInput{
					{ServicioIDOrigen: &id, NombreTexto: "Referencia variable", PrecioUnitarioTexto: &cero, SesionesContratadas: 3, Orden: 0},
					{NombreTexto: "Importe pactado", PrecioUnitarioTexto: &pactado, SesionesContratadas: 2, Orden: 1},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if repo.input.Subtotal != 160 || repo.input.PrecioTotal != 160 {
				t.Fatalf("importe del plan alterado: %+v", repo.input)
			}
			if *repo.input.Servicios[0].PrecioUnitarioTexto != 0 || *repo.input.Servicios[1].PrecioUnitarioTexto != 80 {
				t.Fatal("snapshots alterados")
			}
			totalCuotas := 0.0
			for _, cuota := range repo.input.Cuotas {
				totalCuotas += cuota.Monto
			}
			if redondearImportePlan(totalCuotas) != 160 {
				t.Fatalf("cuotas = %v", totalCuotas)
			}
		})
	}
}
