SELECT
  "$producer" AS gateway_node_id,
  COUNT(*) AS hello_world_count,
  MAX("$when") AS last_seen_at
FROM v_messages
WHERE "$type" = 'HelloWorld'
GROUP BY "$producer"
ORDER BY hello_world_count DESC, gateway_node_id ASC;
