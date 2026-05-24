package repository

import "atrevida-agenda-api/models"

type CrearComboServicioInput struct {
	ComboID       int
	ServicioID    *int
	ServicioTexto string
	Tiempo        string
	Costo         *float64
	Sesiones      int
	Orden         int
}

type ActualizarComboServicioInput struct {
	ID            int
	ServicioID    *int
	ServicioTexto *string
	Tiempo        *string
	Costo         *float64
	Sesiones      *int
	Orden         *int
}

type ComboServiciosRepository interface {
	GetComboServicioByID(id int) (*models.ComboServicioDetallePG, error)
	GetComboServiciosByComboID(comboID int) ([]models.ComboServicioDetallePG, error)
	CreateComboServicio(input CrearComboServicioInput) (int, error)
	UpdateComboServicio(input ActualizarComboServicioInput) error
	DeleteComboServicio(id int) error
}
