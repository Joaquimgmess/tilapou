package product_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"

	"github.com/Joaquimgmess/catalog/internal/product"
)

type fakeStore struct {
	items map[uuid.UUID]product.Product
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: make(map[uuid.UUID]product.Product)}
}

func (f *fakeStore) Insert(_ context.Context, p product.Product) error {
	f.items[p.ID] = p
	return nil
}

func (f *fakeStore) ByID(_ context.Context, id uuid.UUID) (product.Product, error) {
	p, ok := f.items[id]
	if !ok {
		return product.Product{}, product.ErrNotFound
	}
	return p, nil
}

var fixedNow = time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC)

func newTestAPI(t *testing.T) (humatest.TestAPI, *fakeStore) {
	t.Helper()

	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	store := newFakeStore()
	product.RegisterRoutes(api, store, func() time.Time { return fixedNow })

	return api, store
}

func TestCreateProduct(t *testing.T) {
	t.Parallel()

	api, store := newTestAPI(t)

	resp := api.Post("/products", map[string]any{"name": "Coffee", "price_cents": 1500})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusCreated, resp.Body)
	}
	if len(store.items) != 1 {
		t.Fatalf("store has %d items, want 1", len(store.items))
	}
	for _, p := range store.items {
		if !p.CreatedAt.Equal(fixedNow) {
			t.Errorf("CreatedAt = %v, want %v", p.CreatedAt, fixedNow)
		}
	}
}

func TestCreateProductInvalidPrice(t *testing.T) {
	t.Parallel()

	api, _ := newTestAPI(t)

	resp := api.Post("/products", map[string]any{"name": "Coffee", "price_cents": 0})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body)
	}
}

func TestGetProductNotFound(t *testing.T) {
	t.Parallel()

	api, _ := newTestAPI(t)

	resp := api.Get("/products/" + uuid.New().String())
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusNotFound, resp.Body)
	}
}
