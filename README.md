# ゴミ出しAPI

「ゴミの日 × サービス」。 東京都オープンデータの「ごみ収集日」データを取り込み、住所(町丁目)から「今日は何のごみの日か」を答える REST API。

## 課題感

集合住宅のごみ置き場の掲示には「指定された日時に出しましょう」とあるが、肝心の「この地区では何曜日に何を出すのか」は書かれていない。
守られない原因は「知らない」「忘れる」が大半で、正しい情報が必要なときに手元に届いていないことが本質。
自治体のCSVは区ごとに形式・用語・文字コードが違い、「第2・第4木曜」のようなルールは人がカレンダーを見て計算する必要がある。
このAPIはそのCSVを共通の形に正規化して保存し、住所を選ぶだけで今日・明日のごみ、月間カレンダー、次回収集日を返す。

## 起動手順

前提: Go 1.25、Docker

```bash
docker compose up -d
go run ./cmd/importer
go run ./cmd/api
```

## 使い方

```bash
# 地区を探す(品川区 = 13109)
curl "localhost:8080/v1/municipalities/13109/areas?q=大井"

# 今日・明日のごみ
curl localhost:8080/v1/areas/1/today
# → {"message":"今日は資源の日です。明日は収集がありません。", ...}

# 月間カレンダー
curl "localhost:8080/v1/areas/1/schedule?from=2026-10-01&to=2026-10-31"

# 次の燃やさないごみの日
curl "localhost:8080/v1/areas/1/next?category=non_burnable"
```

## エンドポイント

| Method | Path                              | 説明                                       |
| ------ | --------------------------------- | ------------------------------------------ |
| GET    | /v1/municipalities                | 対応自治体                                 |
| GET    | /v1/municipalities/:code/areas?q= | 町丁目の前方一致検索                       |
| GET    | /v1/areas/:id                     | 町丁目の詳細                               |
| GET    | /v1/areas/:id/rules               | 収集ルール(共通コード + CSV の元の文字列)  |
| GET    | /v1/areas/:id/schedule?from&to    | ルールを日付に展開したカレンダー(最大92日) |
| GET    | /v1/areas/:id/today               | 今日・明日のごみと一言メッセージ           |
| GET    | /v1/areas/:id/next?category=      | 指定種別の次回収集                         |

### db

- `collection_rules` 1行 = 地区 × ごみ種別 × 曜日。「火・金」は2行、「第2・第4木」は weekday=木, nth_weeks="2,4" の1行
- 「10月8日は不燃ごみ」のような日付ごとの行は持たず、問い合わせのたびに計算する
- `category_aliases` で区ごとの種別名(「陶器・ガラス・金属ごみ」など)を共通コード(non_burnable など)に寄せる。区を足すときは行を足すだけ
- `data_sources` に取得元(CSV の URL、文字コード、使うパーサー、前回の更新日時とハッシュ)、`import_jobs` に取り込み履歴

### 曜日文字列の解釈(internal/rule)

CSV の「第２木・第４木」「火・金」を、前処理(全角数字→半角、「曜日」の削除、区切り文字の統一)で揃えてから解釈し、  
曜日・頻度(weekly / nth_week)・第N週の組み合わせに変換する。「第2木」は「第2週」ではなく「その月の2回目の木曜」として `(日-1)/7+1` で判定する。
