package cripto

import (
	"crypto/hmac"
	"crypto/sha512"
	"hash"
)

const (
	sha512Tamanho = 64
)

func hkdfExtrair(pseudoChave []byte, salt []byte, h func() hash.Hash) []byte {
	if len(salt) == 0 {
		salt = make([]byte, h().Size())
	}
	mac := hmac.New(h, salt)
	mac.Write(pseudoChave)
	return mac.Sum(nil)
}

func hkdfExpandir(prk []byte, info string, comprimento int, h func() hash.Hash) []byte {
	tamanhoHash := h().Size()
	if comprimento > 255*tamanhoHash {
		panic("hkdf: comprimento requerido muito grande")
	}

	blocos := make([]byte, 0, comprimento)
	t := make([]byte, 0, tamanhoHash)

	for len(blocos) < comprimento {
		mac := hmac.New(h, prk)
		mac.Write(t)
		mac.Write([]byte(info))
		mac.Write([]byte{byte(len(blocos)/tamanhoHash + 1)})
		t = mac.Sum(nil)
		blocos = append(blocos, t...)
	}

	return blocos[:comprimento]
}

func HKDF(chaveMestra []byte, salt []byte, info string, comprimento int) []byte {
	h := sha512.New
	prk := hkdfExtrair(chaveMestra, salt, h)
	return hkdfExpandir(prk, info, comprimento, h)
}
