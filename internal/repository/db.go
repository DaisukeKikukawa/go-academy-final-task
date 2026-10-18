// Package repository は MySQL への読み書きをまとめる。
// SQL はこのパッケージの中にだけ書き、handler や importer からは関数として呼ぶ。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	// ドライバを登録するためだけに import する("mysql" という名前で sql.Open から使えるようになる)
	_ "github.com/go-sql-driver/mysql"
)

// Config は接続先。環境変数から組み立てる。
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// ConfigFromEnv は DB_HOST などの環境変数から Config を作る。未設定なら compose.yaml の値を使う。
func ConfigFromEnv() Config {
	return Config{
		Host:     getenv("DB_HOST", "127.0.0.1"),
		Port:     getenv("DB_PORT", "3306"),
		User:     getenv("DB_USER", "user"),
		Password: getenv("DB_PASSWORD", "password"),
		Name:     getenv("DB_NAME", "gomi"),
	}
}

// DSN は go-sql-driver/mysql が読む接続文字列を返す。
//
//	user:password@tcp(127.0.0.1:3306)/gomi?parseTime=true&loc=Asia%2FTokyo&charset=utf8mb4
//
// parseTime=true: DATETIME / DATE 列を string ではなく time.Time として受け取る
// loc=Asia/Tokyo: その time.Time のタイムゾーンを日本時間にする
func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FTokyo&charset=utf8mb4",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

// Open は接続プールを作り、実際に繋がることを Ping で確かめてから返す。
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	// プールの上限。小さなサービスなので控えめにする
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("db ping (%s:%s/%s): %w", cfg.Host, cfg.Port, cfg.Name, err)
	}
	return db, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
