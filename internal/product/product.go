package product

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID         uuid.UUID
	Name       string
	PriceCents int64
	CreatedAt  time.Time
}

const MaxNameLen = 120

func New(id uuid.UUID, name string, priceCents int64, createdAt time.Time) (Product, error) {
	name = strings.TrimSpace(name)

	switch {
	case id == uuid.Nil:
		return Product{}, invalidf("id is required")
	case name == "":
		return Product{}, invalidf("name is required")
	case len(name) > MaxNameLen:
		return Product{}, invalidf("name exceeds %d characters", MaxNameLen)
	case priceCents <= 0:
		return Product{}, invalidf("price must be greater than zero")
	}

	return Product{
		ID:         id,
		Name:       name,
		PriceCents: priceCents,
		CreatedAt:  createdAt.UTC(),
	}, nil
}
