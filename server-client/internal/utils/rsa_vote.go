package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateRSAForElection(bits int) (mStr string, eInt int, dStr string, err error) {
	if bits <= 0 {
		bits = 2048
	}

	one := big.NewInt(1)
	e := big.NewInt(65537)

	var p, q, n, phi *big.Int

	for {
		p, err = rand.Prime(rand.Reader, bits/2)
		if err != nil {
			return "", 0, "", err
		}
		q, err = rand.Prime(rand.Reader, bits/2)
		if err != nil {
			return "", 0, "", err
		}

		n = new(big.Int).Mul(p, q)

		pm1 := new(big.Int).Sub(p, one)
		qm1 := new(big.Int).Sub(q, one)
		phi = new(big.Int).Mul(pm1, qm1)

		g := new(big.Int).GCD(nil, nil, e, phi)
		if g.Cmp(one) == 0 {
			break
		}
	}

	d := new(big.Int).ModInverse(e, phi)
	if d == nil {
		return "", 0, "", err
	}

	return n.String(), int(e.Int64()), d.String(), nil
}

func RSADecodeCipher(cipherStr, mStr, dStr string) (string, error) {
	n := new(big.Int)
	if _, ok := n.SetString(mStr, 10); !ok {
		return "", ErrParseBigInt("m")
	}

	d := new(big.Int)
	if _, ok := d.SetString(dStr, 10); !ok {
		return "", ErrParseBigInt("d")
	}

	c := new(big.Int)
	if _, ok := c.SetString(cipherStr, 10); !ok {
		return "", ErrParseBigInt("cipher")
	}

	m := new(big.Int).Exp(c, d, n)
	return m.String(), nil
}

func BigIntProdMod(values []string, mStr string) (string, error) {
	n := new(big.Int)
	if _, ok := n.SetString(mStr, 10); !ok {
		return "", ErrParseBigInt("m")
	}

	prod := big.NewInt(1)

	tmp := new(big.Int)
	for _, s := range values {
		if _, ok := tmp.SetString(s, 10); !ok {
			return "", ErrParseBigInt("value")
		}
		prod.Mul(prod, tmp)
		prod.Mod(prod, n)
	}

	return prod.String(), nil
}

type parseBigIntError struct {
	field string
}

func (e parseBigIntError) Error() string {
	return "failed to parse big.Int from field: " + e.field
}

func ErrParseBigInt(field string) error {
	return parseBigIntError{field: field}
}
