package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// Las imagenes se convierten a webp en el front antes de subirse.
	imagenContentType = "image/webp"
	// Cada subida usa una clave nueva (uuid), por eso el objeto es inmutable y se puede cachear un anio.
	imagenCacheControl = "public, max-age=31536000, immutable"
	urlSubidaTTL       = 5 * time.Minute
)

// S3Storage implementa ObjectStorage sobre un bucket de AWS S3. El front sube directo con
// la URL firmada; el backend nunca recibe los bytes.
type S3Storage struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	region  string
	// publicBaseURL es el dominio de lectura (p. ej. CloudFront). Vacio = URL virtual-hosted del bucket.
	publicBaseURL string
}

// NewS3Storage devuelve nil si no hay bucket configurado: la funcionalidad de imagenes
// queda deshabilitada de forma segura (los endpoints responden 503).
// Las credenciales salen de la cadena por defecto de AWS (rol de la Lambda, variables de entorno o perfil local).
func NewS3Storage(ctx context.Context, bucket, region, publicBaseURL string) (*S3Storage, error) {
	if bucket == "" {
		return nil, nil
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("error al cargar configuracion de AWS: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &S3Storage{
		client:        client,
		presign:       s3.NewPresignClient(client),
		bucket:        bucket,
		region:        region,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}, nil
}

// CrearURLSubida firma un PUT para el path dado. Content-Type y Cache-Control quedan firmados,
// asi que el cliente debe enviar exactamente los headers devueltos en SubidaFirmada.Headers.
func (s *S3Storage) CrearURLSubida(path string) (SubidaFirmada, error) {
	req, err := s.presign.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(path),
		ContentType:  aws.String(imagenContentType),
		CacheControl: aws.String(imagenCacheControl),
	}, s3.WithPresignExpires(urlSubidaTTL))
	if err != nil {
		return SubidaFirmada{}, fmt.Errorf("error al firmar URL de subida en S3: %w", err)
	}
	return SubidaFirmada{
		UploadURL: req.URL,
		Path:      path,
		Headers: map[string]string{
			"Content-Type":  imagenContentType,
			"Cache-Control": imagenCacheControl,
		},
	}, nil
}

// Eliminar borra el objeto. S3 responde exito aunque la clave no exista.
func (s *S3Storage) Eliminar(path string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("error al borrar objeto en S3: %w", err)
	}
	return nil
}

// URLPublica arma la URL de lectura: el dominio publico (CloudFront) si esta configurado,
// o la URL del bucket (requiere lectura publica en el bucket).
func (s *S3Storage) URLPublica(path string) string {
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + path
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, path)
}
