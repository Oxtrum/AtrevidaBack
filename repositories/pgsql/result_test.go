package pgsql

import (
	"errors"
	"strings"
	"testing"
)

type resultStub struct {
	rows int64
	err  error
}

func (r resultStub) LastInsertId() (int64, error) { return 0, nil }
func (r resultStub) RowsAffected() (int64, error) { return r.rows, r.err }

func TestAffectedRows(t *testing.T) {
	got, err := affectedRows(resultStub{rows: 7}, "actualizar prueba")
	if err != nil {
		t.Fatalf("affectedRows() error = %v", err)
	}
	if got != 7 {
		t.Fatalf("affectedRows() = %d, want 7", got)
	}
}

func TestAffectedRowsPropagaError(t *testing.T) {
	want := errors.New("driver failure")
	_, err := affectedRows(resultStub{err: want}, "actualizar prueba")
	if !errors.Is(err, want) {
		t.Fatalf("affectedRows() error = %v, want wrapped %v", err, want)
	}
	if !strings.Contains(err.Error(), "actualizar prueba") {
		t.Fatalf("affectedRows() error = %q, want operation context", err)
	}
}
