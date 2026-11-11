package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flashcart/backend/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func contextWithTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), time.Second)
}

type requestIDContextKey struct{}

func New(db *pgxpool.Pool, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := contextWithTimeout(r)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			writeError(w, 503, "NOT_READY", "Serviço temporariamente indisponível.")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"currency": "BRL", "reservation_ttl_seconds": cfg.ReservationTTLSeconds, "assistant_mode": cfg.AssistantMode, "demo": true})
	})
	mux.HandleFunc("GET /api/v1/products", listProducts(db))
	mux.HandleFunc("GET /api/v1/products/{id}", getProduct(db))
	mux.HandleFunc("POST /api/v1/session", createSession(db, cfg))
	mux.HandleFunc("POST /api/v1/orders", createOrder(db, cfg))
	mux.HandleFunc("GET /api/v1/orders", listOrders(db))
	mux.HandleFunc("GET /api/v1/orders/{id}", getOrder(db))
	mux.HandleFunc("POST /api/v1/orders/{id}/simulate-payment", simulatePayment(db, cfg))
	mux.HandleFunc("POST /api/v1/orders/{id}/assistant", assistant(db, cfg))
	static := func(w http.ResponseWriter, r *http.Request) {
		root := filepath.Clean(cfg.FrontendDist)
		relative := filepath.Clean(r.PathValue("path"))
		if relative == "." || relative == "/" {
			relative = "index.html"
		}
		candidate := filepath.Join(root, relative)
		if !strings.HasPrefix(candidate, root+string(os.PathSeparator)) {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(candidate); err != nil {
			if filepath.Ext(relative) != "" {
				http.NotFound(w, r)
				return
			}
			candidate = filepath.Join(root, "index.html")
		}
		if filepath.Base(candidate) == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeFile(w, r, candidate)
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(cfg.FrontendDist, "index.html"))
	})
	mux.HandleFunc("GET /{path...}", static)
	return requestLogger(mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func isJSONContentType(value string) bool {
	media, _, err := mime.ParseMediaType(value)
	return err == nil && media == "application/json"
}

func decodeEmptyJSON(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var body map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil || body == nil || len(body) != 0 {
		writeError(w, 400, "INVALID_REQUEST", "O corpo deve ser um objeto vazio.")
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(w, 400, "INVALID_REQUEST", "Envie apenas um objeto JSON.")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	problem := map[string]string{"code": code, "message": message}
	if request, ok := w.(interface{ Request() *http.Request }); ok {
		if id, ok := request.Request().Context().Value(requestIDContextKey{}).(string); ok {
			problem["request_id"] = id
		}
	}
	writeJSON(w, status, map[string]any{"error": problem})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.URL.Path == "/api/v1/session" || r.URL.Path == "/api/v1/orders" || len(r.URL.Path) >= 15 && r.URL.Path[:15] == "/api/v1/orders/" || r.URL.Path == "/api/v1/config" {
			w.Header().Set("Cache-Control", "no-store")
		}
		started := time.Now()
		r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID))
		recorder := &statusRecorder{ResponseWriter: w, status: 200, request: r}
		next.ServeHTTP(recorder, r)
		pattern := r.Pattern
		if pattern == "" {
			pattern = "unmatched"
		}
		slog.Info("http request", "request_id", requestID, "method", r.Method, "route", pattern, "status", recorder.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	request *http.Request
}

func (w *statusRecorder) Request() *http.Request { return w.request }

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusRecorder) Write(b []byte) (int, error) { return w.ResponseWriter.Write(b) }
