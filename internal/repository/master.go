package repository

import (
	"context"
	"database/sql"
)

// Municipality は municipalities の1行。
type Municipality struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// Area は areas の1行(自治体名も付ける)。
type Area struct {
	ID               int    `json:"id"`
	MunicipalityCode string `json:"municipality_code"`
	MunicipalityName string `json:"municipality_name"`
	Town             string `json:"town"`
	Chome            string `json:"chome"`
	BlockRange       string `json:"block_range"`
	Name             string `json:"name"` // town + chome + block_range
}

// ListMunicipalities は対応自治体を返す。
func ListMunicipalities(ctx context.Context, db *sql.DB) ([]Municipality, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, code, name FROM municipalities ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Municipality
	for rows.Next() {
		var m Municipality
		if err := rows.Scan(&m.ID, &m.Code, &m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

const areaSelect = `
	SELECT a.id, m.code, m.name, a.town, a.chome, a.block_range
	FROM areas a JOIN municipalities m ON m.id = a.municipality_id`

// SearchAreas は自治体コード内の町丁目を、町名の前方一致で検索する。q が空なら全件。
func SearchAreas(ctx context.Context, db *sql.DB, municipalityCode, q string, limit int) ([]Area, error) {
	rows, err := db.QueryContext(ctx, areaSelect+`
		WHERE m.code = ? AND a.town LIKE CONCAT(?, '%')
		ORDER BY a.town, a.chome, a.block_range
		LIMIT ?`, municipalityCode, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAreas(rows)
}

// GetArea は id で町丁目を1件返す。無ければ sql.ErrNoRows。
func GetArea(ctx context.Context, db *sql.DB, id int) (*Area, error) {
	rows, err := db.QueryContext(ctx, areaSelect+` WHERE a.id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	areas, err := scanAreas(rows)
	if err != nil {
		return nil, err
	}
	if len(areas) == 0 {
		return nil, sql.ErrNoRows
	}
	return &areas[0], nil
}

func scanAreas(rows *sql.Rows) ([]Area, error) {
	var out []Area
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.MunicipalityCode, &a.MunicipalityName, &a.Town, &a.Chome, &a.BlockRange); err != nil {
			return nil, err
		}
		a.Name = a.Town + a.Chome + a.BlockRange
		out = append(out, a)
	}
	return out, rows.Err()
}
