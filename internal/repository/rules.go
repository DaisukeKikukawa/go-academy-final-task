package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/rule"
)

// CollectionRule は collection_rules の1行に、ごみ種別の共通コードと表示名を付けたもの。
type CollectionRule struct {
	ID                int64  `json:"id"`
	AreaID            int    `json:"area_id"`
	Category          string `json:"category"`      // burnable など
	CategoryName      string `json:"category_name"` // 燃やすごみ
	Weekday           int    `json:"weekday"`       // 0=日 .. 6=土
	Frequency         string `json:"frequency"`     // weekly / nth_week / biweekly
	NthWeeks          []int  `json:"nth_weeks,omitempty"`
	CollectsOnHoliday bool   `json:"collects_on_holiday"`
	RawText           string `json:"raw_text"`
}

// Rule は rule パッケージの判定ロジックに渡せる形に変換する。
func (c CollectionRule) Rule() rule.Rule {
	return rule.Rule{
		Weekday:  time.Weekday(c.Weekday),
		Freq:     rule.Frequency(c.Frequency),
		NthWeeks: c.NthWeeks,
		RawText:  c.RawText,
	}
}

// ListRulesByArea は地区の収集ルールを、種別 → 曜日の順で返す。
func ListRulesByArea(ctx context.Context, db *sql.DB, areaID int) ([]CollectionRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.area_id, w.code, w.name_ja, r.weekday, r.frequency, r.nth_weeks, r.collects_on_holiday, r.raw_text
		FROM collection_rules r
		JOIN waste_categories w ON w.id = r.waste_category_id
		WHERE r.area_id = ?
		  AND (r.valid_from IS NULL OR r.valid_from <= CURDATE())
		  AND (r.valid_to   IS NULL OR r.valid_to   >= CURDATE())
		ORDER BY w.id, r.weekday`, areaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CollectionRule
	for rows.Next() {
		var c CollectionRule
		var nth sql.NullString
		if err := rows.Scan(&c.ID, &c.AreaID, &c.Category, &c.CategoryName, &c.Weekday, &c.Frequency, &nth, &c.CollectsOnHoliday, &c.RawText); err != nil {
			return nil, err
		}
		if nth.Valid {
			c.NthWeeks = parseNthWeeks(nth.String)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// parseNthWeeks は "2,4" → [2, 4]。importer が入れた形式を読み戻す。
func parseNthWeeks(s string) []int {
	var out []int
	for _, p := range strings.Split(s, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			out = append(out, n)
		}
	}
	return out
}
