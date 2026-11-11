package jobs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventEnvelope struct {
	EventID       uuid.UUID `json:"event_id"`
	SchemaVersion int       `json:"schema_version"`
	Type          string    `json:"type"`
	OrderID       uuid.UUID `json:"order_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type claimedEvent struct {
	id, orderID uuid.UUID
	eventType   string
	payload     []byte
	token       uuid.UUID
	attempts    int
}

func PublishOutbox(ctx context.Context, db *pgxpool.Pool, endpoint, project, topic, emulator string, limit int) (int, error) {
	if limit < 1 || limit > 500 {
		return 0, fmt.Errorf("limit must be between 1 and 500")
	}
	if endpoint == "" {
		if emulator != "" {
			endpoint = "http://" + emulator
		} else {
			endpoint = "https://pubsub.googleapis.com"
		}
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if project == "" || topic == "" {
		return 0, fmt.Errorf("project and topic are required")
	}
	events, err := claimEvents(ctx, db, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	client := &http.Client{Timeout: 10 * time.Second}
	for _, event := range events {
		if err := publishOne(ctx, client, endpoint, project, topic, emulator, event); err != nil {
			slog.Warn("outbox publish failed", "event_id", event.id, "attempt", event.attempts+1, "error", err)
			if markErr := releaseEvent(ctx, db, event, err); markErr != nil {
				return published, markErr
			}
			continue
		}
		if err := markPublished(ctx, db, event); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func claimEvents(ctx context.Context, db *pgxpool.Pool, limit int) ([]claimedEvent, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,order_id,type,payload,attempt_count FROM outbox_events WHERE published_at IS NULL AND available_at<=clock_timestamp() AND (lease_until IS NULL OR lease_until<clock_timestamp()) ORDER BY available_at,created_at LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	events := []claimedEvent{}
	for rows.Next() {
		var e claimedEvent
		if err := rows.Scan(&e.id, &e.orderID, &e.eventType, &e.payload, &e.attempts); err != nil {
			rows.Close()
			return nil, err
		}
		e.token = uuid.New()
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range events {
		if _, err := tx.Exec(ctx, `UPDATE outbox_events SET lease_token=$2,lease_until=clock_timestamp()+interval '60 seconds',attempt_count=attempt_count+1 WHERE id=$1`, events[i].id, events[i].token); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return events, nil
}

func publishOne(ctx context.Context, client *http.Client, endpoint, project, topic, emulator string, event claimedEvent) error {
	var envelope eventEnvelope
	if err := json.Unmarshal(event.payload, &envelope); err != nil {
		return fmt.Errorf("decode outbox event: %w", err)
	}
	if envelope.EventID == uuid.Nil || envelope.OrderID != event.orderID || envelope.Type != event.eventType || envelope.SchemaVersion != 1 {
		return fmt.Errorf("outbox event does not match its indexed columns")
	}
	body, _ := json.Marshal(map[string]any{"messages": []any{map[string]any{"data": base64.StdEncoding.EncodeToString(event.payload), "attributes": map[string]string{"event_id": envelope.EventID.String(), "type": envelope.Type}}}})
	path := endpoint + "/v1/projects/" + url.PathEscape(project) + "/topics/" + url.PathEscape(topic) + ":publish"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if emulator == "" {
		token, err := metadataToken(ctx, client)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pubsub publish returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func metadataToken(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get service token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata token endpoint returned HTTP %d", resp.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("metadata token response was empty")
	}
	return token.AccessToken, nil
}

func markPublished(ctx context.Context, db *pgxpool.Pool, e claimedEvent) error {
	_, err := db.Exec(ctx, `UPDATE outbox_events SET published_at=clock_timestamp(),lease_token=NULL,lease_until=NULL,last_error=NULL WHERE id=$1 AND lease_token=$2 AND published_at IS NULL`, e.id, e.token)
	return err
}
func releaseEvent(ctx context.Context, db *pgxpool.Pool, e claimedEvent, cause error) error {
	delay := 1 << min(e.attempts, 8)
	if delay > 300 {
		delay = 300
	}
	msg := cause.Error()
	if len(msg) > 500 {
		msg = msg[:500]
	}
	_, err := db.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp()+($3::text||' seconds')::interval,lease_token=NULL,lease_until=NULL,last_error=$4 WHERE id=$1 AND lease_token=$2 AND published_at IS NULL`, e.id, e.token, strconv.Itoa(delay), msg)
	return err
}

type pushRequest struct {
	Message struct {
		Data      string `json:"data"`
		MessageID string `json:"messageId"`
	} `json:"message"`
}

// PubSubPushHandler acknowledges only committed events and safe duplicates.
func PubSubPushHandler(db *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		var req pushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid message", http.StatusBadRequest)
			return
		}
		raw, err := base64.StdEncoding.DecodeString(req.Message.Data)
		if err != nil {
			http.Error(w, "invalid message data", http.StatusBadRequest)
			return
		}
		var event eventEnvelope
		if err := json.Unmarshal(raw, &event); err != nil || event.EventID == uuid.Nil || event.OrderID == uuid.Nil || event.SchemaVersion != 1 || (event.Type != "ORDER_PAID" && event.Type != "ORDER_EXPIRED") {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		tx, err := db.Begin(r.Context())
		if err != nil {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		defer tx.Rollback(r.Context())
		var inserted uuid.UUID
		err = tx.QueryRow(r.Context(), `INSERT INTO processed_events(event_id) VALUES($1) ON CONFLICT DO NOTHING RETURNING event_id`, event.EventID).Scan(&inserted)
		if err == pgx.ErrNoRows {
			if err = tx.Commit(r.Context()); err != nil {
				http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		var storedOrder uuid.UUID
		var storedType string
		var schema int
		err = tx.QueryRow(r.Context(), `SELECT order_id,type,schema_version FROM outbox_events WHERE id=$1`, event.EventID).Scan(&storedOrder, &storedType, &schema)
		if err != nil || storedOrder != event.OrderID || storedType != event.Type || schema != event.SchemaVersion {
			http.Error(w, "event does not match outbox", http.StatusBadRequest)
			return
		}
		message := "Pedido confirmado."
		if event.Type == "ORDER_EXPIRED" {
			message = "A reserva expirou e o estoque foi liberado."
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO notifications(event_id,order_id,kind,message) VALUES($1,$2,$3,$4)`, event.EventID, event.OrderID, event.Type, message)
		if err != nil {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
