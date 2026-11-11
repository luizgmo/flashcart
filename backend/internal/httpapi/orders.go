package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"flashcart/backend/internal/config"
	domain "flashcart/backend/internal/orders"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type createOrderRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}
type orderDTO struct {
	ID                 uuid.UUID  `json:"id"`
	ProductID          uuid.UUID  `json:"product_id"`
	ProductName        string     `json:"product_name"`
	Quantity           int        `json:"quantity"`
	UnitPriceCents     int64      `json:"unit_price_cents"`
	TotalCents         int64      `json:"total_cents"`
	Currency           string     `json:"currency"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	ExpiresAt          time.Time  `json:"expires_at"`
	PaidAt             *time.Time `json:"paid_at"`
	ExpiredAt          *time.Time `json:"expired_at"`
	CanPay             bool       `json:"can_pay"`
	ReservationElapsed bool       `json:"reservation_elapsed"`
}

func createOrder(db *pgxpool.Pool, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validOrigin(r, cfg.AppOrigin) {
			writeError(w, 403, "ORIGIN_FORBIDDEN", "Origem não permitida.")
			return
		}
		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Envie application/json.")
			return
		}
		sid, ok := sessionID(db, r)
		if !ok {
			writeError(w, 401, "SESSION_REQUIRED", "Inicie uma sessão para continuar.")
			return
		}
		key, e := uuid.Parse(r.Header.Get("Idempotency-Key"))
		if e != nil {
			writeError(w, 400, "IDEMPOTENCY_KEY_REQUIRED", "Envie uma chave Idempotency-Key válida.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		var req createOrderRequest
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if e = d.Decode(&req); e != nil || req.ProductID == uuid.Nil || req.Quantity < 1 || req.Quantity > 5 {
			writeError(w, 400, "INVALID_REQUEST", "Produto ou quantidade inválidos.")
			return
		}
		if e = d.Decode(&struct{}{}); e != io.EOF {
			writeError(w, 400, "INVALID_REQUEST", "Envie apenas um objeto JSON.")
			return
		}
		canonical, _ := json.Marshal(req)
		hash := sha256.Sum256(canonical)
		attemptMinute, attemptDay, orderDay := 10, 20, 500
		if cfg.RateLimitMode == "load" {
			attemptMinute, attemptDay, orderDay = 1000, 10000, 10000
		}
		allowed, e := consumeQuota(r.Context(), db, "session", quotaSubject(sid), "create_order_attempt", "minute", attemptMinute)
		if e == nil && allowed {
			allowed, e = consumeQuota(r.Context(), db, "session", quotaSubject(sid), "create_order_attempt", "day", attemptDay)
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível validar o limite de pedidos.")
			return
		}
		if !allowed {
			writeError(w, 429, "RATE_LIMITED", "Limite de tentativas atingido.")
			return
		}
		tx, e := db.Begin(r.Context())
		if e != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		defer tx.Rollback(r.Context())
		var live bool
		if e = tx.QueryRow(r.Context(), `SELECT true FROM sessions WHERE id=$1 AND expires_at>clock_timestamp() FOR UPDATE`, sid).Scan(&live); e != nil {
			writeError(w, 401, "SESSION_REQUIRED", "Inicie uma sessão para continuar.")
			return
		}
		var existing uuid.UUID
		var oldHash []byte
		e = tx.QueryRow(r.Context(), `SELECT order_id,request_hash FROM idempotency_keys WHERE session_id=$1 AND key=$2`, sid, key).Scan(&existing, &oldHash)
		if e == nil {
			if !equalBytes(hash[:], oldHash) {
				writeError(w, 409, "IDEMPOTENCY_CONFLICT", "Esta chave já foi usada com outros dados.")
				return
			}
			o, e := loadOrder(r.Context(), tx, sid, existing)
			if e != nil {
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível recuperar o pedido.")
				return
			}
			if e = tx.Commit(r.Context()); e != nil {
				writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
				return
			}
			w.Header().Set("Idempotency-Replayed", "true")
			writeJSON(w, 200, map[string]any{"order": o, "server_time": time.Now().UTC()})
			return
		} else if !errors.Is(e, pgx.ErrNoRows) {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível consultar o pedido.")
			return
		}
		var pending int
		if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM orders WHERE session_id=$1 AND status='PENDING_PAYMENT'`, sid).Scan(&pending); e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível validar o limite de pedidos.")
			return
		}
		if pending >= 3 {
			writeError(w, 409, "PENDING_ORDER_LIMIT", "Finalize ou aguarde a liberação de um pedido pendente.")
			return
		}
		allowed, e = consumeQuota(r.Context(), tx, "global", "orders", "new_order", "day", orderDay)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível validar o limite de pedidos.")
			return
		}
		if !allowed {
			writeError(w, 429, "RATE_LIMITED", "Limite diário de pedidos atingido.")
			return
		}
		var name string
		var price int64
		e = tx.QueryRow(r.Context(), `UPDATE products SET stock_available=stock_available-$2 WHERE id=$1 AND stock_available >= $2 RETURNING name,price_cents`, req.ProductID, req.Quantity).Scan(&name, &price)
		if errors.Is(e, pgx.ErrNoRows) {
			var exists bool
			_ = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1)`, req.ProductID).Scan(&exists)
			if exists {
				writeError(w, 409, "INSUFFICIENT_STOCK", "Estoque insuficiente.")
			} else {
				writeError(w, 404, "RESOURCE_NOT_FOUND", "Produto não encontrado.")
			}
			return
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível reservar o estoque.")
			return
		}
		var o orderDTO
		e = tx.QueryRow(r.Context(), `INSERT INTO orders(session_id,product_id,product_name,quantity,unit_price_cents,status,created_at,expires_at,updated_at) VALUES($1,$2,$3,$4,$5,'PENDING_PAYMENT',clock_timestamp(),clock_timestamp()+($6 * interval '1 second'),clock_timestamp()) RETURNING id,product_id,product_name,quantity,unit_price_cents,total_cents,status,created_at,expires_at`, sid, req.ProductID, name, req.Quantity, price, cfg.ReservationTTLSeconds).Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPriceCents, &o.TotalCents, &o.Status, &o.CreatedAt, &o.ExpiresAt)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível criar o pedido.")
			return
		}
		_, e = tx.Exec(r.Context(), `INSERT INTO idempotency_keys(session_id,key,request_hash,order_id) VALUES($1,$2,$3,$4)`, sid, key, hash[:], o.ID)
		if e == nil {
			_, e = tx.Exec(r.Context(), `INSERT INTO order_events(order_id,type) VALUES($1,'ORDER_CREATED')`, o.ID)
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível registrar o pedido.")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		o.Currency = "BRL"
		o.CanPay = true
		writeJSON(w, 201, map[string]any{"order": o, "server_time": time.Now().UTC()})
	}
}

func sessionID(db *pgxpool.Pool, r *http.Request) (uuid.UUID, bool) {
	c, e := r.Cookie("flashcart_session")
	if e != nil {
		return uuid.Nil, false
	}
	h := sha256.Sum256([]byte(c.Value))
	var id uuid.UUID
	e = db.QueryRow(r.Context(), `SELECT id FROM sessions WHERE token_hash=$1 AND expires_at>clock_timestamp()`, h[:]).Scan(&id)
	return id, e == nil
}
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var x byte
	for i := range a {
		x |= a[i] ^ b[i]
	}
	return x == 0
}
func loadOrder(ctx context.Context, tx pgx.Tx, sid, id uuid.UUID) (orderDTO, error) {
	var o orderDTO
	e := tx.QueryRow(ctx, `SELECT id,product_id,product_name,quantity,unit_price_cents,total_cents,status,created_at,expires_at,paid_at,expired_at,(status='PENDING_PAYMENT' AND expires_at>clock_timestamp()),(expires_at<=clock_timestamp()) FROM orders WHERE session_id=$1 AND id=$2`, sid, id).Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPriceCents, &o.TotalCents, &o.Status, &o.CreatedAt, &o.ExpiresAt, &o.PaidAt, &o.ExpiredAt, &o.CanPay, &o.ReservationElapsed)
	o.Currency = "BRL"
	return o, e
}

func listOrders(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, ok := sessionID(db, r)
		if !ok {
			writeError(w, 401, "SESSION_REQUIRED", "Inicie uma sessão para continuar.")
			return
		}
		page, size, e := pagination(r)
		if e != nil || size > 20 {
			writeError(w, 400, "INVALID_REQUEST", "Paginação inválida.")
			return
		}
		rows, e := db.Query(r.Context(), `SELECT id,product_id,product_name,quantity,unit_price_cents,total_cents,status,created_at,expires_at,paid_at,expired_at,(status='PENDING_PAYMENT' AND expires_at>clock_timestamp()),(expires_at<=clock_timestamp()) FROM orders WHERE session_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, sid, size, (page-1)*size)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os pedidos.")
			return
		}
		defer rows.Close()
		items := []orderDTO{}
		for rows.Next() {
			var o orderDTO
			if e = rows.Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPriceCents, &o.TotalCents, &o.Status, &o.CreatedAt, &o.ExpiresAt, &o.PaidAt, &o.ExpiredAt, &o.CanPay, &o.ReservationElapsed); e != nil {
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os pedidos.")
				return
			}
			o.Currency = "BRL"
			items = append(items, o)
		}
		var total int
		if e = db.QueryRow(r.Context(), `SELECT count(*) FROM orders WHERE session_id=$1`, sid).Scan(&total); e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os pedidos.")
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": size, "total": total, "server_time": time.Now().UTC()})
	}
}

func getOrder(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sid, ok := sessionID(db, r)
		if !ok {
			writeError(w, 401, "SESSION_REQUIRED", "Inicie uma sessão para continuar.")
			return
		}
		id, e := uuid.Parse(r.PathValue("id"))
		if e != nil {
			writeError(w, 400, "INVALID_REQUEST", "Identificador inválido.")
			return
		}
		var o orderDTO
		e = db.QueryRow(r.Context(), `SELECT id,product_id,product_name,quantity,unit_price_cents,total_cents,status,created_at,expires_at,paid_at,expired_at,(status='PENDING_PAYMENT' AND expires_at>clock_timestamp()),(expires_at<=clock_timestamp()) FROM orders WHERE id=$1 AND session_id=$2`, id, sid).Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPriceCents, &o.TotalCents, &o.Status, &o.CreatedAt, &o.ExpiresAt, &o.PaidAt, &o.ExpiredAt, &o.CanPay, &o.ReservationElapsed)
		if errors.Is(e, pgx.ErrNoRows) {
			writeError(w, 404, "RESOURCE_NOT_FOUND", "Pedido não encontrado.")
			return
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar o pedido.")
			return
		}
		o.Currency = "BRL"
		events := []map[string]any{}
		rows, e := db.Query(r.Context(), `SELECT type,occurred_at FROM order_events WHERE order_id=$1 ORDER BY occurred_at,id`, id)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar o pedido.")
			return
		}
		for rows.Next() {
			var t string
			var at time.Time
			if e = rows.Scan(&t, &at); e != nil {
				rows.Close()
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar o pedido.")
				return
			}
			events = append(events, map[string]any{"type": t, "occurred_at": at})
		}
		rows.Close()
		notifications := []map[string]any{}
		rows, e = db.Query(r.Context(), `SELECT id,kind,message,created_at FROM notifications WHERE order_id=$1 ORDER BY created_at,id`, id)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar o pedido.")
			return
		}
		for rows.Next() {
			var nid uuid.UUID
			var kind, msg string
			var at time.Time
			if e = rows.Scan(&nid, &kind, &msg, &at); e != nil {
				rows.Close()
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar o pedido.")
				return
			}
			notifications = append(notifications, map[string]any{"id": nid, "kind": kind, "message": msg, "created_at": at})
		}
		rows.Close()
		writeJSON(w, 200, map[string]any{"order": map[string]any{"id": o.ID, "product_id": o.ProductID, "product_name": o.ProductName, "quantity": o.Quantity, "unit_price_cents": o.UnitPriceCents, "total_cents": o.TotalCents, "currency": o.Currency, "status": o.Status, "created_at": o.CreatedAt, "expires_at": o.ExpiresAt, "paid_at": o.PaidAt, "expired_at": o.ExpiredAt, "can_pay": o.CanPay, "reservation_elapsed": o.ReservationElapsed, "timeline": events, "notifications": notifications}, "server_time": time.Now().UTC()})
	}
}

func simulatePayment(db *pgxpool.Pool, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validOrigin(r, cfg.AppOrigin) {
			writeError(w, 403, "ORIGIN_FORBIDDEN", "Origem não permitida.")
			return
		}
		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "Envie application/json.")
			return
		}
		if !decodeEmptyJSON(w, r) {
			return
		}
		sid, ok := sessionID(db, r)
		if !ok {
			writeError(w, 401, "SESSION_REQUIRED", "Inicie uma sessão para continuar.")
			return
		}
		id, e := uuid.Parse(r.PathValue("id"))
		if e != nil {
			writeError(w, 400, "INVALID_REQUEST", "Identificador inválido.")
			return
		}
		allowed, e := consumeQuota(r.Context(), db, "session", quotaSubject(sid), "payment_attempt", "minute", 20)
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível validar o limite.")
			return
		}
		if !allowed {
			writeError(w, 429, "RATE_LIMITED", "Limite de tentativas atingido.")
			return
		}
		tx, e := db.Begin(r.Context())
		if e != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		defer tx.Rollback(r.Context())
		var o orderDTO
		e = tx.QueryRow(r.Context(), `SELECT id,product_id,product_name,quantity,unit_price_cents,total_cents,status,created_at,expires_at,paid_at,expired_at FROM orders WHERE id=$1 AND session_id=$2 FOR UPDATE`, id, sid).Scan(&o.ID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPriceCents, &o.TotalCents, &o.Status, &o.CreatedAt, &o.ExpiresAt, &o.PaidAt, &o.ExpiredAt)
		if errors.Is(e, pgx.ErrNoRows) {
			writeError(w, 404, "RESOURCE_NOT_FOUND", "Pedido não encontrado.")
			return
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível atualizar o pedido.")
			return
		}
		if o.Status == domain.Paid {
			if e = tx.Commit(r.Context()); e != nil {
				writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
				return
			}
			o.Currency = "BRL"
			writeJSON(w, 200, map[string]any{"order": o, "server_time": time.Now().UTC()})
			return
		}
		var now time.Time
		if e = tx.QueryRow(r.Context(), `SELECT clock_timestamp()`).Scan(&now); e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível atualizar o pedido.")
			return
		}
		if o.Status == domain.Expired || domain.ShouldExpire(o.Status, now, o.ExpiresAt) {
			if o.Status != domain.Expired {
				var expired time.Time
				e = tx.QueryRow(r.Context(), `UPDATE orders SET status='EXPIRED',expired_at=$2,updated_at=$2 WHERE id=$1 RETURNING expired_at`, id, now).Scan(&expired)
				if e == nil {
					_, e = tx.Exec(r.Context(), `UPDATE products SET stock_available=stock_available+$2 WHERE id=$1`, o.ProductID, o.Quantity)
				}
				if e == nil {
					_, e = tx.Exec(r.Context(), `INSERT INTO order_events(order_id,type,occurred_at) VALUES($1,'ORDER_EXPIRED',$2)`, id, now)
				}
				eventID := uuid.New()
				payload, _ := json.Marshal(map[string]any{"event_id": eventID, "schema_version": 1, "type": "ORDER_EXPIRED", "order_id": id, "occurred_at": now})
				if e == nil {
					_, e = tx.Exec(r.Context(), `INSERT INTO outbox_events(id,order_id,type,schema_version,payload) VALUES($1,$2,'ORDER_EXPIRED',1,$3)`, eventID, id, payload)
				}
			}
			if e != nil {
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível liberar a reserva.")
				return
			}
			if e = tx.Commit(r.Context()); e != nil {
				writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
				return
			}
			writeError(w, 409, "ORDER_EXPIRED", "A reserva expirou e o estoque foi liberado.")
			return
		}
		_, e = tx.Exec(r.Context(), `UPDATE orders SET status='PAID',paid_at=$2,updated_at=$2 WHERE id=$1`, id, now)
		if e == nil {
			_, e = tx.Exec(r.Context(), `INSERT INTO order_events(order_id,type,occurred_at) VALUES($1,'ORDER_PAID',$2)`, id, now)
		}
		eventID := uuid.New()
		payload, _ := json.Marshal(map[string]any{"event_id": eventID, "schema_version": 1, "type": "ORDER_PAID", "order_id": id, "occurred_at": now})
		if e == nil {
			_, e = tx.Exec(r.Context(), `INSERT INTO outbox_events(id,order_id,type,schema_version,payload) VALUES($1,$2,'ORDER_PAID',1,$3)`, eventID, id, payload)
		}
		if e != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível confirmar o pagamento.")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			writeError(w, 503, "RETRYABLE_CONFLICT", "Tente novamente.")
			return
		}
		o.Status = "PAID"
		o.PaidAt = &now
		o.CanPay = false
		o.Currency = "BRL"
		writeJSON(w, 200, map[string]any{"order": o, "server_time": time.Now().UTC()})
	}
}
