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

type Options struct {
	Title          string
	Version        string
	APIPrefix      string
	RequestTimeout time.Duration
	TrustedProxies int
	ErrorDocsURL   string
}

func NewAPI(logger *slog.Logger, opts Options) (chi.Router, *huma.Group) {
	metrics := NewMetrics()
	problem := problems{docsPrefix: opts.ErrorDocsURL}

	router := chi.NewRouter()
	router.NotFound(problem.notFound)
	router.MethodNotAllowed(problem.methodNotAllowed)
	router.Use(middleware.RequestID)
	router.Use(clientIP(opts.TrustedProxies))
	router.Use(observe(logger, metrics))
	router.Use(problem.recoverer)
	router.Use(middleware.Timeout(opts.RequestTimeout))
	router.Use(middleware.RequestSize(maxRequestBytes))
	router.Use(problem.allowContentType("application/json"))
	router.Use(middleware.Compress(compressionLevel))

	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	cfg.RejectUnknownQueryParameters = true
	cfg.Transformers = append(cfg.Transformers, problem.transformer)

	router.Get("/metrics", metrics.Handler())

	api := humachi.New(router, cfg)

	return router, huma.NewGroup(api, opts.APIPrefix)
}

func clientIP(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies <= 0 {
		return middleware.ClientIPFromRemoteAddr
	}
	return middleware.ClientIPFromXFFTrustedProxies(trustedProxies)
}

func observe(logger *slog.Logger, metrics *Metrics) func(http.Handler) http.Handler {
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

			metrics.observe(r.Method, route, ww.Status(), elapsed)

			reqLogger.InfoContext(ctx, "http request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", elapsed),
			)
		})
	}
}
