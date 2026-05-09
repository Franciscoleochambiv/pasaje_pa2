package repository

import (
	"strings"
	"testing"
)

func TestGenerateReservationCode(t *testing.T) {
	code := GenerateReservationCode()
	if len(code) != 8 {
		t.Errorf("reservation code length = %d, want 8", len(code))
	}
	for _, c := range code {
		if !strings.ContainsRune(alphanumericChars, c) {
			t.Errorf("invalid char %c in reservation code %s", c, code)
		}
	}
}

func TestGenerateParcelCode(t *testing.T) {
	code := GenerateParcelCode()
	if !strings.HasPrefix(code, "ENC-") {
		t.Errorf("parcel code %q should start with 'ENC-'", code)
	}
	if len(code) != 12 { // "ENC-" + 8 chars
		t.Errorf("parcel code length = %d, want 12", len(code))
	}
	suffix := code[4:]
	for _, c := range suffix {
		if !strings.ContainsRune(alphanumericChars, c) {
			t.Errorf("invalid char %c in parcel code suffix %s", c, suffix)
		}
	}
}

func TestCodeUniqueness(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code := GenerateParcelCode()
		if codes[code] {
			t.Errorf("duplicate code generated: %s after %d iterations", code, i)
		}
		codes[code] = true
	}
}

func TestRandomAlphanumeric(t *testing.T) {
	for _, length := range []int{1, 4, 8, 16, 32} {
		result := randomAlphanumeric(length)
		if len(result) != length {
			t.Errorf("randomAlphanumeric(%d) returned length %d", length, len(result))
		}
	}
}
