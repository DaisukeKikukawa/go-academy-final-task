-- 初期データ: 共通コード、品川区、品川区の種別対応表、品川区CSVの取得元
-- 初期化スクリプトは文字コード指定なしで流されるため、日本語を正しく保存するよう宣言する
SET NAMES utf8mb4;

INSERT INTO waste_categories (id, code, name_ja) VALUES
  (1, 'burnable',     '燃やすごみ'),
  (2, 'non_burnable', '燃やさないごみ'),
  (3, 'recyclable',   '資源'),
  (4, 'plastic',      'プラスチック'),
  (5, 'bulky',        '粗大ごみ');

INSERT INTO municipalities (id, code, name) VALUES
  (1, '13109', '品川区');

-- 品川区CSVの「ゴミ分類区分」3種類 → 共通コード
INSERT INTO category_aliases (municipality_id, raw_name, waste_category_id) VALUES
  (1, '燃やすごみ',             1),
  (1, '陶器・ガラス・金属ごみ', 2),
  (1, '資源',                   3);

INSERT INTO data_sources (id, municipality_id, resource_url, encoding) VALUES
  (1, 1, 'http://www.city.shinagawa.tokyo.jp/ct/other000081600/gomisyusyubi.csv', 'cp932');
