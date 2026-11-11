package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "flashcart/backend/internal/orders"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExpireBatch releases overdue reservations using one short transaction per order.
func ExpireBatch(ctx context.Context, db *pgxpool.Pool, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, fmt.Errorf("limit must be between 1 and 1000")
	}
	rows, err := db.Query(ctx, `SELECT id FROM orders WHERE status='PENDING_PAYMENT' AND expires_at<=clock_timestamp() ORDER BY expires_at,id LIMIT $1`, limit)
	if err != nil {
		return 0, err
	}
	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	count := 0
	for _, id := range ids {
		changed, err := expireOne(ctx, db, id)
		if err != nil {
			return count, err
		}
		if changed {
			count++
		}
	}
	if _, err := db.Exec(ctx, `DELETE FROM quota_buckets WHERE bucket_start < clock_timestamp()-interval '48 hours'`); err != nil {
		return count, err
	}
	return count, nil
}

func expireOne(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var productID uuid.UUID
	var quantity int
	var status string
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT product_id,quantity,status,expires_at FROM orders WHERE id=$1 FOR UPDATE SKIP LOCKED`, id).Scan(&productID, &quantity, &status, &expires)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if status != domain.PendingPayment {
		return false, nil
	}
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return false, err
	}
	if !domain.ShouldExpire(status, now, expires) {
		return false, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE orders SET status='EXPIRED',expired_at=$2,updated_at=$2 WHERE id=$1`, id, now); err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE products SET stock_available=stock_available+$2 WHERE id=$1`, productID, quantity); err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO order_events(order_id,type,occurred_at) VALUES($1,'ORDER_EXPIRED',$2) ON CONFLICT DO NOTHING`, id, now); err != nil {
		return false, err
	}
	eventID := uuid.New()
	payload, err := json.Marshal(map[string]any{"event_id": eventID, "schema_version": 1, "type": "ORDER_EXPIRED", "order_id": id, "occurred_at": now})
	if err != nil {
		return false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,order_id,type,schema_version,payload) VALUES($1,$2,'ORDER_EXPIRED',1,$3) ON CONFLICT(order_id,type) DO NOTHING`, eventID, id, payload); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
