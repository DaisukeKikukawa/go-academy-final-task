// Package opendata は東京都オープンデータの取得まわり(文字コード変換、CSV取得、CKAN API)を扱う。
package opendata

import (
	"fmt"
	"io"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// NewUTF8Reader は encoding(例: "cp932", "utf-8", "utf-8-sig")で書かれた r を、
// UTF-8 として読める io.Reader に包んで返す。
// データを一度に全部読み込まず、読まれるそばから変換する。
func NewUTF8Reader(r io.Reader, encoding string) (io.Reader, error) {
	switch encoding {
	case "cp932", "shift_jis", "sjis":
		return transform.NewReader(r, japanese.ShiftJIS.NewDecoder()), nil
	case "utf-8", "utf8":
		return r, nil
	case "utf-8-sig":
		// 先頭の BOM(バイト順マーク)を取り除く。中野区のCSVがこの形式
		return transform.NewReader(r, unicode.UTF8BOM.NewDecoder()), nil
	default:
		return nil, fmt.Errorf("未対応の文字コード: %q", encoding)
	}
}
