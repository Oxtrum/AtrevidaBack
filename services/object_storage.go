package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

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

// ObjectStorageFirmado agrega la lectura privada: para objetos que no son publicos (fotos del personal)
// se emite una URL firmada que caduca.
type ObjectStorageFirmado interface {
	ObjectStorage
	// URLLecturaFirmada emite una URL de lectura temporal para un objeto privado.
	URLLecturaFirmada(path string, ttl time.Duration) (string, error)
}

// nuevaClaveImagen genera una clave nueva por subida (prefijo + uuid + .webp): asi el navegador/CDN
// nunca sirve la imagen vieja al reemplazarla. El prefijo termina en "/".
func nuevaClaveImagen(prefijo string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("error al generar clave de imagen: %w", err)
	}
	return prefijo + hex.EncodeToString(b) + ".webp", nil
}

// validarClaveImagen exige que la clave confirmada cuelgue del prefijo y tenga la forma que emite el backend.
func validarClaveImagen(prefijo, campo, key string) error {
	nombre := strings.TrimPrefix(key, prefijo)
	base := strings.TrimSuffix(nombre, ".webp")
	if !strings.HasPrefix(key, prefijo) || base == nombre || base == "" || strings.ContainsAny(base, `/\.`) {
		return errors.New(campo + " invalida: debe ser la clave emitida por upload-url para este registro")
	}
	return nil
}
