package pgsql

import (
	"context"
	"testing"
)

func TestQueryContextConservaCancelacion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := queryContext(ctx)
	if got != ctx {
		t.Fatal("queryContext() reemplazo el contexto recibido")
	}
	if got.Err() != context.Canceled {
		t.Fatalf("queryContext().Err() = %v, want context.Canceled", got.Err())
	}
}

func TestQueryContextAceptaNil(t *testing.T) {
	if got := queryContext(nil); got == nil {
		t.Fatal("queryContext(nil) devolvio nil")
	}
}
