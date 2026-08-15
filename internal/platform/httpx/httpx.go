package httpx

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Joaquimgmess/catalog/internal/platform/logging"
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
}

func SetErrorDocsPrefix(prefix string) {
	huma.NewErrorWithContext = errorWithInstance(prefix)
}

func NewAPI(logger *slog.Logger, opts Options) (chi.Router, *huma.Group) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(clientIP(opts.TrustedProxies))
	router.Use(middleware.Recoverer)
	router.Use(requestLogger(logger))
	router.Use(middleware.Timeout(opts.RequestTimeout))
	router.Use(middleware.RequestSize(maxRequestBytes))
	router.Use(middleware.AllowContentType("application/json"))
	router.Use(middleware.Compress(compressionLevel))

	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	cfg.RejectUnknownQueryParameters = true

	api := humachi.New(router, cfg)

	return router, huma.NewGroup(api, opts.APIPrefix)
}

func clientIP(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies <= 0 {
		return middleware.ClientIPFromRemoteAddr
	}
	return middleware.ClientIPFromXFFTrustedProxies(trustedProxies)
}

func errorWithInstance(docsPrefix string) func(huma.Context, int, string, ...error) huma.StatusError {
	return func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		err := huma.NewError(status, msg, errs...)

		model, ok := err.(*huma.ErrorModel)
		if !ok {
			return err
		}
		if docsPrefix != "" {
			model.Type = docsPrefix + strconv.Itoa(status)
		}
		if ctx != nil {
			model.Instance = middleware.GetReqID(ctx.Context())
		}

		return model
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
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

			reqLogger.InfoContext(ctx, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
