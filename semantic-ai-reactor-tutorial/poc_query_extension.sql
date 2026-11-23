-- Combined Vaccine Risk & Compressor Integrity Cockpit Query
-- This query retrieves the latest StorageRiskAssessment and MachineryHealthAssessment
-- side-by-side for each active freezer across the mesh, joining them by their gateway node.

WITH LatestRisk AS (
  SELECT 
    "$producer" AS gateway_node,
    json_extract(raw_json, '$.fields.target_producer') AS target_producer,
    json_extract(raw_json, '$.fields.risk_level') AS vaccine_risk,
    CAST(json_extract(raw_json, '$.fields.estimated_minutes_to_spoilage') AS INTEGER) AS minutes_spoilage,
    json_extract(raw_json, '$.fields.root_cause_prediction') AS spoilage_reason,
    "$when" AS risk_assessed_at,
    ROW_NUMBER() OVER (PARTITION BY "$producer" ORDER BY datetime("$when") DESC) as rn
  FROM v_messages
  WHERE "$type" = 'StorageRiskAssessment'
),
LatestMechanical AS (
  SELECT 
    "$producer" AS gateway_node,
    COALESCE(json_extract(raw_json, '$.fields.target_producer'), "$producer") AS target_producer,
    json_extract(raw_json, '$.fields.health_status') AS compressor_health,
    CAST(json_extract(raw_json, '$.fields.predicted_days_to_failure') AS INTEGER) AS days_to_failure,
    json_extract(raw_json, '$.fields.fault_prediction') AS machinery_fault,
    json_extract(raw_json, '$.fields.action_priority') AS repair_priority,
    "$when" AS mechanical_assessed_at,
    ROW_NUMBER() OVER (PARTITION BY "$producer" ORDER BY datetime("$when") DESC) as rn
  FROM v_messages
  WHERE "$type" = 'MachineryHealthAssessment'
),
FilteredRisk AS (
  SELECT * FROM LatestRisk WHERE rn = 1
),
FilteredMechanical AS (
  SELECT * FROM LatestMechanical WHERE rn = 1
)
SELECT * FROM (
  SELECT 
    COALESCE(r.target_producer, m.target_producer, r.gateway_node) AS asset_id,
    COALESCE(r.vaccine_risk, 'NOMINAL') AS vaccine_risk_status,
    r.minutes_spoilage AS minutes_spoilage,
    r.spoilage_reason AS spoilage_reason,
    COALESCE(m.compressor_health, 'NOMINAL') AS compressor_health_status,
    m.days_to_failure AS compressor_days_to_failure,
    m.machinery_fault AS mechanical_fault_details,
    m.repair_priority AS technician_dispatch_priority,
    CASE 
      WHEN r.risk_assessed_at IS NULL THEN m.mechanical_assessed_at
      WHEN m.mechanical_assessed_at IS NULL THEN r.risk_assessed_at
      WHEN r.risk_assessed_at > m.mechanical_assessed_at THEN r.risk_assessed_at
      ELSE m.mechanical_assessed_at
    END AS last_updated
  FROM FilteredRisk r
  LEFT JOIN FilteredMechanical m ON r.gateway_node = m.gateway_node
  UNION
  SELECT 
    COALESCE(r.target_producer, m.target_producer, m.gateway_node) AS asset_id,
    COALESCE(r.vaccine_risk, 'NOMINAL') AS vaccine_risk_status,
    r.minutes_spoilage AS minutes_spoilage,
    r.spoilage_reason AS spoilage_reason,
    COALESCE(m.compressor_health, 'NOMINAL') AS compressor_health_status,
    m.days_to_failure AS compressor_days_to_failure,
    m.machinery_fault AS mechanical_fault_details,
    m.repair_priority AS technician_dispatch_priority,
    CASE 
      WHEN r.risk_assessed_at IS NULL THEN m.mechanical_assessed_at
      WHEN m.mechanical_assessed_at IS NULL THEN r.risk_assessed_at
      WHEN r.risk_assessed_at > m.mechanical_assessed_at THEN r.risk_assessed_at
      ELSE m.mechanical_assessed_at
    END AS last_updated
  FROM FilteredMechanical m
  LEFT JOIN FilteredRisk r ON m.gateway_node = r.gateway_node
)
ORDER BY 
  CASE 
    WHEN vaccine_risk_status = 'CRITICAL' OR compressor_health_status = 'FAILURE_IMMINENT' THEN 0 
    WHEN vaccine_risk_status = 'HIGH' OR compressor_health_status = 'CRITICAL' THEN 1
    ELSE 2 
  END,
  asset_id;
