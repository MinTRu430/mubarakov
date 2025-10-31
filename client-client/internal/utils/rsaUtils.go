package utils

import (
	"crypto/rsa"
	"crypto/sha256"
	"math/big"
)

func VerifyRSASignature(A, g, p, h string, e int, nStr string) bool {
	n := new(big.Int)
	n.SetString(nStr, 10)

	pub := &rsa.PublicKey{
		N: n,
		E: e,
	}

	data := []byte(A + g + p)
	hash := sha256.Sum256(data)

	sigBytes := new(big.Int)
	sigBytes.SetString(h, 16)

	err := rsa.VerifyPKCS1v15(pub, 0, hash[:], sigBytes.Bytes())
	return err == nil
}
