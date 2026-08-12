package pagination

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultLimit  = 50
	MaxLimit      = 100
	cursorVersion = 1
)

var ErrInvalid = errors.New("paginacion invalida")

type Request struct {
	Enabled bool
	Limit   int
	Cursor  string
}

type Metadata struct {
	Limit        int     `json:"limit" example:"50"`
	HasMore      bool    `json:"has_more" example:"true"`
	NextCursor   *string `json:"next_cursor" extensions:"x-nullable"`
	TotalRecords *int    `json:"total_registros,omitempty" example:"2500"`
	TotalPages   *int    `json:"total_paginas,omitempty" example:"50"`
}

type envelope struct {
	Version int             `json:"v"`
	Scope   string          `json:"s"`
	Filters string          `json:"f"`
	Values  json.RawMessage `json:"p"`
	Check   string          `json:"c"`
}

func Parse(rawLimit, rawCursor string) (Request, error) {
	rawLimit = strings.TrimSpace(rawLimit)
	rawCursor = strings.TrimSpace(rawCursor)
	if rawLimit == "" && rawCursor == "" {
		return Request{}, nil
	}
	limit := DefaultLimit
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > MaxLimit {
			return Request{}, fmt.Errorf("%w: limit debe estar entre 1 y %d", ErrInvalid, MaxLimit)
		}
		limit = parsed
	}
	return Request{Enabled: true, Limit: limit, Cursor: rawCursor}, nil
}

func (r Request) QueryLimit() int {
	if !r.Enabled {
		return 0
	}
	return r.Limit + 1
}

func Decode(cursor, scope string, filters any, target any) error {
	if strings.TrimSpace(cursor) == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return fmt.Errorf("%w: cursor invalido", ErrInvalid)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Version != cursorVersion || env.Scope != scope {
		return fmt.Errorf("%w: cursor invalido", ErrInvalid)
	}
	fingerprint, err := hashJSON(filters)
	if err != nil || env.Filters != fingerprint || env.Check != checksum(env.Version, env.Scope, env.Filters, env.Values) {
		return fmt.Errorf("%w: cursor no corresponde a estos filtros", ErrInvalid)
	}
	if err := json.Unmarshal(env.Values, target); err != nil {
		return fmt.Errorf("%w: cursor incompleto", ErrInvalid)
	}
	return nil
}

func Build[T any](items []T, request Request, scope string, filters any, position func(T) any) ([]T, *Metadata, error) {
	if !request.Enabled {
		return items, nil, nil
	}
	hasMore := len(items) > request.Limit
	if hasMore {
		items = items[:request.Limit]
	}
	meta := &Metadata{Limit: request.Limit, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		cursor, err := encode(scope, filters, position(items[len(items)-1]))
		if err != nil {
			return nil, nil, err
		}
		meta.NextCursor = &cursor
	}
	return items, meta, nil
}

func AddTotal(metadata *Metadata, total int) {
	if metadata == nil {
		return
	}
	pages := (total + metadata.Limit - 1) / metadata.Limit
	if pages < 1 {
		pages = 1
	}
	metadata.TotalRecords = &total
	metadata.TotalPages = &pages
}

func encode(scope string, filters, values any) (string, error) {
	fingerprint, err := hashJSON(filters)
	if err != nil {
		return "", err
	}
	position, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	env := envelope{Version: cursorVersion, Scope: scope, Filters: fingerprint, Values: position}
	env.Check = checksum(env.Version, env.Scope, env.Filters, env.Values)
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}

func checksum(version int, scope, filters string, values []byte) string {
	raw := fmt.Sprintf("atrevida-cursor|%d|%s|%s|%s", version, scope, filters, values)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
