// Package parser は収集日CSVを、共通の Row に変換する。
package parser

import (
	"fmt"
	"regexp"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/rule"
)

// Row はCSV1行から得られる「ある地区の、あるごみ種別の、1つの収集ルール」。
// "火・金" のように曜日が2つある行は Row も2つになる。
type Row struct {
	Town        string    // 大井
	Chome       string    // 1丁目(無ければ "全域")
	BlockRange  string    // 18番、1～10番 など。無ければ空
	RawCategory string    // CSVに書かれていたごみ種別名(燃やすごみ など)。共通コードへの変換は取り込み側で行う
	Rule        rule.Rule // 曜日と頻度
}

// AreaName は Town + Chome + BlockRange を結合した表示用の地区名。
func (r Row) AreaName() string {
	return r.Town + r.Chome + r.BlockRange
}

// LineError はCSVの特定の行で起きた変換エラー。
// 1行の失敗で全体を止めず、呼び出し側が記録できるようにする。
type LineError struct {
	Line int // 1始まりの行番号(ヘッダ行が1)
	Err  error
}

func (e LineError) Error() string {
	return fmt.Sprintf("%d行目: %v", e.Line, e.Err)
}

// Result は変換結果。Rows は成功した行、Errors は失敗した行。
type Result struct {
	Rows   []Row
	Errors []LineError
}

// areaPattern は「大井1丁目」「南大井6丁目18番」「西品川1丁目（品川エリア）」を
// 町名 / 丁目 / 残り に分ける。
var areaPattern = regexp.MustCompile(`^(.+?)(\d+丁目)(.*)$`)

// SplitArea は地区名を (町名, 丁目, 番地範囲) に分ける。
// 丁目が無い場合は 丁目="全域" とする。
func SplitArea(name string) (town, chome, blockRange string) {
	m := areaPattern.FindStringSubmatch(name)
	if m == nil {
		return name, "全域", ""
	}
	return m[1], m[2], m[3]
}
