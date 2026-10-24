package rule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Rule struct {
	Weekday time.Weekday
	Nth     []int
	RawText string
}

var weekdayOf = map[string]time.Weekday{
	"日": time.Sunday, "月": time.Monday, "火": time.Tuesday, "水": time.Wednesday,
	"木": time.Thursday, "金": time.Friday, "土": time.Saturday,
}

func normalize(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		return r
	}, s)
	s = strings.ReplaceAll(s, "曜日", "")
	s = strings.ReplaceAll(s, "曜", "")
	for _, sep := range []string{"、", ";", "；", ",", "，", "/", "／"} {
		s = strings.ReplaceAll(s, sep, "・")
	}
	s = strings.ReplaceAll(s, " ", "")
	return strings.ReplaceAll(s, "　", "")
}

func Parse(raw string) ([]Rule, error) {
	s := normalize(raw)
	if s == "" {
		return nil, fmt.Errorf("曜日が空です: %q", raw)
	}
	var rules []Rule
	index := map[time.Weekday]int{}
	var pending []int
	for _, tok := range strings.Split(s, "・") {
		if !strings.HasPrefix(tok, "第") {
			wd, ok := weekdayOf[tok]
			if !ok {
				return nil, fmt.Errorf("曜日として読めません: %q", raw)
			}
			rules = append(rules, Rule{Weekday: wd, RawText: raw})
			continue
		}
		body := strings.TrimPrefix(tok, "第")
		digits := strings.TrimRightFunc(body, func(r rune) bool { return r < '0' || r > '9' })
		n, err := strconv.Atoi(digits)
		if err != nil {
			return nil, fmt.Errorf("第Nの数字が読めません: %q", raw)
		}
		pending = append(pending, n)
		wdStr := strings.TrimPrefix(body, digits)
		if wdStr == "" {
			continue
		}
		wd, ok := weekdayOf[wdStr]
		if !ok {
			return nil, fmt.Errorf("曜日として読めません: %q", raw)
		}
		if i, ok := index[wd]; ok {
			rules[i].Nth = append(rules[i].Nth, pending...)
		} else {
			index[wd] = len(rules)
			rules = append(rules, Rule{Weekday: wd, Nth: pending, RawText: raw})
		}
		pending = nil
	}
	if len(pending) > 0 {
		return nil, fmt.Errorf("第Nに曜日がありません: %q", raw)
	}
	return rules, nil
}

func (r Rule) Matches(d time.Time) bool {
	if d.Weekday() != r.Weekday {
		return false
	}
	if len(r.Nth) == 0 {
		return true
	}
	n := (d.Day()-1)/7 + 1
	for _, w := range r.Nth {
		if w == n {
			return true
		}
	}
	return false
}
