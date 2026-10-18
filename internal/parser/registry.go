package parser

import "fmt"

// registry は data_sources.parser_key → パーサー実装の対応表。
// 自治体を追加したらここに1行足す。
var registry = map[string]Parser{
	"shinagawa_v1": Shinagawa{},
}

// ByKey は parser_key に対応するパーサーを返す。
func ByKey(key string) (Parser, error) {
	p, ok := registry[key]
	if !ok {
		return nil, fmt.Errorf("未知の parser_key: %q", key)
	}
	return p, nil
}
