package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Joaquimgmess/catalog/internal/platform/logging"
)

const problemContentType = "application/problem+json"

type problems struct {
	docsPrefix string
}

func (p problems) decorate(model *huma.ErrorModel, status int, reqID string) {
	if p.docsPrefix != "" {
		model.Type = p.docsPrefix + strconv.Itoa(status)
	}
	model.Instance = reqID
}

func (p problems) transformer(ctx huma.Context, _ string, v any) (any, error) {
	model, ok := v.(*huma.ErrorModel)
	if !ok {
		return v, nil
	}

	status := model.Status
	if status == 0 {
		status = ctx.Status()
	}
	p.decorate(model, status, middleware.GetReqID(ctx.Context()))

	return model, nil
}

func (p problems) write(w http.ResponseWriter, r *http.Request, status int, detail string) {
	model := &huma.ErrorModel{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
	p.decorate(model, status, middleware.GetReqID(r.Context()))

	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(model); err != nil {
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "writing problem response", slog.Any("error", err))
	}
}

func (p problems) notFound(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, http.StatusNotFound, "the requested resource does not exist")
}

func (p problems) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, http.StatusMethodNotAllowed, "method not allowed for this resource")
}

func (p problems) allowContentType(allowed string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength == 0 || mediaType(r.Header.Get("Content-Type")) == allowed {
				next.ServeHTTP(w, r)
				return
			}

			p.write(w, r, http.StatusUnsupportedMediaType, "content type must be "+allowed)
		})
	}
}

func (p problems) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			logging.FromContext(ctx).ErrorContext(ctx, "panic recovered",
				slog.Any("panic", rec),
				slog.String("stack", string(debug.Stack())),
			)
			p.write(w, r, http.StatusInternalServerError, "unexpected error occurred")
		}()

		next.ServeHTTP(w, r)
	})
}

func mediaType(header string) string {
	if i := strings.IndexByte(header, ';'); i >= 0 {
		header = header[:i]
	}
	return strings.ToLower(strings.TrimSpace(header))
}
