package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
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

func ParseRSAPrivateKeyFromPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid PEM type: %s", block.Type)
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS1 private key: %w", err)
	}
	return priv, nil
}

func SignBWithPrivateKey(priv *rsa.PrivateKey, B string) (string, error) {
	h := sha256.Sum256([]byte(B))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}
