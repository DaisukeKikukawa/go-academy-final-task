// Package importer はオープンデータを取得し、変換して MySQL に入れる一連の流れをまとめる。
//
//	取得 → 文字コード変換 → パーサー → 種別の共通コード化 → 地区登録 → ルール入れ替え
//
// 毎回取り直して入れ替える(冪等)ので、cron で毎日実行すれば常に最新になる。
package importer

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/opendata"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/parser"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

// Importer は1つの取得元を処理する。
type Importer struct {
	DB   *sql.DB
	HTTP *http.Client
}

// New は既定の HTTP クライアント(60秒タイムアウト)で Importer を作る。
func New(db *sql.DB) *Importer {
	return &Importer{DB: db, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

// Summary は1回の取り込みの結果。
type Summary struct {
	RulesSaved int
	Areas      int
	LineErrors []parser.LineError
}

// Run は取得元 ds の CSV を取得し、収集ルールを入れ替える。
func (im *Importer) Run(ctx context.Context, ds repository.DataSource) (Summary, error) {
	var sum Summary

	// 1. 取得(数十KB なので一度メモリに持つ)
	raw, err := im.fetch(ctx, ds.ResourceURL)
	if err != nil {
		return sum, fmt.Errorf("取得失敗 %s: %w", ds.ResourceURL, err)
	}

	// 2. 文字コード変換 → パーサー
	utf8r, err := opendata.NewUTF8Reader(bytes.NewReader(raw), ds.Encoding)
	if err != nil {
		return sum, err
	}
	res, err := parser.Shinagawa{}.Parse(utf8r)
	if err != nil {
		return sum, fmt.Errorf("パース失敗: %w", err)
	}
	sum.LineErrors = res.Errors
	if len(res.Rows) == 0 {
		return sum, fmt.Errorf("変換できた行が0件です(行エラー %d件)", len(res.Errors))
	}

	// 3. ここから DB 書き込み。全部まとめて1トランザクション
	tx, err := im.DB.BeginTx(ctx, nil)
	if err != nil {
		return sum, err
	}
	defer tx.Rollback() // Commit 済みなら何もしない。途中で return したら取り消し

	aliases, err := repository.LoadCategoryAliases(ctx, tx, ds.MunicipalityID)
	if err != nil {
		return sum, err
	}
	if err := repository.DeleteRulesBySource(ctx, tx, ds.ID); err != nil {
		return sum, err
	}

	areaIDs := map[string]int{} // 地区名 → areas.id(同じ地区を何度も UPSERT しないためのキャッシュ)
	for _, row := range res.Rows {
		catID, ok := aliases[row.RawCategory]
		if !ok {
			return sum, fmt.Errorf("category_aliases に %q がありません(municipality_id=%d)", row.RawCategory, ds.MunicipalityID)
		}
		areaID, ok := areaIDs[row.AreaName()]
		if !ok {
			areaID, err = repository.UpsertArea(ctx, tx, ds.MunicipalityID, ds.MunicipalityCode, row.Town, row.Chome, row.BlockRange)
			if err != nil {
				return sum, err
			}
			areaIDs[row.AreaName()] = areaID
		}
		if err := repository.InsertRule(ctx, tx, repository.RuleRow{
			AreaID:          areaID,
			WasteCategoryID: catID,
			DataSourceID:    ds.ID,
			Weekday:         int(row.Rule.Weekday),
			Frequency:       string(row.Rule.Freq),
			NthWeeks:        nthWeeksString(row.Rule.NthWeeks),
			RawText:         row.Rule.RawText,
		}); err != nil {
			return sum, err
		}
		sum.RulesSaved++
	}
	sum.Areas = len(areaIDs)

	if err := repository.MarkDataSourceFetched(ctx, tx, ds.ID); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	return sum, nil
}

// fetch は URL から本文を HTTP GET で取る。
func (im *Importer) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := im.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// nthWeeksString は [2,4] → "2,4"。空なら nil(DB では NULL)。
func nthWeeksString(ns []int) *string {
	if len(ns) == 0 {
		return nil
	}
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	s := strings.Join(parts, ",")
	return &s
}
