package services

// ObjectStorage es el contrato de almacenamiento de imagenes que usa inventario.
// La implementacion S3 (ver ARQUITECTURA_TRASTIENDA.md) debe cumplir estos tres metodos.
type ObjectStorage interface {
	// CrearURLSubida emite una URL firmada para que el front suba el archivo directo.
	CrearURLSubida(path string) (SubidaFirmada, error)
	// Eliminar borra el objeto; si no existe se considera exito.
	Eliminar(path string) error
	// URLPublica arma la URL de lectura del objeto.
	URLPublica(path string) string
}
