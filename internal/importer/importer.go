// Package importer はオープンデータを取得し、変換して MySQL に入れる一連の流れをまとめる。
//
//	取得(Fetch) → 文字コード変換 → パーサー → 種別の共通コード化 → 地区登録 → ルール入れ替え → 履歴記録
package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/opendata"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/parser"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

// Fetcher は URL から本文を取ってくる。本番は HTTP、テストや開発ではローカルファイルに差し替える。
type Fetcher func(ctx context.Context, url string) (io.ReadCloser, error)

// Importer は1つの取得元を処理する。
type Importer struct {
	DB    *sql.DB
	Fetch Fetcher
	CKAN  *opendata.CKANClient // nil なら更新日時チェックを行わない(開発時のローカルファイル取り込み)
}

// Summary は1回の取り込みの結果。
type Summary struct {
	JobID      int64
	Status     string // success / failed / skipped
	Reason     string // skipped の理由(catalog-unchanged / content-unchanged)
	RulesSaved int
	Areas      int
	LineErrors []parser.LineError
	Warnings   []string // 致命的ではないが記録したいこと(CKAN に繋がらなかった等)
}

// Run は取得元 ds を取り込む。失敗しても import_jobs に記録してから error を返す。
func (im *Importer) Run(ctx context.Context, ds repository.DataSource) (Summary, error) {
	jobID, err := repository.StartImportJob(ctx, im.DB, ds.ID)
	if err != nil {
		return Summary{}, fmt.Errorf("import_jobs に開始を記録できません: %w", err)
	}
	sum := Summary{JobID: jobID}

	sum, err = im.run(ctx, ds, sum)
	if err != nil {
		sum.Status = "failed"
		_ = repository.FinishImportJob(ctx, im.DB, jobID, "failed", 0, err.Error())
		return sum, err
	}
	note := strings.TrimSpace(strings.Join(append(sum.Warnings, joinLineErrors(sum.LineErrors)), "\n"))
	if err := repository.FinishImportJob(ctx, im.DB, jobID, sum.Status, sum.RulesSaved, note); err != nil {
		return sum, fmt.Errorf("import_jobs に結果を記録できません: %w", err)
	}
	return sum, nil
}

func (im *Importer) run(ctx context.Context, ds repository.DataSource, sum Summary) (Summary, error) {
	// 0. カタログ側の更新日時を見て、前回取り込み時から変わっていなければ CSV を取りに行かない
	var remoteModified *time.Time
	if im.CKAN != nil && ds.CKANPackageID != "" {
		info, err := im.CKAN.PackageShow(ctx, ds.CKANPackageID)
		if err != nil {
			// カタログに繋がらなくても CSV 本体の取得は試す(ハッシュ比較で二重取り込みは防げる)
			sum.Warnings = append(sum.Warnings, "CKAN 確認失敗: "+err.Error())
		} else {
			// DB の DATETIME は秒までしか持たないので、比較も保存も秒に丸めた値で行う
			t := info.MetadataModified.Truncate(time.Second)
			remoteModified = &t
			if ds.RemoteModifiedAt.Valid && ds.RemoteModifiedAt.Time.Equal(t) {
				sum.Status, sum.Reason = "skipped", "catalog-unchanged"
				return sum, nil
			}
		}
	}

	// 1. 取得。本文はハッシュ計算と変換の両方で使うので一度メモリに持つ(数十KB)
	body, err := im.Fetch(ctx, ds.ResourceURL)
	if err != nil {
		return sum, fmt.Errorf("取得失敗 %s: %w", ds.ResourceURL, err)
	}
	raw, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return sum, fmt.Errorf("本文の読み込み失敗: %w", err)
	}
	hash := sha256Hex(raw)

	// 2. 前回と本文が同じなら、ルールは触らず skipped(カタログの更新日時だけ控えておく)
	if ds.ContentHash.Valid && ds.ContentHash.String == hash {
		sum.Status, sum.Reason = "skipped", "content-unchanged"
		if err := repository.MarkDataSourceFetched(ctx, im.DB, ds.ID, hash, remoteModified); err != nil {
			return sum, err
		}
		return sum, nil
	}

	// 3. 文字コード変換 → パーサー
	utf8r, err := opendata.NewUTF8Reader(bytes.NewReader(raw), ds.Encoding)
	if err != nil {
		return sum, err
	}
	p, err := parser.ByKey(ds.ParserKey)
	if err != nil {
		return sum, err
	}
	res, err := p.Parse(utf8r)
	if err != nil {
		return sum, fmt.Errorf("パース失敗: %w", err)
	}
	sum.LineErrors = res.Errors
	if len(res.Rows) == 0 {
		return sum, fmt.Errorf("変換できた行が0件です(行エラー %d件)", len(res.Errors))
	}

	// 4. ここから DB 書き込み。全部まとめて1トランザクション
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
			AreaID:            areaID,
			WasteCategoryID:   catID,
			DataSourceID:      ds.ID,
			Weekday:           int(row.Rule.Weekday),
			Frequency:         string(row.Rule.Freq),
			NthWeeks:          nthWeeksString(row.Rule.NthWeeks),
			CollectsOnHoliday: row.CollectsOnHoliday,
			RawText:           row.Rule.RawText,
		}); err != nil {
			return sum, err
		}
		sum.RulesSaved++
	}
	sum.Areas = len(areaIDs)

	if err := repository.MarkDataSourceFetched(ctx, tx, ds.ID, hash, remoteModified); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	sum.Status = "success"
	return sum, nil
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

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func joinLineErrors(errs []parser.LineError) string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Error())
	}
	return strings.Join(parts, "\n")
}
