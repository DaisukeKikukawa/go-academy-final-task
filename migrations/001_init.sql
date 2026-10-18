-- ゴミ出し支援API 初期スキーマ
-- 初期化スクリプトは文字コード指定なしで流されるため、日本語を正しく保存するよう宣言する
SET NAMES utf8mb4;

-- A. 正解表 ---------------------------------------------------------------

CREATE TABLE municipalities (
  id   INT          NOT NULL AUTO_INCREMENT,
  code CHAR(5)      NOT NULL COMMENT '標準地域コード(5桁)。品川区=13109',
  name VARCHAR(64)  NOT NULL COMMENT '表示名。品川区',
  PRIMARY KEY (id),
  UNIQUE KEY uq_municipalities_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='自治体';

CREATE TABLE areas (
  id              INT           NOT NULL AUTO_INCREMENT,
  municipality_id INT           NOT NULL,
  town            VARCHAR(64)   NOT NULL COMMENT '町名。大井',
  chome           VARCHAR(16)   NOT NULL COMMENT '丁目。1丁目 / 全域',
  block_range     VARCHAR(64)   NOT NULL DEFAULT '' COMMENT '番地範囲や注記。18番 / （品川エリア）',
  area_key        VARCHAR(191)  NOT NULL COMMENT '重複チェック用。code:town:chome:block_range',
  PRIMARY KEY (id),
  UNIQUE KEY uq_areas_key (area_key),
  KEY idx_areas_municipality_town (municipality_id, town),
  CONSTRAINT fk_areas_municipality FOREIGN KEY (municipality_id) REFERENCES municipalities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='町丁目(収集ルールが決まる最小単位)';

CREATE TABLE waste_categories (
  id      INT          NOT NULL AUTO_INCREMENT,
  code    VARCHAR(32)  NOT NULL COMMENT 'burnable / non_burnable / recyclable / plastic / bulky',
  name_ja VARCHAR(64)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_waste_categories_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='ごみ種別の共通コード';

-- B. 取り込み管理 ---------------------------------------------------------

CREATE TABLE data_sources (
  id                 INT           NOT NULL AUTO_INCREMENT,
  municipality_id    INT           NOT NULL,
  resource_url       VARCHAR(512)  NOT NULL COMMENT 'CSVの実URL',
  encoding           VARCHAR(16)   NOT NULL DEFAULT 'utf-8' COMMENT 'cp932 / utf-8 / utf-8-sig',
  last_fetched_at    DATETIME      NULL COMMENT '最後に取り込んだ日時',
  PRIMARY KEY (id),
  KEY idx_data_sources_municipality (municipality_id),
  CONSTRAINT fk_data_sources_municipality FOREIGN KEY (municipality_id) REFERENCES municipalities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='オープンデータの取得元';

CREATE TABLE category_aliases (
  id                INT          NOT NULL AUTO_INCREMENT,
  municipality_id   INT          NOT NULL,
  raw_name          VARCHAR(64)  NOT NULL COMMENT 'CSVに書かれている種別名。陶器・ガラス・金属ごみ',
  waste_category_id INT          NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_category_aliases (municipality_id, raw_name),
  CONSTRAINT fk_category_aliases_municipality FOREIGN KEY (municipality_id) REFERENCES municipalities (id),
  CONSTRAINT fk_category_aliases_category FOREIGN KEY (waste_category_id) REFERENCES waste_categories (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='自治体ごとのごみ種別名 → 共通コード';

-- 収集ルール(正解表の本体。data_sources を参照するので最後に作る) -------------

CREATE TABLE collection_rules (
  id                  BIGINT       NOT NULL AUTO_INCREMENT,
  area_id             INT          NOT NULL,
  waste_category_id   INT          NOT NULL,
  data_source_id      INT          NOT NULL,
  weekday             TINYINT      NOT NULL COMMENT '0=日 .. 6=土',
  frequency           ENUM('weekly','nth_week') NOT NULL,
  nth_weeks           VARCHAR(16)  NULL COMMENT 'nth_week のとき "2,4"',
  raw_text            VARCHAR(64)  NOT NULL COMMENT 'CSVの元の文字列。第２木・第４木',
  PRIMARY KEY (id),
  KEY idx_collection_rules_area (area_id),
  KEY idx_collection_rules_weekday (weekday),
  KEY idx_collection_rules_source (data_source_id),
  CONSTRAINT fk_collection_rules_area     FOREIGN KEY (area_id)           REFERENCES areas (id),
  CONSTRAINT fk_collection_rules_category FOREIGN KEY (waste_category_id) REFERENCES waste_categories (id),
  CONSTRAINT fk_collection_rules_source   FOREIGN KEY (data_source_id)    REFERENCES data_sources (id),
  CONSTRAINT chk_collection_rules_weekday CHECK (weekday BETWEEN 0 AND 6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='収集ルール(1行 = 地区 × 種別 × 曜日)';
