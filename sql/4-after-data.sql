ALTER TABLE chairs ADD COLUMN latitude  INTEGER DEFAULT NULL COMMENT '経度';
ALTER TABLE chairs ADD COLUMN longitude INTEGER DEFAULT NULL COMMENT '緯度';
ALTER TABLE chairs ADD KEY latitude_longitude(latitude, longitude);

UPDATE chairs c
JOIN (
    SELECT chair_id, latitude, longitude
    FROM chair_locations
    WHERE (chair_id, created_at) IN (
        SELECT chair_id, MAX(created_at)
        FROM chair_locations
        GROUP BY chair_id
    )
) cl ON c.id = cl.chair_id
SET c.latitude = cl.latitude,
    c.longitude = cl.longitude;

DROP TRIGGER IF EXISTS after_insert_chair_locations2;
DROP TRIGGER IF EXISTS after_update_chair_locations;

DELIMITER //

CREATE TRIGGER after_update_chair_locations
AFTER UPDATE ON chair_locations
FOR EACH ROW
BEGIN
    -- 最新の `latitude` と `longitude` を取得
    DECLARE latest_latitude INT;
    DECLARE latest_longitude INT;

    SELECT latitude, longitude
    INTO latest_latitude, latest_longitude
    FROM chair_locations
    WHERE chair_id = NEW.chair_id
    ORDER BY created_at DESC
    LIMIT 1;

    -- `chairs` テーブルを更新
    UPDATE chairs
    SET latitude = latest_latitude,
        longitude = latest_longitude
    WHERE id = NEW.chair_id;
END //

CREATE TRIGGER after_insert_chair_locations2
AFTER INSERT ON chair_locations
FOR EACH ROW
BEGIN
    -- 最新の `latitude` と `longitude` を取得
    DECLARE latest_latitude INT;
    DECLARE latest_longitude INT;

    SELECT latitude, longitude
    INTO latest_latitude, latest_longitude
    FROM chair_locations
    WHERE chair_id = NEW.chair_id
    ORDER BY created_at DESC
    LIMIT 1;

    -- `chairs` テーブルを更新
    UPDATE chairs
    SET latitude = latest_latitude,
        longitude = latest_longitude
    WHERE id = NEW.chair_id;
END //

DELIMITER ;
