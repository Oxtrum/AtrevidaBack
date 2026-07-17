package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SubidaFirmada contiene los datos que el frontend necesita para subir el
// archivo directamente a Supabase Storage sin pasar los bytes por el backend.
type SubidaFirmada struct {
	// URL absoluta a la que el cliente hace PUT con el archivo.
	UploadURL string `json:"upload_url"`
	// Token de la subida firmada (incluido tambien en UploadURL).
	Token string `json:"token"`
	// Path del objeto dentro del bucket.
	Path string `json:"path"`
}

// SupabaseStorage es un cliente HTTP fino sobre la API de Supabase Storage.
// Emite URLs firmadas de subida, borra objetos y deriva URLs publicas.
type SupabaseStorage struct {
	baseURL   string
	secretKey string
	bucket    string
	client    *http.Client
}

// NewSupabaseStorage devuelve nil si falta configuracion (URL o secret): en ese
// caso la funcionalidad de imagenes queda deshabilitada de forma segura.
func NewSupabaseStorage(baseURL, secretKey, bucket string) *SupabaseStorage {
	if baseURL == "" || secretKey == "" || bucket == "" {
		return nil
	}
	return &SupabaseStorage{
		baseURL:   strings.TrimRight(baseURL, "/"),
		secretKey: secretKey,
		bucket:    bucket,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// CrearURLSubida pide a Supabase una URL firmada de subida para el path dado.
func (s *SupabaseStorage) CrearURLSubida(path string) (SubidaFirmada, error) {
	endpoint := fmt.Sprintf("%s/storage/v1/object/upload/sign/%s/%s", s.baseURL, s.bucket, path)
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return SubidaFirmada{}, err
	}
	s.setAuth(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return SubidaFirmada{}, fmt.Errorf("error al contactar Supabase Storage: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return SubidaFirmada{}, fmt.Errorf("Supabase Storage respondio %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.URL == "" {
		return SubidaFirmada{}, fmt.Errorf("respuesta de firma invalida de Supabase Storage")
	}

	uploadURL := s.baseURL + "/storage/v1" + parsed.URL
	token := ""
	if parsedURL, err := url.Parse(parsed.URL); err == nil {
		token = parsedURL.Query().Get("token")
	}
	return SubidaFirmada{UploadURL: uploadURL, Token: token, Path: path}, nil
}

// Eliminar borra el objeto del bucket. Un 404 se considera exito idempotente.
func (s *SupabaseStorage) Eliminar(path string) error {
	endpoint := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, path)
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	s.setAuth(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error al contactar Supabase Storage: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Supabase Storage respondio %d al borrar: %s", resp.StatusCode, string(body))
	}
	return nil
}

// URLPublica arma la URL de lectura publica del objeto (bucket publico).
func (s *SupabaseStorage) URLPublica(path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.baseURL, s.bucket, path)
}

func (s *SupabaseStorage) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("apikey", s.secretKey)
}
