package telefono

import "testing"

func TestNormalizeE164(t *testing.T) {
	value, err := NormalizeE164("+591 700-11223")
	if err != nil || value != "+59170011223" {
		t.Fatalf("NormalizeE164() = %q, %v", value, err)
	}
}

func TestNormalizeE164RejectsNationalNumber(t *testing.T) {
	if _, err := NormalizeE164("70011223"); err == nil {
		t.Fatal("NormalizeE164() acepto un numero nacional ambiguo")
	}
}

func TestTryNormalizeLegacy(t *testing.T) {
	value, ok := TryNormalizeLegacy("70011223")
	if !ok || value != "+59170011223" {
		t.Fatalf("TryNormalizeLegacy() = %q, %v", value, ok)
	}
}

func TestResolveE164PriorizaElValorExplicito(t *testing.T) {
	explicit := "+549 11 2345-6789"
	value, err := ResolveE164("70011223", &explicit)
	if err != nil || value == nil || *value != "+5491123456789" {
		t.Fatalf("ResolveE164() = %v, %v", value, err)
	}
}

func TestDigitsForWhatsApp(t *testing.T) {
	if got := DigitsForWhatsApp("+59170011223"); got != "59170011223" {
		t.Fatalf("DigitsForWhatsApp() = %q", got)
	}
}
