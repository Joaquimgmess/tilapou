package product

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/catalog/internal/platform/logging"
)

type CreateCommand struct {
	Name       string
	PriceCents int64
}

func Create(ctx context.Context, store Store, clock func() time.Time, cmd CreateCommand) (Product, error) {
	p, err := New(uuid.New(), cmd.Name, cmd.PriceCents, clock())
	if err != nil {
		return Product{}, err
	}

	if err := store.Insert(ctx, p); err != nil {
		return Product{}, err
	}

	logging.FromContext(ctx).InfoContext(ctx, "product created", slog.String("product_id", p.ID.String()))
	return p, nil
}
