package services

import (
	"atrevida-agenda-api/models"
	repository "atrevida-agenda-api/repositories"
	"testing"
	"time"
)

type catalogoCostoReserva struct {
	repository.ServiciosRepository
	variable  bool
	consultas int
}

type reservaCostoUpdateRepo struct {
	reservasResumenRepo
	update repository.UpdateReservaInput
	estado repository.UpdateReservaEstadoInput
	creado repository.CreateReservaInput
}

func (r *reservaCostoUpdateRepo) CreateReserva(input repository.CreateReservaInput) (int, error) {
	r.creado = input
	return 1, nil
}

func (r *reservaCostoUpdateRepo) UpdateReserva(input repository.UpdateReservaInput) error {
	r.update = input
	return nil
}
func (r *reservaCostoUpdateRepo) UpdateReservaEstado(input repository.UpdateReservaEstadoInput) error {
	r.estado = input
	return nil
}

func TestReservaCambioVariableLimpiaPrecioSalvoImporteExplicito(t *testing.T) {
	for _, precio := range []*float64{nil, floatPtr(0), floatPtr(125.5)} {
		estado, servicio, nuevo := "AGENDADO", "Fijo", "Variable"
		fijo := false
		current := &models.ReservaPGCompleta{}
		current.Estado = &estado
		current.ServicioNombre = &servicio
		current.CostoVariable = &fijo
		current.LocalNombre = "SAN MARTIN"
		current.Fecha = time.Date(2099, 1, 5, 0, 0, 0, 0, time.UTC)
		current.HoraDesde = "10:00"
		current.HoraHasta = "11:00"
		current.Precio = floatPtr(70)
		repo := &reservaCostoUpdateRepo{reservasResumenRepo: reservasResumenRepo{reserva: current}}
		svc := &ReservasPGService{repo: repo, serviciosRepo: &catalogoCostoReserva{variable: true}}
		if err := svc.ActualizarReserva(ActualizarReservaPGInput{Id: 1, Local: "SAN MARTIN", NuevoServicio: nuevo, NuevoPrecio: precio}); err != nil {
			t.Fatal(err)
		}
		if !repo.update.NuevoCostoVariableSet || repo.update.NuevoCostoVariable == nil || !*repo.update.NuevoCostoVariable {
			t.Fatal("modalidad no actualizada")
		}
		if precio == nil {
			if !repo.update.NuevoPrecioSet || repo.update.NuevoPrecio != nil {
				t.Fatal("precio anterior no se limpio")
			}
		} else if repo.update.NuevoPrecio == nil || *repo.update.NuevoPrecio != *precio {
			t.Fatal("importe explicito perdido")
		}
	}
}

func (r *catalogoCostoReserva) GetServicioByNombre(nombre string) (*models.ServicioItem, error) {
	r.consultas++
	return &models.ServicioItem{Nombre: nombre, CostoVariable: r.variable}, nil
}

func TestCrearReservaVariablePublicaYAdministrativa(t *testing.T) {
	for _, local := range []string{"SAN MARTIN", "PASEO ARANJUEZ"} {
		for _, estado := range []string{"PENDIENTE", "AGENDADO"} {
			repo := &reservaCostoUpdateRepo{}
			svc := &ReservasPGService{repo: repo, serviciosRepo: &catalogoCostoReserva{variable: true}}
			_, err := svc.CrearReserva(CrearReservaPGInput{
				Local: local, Fecha: "2099-01-05", HoraDesde: "10:00", HoraHasta: "11:00", Tipo: "M",
				Cliente: "Prueba", Telefono: "70711360", Servicio: "Variable", ServicioSolicitado: "Variable", Estado: estado,
			})
			if err != nil {
				t.Fatal(err)
			}
			if repo.creado.CostoVariable == nil || !*repo.creado.CostoVariable || repo.creado.Precio != nil {
				t.Fatal("variable debe conservar modalidad sin inventar un importe")
			}
			if repo.creado.Estado != estado {
				t.Fatalf("estado = %s", repo.creado.Estado)
			}
			if estado == "PENDIENTE" && repo.creado.ServicioConfirmado != nil {
				t.Fatal("reserva publica pendiente no debe confirmarse")
			}
		}
	}
}

func TestAprobacionVariableSinPrecioYCeroExplicito(t *testing.T) {
	for _, precio := range []*float64{nil, floatPtr(0), floatPtr(100)} {
		estado, servicio, nuevo := "PENDIENTE", "Fijo", "Variable"
		current := &models.ReservaPGCompleta{}
		current.Estado = &estado
		current.ServicioNombre = &servicio
		current.Precio = floatPtr(70)
		repo := &reservaCostoUpdateRepo{reservasResumenRepo: reservasResumenRepo{reserva: current}}
		svc := &ReservasPGService{repo: repo, serviciosRepo: &catalogoCostoReserva{variable: true}}
		if err := svc.ActualizarEstadoReserva(ActualizarEstadoReservaInput{Id: 1, Estado: "AGENDADO", ServicioConfirmado: &nuevo, Precio: precio}); err != nil {
			t.Fatal(err)
		}
		if !repo.estado.CostoVariableSet || repo.estado.CostoVariable == nil || !*repo.estado.CostoVariable {
			t.Fatal("modalidad no fijada")
		}
		if precio == nil && !repo.estado.PrecioSet {
			t.Fatal("importe anterior no limpiado")
		}
		if precio != nil && (repo.estado.Precio == nil || *repo.estado.Precio != *precio) {
			t.Fatal("importe explicito perdido")
		}
	}
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
