// Package httpx builds the daemon HTTP router.
package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Joaquimgmess/tilapou/internal/platform/logging"
)

const (
	maxRequestBytes  = 1 << 20
	compressionLevel = 5
)

type options struct {
	trustedProxies int
	errorDocsURL   string
}

// Option tunes what NewAPI leaves optional.
type Option func(*options)

// WithTrustedProxies takes the client IP from X-Forwarded-For, skipping n
// trusted hops. Without it the IP comes from the connection.
func WithTrustedProxies(n int) Option {
	return func(o *options) { o.trustedProxies = n }
}

// WithErrorDocs prefixes the problem+json type with url, linking each error to
// its documentation.
func WithErrorDocs(url string) Option {
	return func(o *options) { o.errorDocsURL = url }
}

// NewAPI returns the chi router, with /metrics already registered, and the huma
// group mounted under prefix. Every request is cut off at requestTimeout.
func NewAPI(
	logger *slog.Logger,
	title, version, prefix string,
	requestTimeout time.Duration,
	opts ...Option,
) (chi.Router, *huma.Group) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	metrics := newMetrics()
	problem := problems{docsPrefix: o.errorDocsURL}

	router := chi.NewRouter()
	router.NotFound(problem.notFound)
	router.MethodNotAllowed(problem.methodNotAllowed)
	router.Use(middleware.RequestID)
	router.Use(clientIP(o.trustedProxies))
	router.Use(observe(logger, metrics))
	router.Use(problem.recoverer)
	router.Use(middleware.Timeout(requestTimeout))
	router.Use(middleware.RequestSize(maxRequestBytes))
	router.Use(problem.allowContentType("application/json"))
	router.Use(middleware.Compress(compressionLevel))

	cfg := huma.DefaultConfig(title, version)
	cfg.RejectUnknownQueryParameters = true
	cfg.Transformers = append(cfg.Transformers, problem.transformer)

	router.Get("/metrics", metrics.Handler())

	api := humachi.New(router, cfg)

	return router, huma.NewGroup(api, prefix)
}

func clientIP(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies <= 0 {
		return middleware.ClientIPFromRemoteAddr
	}
	return middleware.ClientIPFromXFFTrustedProxies(trustedProxies)
}

func observe(logger *slog.Logger, m *metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqLogger := logger.With(
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("client_ip", middleware.GetClientIP(r.Context())),
			)
			ctx := logging.WithLogger(r.Context(), reqLogger)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r.WithContext(ctx))

			elapsed := time.Since(start)
			route := routePattern(r)

			m.observe(r.Method, route, ww.Status(), elapsed)

			reqLogger.InfoContext(ctx, "http request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int64("duration_ms", elapsed.Milliseconds()),
			)
		})
	}
}
