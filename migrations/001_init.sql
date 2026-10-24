SET NAMES utf8mb4;

CREATE TABLE municipalities (
  id   INT         NOT NULL AUTO_INCREMENT,
  code CHAR(5)     NOT NULL,
  name VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (code)
);

CREATE TABLE areas (
  id              INT         NOT NULL AUTO_INCREMENT,
  municipality_id INT         NOT NULL,
  town            VARCHAR(64) NOT NULL,
  chome           VARCHAR(16) NOT NULL,
  block_range     VARCHAR(64) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  UNIQUE KEY (municipality_id, town, chome, block_range),
  FOREIGN KEY (municipality_id) REFERENCES municipalities (id)
);

CREATE TABLE waste_categories (
  id      INT         NOT NULL AUTO_INCREMENT,
  code    VARCHAR(32) NOT NULL,
  name_ja VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (code)
);

CREATE TABLE category_aliases (
  id                INT         NOT NULL AUTO_INCREMENT,
  municipality_id   INT         NOT NULL,
  raw_name          VARCHAR(64) NOT NULL,
  waste_category_id INT         NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY (municipality_id, raw_name),
  FOREIGN KEY (municipality_id) REFERENCES municipalities (id),
  FOREIGN KEY (waste_category_id) REFERENCES waste_categories (id)
);

CREATE TABLE collection_rules (
  id                BIGINT      NOT NULL AUTO_INCREMENT,
  area_id           INT         NOT NULL,
  waste_category_id INT         NOT NULL,
  weekday           TINYINT     NOT NULL,
  nth_weeks         VARCHAR(16) NULL,
  raw_text          VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  KEY (area_id),
  FOREIGN KEY (area_id) REFERENCES areas (id),
  FOREIGN KEY (waste_category_id) REFERENCES waste_categories (id)
);
