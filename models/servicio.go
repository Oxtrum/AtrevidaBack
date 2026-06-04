package models

type ServicioItem struct {
	Id                    int    `json:"id" example:"8"`
	Nombre                string `json:"nombre" example:"Depilacion Laser Piernas"`
	Categoria             string `json:"categoria" example:"Corporal"`
	Local                 string `json:"local" example:"ARANJUEZ"`
	Tiempo                string `json:"tiempo" example:"01:00"`
	Costo                 string `json:"costo" example:"350"`
	Sesiones              int    `json:"sesiones" example:"6"`
	TipoEspacio           string `json:"tipoEspacio" example:"M"`
	RequiereEvaluacion    bool   `json:"requiere_evaluacion" example:"true"`
	VisiblePacienteNuevo  bool   `json:"visible_paciente_nuevo" example:"true"`
}
