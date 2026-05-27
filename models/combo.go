package models

type ServicioIncluido struct {
	Nombre   string `json:"nombre" example:"Depilacion Laser Piernas"`
	Tiempo   string `json:"tiempo" example:"01:00"`
	Costo    string `json:"costo" example:"350"`
	Sesiones int    `json:"sesiones" example:"6"`
}

type ComboItem struct {
	ID                 int                `json:"id,omitempty" example:"12"`
	Nombre             string             `json:"nombre" example:"Relax Total"`
	Categoria          string             `json:"categoria" example:"Corporal"`
	Local              string             `json:"local" example:"ARANJUEZ"`
	CostoTotal         string             `json:"costo_total" example:"1200"`
	SesionesTotales    int                `json:"sesiones_totales" example:"4"`
	ServiciosIncluidos []ServicioIncluido `json:"servicios_incluidos"`
}
