# ゴミ出し支援API

GO Academy Drive 最終課題①「ゴミの日 × サービス」。
東京都オープンデータの「ごみ収集日」データを取り込み、住所(町丁目)から「今日は何のごみの日か」を答える REST API。
Go + MySQL で実装。対応自治体は品川区。

## 課題感

集合住宅のごみ置き場の掲示には「指定された日時に出しましょう」とあるが、肝心の「この地区では何曜日に何を出すのか」は書かれていない。
守られない原因は「知らない」「忘れる」が大半で、正しい情報が必要なときに手元に届いていないことが本質。
自治体のCSVは区ごとに形式・用語・文字コードが違い、「第2・第4木曜」のようなルールは人がカレンダーを見て計算する必要がある。
このAPIはそのCSVを共通の形に正規化して保存し、住所を選ぶだけで今日・明日のごみ、月間カレンダー、次回収集日を返す。

## 起動手順

前提: Go 1.25、Docker

```bash
# 1. MySQL を起動(初回は migrations/ のスキーマと seed が自動で入る)
docker compose up -d

# 2. オープンデータを取り込む(品川区の CSV を取得して DB に保存)
go run ./cmd/importer
#    ネットが無い場合は同梱の CSV から:
#    go run ./cmd/importer -source 1 -file testdata/shinagawa.csv

# 3. API サーバを起動(http://localhost:8080)
go run ./cmd/api
```

スキーマを変えたときは `docker compose down && rm -rf data && docker compose up -d` で作り直す。
接続先は環境変数 `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` `PORT` で変更できる(既定は compose.yaml の値)。

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

| Method | Path | 説明 |
|---|---|---|
| GET | /healthz | 死活監視(DB に Ping) |
| GET | /v1/municipalities | 対応自治体 |
| GET | /v1/municipalities/:code/areas?q= | 町丁目の前方一致検索 |
| GET | /v1/areas/:id | 町丁目の詳細 |
| GET | /v1/areas/:id/rules | 収集ルール(共通コード + CSV の元の文字列) |
| GET | /v1/areas/:id/schedule?from&to | ルールを日付に展開したカレンダー(最大92日) |
| GET | /v1/areas/:id/today | 今日・明日のごみと一言メッセージ |
| GET | /v1/areas/:id/next?category= | 指定種別の次回収集日 |

エラーは全て `{"error": {"code": "...", "message": "..."}}` の形で、400 / 404 / 500 / 503 を使い分ける。

## 設計の要点

### 全体像

```
東京都オープンデータ(カタログ + 区の CSV)
   ↓ importer(毎日1回のバッチ)      文字コード変換 → パーサー → 共通コード化 → DB 入れ替え
MySQL(正解表: collection_rules)
   ↑ API(gin)                         ルールを日付に展開して JSON で返す
利用者
```

API は MySQL だけを見るので、利用者のリクエストで外部サイトにはアクセスしない。

### 定期更新の仕組み(cmd/importer)

1. 取得元(data_sources)ごとに CSV を取得し、文字コードを変換してパーサーに通す
2. 「同じ取得元の古いルールを DELETE → 新しいルールを INSERT」を1トランザクションで行う。途中で失敗したら Rollback で古いデータが残る
3. 地区(areas)は UNIQUE キーで UPSERT するので、何度取り込んでも増えない
4. 結果(success / failed、件数、行エラー)を import_jobs に毎回記録する

取り込みは冪等なので、cron で毎日実行すれば常に最新の状態になる: `0 4 * * * cd /path/to/repo && go run ./cmd/importer`

### データモデル

- `collection_rules` が核。1行 = 地区 × ごみ種別 × 曜日。「火・金」は2行、「第2・第4木」は weekday=木, nth_weeks="2,4" の1行
- 「10月8日は不燃ごみ」のような日付ごとの行は持たず、問い合わせのたびにルールから計算する
- `category_aliases` で区ごとの種別名(「陶器・ガラス・金属ごみ」など)を共通コード(non_burnable など)に寄せる。区を足すときは行を足すだけ
- `data_sources` に取得元(CSV の URL、文字コード、使うパーサー、最終取得日時)、`import_jobs` に取り込み履歴

### 曜日文字列の解釈(internal/rule)

CSV の「第２木・第４木」「火・金」を、前処理(全角数字→半角、「曜日」の削除、区切り文字の統一)で揃えてから解釈し、
曜日・頻度(weekly / nth_week)・第N週の組み合わせに変換する。「第2木」は「第2週」ではなく「その月の2回目の木曜」として `(日-1)/7+1` で判定する。

## ディレクトリ構成

```
cmd/api/          HTTP サーバ
cmd/importer/     オープンデータ取り込み CLI
internal/
  rule/           曜日文字列の解釈と日付判定(DB を知らない純粋ロジック)
  parser/         自治体ごとの CSV パーサー(品川区)
  opendata/       文字コード変換
  repository/     MySQL への読み書き(SQL はここにだけ書く)
  importer/       取得 → 変換 → DB 投入の流れ
  schedule/       ルールを日付に展開
  handler/        gin のハンドラ
migrations/       スキーマと seed(compose が初回起動時に流す)
testdata/         品川区 CSV(ネットが無いときの取り込み用)
```

## 未実装・仮定

- 祝日: 判定の差し込み口(HolidayFunc)は用意してあるが、祝日テーブルと内閣府 CSV の取り込みは未実装
- 年末年始の休止はデータに含まれない
- 他の区: `parser.Parser` を実装し `registry` と `data_sources` に1行足せば追加できる設計
