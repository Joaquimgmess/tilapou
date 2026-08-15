package httpx_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Joaquimgmess/catalog/internal/platform/httpx"
)

type echoBody struct {
	Name string `json:"name"`
}

type echoInput struct {
	Body echoBody
}

type echoOutput struct {
	Body echoBody
}

func TestMain(m *testing.M) {
	httpx.SetErrorDocsPrefix(errorDocs)
	m.Run()
}

func newServer(t *testing.T, ready func(ctx context.Context) error) http.Handler {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	router, api := httpx.NewAPI(logger, httpx.Options{
		Title:          "test",
		Version:        "1.0.0",
		APIPrefix:      "/v1",
		RequestTimeout: time.Second,
	})
	httpx.RegisterHealth(router, ready)

	huma.Register(api, huma.Operation{
		OperationID: "echo",
		Method:      http.MethodPost,
		Path:        "/echo",
	}, func(_ context.Context, in *echoInput) (*echoOutput, error) {
		out := &echoOutput{}
		out.Body.Name = in.Body.Name
		return out, nil
	})

	return router
}

func do(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

const errorDocs = "https://errors.test/"

const echoPayload = `{"name":"a"}`

func jsonPost(target string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, target, strings.NewReader(echoPayload))
	req.Header.Set("Content-Type", "application/json")

	return req
}

func TestGroupPrefixesRoutes(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })

	if got := do(t, handler, jsonPost("/v1/echo")).Code; got != http.StatusOK {
		t.Errorf("POST /v1/echo = %d, want %d", got, http.StatusOK)
	}
	if got := do(t, handler, jsonPost("/echo")).Code; got != http.StatusNotFound {
		t.Errorf("POST /echo = %d, want %d", got, http.StatusNotFound)
	}
}

func TestRejectsNonJSONContentType(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/echo", strings.NewReader("name=a"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if got := do(t, handler, req).Code; got != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want %d", got, http.StatusUnsupportedMediaType)
	}
}

func TestRejectsUnknownQueryParameters(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })

	if got := do(t, handler, jsonPost("/v1/echo?bogus=1")).Code; got != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", got, http.StatusUnprocessableEntity)
	}
}

func TestErrorCarriesTypeAndInstance(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	rec := do(t, handler, jsonPost("/v1/echo?bogus=1"))

	var model huma.ErrorModel
	if err := json.Unmarshal(rec.Body.Bytes(), &model); err != nil {
		t.Fatalf("decoding error body: %v", err)
	}
	if model.Type != errorDocs+"422" {
		t.Errorf("type = %q, want %q", model.Type, errorDocs+"422")
	}
	if model.Instance == "" {
		t.Error("instance is empty, want the request id")
	}
}

func TestReadyzReportsDependencyFailure(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return context.DeadlineExceeded })

	if got := do(t, handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", http.NoBody)).Code; got != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if got := do(t, handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", http.NoBody)).Code; got != http.StatusOK {
		t.Errorf("healthz = %d, want %d", got, http.StatusOK)
	}
}
