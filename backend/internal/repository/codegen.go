package repository

import (
	"crypto/rand"
	"math/big"
)

const alphanumericChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomAlphanumeric genera una cadena aleatoria alfanumérica en mayúsculas.
func randomAlphanumeric(n int) string {
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumericChars))))
		b[i] = alphanumericChars[idx.Int64()]
	}
	return string(b)
}

// GenerateReservationCode genera un código de reserva de 8 caracteres.
func GenerateReservationCode() string {
	return randomAlphanumeric(8)
}
