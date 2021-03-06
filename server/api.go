package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"

	"github.com/philips-labs/garo/rpc/garo"
)

var (
	//ContentTypePlainText holds HTTP Content-Type text/plain
	ContentTypePlainText = "text/plain; charset=utf-8"
)

func configureAPI(twirpServer garo.TwirpServer, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(zapLogger(logger))
	r.Use(middleware.RedirectSlashes)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypePlainText)
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", ContentTypePlainText)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong\n"))
	})
	r.Mount(twirpServer.PathPrefix(), twirpServer)

	logRoutes(r, logger)

	return r
}

func zapLogger(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				l.Info("Served",
					zap.String("proto", r.Proto),
					zap.String("path", r.URL.Path),
					zap.Duration("lat", time.Since(t1)),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.String("reqId", middleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func logRoutes(r *chi.Mux, logger *zap.Logger) {
	if err := chi.Walk(r, zapPrintRoute(logger)); err != nil {
		logger.Error("Failed to walk routes:", zap.Error(err))
	}
}

func zapPrintRoute(logger *zap.Logger) chi.WalkFunc {
	return func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		route = strings.Replace(route, "/*/", "/", -1)
		logger.Debug("Registering route", zap.String("method", method), zap.String("route", route))
		return nil
	}
}
