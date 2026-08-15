package product

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type fakeStore struct {
	items map[uuid.UUID]Product
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: make(map[uuid.UUID]Product)}
}

func (f *fakeStore) Insert(_ context.Context, p Product) error {
	f.items[p.ID] = p
	return nil
}

func (f *fakeStore) ByID(_ context.Context, id uuid.UUID) (Product, error) {
	p, ok := f.items[id]
	if !ok {
		return Product{}, ErrNotFound
	}
	return p, nil
}

func newTestAPI(t *testing.T) (humatest.TestAPI, *fakeStore) {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	store := newFakeStore()
	RegisterRoutes(api, store)
	return api, store
}

func TestCreateProduct(t *testing.T) {
	api, store := newTestAPI(t)

	resp := api.Post("/products", map[string]any{"name": "Coffee", "price_cents": 1500})
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusCreated, resp.Body)
	}
	if len(store.items) != 1 {
		t.Fatalf("store has %d items, want 1", len(store.items))
	}
}

func TestCreateProductInvalidPrice(t *testing.T) {
	api, _ := newTestAPI(t)

	resp := api.Post("/products", map[string]any{"name": "Coffee", "price_cents": 0})
	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body)
	}
}

func TestGetProductNotFound(t *testing.T) {
	api, _ := newTestAPI(t)

	resp := api.Get("/products/" + uuid.New().String())
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusNotFound, resp.Body)
	}
}
