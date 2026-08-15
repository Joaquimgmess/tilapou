package product

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type Response struct {
	ID         uuid.UUID `json:"id" doc:"Product identifier"`
	Name       string    `json:"name" doc:"Product name"`
	PriceCents int64     `json:"price_cents" doc:"Price in cents"`
	CreatedAt  time.Time `json:"created_at" doc:"Creation timestamp"`
}

type CreateInput struct {
	Body struct {
		Name       string `json:"name" maxLength:"120" doc:"Product name"`
		PriceCents int64  `json:"price_cents" minimum:"1" doc:"Price in cents"`
	}
}

type CreateOutput struct {
	Body Response
}

type GetInput struct {
	ID uuid.UUID `path:"id" doc:"Product identifier"`
}

type GetOutput struct {
	Body Response
}

func RegisterRoutes(api huma.API, store Store) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-product",
		Method:        http.MethodPost,
		Path:          "/products",
		Summary:       "Create a product",
		Tags:          []string{"products"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateInput) (*CreateOutput, error) {
		p, err := Create(ctx, store, CreateCommand{Name: in.Body.Name, PriceCents: in.Body.PriceCents})
		if err != nil {
			return nil, toHTTPError(err)
		}
		return &CreateOutput{Body: toResponse(p)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-product",
		Method:      http.MethodGet,
		Path:        "/products/{id}",
		Summary:     "Get a product by id",
		Tags:        []string{"products"},
	}, func(ctx context.Context, in *GetInput) (*GetOutput, error) {
		p, err := Get(ctx, store, in.ID)
		if err != nil {
			return nil, toHTTPError(err)
		}
		return &GetOutput{Body: toResponse(p)}, nil
	})
}

func toResponse(p Product) Response {
	return Response{ID: p.ID, Name: p.Name, PriceCents: p.PriceCents, CreatedAt: p.CreatedAt}
}

func toHTTPError(err error) error {
	var invalid InvalidError
	switch {
	case errors.Is(err, ErrNotFound):
		return huma.Error404NotFound("product not found")
	case errors.As(err, &invalid):
		return huma.Error422UnprocessableEntity(invalid.Reason)
	default:
		return err
	}
}
