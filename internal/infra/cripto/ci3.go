package cripto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

var saltNulo = make([]byte, sha512Tamanho)

func CI3Encrypt(plaintext interface{}, chaveMestra []byte) (string, error) {
	jsonBytes, err := json.Marshal(plaintext)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar JSON: %w", err)
	}

	dado := string(jsonBytes)

	chaveCripto := HKDF(chaveMestra, saltNulo, "encryption", 16)
	chaveHmac := HKDF(chaveMestra, saltNulo, "authentication", sha512Tamanho)

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("erro ao gerar IV: %w", err)
	}

	bloco, err := aes.NewCipher(chaveCripto)
	if err != nil {
		return "", fmt.Errorf("erro ao criar cifra AES: %w", err)
	}

	padded := pkcs7Padding([]byte(dado), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(bloco, iv)
	mode.CryptBlocks(ciphertext, padded)

	dadoB64 := base64.StdEncoding.EncodeToString(append(iv, ciphertext...))

	mac := hmac.New(sha512.New, chaveHmac)
	mac.Write([]byte(dadoB64))
	hmacHex := hex.EncodeToString(mac.Sum(nil))

	resultadoB64 := base64.StdEncoding.EncodeToString([]byte(hmacHex + dadoB64))

	return resultadoB64, nil
}

func pkcs7Padding(dado []byte, tamanhoBloco int) []byte {
	pad := tamanhoBloco - len(dado)%tamanhoBloco
	bytesPad := make([]byte, pad)
	for i := range bytesPad {
		bytesPad[i] = byte(pad)
	}
	return append(dado, bytesPad...)
}
