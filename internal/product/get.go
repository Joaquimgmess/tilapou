package product

import (
	"context"

	"github.com/google/uuid"
)

func Get(ctx context.Context, store Store, id uuid.UUID) (Product, error) {
	return store.ByID(ctx, id)
}
