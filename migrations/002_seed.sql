SET NAMES utf8mb4;

INSERT INTO waste_categories (id, code, name_ja) VALUES
  (1, 'burnable',     '燃やすごみ'),
  (2, 'non_burnable', '燃やさないごみ'),
  (3, 'recyclable',   '資源');

INSERT INTO municipalities (id, code, name) VALUES
  (1, '13109', '品川区');

INSERT INTO category_aliases (municipality_id, raw_name, waste_category_id) VALUES
  (1, '燃やすごみ',             1),
  (1, '陶器・ガラス・金属ごみ', 2),
  (1, '資源',                   3);
