package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/handler"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/service"
)

func main() {
	db, err := sql.Open("mysql", dsn())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	h := handler.New(service.New(repository.New(db)))
	r := gin.Default()
	h.Register(r)
	log.Fatal(r.Run(":" + env("PORT", "8080")))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FTokyo&charset=utf8mb4",
		env("DB_USER", "user"), env("DB_PASSWORD", "password"), env("DB_HOST", "127.0.0.1"), env("DB_PORT", "3306"), env("DB_NAME", "gomi"))
}
