package rule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// weekdayOf は "月" "火" ... の1文字を time.Weekday に対応させる表。
var weekdayOf = map[string]time.Weekday{
	"日": time.Sunday,
	"月": time.Monday,
	"火": time.Tuesday,
	"水": time.Wednesday,
	"木": time.Thursday,
	"金": time.Friday,
	"土": time.Saturday,
}

// Parse は自治体CSVの曜日文字列を Rule の並びに変換する。
//
//	"火・金"        → [火 weekly, 金 weekly]
//	"第2木・第4木"  → [木 nth_week {2,4}]
//	"第1・第3月"    → [月 nth_week {1,3}]   ("第1" は曜日が省略され、後ろの "第3月" の曜日を使う)
//
// 空文字や曜日として読めない文字列はエラーを返す。
func Parse(raw string) ([]Rule, error) {
	s := normalize(raw)
	if s == "" {
		return nil, fmt.Errorf("曜日文字列が空です")
	}

	var (
		rules   []Rule
		nthIdx  = map[time.Weekday]int{} // 曜日 → rules の中の位置(同じ曜日の第Nを1つにまとめる)
		pending []int                    // "第1・第3月" の "第1" のように曜日が後回しになった N
	)

	for _, tok := range strings.Split(s, "・") {
		if !strings.HasPrefix(tok, "第") {
			// 「火」のような毎週ルール
			wd, ok := weekdayOf[tok]
			if !ok {
				return nil, fmt.Errorf("曜日として読めません: %q (元の文字列 %q)", tok, raw)
			}
			rules = append(rules, Rule{Weekday: wd, Freq: Weekly, RawText: raw})
			continue
		}

		// 「第2木」「第1」のような第Nルール。"第" の後ろを数字と曜日に分ける
		n, wdStr, err := splitNth(strings.TrimPrefix(tok, "第"))
		if err != nil {
			return nil, fmt.Errorf("%v (元の文字列 %q)", err, raw)
		}
		pending = append(pending, n)
		if wdStr == "" {
			continue // 曜日はまだ分からない。次のトークンに持ち越す
		}
		wd, ok := weekdayOf[wdStr]
		if !ok {
			return nil, fmt.Errorf("曜日として読めません: %q (元の文字列 %q)", wdStr, raw)
		}
		if i, exists := nthIdx[wd]; exists {
			rules[i].NthWeeks = append(rules[i].NthWeeks, pending...)
		} else {
			nthIdx[wd] = len(rules)
			rules = append(rules, Rule{Weekday: wd, Freq: NthWeek, NthWeeks: pending, RawText: raw})
		}
		pending = nil
	}

	if len(pending) > 0 {
		return nil, fmt.Errorf("第Nの曜日が指定されていません (元の文字列 %q)", raw)
	}
	return rules, nil
}

// splitNth は "2木" を (2, "木")、"1" を (1, "") に分ける。
func splitNth(s string) (int, string, error) {
	digits := strings.TrimRightFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	if digits == "" {
		return 0, "", fmt.Errorf("第の後に数字がありません: %q", "第"+s)
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 || n > 5 {
		return 0, "", fmt.Errorf("第Nの数字が不正です: %q", "第"+s)
	}
	return n, strings.TrimPrefix(s, digits), nil
}
