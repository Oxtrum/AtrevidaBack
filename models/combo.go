package models

type ServicioIncluido struct {
	Nombre   string `json:"nombre"`
	Tiempo   string `json:"tiempo"`
	Costo    string `json:"costo"`
	Sesiones int    `json:"sesiones"`
}

type ComboItem struct {
	Nombre             string             `json:"nombre"`
	Categoria          string             `json:"categoria"`
	Local              string             `json:"local"`
	CostoTotal         string             `json:"costo_total"`
	SesionesTotales    int                `json:"sesiones_totales"`
	ServiciosIncluidos []ServicioIncluido `json:"servicios_incluidos"`
}
