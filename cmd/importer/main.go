package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/rule"
)

// ステップ1: importer の準備
const (
	municipalityCode = "13109"
	csvURL           = "http://www.city.shinagawa.tokyo.jp/ct/other000081600/gomisyusyubi.csv"
)

var areaPattern = regexp.MustCompile(`^(.+?)(\d+丁目)(.*)$`)

func main() {
	db, err := sql.Open("mysql", dsn())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	resp, err := http.Get(csvURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("HTTP %d", resp.StatusCode)
	}
	records, err := csv.NewReader(transform.NewReader(resp.Body, japanese.ShiftJIS.NewDecoder())).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[h] = i
	}

	var municipalityID int
	if err := db.QueryRow(`SELECT id FROM municipalities WHERE code = ?`, municipalityCode).Scan(&municipalityID); err != nil {
		log.Fatal(err)
	}
	aliases := map[string]int{}
	rows, err := db.Query(`SELECT raw_name, waste_category_id FROM category_aliases WHERE municipality_id = ?`, municipalityID)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var name string
		var id int
		if err := rows.Scan(&name, &id); err != nil {
			log.Fatal(err)
		}
		aliases[name] = id
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE r FROM collection_rules r JOIN areas a ON a.id = r.area_id WHERE a.municipality_id = ?`, municipalityID); err != nil {
		log.Fatal(err)
	}

	areaIDs := map[string]int{}
	saved := 0
	for i, rec := range records[1:] {
		rules, err := rule.Parse(rec[col["収集曜日"]])
		if err != nil {
			log.Printf("%d行目をスキップ: %v", i+2, err)
			continue
		}
		catID, ok := aliases[rec[col["ゴミ分類区分"]]]
		if !ok {
			log.Fatalf("category_aliases にありません: %q", rec[col["ゴミ分類区分"]])
		}
		name := rec[col["地区名"]]
		areaID, ok := areaIDs[name]
		if !ok {
			town, chome, block := name, "全域", ""
			if m := areaPattern.FindStringSubmatch(name); m != nil {
				town, chome, block = m[1], m[2], m[3]
			}
			res, err := tx.Exec(`INSERT INTO areas (municipality_id, town, chome, block_range) VALUES (?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, municipalityID, town, chome, block)
			if err != nil {
				log.Fatal(err)
			}
			id, _ := res.LastInsertId()
			areaID = int(id)
			areaIDs[name] = areaID
		}
		for _, r := range rules {
			var nth *string
			if len(r.Nth) > 0 {
				s := joinInts(r.Nth)
				nth = &s
			}
			if _, err := tx.Exec(`INSERT INTO collection_rules (area_id, waste_category_id, weekday, nth_weeks, raw_text) VALUES (?, ?, ?, ?, ?)`,
				areaID, catID, int(r.Weekday), nth, r.RawText); err != nil {
				log.Fatal(err)
			}
			saved++
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	log.Printf("rules=%d areas=%d", saved, len(areaIDs))
}

func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func dsn() string {
	env := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Asia%%2FTokyo&charset=utf8mb4",
		env("DB_USER", "user"), env("DB_PASSWORD", "password"), env("DB_HOST", "127.0.0.1"), env("DB_PORT", "3306"), env("DB_NAME", "gomi"))
}
