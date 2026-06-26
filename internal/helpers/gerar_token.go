package helpers

import (
	"crypto/rand"
	"encoding/hex"
)

// GerarToken gera um token hexadecimal aleatório de 32 caracteres
// utilizando criptografia segura (crypto/rand).
func GerarToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
