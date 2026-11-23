WITH ranked AS (
  SELECT
    "$producer" AS node_id,
    "$when" AS updated_at,
    "$where" AS where_json,
    json_extract(fields, '$.color') AS color,
    json_extract(fields, '$.address') AS address,
    ROW_NUMBER() OVER (
      PARTITION BY "$producer"
      ORDER BY datetime("$when") DESC, "$id" DESC
    ) AS rn
  FROM v_messages
  WHERE "$type" = 'NodeGeoLocation'
)
SELECT
  node_id,
  updated_at,
  where_json,
  json_extract(where_json, '$.locagram') AS locagram,
  json_extract(where_json, '$.geojson') AS geojson,
  color,
  address
FROM ranked
WHERE rn = 1
ORDER BY node_id ASC;
