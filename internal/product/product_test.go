package product_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Joaquimgmess/catalog/internal/product"
)

func TestNewEnforcesInvariants(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := map[string]struct {
		id         uuid.UUID
		name       string
		priceCents int64
		wantErr    bool
	}{
		"valid":          {uuid.New(), "Coffee", 1500, false},
		"nil id":         {uuid.Nil, "Coffee", 1500, true},
		"blank name":     {uuid.New(), "   ", 1500, true},
		"name too long":  {uuid.New(), strings.Repeat("a", product.MaxNameLen+1), 1500, true},
		"zero price":     {uuid.New(), "Coffee", 0, true},
		"negative price": {uuid.New(), "Coffee", -1, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := product.New(tt.id, tt.name, tt.priceCents, now)
			if tt.wantErr {
				var invalid product.InvalidError
				if !errors.As(err, &invalid) {
					t.Fatalf("New() error = %v, want product.InvalidError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("New() unexpected error = %v", err)
			}
		})
	}
}

func TestNewNormalizesNameAndTime(t *testing.T) {
	t.Parallel()

	p, err := product.New(uuid.New(), "  Coffee  ", 1500, time.Now().In(time.FixedZone("BRT", -3*60*60)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.Name != "Coffee" {
		t.Errorf("Name = %q, want %q", p.Name, "Coffee")
	}
	if p.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt in %v, want UTC", p.CreatedAt.Location())
	}
}
