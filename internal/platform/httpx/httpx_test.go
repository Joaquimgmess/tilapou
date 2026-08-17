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

	"github.com/Joaquimgmess/tilapou/internal/platform/httpx"
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

func newServer(t *testing.T, ready func(ctx context.Context) error) http.Handler {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)
	router, api := httpx.NewAPI(logger, "test", "1.0.0", "/v1", time.Second,
		httpx.WithErrorDocs(errorDocs))
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

	huma.Register(api, huma.Operation{
		OperationID: "ping",
		Method:      http.MethodPost,
		Path:        "/ping",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
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

func TestOMiddlewareRecusaOContentTypeMesmoSemBodyDeclarado(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/ping", strings.NewReader("x=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if got := do(t, handler, req).Code; got != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want %d: quem recusa aqui e o middleware, nao o huma", got, http.StatusUnsupportedMediaType)
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

func problemFrom(t *testing.T, rec *httptest.ResponseRecorder) huma.ErrorModel {
	t.Helper()

	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content type = %q, want application/problem+json", ct)
	}

	var model huma.ErrorModel
	if err := json.Unmarshal(rec.Body.Bytes(), &model); err != nil {
		t.Fatalf("decoding problem body: %v", err)
	}
	if model.Instance == "" {
		t.Error("instance is empty, want the request id")
	}

	return model
}

func TestUnmatchedRouteReturnsProblem(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	rec := do(t, handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/nope", http.NoBody))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	problemFrom(t, rec)
}

func TestWrongMethodReturnsProblem(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	rec := do(t, handler, httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/echo", http.NoBody))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	problemFrom(t, rec)
}

func TestUnsupportedMediaTypeReturnsProblem(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/echo", strings.NewReader("name=a"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := do(t, handler, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
	problemFrom(t, rec)
}

func TestPanicReturnsProblemAndIsLogged(t *testing.T) {
	t.Parallel()

	var logged strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logged, nil))
	router, api := httpx.NewAPI(logger, "t", "1.0.0", "/v1", time.Second, httpx.WithErrorDocs(errorDocs))
	huma.Register(api, huma.Operation{OperationID: "boom", Method: http.MethodGet, Path: "/boom"},
		func(context.Context, *struct{}) (*struct{}, error) { panic("boom") })

	rec := do(t, router, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/boom", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	model := problemFrom(t, rec)
	if strings.Contains(model.Detail, "boom") {
		t.Errorf("detail %q leaks the panic value", model.Detail)
	}

	out := logged.String()
	if !strings.Contains(out, "panic recovered") || !strings.Contains(out, "request_id") {
		t.Errorf("panic log missing structure: %s", out)
	}
	if !strings.Contains(out, `"msg":"http request"`) {
		t.Errorf("request log line missing for panic: %s", out)
	}
}

func TestMetricsExposeRequestHistogram(t *testing.T) {
	t.Parallel()

	handler := newServer(t, func(context.Context) error { return nil })
	do(t, handler, jsonPost("/v1/echo"))

	rec := do(t, handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `http_request_duration_seconds_count{method="POST",route="/v1/echo",status="200"} 1`) {
		t.Errorf("metrics missing the echo series:\n%s", body)
	}
}

func TestHandlerErrorsCarryTypeAndInstance(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	router, api := httpx.NewAPI(logger, "t", "1.0.0", "/v1", time.Second,
		httpx.WithErrorDocs(errorDocs))
	huma.Register(api, huma.Operation{OperationID: "missing", Method: http.MethodGet, Path: "/missing"},
		func(context.Context, *struct{}) (*struct{}, error) {
			return nil, huma.Error404NotFound("thing not found")
		})

	rec := do(t, router, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/missing", http.NoBody))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var model huma.ErrorModel
	if err := json.Unmarshal(rec.Body.Bytes(), &model); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if model.Type != errorDocs+"404" {
		t.Errorf("type = %q, want %q", model.Type, errorDocs+"404")
	}
	if model.Instance == "" {
		t.Error("instance is empty, want the request id")
	}
}
