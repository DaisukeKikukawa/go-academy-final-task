package repository

import (
	"context"
	"database/sql"
	"time"
)

// DataSource は data_sources の1行(取得元)。自治体コードも一緒に持つ。
type DataSource struct {
	ID               int
	MunicipalityID   int
	MunicipalityCode string // 13109
	CKANPackageID    string
	ResourceURL      string
	Format           string
	Encoding         string
	ParserKey        string
	RemoteModifiedAt sql.NullTime
	ContentHash      sql.NullString
}

// ListDataSources は取得元をすべて返す(importer が上から順に処理する)。
func ListDataSources(ctx context.Context, db *sql.DB) ([]DataSource, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT d.id, d.municipality_id, m.code, d.ckan_package_id, d.resource_url,
		       d.format, d.encoding, d.parser_key, d.remote_modified_at, d.content_hash
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
		if err := rows.Scan(&d.ID, &d.MunicipalityID, &d.MunicipalityCode, &d.CKANPackageID, &d.ResourceURL,
			&d.Format, &d.Encoding, &d.ParserKey, &d.RemoteModifiedAt, &d.ContentHash); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkDataSourceFetched は取得結果(本文ハッシュ、カタログ側の更新日時)を記録する。
func MarkDataSourceFetched(ctx context.Context, q Querier, sourceID int, contentHash string, remoteModified *time.Time) error {
	_, err := q.ExecContext(ctx, `
		UPDATE data_sources
		SET last_fetched_at = NOW(), content_hash = ?, remote_modified_at = ?
		WHERE id = ?`, contentHash, remoteModified, sourceID)
	return err
}
