SET CHARACTER_SET_CLIENT = utf8mb4;
SET CHARACTER_SET_CONNECTION = utf8mb4;

USE isuride;

DROP TABLE IF EXISTS settings;
CREATE TABLE settings
(
  name  VARCHAR(30) NOT NULL COMMENT '設定名',
  value TEXT        NOT NULL COMMENT '設定値',
  PRIMARY KEY (name)
)
  COMMENT = 'システム設定テーブル';

DROP TABLE IF EXISTS chair_models;
CREATE TABLE chair_models
(
  name  VARCHAR(50) NOT NULL COMMENT '椅子モデル名',
  speed INTEGER     NOT NULL COMMENT '移動速度',
  PRIMARY KEY (name)
)
  COMMENT = '椅子モデルテーブル';

DROP TABLE IF EXISTS chairs;
CREATE TABLE chairs
(
  id           VARCHAR(26)  NOT NULL COMMENT '椅子ID',
  owner_id     VARCHAR(26)  NOT NULL COMMENT 'オーナーID',
  name         VARCHAR(30)  NOT NULL COMMENT '椅子の名前',
  model        TEXT         NOT NULL COMMENT '椅子のモデル',
  is_active    TINYINT(1)   NOT NULL COMMENT '配椅子受付中かどうか',
  access_token VARCHAR(255) NOT NULL COMMENT 'アクセストークン',
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '登録日時',
  updated_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '更新日時',
  PRIMARY KEY (id),
  KEY owner_id(owner_id)
)
  COMMENT = '椅子情報テーブル';

DROP TABLE IF EXISTS chair_locations;
CREATE TABLE chair_locations
(
  id         VARCHAR(26) NOT NULL,
  chair_id   VARCHAR(26) NOT NULL COMMENT '椅子ID',
  latitude   INTEGER     NOT NULL COMMENT '経度',
  longitude  INTEGER     NOT NULL COMMENT '緯度',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '登録日時',
  PRIMARY KEY (id),
  KEY chair_locations(chair_id, created_at),
  KEY latitude_longitude(latitude, longitude)
)
  COMMENT = '椅子の現在位置情報テーブル';

DROP TABLE IF EXISTS users;
CREATE TABLE users
(
  id              VARCHAR(26)  NOT NULL COMMENT 'ユーザーID',
  username        VARCHAR(30)  NOT NULL COMMENT 'ユーザー名',
  firstname       VARCHAR(30)  NOT NULL COMMENT '本名(名前)',
  lastname        VARCHAR(30)  NOT NULL COMMENT '本名(名字)',
  date_of_birth   VARCHAR(30)  NOT NULL COMMENT '生年月日',
  access_token    VARCHAR(255) NOT NULL COMMENT 'アクセストークン',
  invitation_code VARCHAR(30)  NOT NULL COMMENT '招待トークン',
  created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '登録日時',
  updated_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '更新日時',
  PRIMARY KEY (id),
  UNIQUE (username),
  UNIQUE (access_token),
  UNIQUE (invitation_code)
)
  COMMENT = '利用者情報テーブル';

DROP TABLE IF EXISTS payment_tokens;
CREATE TABLE payment_tokens
(
  user_id    VARCHAR(26)  NOT NULL COMMENT 'ユーザーID',
  token      VARCHAR(255) NOT NULL COMMENT '決済トークン',
  created_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '登録日時',
  PRIMARY KEY (user_id)
)
  COMMENT = '決済トークンテーブル';

DROP TABLE IF EXISTS rides;
CREATE TABLE rides
(
  id                    VARCHAR(26) NOT NULL COMMENT 'ライドID',
  user_id               VARCHAR(26) NOT NULL COMMENT 'ユーザーID',
  chair_id              VARCHAR(26) NULL     COMMENT '割り当てられた椅子ID',
  pickup_latitude       INTEGER     NOT NULL COMMENT '配車位置(経度)',
  pickup_longitude      INTEGER     NOT NULL COMMENT '配車位置(緯度)',
  destination_latitude  INTEGER     NOT NULL COMMENT '目的地(経度)',
  destination_longitude INTEGER     NOT NULL COMMENT '目的地(緯度)',
  evaluation            INTEGER     NULL     COMMENT '評価',
  created_at            DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '要求日時',
  updated_at            DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '状態更新日時',
  PRIMARY KEY (id)
)
  COMMENT = 'ライド情報テーブル';

DROP TABLE IF EXISTS ride_statuses;
CREATE TABLE ride_statuses
(
  id              VARCHAR(26)                                                                NOT NULL,
  ride_id VARCHAR(26)                                                                        NOT NULL COMMENT 'ライドID',
  status          ENUM ('MATCHING', 'ENROUTE', 'PICKUP', 'CARRYING', 'ARRIVED', 'COMPLETED') NOT NULL COMMENT '状態',
  created_at      DATETIME(6)                                                                NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '状態変更日時',
  app_sent_at     DATETIME(6)                                                                NULL COMMENT 'ユーザーへの状態通知日時',
  chair_sent_at   DATETIME(6)                                                                NULL COMMENT '椅子への状態通知日時',
  PRIMARY KEY (id),
  KEY ride_id_created_at(ride_id, created_at DESC)
)
  COMMENT = 'ライドステータスの変更履歴テーブル';

DROP TABLE IF EXISTS owners;
CREATE TABLE owners
(
  id                   VARCHAR(26)  NOT NULL COMMENT 'オーナーID',
  name                 VARCHAR(30)  NOT NULL COMMENT 'オーナー名',
  access_token         VARCHAR(255) NOT NULL COMMENT 'アクセストークン',
  chair_register_token VARCHAR(255) NOT NULL COMMENT '椅子登録トークン',
  created_at           DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '登録日時',
  updated_at           DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6) COMMENT '更新日時',
  PRIMARY KEY (id),
  UNIQUE (name),
  UNIQUE (access_token),
  UNIQUE (chair_register_token)
)
  COMMENT = '椅子のオーナー情報テーブル';

DROP TABLE IF EXISTS coupons;
CREATE TABLE coupons
(
  user_id    VARCHAR(26)  NOT NULL COMMENT '所有しているユーザーのID',
  code       VARCHAR(255) NOT NULL COMMENT 'クーポンコード',
  discount   INTEGER      NOT NULL COMMENT '割引額',
  created_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '付与日時',
  used_by    VARCHAR(26)  NULL COMMENT 'クーポンが適用されたライドのID',
  PRIMARY KEY (user_id, code)
)
  COMMENT 'クーポンテーブル';

DROP TABLE  IF EXISTS `distance_table`;
CREATE TABLE distance_table
(
    chair_id                  VARCHAR(26)  NOT NULL COMMENT '椅子ID',
    total_distance            INT          NOT NULL,
    total_distance_updated_at DATETIME(6)  NOT NULL,
    PRIMARY KEY (chair_id)
);

DROP TABLE  IF EXISTS `tmp_distance_table`;
CREATE TABLE tmp_distance_table (
    chair_id          VARCHAR(26) NOT NULL,
    created_at        DATETIME(6) NOT NULL,
    prev_latitude     INTEGER NULL,
    prev_longitude    INTEGER NULL,
    current_latitude  INTEGER NOT NULL,
    current_longitude INTEGER NOT NULL,
    distance          FLOAT NOT NULL,
    PRIMARY KEY (chair_id, created_at)
);

DROP TRIGGER IF EXISTS after_insert_chair_locations;
DROP TRIGGER IF EXISTS after_insert_tmp_distance_table;
DELIMITER //

CREATE TRIGGER after_insert_chair_locations
AFTER INSERT ON chair_locations
FOR EACH ROW
BEGIN
    DECLARE prev_lat INTEGER;
    DECLARE prev_long INTEGER;
    DECLARE distance FLOAT;

    -- 前回の座標を取得 (同じ chair_id の最新データ)
    SELECT latitude, longitude
    INTO prev_lat, prev_long
    FROM chair_locations
    WHERE chair_id = NEW.chair_id
      AND created_at < NEW.created_at
    ORDER BY created_at DESC
    LIMIT 1;

    -- 距離を計算 (前回の座標がある場合のみ)
    IF prev_lat IS NOT NULL AND prev_long IS NOT NULL THEN
        SET distance = ABS(NEW.latitude - prev_lat) + ABS(NEW.longitude - prev_long);
    ELSE
        SET distance = 0;  -- 最初のデータの場合は距離を0に設定
    END IF;

    -- tmp_distance_table に挿入
    INSERT INTO tmp_distance_table (chair_id, created_at, prev_latitude, prev_longitude, 
                                    current_latitude, current_longitude, distance)
    VALUES (NEW.chair_id, NEW.created_at, prev_lat, prev_long, NEW.latitude, NEW.longitude, distance);
END //

CREATE TRIGGER after_insert_tmp_distance_table
AFTER INSERT ON tmp_distance_table
FOR EACH ROW
BEGIN
    DECLARE total_distance INT;
    DECLARE total_distance_updated_at DATETIME(6);

    -- tmp_distance_table から total_distance を計算
    SELECT SUM(IFNULL(distance, 0)), MAX(created_at)
    INTO total_distance, total_distance_updated_at
    FROM tmp_distance_table
    WHERE chair_id = NEW.chair_id;

    -- distance_table を更新
    IF EXISTS (SELECT 1 FROM distance_table WHERE chair_id = NEW.chair_id) THEN
        UPDATE distance_table
        SET total_distance = total_distance,
            total_distance_updated_at = total_distance_updated_at
        WHERE chair_id = NEW.chair_id;
    ELSE
        INSERT INTO distance_table (chair_id, total_distance, total_distance_updated_at)
        VALUES (NEW.chair_id, total_distance, total_distance_updated_at);
    END IF;
END //

DELIMITER ;