// importer: data_sources に登録された取得元を順に取り込む。
//
//	go run ./cmd/importer                              # 全取得元を HTTP で取得
//	go run ./cmd/importer -source 1 -file testdata/shinagawa.csv   # 1件だけローカルファイルから
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/importer"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/opendata"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

func main() {
	sourceID := flag.Int("source", 0, "処理する data_sources.id(0なら全部)")
	file := flag.String("file", "", "URL の代わりに読むローカルファイル(-source と併用)")
	flag.Parse()

	ctx := context.Background()
	db, err := repository.Open(ctx, repository.ConfigFromEnv())
	if err != nil {
		log.Fatalf("DB に接続できません: %v", err)
	}
	defer db.Close()

	im := &importer.Importer{
		DB:    db,
		Fetch: importer.HTTPFetcher(60 * time.Second),
		CKAN:  opendata.NewCKANClient(30 * time.Second),
	}
	if *file != "" {
		if *sourceID == 0 {
			log.Fatal("-file を使うときは -source も指定してください")
		}
		im.Fetch = importer.FileFetcher(*file)
		im.CKAN = nil // ローカルファイルのときはカタログを見ない
	}

	sources, err := repository.ListDataSources(ctx, db)
	if err != nil {
		log.Fatalf("data_sources を読めません: %v", err)
	}

	failed := 0
	for _, ds := range sources {
		if *sourceID != 0 && ds.ID != *sourceID {
			continue
		}
		sum, err := im.Run(ctx, ds)
		if err != nil {
			failed++
			log.Printf("[source %d] FAILED (job %d): %v", ds.ID, sum.JobID, err)
			continue
		}
		log.Printf("[source %d] %s %s (job %d): rules=%d areas=%d lineErrors=%d warnings=%d",
			ds.ID, sum.Status, sum.Reason, sum.JobID, sum.RulesSaved, sum.Areas, len(sum.LineErrors), len(sum.Warnings))
	}
	if failed > 0 {
		log.Fatalf("%d 件の取得元が失敗しました", failed)
	}
}
