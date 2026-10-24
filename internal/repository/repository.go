package repository

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/rule"
)

var ErrNotFound = errors.New("not found")

type Municipality struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Area struct {
	ID           int    `json:"id"`
	Municipality string `json:"municipality"`
	Name         string `json:"name"`
}

type Rule struct {
	rule.Rule
	Category string
	Name     string
}

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListMunicipalities() ([]Municipality, error) {
	rows, err := r.db.Query(`SELECT id, code, name FROM municipalities ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Municipality{}
	for rows.Next() {
		var m Municipality
		if err := rows.Scan(&m.ID, &m.Code, &m.Name); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *Repository) SearchAreas(code, q string) ([]Area, error) {
	rows, err := r.db.Query(`SELECT a.id, m.name, CONCAT(a.town, a.chome, a.block_range)
		FROM areas a JOIN municipalities m ON m.id = a.municipality_id
		WHERE m.code = ? AND a.town LIKE CONCAT(?, '%')
		ORDER BY a.town, a.chome, a.block_range`, code, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Area{}
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.Municipality, &a.Name); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *Repository) FindArea(id int) (Area, error) {
	var a Area
	err := r.db.QueryRow(`SELECT a.id, m.name, CONCAT(a.town, a.chome, a.block_range)
		FROM areas a JOIN municipalities m ON m.id = a.municipality_id WHERE a.id = ?`, id).
		Scan(&a.ID, &a.Municipality, &a.Name)
	if err == sql.ErrNoRows {
		return a, ErrNotFound
	}
	return a, err
}

func (r *Repository) ListRules(areaID int) ([]Rule, error) {
	rows, err := r.db.Query(`SELECT r.weekday, r.nth_weeks, r.raw_text, w.code, w.name_ja
		FROM collection_rules r JOIN waste_categories w ON w.id = r.waste_category_id
		WHERE r.area_id = ? ORDER BY w.id, r.weekday`, areaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Rule
	for rows.Next() {
		var ru Rule
		var weekday int
		var nth sql.NullString
		if err := rows.Scan(&weekday, &nth, &ru.RawText, &ru.Category, &ru.Name); err != nil {
			return nil, err
		}
		ru.Weekday = time.Weekday(weekday)
		for _, p := range strings.Split(nth.String, ",") {
			if n, err := strconv.Atoi(p); err == nil {
				ru.Nth = append(ru.Nth, n)
			}
		}
		list = append(list, ru)
	}
	return list, rows.Err()
}
