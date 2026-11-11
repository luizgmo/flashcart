package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type seedProduct struct {
	id, slug, name, description, category, image string
	price                                        int64
	stock                                        int
}

var demoProducts = []seedProduct{
	{"00000000-0000-4000-8000-000000000001", "fone-nuvem", "Fone Nuvem", "Som equilibrado para acompanhar o dia.", "Áudio", "/products/headphones.svg", 28900, 24},
	{"00000000-0000-4000-8000-000000000002", "mouse-trilha", "Mouse Trilha", "Precisão confortável para longas jornadas.", "Periféricos", "/products/mouse.svg", 17900, 38},
	{"00000000-0000-4000-8000-000000000003", "teclado-linha", "Teclado Linha", "Teclas silenciosas em perfil compacto.", "Periféricos", "/products/keyboard.svg", 34900, 20},
	{"00000000-0000-4000-8000-000000000004", "hub-caminho", "Hub Caminho", "Mais conexões, menos complicação.", "Acessórios", "/products/hub.svg", 12900, 52},
	{"00000000-0000-4000-8000-000000000005", "webcam-foco", "Webcam Foco", "Imagem clara para conversas e reuniões.", "Acessórios", "/products/webcam.svg", 23900, 18},
	{"00000000-0000-4000-8000-000000000006", "luminaria-aurora", "Luminária Aurora", "Luz suave para a mesa de trabalho.", "Acessórios", "/products/lamp.svg", 19900, 31},
	{"00000000-0000-4000-8000-000000000007", "caixa-pulso", "Caixa Pulso", "Áudio compacto para espaços pequenos.", "Áudio", "/products/speaker.svg", 15900, 27},
	{"00000000-0000-4000-8000-000000000008", "suporte-eixo", "Suporte Eixo", "Eleva a tela e libera espaço na mesa.", "Acessórios", "/products/stand.svg", 21900, 42},
}

func runSeed(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range demoProducts {
		_, err = tx.ExecContext(ctx, `INSERT INTO products(id,slug,name,description,category,image_path,price_cents,initial_stock,stock_available) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) ON CONFLICT(slug) DO NOTHING`, p.id, p.slug, p.name, p.description, p.category, p.image, p.price, p.stock)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
