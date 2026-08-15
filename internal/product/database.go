package product

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Insert(ctx context.Context, p Product) error
	ByID(ctx context.Context, id uuid.UUID) (Product, error)
}

type DB struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewDB(pool *pgxpool.Pool, timeout time.Duration) *DB {
	return &DB{pool: pool, timeout: timeout}
}

func (d *DB) Insert(ctx context.Context, p Product) error {
	const query = `
		INSERT INTO products (id, name, price_cents, created_at)
		VALUES ($1, $2, $3, $4)`

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	if _, err := d.pool.Exec(ctx, query, p.ID, p.Name, p.PriceCents, p.CreatedAt); err != nil {
		return fmt.Errorf("insert product %s: %w", p.ID, err)
	}
	return nil
}

func (d *DB) ByID(ctx context.Context, id uuid.UUID) (Product, error) {
	const query = `
		SELECT id, name, price_cents, created_at
		FROM products
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var p Product
	err := d.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.PriceCents, &p.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Product{}, ErrNotFound
	case err != nil:
		return Product{}, fmt.Errorf("select product %s: %w", id, err)
	}

	return p, nil
}
