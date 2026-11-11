package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type product struct {
	ID             uuid.UUID `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	ImagePath      string    `json:"image_path"`
	PriceCents     int64     `json:"price_cents"`
	Currency       string    `json:"currency"`
	StockAvailable int       `json:"stock_available"`
}

func listProducts(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, pageSize, err := pagination(r)
		if err != nil {
			writeError(w, 400, "INVALID_REQUEST", "Paginação inválida.")
			return
		}
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		if category != "" && category != "Áudio" && category != "Periféricos" && category != "Acessórios" {
			writeError(w, 400, "INVALID_REQUEST", "Categoria inválida.")
			return
		}
		ctx := r.Context()
		rows, err := db.Query(ctx, `SELECT id,slug,name,description,category,image_path,price_cents,stock_available FROM products WHERE ($1='' OR category=$1) ORDER BY name,id LIMIT $2 OFFSET $3`, category, pageSize, (page-1)*pageSize)
		if err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os produtos.")
			return
		}
		defer rows.Close()
		items := []product{}
		for rows.Next() {
			var p product
			if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Category, &p.ImagePath, &p.PriceCents, &p.StockAvailable); err != nil {
				writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os produtos.")
				return
			}
			p.Currency = "BRL"
			items = append(items, p)
		}
		var total int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM products WHERE ($1='' OR category=$1)`, category).Scan(&total); err != nil {
			writeError(w, 500, "INTERNAL_ERROR", "Não foi possível carregar os produtos.")
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "page": page, "page_size": pageSize, "total": total})
	}
}

func getProduct(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			writeError(w, 400, "INVALID_REQUEST", "Identificador inválido.")
			return
		}
		var p product
		err = db.QueryRow(r.Context(), `SELECT id,slug,name,description,category,image_path,price_cents,stock_available FROM products WHERE id=$1`, id).Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Category, &p.ImagePath, &p.PriceCents, &p.StockAvailable)
		if err != nil {
			writeError(w, 404, "RESOURCE_NOT_FOUND", "Produto não encontrado.")
			return
		}
		p.Currency = "BRL"
		writeJSON(w, 200, map[string]any{"product": p})
	}
}

func pagination(r *http.Request) (int, int, error) {
	page := 1
	size := 12
	if v := r.URL.Query().Get("page"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 {
			return 0, 0, errPagination
		}
		page = n
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > 24 {
			return 0, 0, errPagination
		}
		size = n
	}
	return page, size, nil
}
