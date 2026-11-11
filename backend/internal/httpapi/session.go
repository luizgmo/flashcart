package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"flashcart/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errPagination = &validationError{}

type validationError struct{}

func (*validationError) Error() string { return "invalid pagination" }

func createSession(db *pgxpool.Pool, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validOrigin(r, cfg.AppOrigin) {
			writeError(w, 403, "ORIGIN_FORBIDDEN", "Origem não permitida.")
			return
		}
		if ct := r.Header.Get("Content-Type"); !isJSONContentType(ct) {
			writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Envie application/json.")
			return
		}
		if !decodeEmptyJSON(w, r) {
			return
		}
		if cookie, err := r.Cookie("flashcart_session"); err == nil && cookie.Value != "" {
			hash := sha256.Sum256([]byte(cookie.Value))
			var expires time.Time
			err = db.QueryRow(r.Context(), `SELECT expires_at FROM sessions WHERE token_hash=$1 AND expires_at>clock_timestamp()`, hash[:]).Scan(&expires)
			if err == nil {
				writeJSON(w, 200, map[string]any{"expires_at": expires, "server_time": time.Now().UTC()})
				return
			}
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível iniciar a sessão.")
			return
		}
		token := base64.RawURLEncoding.EncodeToString(raw)
		hash := sha256.Sum256([]byte(token))
		tx, err := db.Begin(r.Context())
		if err != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		defer tx.Rollback(r.Context())
		hourLimit, dayLimit := 100, 500
		if cfg.RateLimitMode == "load" {
			hourLimit, dayLimit = 10000, 100000
		}
		allowed, err := consumeQuota(r.Context(), tx, "global", "sessions", "create", "hour", hourLimit)
		if err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível iniciar a sessão.")
			return
		}
		if allowed {
			allowed, err = consumeQuota(r.Context(), tx, "global", "sessions", "create", "day", dayLimit)
		}
		if err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível iniciar a sessão.")
			return
		}
		if !allowed {
			writeError(w, 429, "RATE_LIMITED", "Limite de sessões atingido.")
			return
		}
		var expires time.Time
		err = tx.QueryRow(r.Context(), `INSERT INTO sessions(token_hash,expires_at) VALUES($1,clock_timestamp()+interval '24 hours') RETURNING expires_at`, hash[:]).Scan(&expires)
		if err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível iniciar a sessão.")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		secure := cfg.AppEnv == "production" || strings.HasPrefix(cfg.AppOrigin, "https://")
		http.SetCookie(w, &http.Cookie{Name: "flashcart_session", Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires})
		writeJSON(w, 201, map[string]any{"expires_at": expires, "server_time": time.Now().UTC()})
	}
}

func validOrigin(r *http.Request, expected string) bool {
	return r.Header.Get("Origin") == expected && expected != ""
}
