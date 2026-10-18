package rule

import "strings"

// normalize は自治体ごとにばらつく曜日文字列の表記を、
// 後続の処理が扱いやすい1つの形に揃える。
//
//	"第２木・第４木"   → "第2木・第4木"   (全角数字を半角に)
//	"月曜日・金曜日"   → "月・金"         ("曜日" "曜" を削る)
//	"第1;第3木曜日"    → "第1・第3木"     (区切り文字を "・" に統一)
//	" 火 ・ 金 "       → "火・金"         (前後と区切りまわりの空白を削る)
func normalize(s string) string {
	// 1. 全角数字 → 半角数字
	s = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		return r
	}, s)

	// 2. "曜日" "曜" を削る("木曜日" → "木")。
	//    長い方を先に消さないと "曜日" の "日" が残るので順番に注意。
	s = strings.ReplaceAll(s, "曜日", "")
	s = strings.ReplaceAll(s, "曜", "")

	// 3. 区切り文字を "・" に統一
	for _, sep := range []string{"、", ";", "；", ",", "，", "/", "／"} {
		s = strings.ReplaceAll(s, sep, "・")
	}

	// 4. 空白(半角・全角)を削る
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "")

	return strings.TrimSpace(s)
}
