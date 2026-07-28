package services

import (
	"testing"
	"time"
)

func igualSlots(t *testing.T, got, want [][2]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("cantidad de slots = %d, se esperaba %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot %d = %v, se esperaba %v", i, got[i], want[i])
		}
	}
}

func TestPartirEnSlots30UnaHoraDaDosBloques(t *testing.T) {
	got := partirEnSlots30("16:00", "17:00")
	igualSlots(t, got, [][2]string{
		{"16:00", "16:30"},
		{"16:30", "17:00"},
	})
}

func TestPartirEnSlots30MediaHoraDaUnBloque(t *testing.T) {
	got := partirEnSlots30("16:30", "17:00")
	igualSlots(t, got, [][2]string{{"16:30", "17:00"}})
}

func TestPartirEnSlots30RecortaElUltimoBloqueParcial(t *testing.T) {
	got := partirEnSlots30("16:00", "16:50")
	igualSlots(t, got, [][2]string{
		{"16:00", "16:30"},
		{"16:30", "16:50"},
	})
}

func TestPartirEnSlots30HoraInvalidaDaNil(t *testing.T) {
	if got := partirEnSlots30("no-es-hora", "17:00"); got != nil {
		t.Errorf("se esperaba nil para hora inválida, se obtuvo %v", got)
	}
}

func TestGenerarSlots30NoPasaDelCierre(t *testing.T) {
	got := generarSlots30("08:00", "09:00")
	igualSlots(t, got, [][2]string{
		{"08:00", "08:30"},
		{"08:30", "09:00"},
	})
}

func TestHorarioLocalEntreSemanaUltimoSlotEs1930(t *testing.T) {
	// 2026-07-29 es miércoles.
	slots := horarioLocal("SAN MARTIN", time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC))
	if len(slots) != 24 {
		t.Fatalf("cantidad de slots = %d, se esperaba 24 (08:00-20:00 en pasos de 30)", len(slots))
	}
	if slots[0] != [2]string{"08:00", "08:30"} {
		t.Errorf("primer slot = %v, se esperaba 08:00-08:30", slots[0])
	}
	if slots[23] != [2]string{"19:30", "20:00"} {
		t.Errorf("último slot = %v, se esperaba 19:30-20:00", slots[23])
	}
}

func TestHorarioLocalSabadoAranjuezUltimoSlotEs1730(t *testing.T) {
	// 2026-08-01 es sábado. PASEO ARANJUEZ cierra 18:00.
	slots := horarioLocal("PASEO ARANJUEZ", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if len(slots) != 20 {
		t.Fatalf("cantidad de slots = %d, se esperaba 20 (08:00-18:00 en pasos de 30)", len(slots))
	}
	if slots[19] != [2]string{"17:30", "18:00"} {
		t.Errorf("último slot = %v, se esperaba 17:30-18:00", slots[19])
	}
}

func TestHorarioLocalSabadoSanMartinUltimoSlotEs1430(t *testing.T) {
	// 2026-08-01 es sábado. SAN MARTIN cierra 15:00.
	slots := horarioLocal("SAN MARTIN", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if len(slots) != 14 {
		t.Fatalf("cantidad de slots = %d, se esperaba 14 (08:00-15:00 en pasos de 30)", len(slots))
	}
	if slots[13] != [2]string{"14:30", "15:00"} {
		t.Errorf("último slot = %v, se esperaba 14:30-15:00", slots[13])
	}
}

func TestHorarioLocalDomingoNoTieneSlots(t *testing.T) {
	// 2026-08-02 es domingo.
	if slots := horarioLocal("SAN MARTIN", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)); slots != nil {
		t.Errorf("se esperaba nil el domingo, se obtuvo %v", slots)
	}
}

func TestSumar60MinSigueSiendoUnaHora(t *testing.T) {
	// La duración por defecto NO cambia con esta tarea.
	if got := sumar60Min("16:00"); got != "17:00" {
		t.Errorf("sumar60Min(16:00) = %s, se esperaba 17:00", got)
	}
}
