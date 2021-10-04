package infrastructure

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type UniqueIdGenerator interface {
	Generate() (string, error)
}

type UuidGenerator struct{}

func (g *UuidGenerator) Generate() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "failed to uuid.NewRandom")
	}

	return uid.String(), nil
}
