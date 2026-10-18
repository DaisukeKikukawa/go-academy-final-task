package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// Querier は *sql.DB と *sql.Tx の両方を受け取れるようにする小さなインタフェース。
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// LoadCategoryAliases は自治体のごみ種別名 → waste_categories.id の対応表を返す。
func LoadCategoryAliases(ctx context.Context, q Querier, municipalityID int) (map[string]int, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT raw_name, waste_category_id FROM category_aliases WHERE municipality_id = ?`, municipalityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := map[string]int{}
	for rows.Next() {
		var name string
		var id int
		if err := rows.Scan(&name, &id); err != nil {
			return nil, err
		}
		m[name] = id
	}
	return m, rows.Err()
}

// UpsertArea は地区を登録し、その id を返す。すでに同じ area_key があれば既存の id を返す。
// area_key に UNIQUE があるので、何度 import しても地区は増えない。
func UpsertArea(ctx context.Context, tx *sql.Tx, municipalityID int, code, town, chome, blockRange string) (int, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", code, town, chome, blockRange)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO areas (municipality_id, town, chome, block_range, area_key)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`,
		municipalityID, town, chome, blockRange, key)
	if err != nil {
		return 0, fmt.Errorf("upsert area %q: %w", key, err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// DeleteRulesBySource は、その取得元から取り込んだ収集ルールをすべて消す(入れ替えの前半)。
func DeleteRulesBySource(ctx context.Context, tx *sql.Tx, sourceID int) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM collection_rules WHERE data_source_id = ?`, sourceID)
	return err
}

// RuleRow は collection_rules に入れる1行。
type RuleRow struct {
	AreaID            int
	WasteCategoryID   int
	DataSourceID      int
	Weekday           int
	Frequency         string  // weekly / nth_week / biweekly
	NthWeeks          *string // "2,4" または nil
	CollectsOnHoliday bool
	RawText           string
}

// InsertRule は収集ルールを1行入れる(入れ替えの後半)。
func InsertRule(ctx context.Context, tx *sql.Tx, r RuleRow) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO collection_rules
		  (area_id, waste_category_id, data_source_id, weekday, frequency, nth_weeks, collects_on_holiday, raw_text)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.AreaID, r.WasteCategoryID, r.DataSourceID, r.Weekday, r.Frequency, r.NthWeeks, r.CollectsOnHoliday, r.RawText)
	return err
}

// StartImportJob は取り込み開始を記録し、その job id を返す。
func StartImportJob(ctx context.Context, db *sql.DB, sourceID int) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO import_jobs (data_source_id, started_at, status) VALUES (?, NOW(), 'running')`, sourceID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishImportJob は取り込み結果を記録する。status は success / failed / skipped。
func FinishImportJob(ctx context.Context, db *sql.DB, jobID int64, status string, rowsLoaded int, errMsg string) error {
	var msg *string
	if errMsg != "" {
		msg = &errMsg
	}
	_, err := db.ExecContext(ctx, `
		UPDATE import_jobs SET finished_at = NOW(), status = ?, rows_loaded = ?, error_message = ?
		WHERE id = ?`, status, rowsLoaded, msg, jobID)
	return err
}
