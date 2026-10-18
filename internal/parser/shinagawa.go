package parser

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/rule"
)

// Shinagawa は品川区の「ごみ収集日」CSV用のパーサー。
//
// 列: ゴミ分類区分, 地区名, 地区名(英語), 収集曜日, 祝日の収集, 特別に収集する日,
// 特別に収集しない日, 適用開始日, 適用終了日, 標準地域コード
//
// 縦持ち(1行 = 地区 × ごみ種別)なので、1行をそのまま Row に変換できる。
type Shinagawa struct{}

// 読み取りに使う列名。列の並びが変わっても名前で探すので壊れにくい。
const (
	colCategory = "ゴミ分類区分"
	colArea     = "地区名"
	colWeekday  = "収集曜日"
)

// Parse は UTF-8 に変換済みの CSV を読み、行ごとに Row へ変換する。
// 返り値の error はヘッダ不正など「ファイル全体として読めない」場合のみ。行単位の失敗は Result.Errors に入る。
func (Shinagawa) Parse(r io.Reader) (*Result, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // 列数のばらつきを許す(足りない列は後で検出する)

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("ヘッダ行が読めません: %w", err)
	}
	idx, err := columnIndex(header, colCategory, colArea, colWeekday)
	if err != nil {
		return nil, err
	}

	res := &Result{}
	line := 1
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			res.Errors = append(res.Errors, LineError{line, err})
			continue
		}
		if len(rec) <= idx[colWeekday] {
			res.Errors = append(res.Errors, LineError{line, fmt.Errorf("列が足りません(%d列)", len(rec))})
			continue
		}

		rules, err := rule.Parse(rec[idx[colWeekday]])
		if err != nil {
			res.Errors = append(res.Errors, LineError{line, err})
			continue
		}

		town, chome, block := SplitArea(rec[idx[colArea]])
		for _, ru := range rules {
			res.Rows = append(res.Rows, Row{
				Town:        town,
				Chome:       chome,
				BlockRange:  block,
				RawCategory: rec[idx[colCategory]],
				Rule:        ru,
			})
		}
	}
	return res, nil
}

// columnIndex はヘッダから必要な列の位置を名前で探す。見つからない列があればエラー。
func columnIndex(header []string, names ...string) (map[string]int, error) {
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}
	for _, n := range names {
		if _, ok := idx[n]; !ok {
			return nil, fmt.Errorf("ヘッダに %q 列がありません: %v", n, header)
		}
	}
	return idx, nil
}
