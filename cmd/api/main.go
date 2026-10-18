package main

import (
	"context"
	"log"
	"os"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/handler"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

func main() {
	ctx := context.Background()

	cfg := repository.ConfigFromEnv()
	db, err := repository.Open(ctx, cfg)
	if err != nil {
		log.Fatalf("DB に接続できません: %v", err)
	}
	defer db.Close()
	log.Printf("db connected: %s:%s/%s", cfg.Host, cfg.Port, cfg.Name)

	addr := ":" + envOr("PORT", "8080")
	r := handler.NewRouter(&handler.Server{DB: db})
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
