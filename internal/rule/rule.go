// Package rule は「いつごみを出すか」の収集ルールを表し、
// 自治体CSVの曜日文字列(例: "火・金", "第２木・第４木")を解釈する。
package rule

import "time"

// Frequency は収集の頻度。
type Frequency string

const (
	Weekly   Frequency = "weekly"   // 毎週
	NthWeek  Frequency = "nth_week" // その月のN回目の曜日(第2・第4木曜など)
	Biweekly Frequency = "biweekly" // 隔週(基準日から14日おき)
)

// Rule は「ある曜日に、ある頻度で収集する」という1つのルール。
// "火・金" のように曜日が複数ある場合は、曜日ごとに別の Rule にする。
type Rule struct {
	Weekday  time.Weekday // 0=日曜 .. 6=土曜
	Freq     Frequency
	NthWeeks []int  // Freq が NthWeek のときだけ使う。例: [2, 4]
	RawText  string // CSVに書かれていた元の文字列(調査用)
}
