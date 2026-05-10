package models

type ServicioItem struct {
	Id          int    `json:"id"`
	Nombre      string `json:"nombre"`
	Categoria   string `json:"categoria"`
	Local       string `json:"local"`
	Tiempo      string `json:"tiempo"`
	Costo       string `json:"costo"`
	Sesiones    int    `json:"sesiones"`
	TipoEspacio string `json:"tipoEspacio"`
}
