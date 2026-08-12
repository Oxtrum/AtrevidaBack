package pgsql

import "testing"

func TestBatchValuesPlaceholders(t *testing.T) {
	got := batchValuesPlaceholders(3, 2)
	want := "($1,$2),($3,$4),($5,$6)"
	if got != want {
		t.Fatalf("batchValuesPlaceholders() = %q, want %q", got, want)
	}
}

func TestBatchValuesPlaceholdersVacio(t *testing.T) {
	if got := batchValuesPlaceholders(0, 3); got != "" {
		t.Fatalf("batchValuesPlaceholders() = %q, want empty", got)
	}
}
