package rule

import "time"

// Matches は日付 d がこのルールの収集日に当たるかを返す。
// 祝日の扱いはここでは見ない(呼び出し側で判断する)。
func (r Rule) Matches(d time.Time) bool {
	if d.Weekday() != r.Weekday {
		return false
	}
	switch r.Freq {
	case Weekly:
		return true
	case NthWeek:
		n := nthWeekdayOfMonth(d)
		for _, want := range r.NthWeeks {
			if n == want {
				return true
			}
		}
		return false
	case Biweekly:
		// 隔週は基準日が必要。基準日はDB側(biweekly_anchor)で持つ予定なので、
		// Rule に基準日を足すまでは未対応として false を返す。
		return false
	}
	return false
}

// nthWeekdayOfMonth は、その日が「その月で何回目のその曜日か」を返す。
// 1日〜7日なら1回目、8日〜14日なら2回目、... という計算になる。
//
//	2026-10-08(木) → 2  (10月の2回目の木曜)
//	2026-10-15(木) → 3
func nthWeekdayOfMonth(d time.Time) int {
	return (d.Day()-1)/7 + 1
}
