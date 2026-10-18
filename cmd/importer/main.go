// importer: data_sources に登録された取得元を順に取り込む。
//
//	go run ./cmd/importer
package main

import (
	"context"
	"log"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/importer"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

func main() {
	ctx := context.Background()
	db, err := repository.Open(ctx, repository.ConfigFromEnv())
	if err != nil {
		log.Fatalf("DB に接続できません: %v", err)
	}
	defer db.Close()

	sources, err := repository.ListDataSources(ctx, db)
	if err != nil {
		log.Fatalf("data_sources を読めません: %v", err)
	}

	im := importer.New(db)
	failed := 0
	for _, ds := range sources {
		sum, err := im.Run(ctx, ds)
		if err != nil {
			failed++
			log.Printf("[source %d] FAILED: %v", ds.ID, err)
			continue
		}
		log.Printf("[source %d] success: rules=%d areas=%d lineErrors=%d", ds.ID, sum.RulesSaved, sum.Areas, len(sum.LineErrors))
		for _, e := range sum.LineErrors {
			log.Printf("[source %d]   %v", ds.ID, e)
		}
	}
	if failed > 0 {
		log.Fatalf("%d 件の取得元が失敗しました", failed)
	}
}
