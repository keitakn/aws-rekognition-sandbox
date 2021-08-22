package infrastructure

import (
	"github.com/google/uuid"
)

type UniqueIdGenerator interface {
	Generate() (string, error)
}

type UuidGenerator struct{}

func (g *UuidGenerator) Generate() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return uid.String(), nil
}
