// Package schedule は DB から読んだ収集ルールを、実際の日付に展開する。
// 「10月8日は大井1丁目で何のごみの日か」はここで計算する(DB に日付ごとの行は持たない)。
package schedule

import (
	"time"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

// JST は日付の基準にする日本時間。
var JST = time.FixedZone("Asia/Tokyo", 9*60*60)

// Item はある日に出せるごみ1種類。
type Item struct {
	Category     string `json:"category"`      // burnable など
	CategoryName string `json:"category_name"` // 燃やすごみ
}

// Day は1日分の結果。
type Day struct {
	Date    string `json:"date"`    // 2026-10-08
	Weekday int    `json:"weekday"` // 0=日 .. 6=土
	Items   []Item `json:"items"`
}

// ItemsOn は日付 d に出せるごみ種別を返す。同じ種別は1つにまとめる。
func ItemsOn(rules []repository.CollectionRule, d time.Time) []Item {
	seen := map[string]bool{}
	items := []Item{}
	for _, r := range rules {
		if !r.Rule().Matches(d) {
			continue
		}
		if seen[r.Category] {
			continue
		}
		seen[r.Category] = true
		items = append(items, Item{Category: r.Category, CategoryName: r.CategoryName})
	}
	return items
}

// Expand は from から to まで(両端含む)の各日について ItemsOn を計算する。
func Expand(rules []repository.CollectionRule, from, to time.Time) []Day {
	from = Truncate(from)
	to = Truncate(to)
	var days []Day
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		days = append(days, Day{
			Date:    d.Format("2006-01-02"),
			Weekday: int(d.Weekday()),
			Items:   ItemsOn(rules, d),
		})
	}
	return days
}

// Truncate は時刻を落として JST の 0 時にする。
func Truncate(t time.Time) time.Time {
	t = t.In(JST)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, JST)
}

// Today は JST での今日の 0 時。
func Today() time.Time { return Truncate(time.Now()) }
