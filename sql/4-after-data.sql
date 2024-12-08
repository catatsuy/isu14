ALTER TABLE chairs ADD COLUMN latitude  INTEGER NOT NULL DEFAULT 0 COMMENT '経度';
ALTER TABLE chairs ADD COLUMN longitude INTEGER NOT NULL DEFAULT 0 COMMENT '緯度';
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
