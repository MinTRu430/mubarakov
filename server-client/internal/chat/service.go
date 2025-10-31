package chat

import (
	"context"
	"math/big"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ComputeSharedKey(ctx context.Context, login, B string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	params, err := s.repo.GetDHParams(ctx, login)
	if err != nil {
		return "", err
	}

	Bnum := new(big.Int)
	Bnum.SetString(B, 10)

	P := new(big.Int)
	P.SetString(params.P, 10)

	A := new(big.Int)
	A.SetString(params.A, 10)

	an := new(big.Int)
	an.SetString(params.APriv, 10)

	K := new(big.Int).Exp(Bnum, an, P)

	return K.String(), nil
}
