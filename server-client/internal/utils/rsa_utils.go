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
)

func GenerateRSAKeyPairPEM(bits int) (publicPEM, privatePEM string, err error) {
	if bits <= 0 {
		bits = 2048
	}

	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate rsa key: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privPEM := pem.EncodeToMemory(privBlock)

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM := pem.EncodeToMemory(pubBlock)

	return string(pubPEM), string(privPEM), nil
}

func VerifyRSASignatureFromPEM(pubPEM, data, signatureBase64 string) (bool, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return false, fmt.Errorf("failed to parse public key PEM")
	}

	pubIfc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	pub, ok := pubIfc.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not RSA public key")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("bad base64 signature: %w", err)
	}

	h := sha256.Sum256([]byte(data))

	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sigBytes); err != nil {
		return false, nil
	}

	return true, nil
}
