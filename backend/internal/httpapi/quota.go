package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type quotaQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func quotaSubject(id uuid.UUID) string { h := sha256.Sum256(id[:]); return hex.EncodeToString(h[:]) }

// consumeQuota atomically increments a fixed UTC time bucket, up to the supplied limit.
func consumeQuota(ctx context.Context, q quotaQuerier, scope, subject, action, window string, limit int) (bool, error) {
	var count int
	err := q.QueryRow(ctx, `INSERT INTO quota_buckets(scope,subject_hash,action,bucket_start,count)
VALUES($1,$2,$3,date_trunc($4,clock_timestamp()),1)
ON CONFLICT(scope,subject_hash,action,bucket_start) DO UPDATE SET count=quota_buckets.count+1
WHERE quota_buckets.count<$5 RETURNING count`, scope, subject, action, window, limit).Scan(&count)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("consume quota: %w", err)
}
