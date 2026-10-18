package repository

import (
	"context"
	"database/sql"
)

// DataSource は data_sources の1行(取得元)。自治体コードも一緒に持つ。
type DataSource struct {
	ID               int
	MunicipalityID   int
	MunicipalityCode string // 13109
	ResourceURL      string // CSV の URL
	Encoding         string // cp932 / utf-8 / utf-8-sig
}

// ListDataSources は取得元をすべて返す(importer が上から順に処理する)。
func ListDataSources(ctx context.Context, db *sql.DB) ([]DataSource, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT d.id, d.municipality_id, m.code, d.resource_url, d.encoding
		FROM data_sources d
		JOIN municipalities m ON m.id = d.municipality_id
		ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DataSource
	for rows.Next() {
		var d DataSource
		if err := rows.Scan(&d.ID, &d.MunicipalityID, &d.MunicipalityCode, &d.ResourceURL, &d.Encoding); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkDataSourceFetched は最後に取得した日時を記録する。
func MarkDataSourceFetched(ctx context.Context, q Querier, sourceID int) error {
	_, err := q.ExecContext(ctx, `UPDATE data_sources SET last_fetched_at = NOW() WHERE id = ?`, sourceID)
	return err
}
