#Compilarlo y build
go build -o jev-simulator main.go
#Correrlo
./jev-simulator -port 8080 -token "mi_token_secreto_123"
#Probarlo
curl -X POST http://localhost:8080/v1/systemone \
  -H "Authorization: Bearer mi_token_secreto_123" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "Telemetry data: temperature is -12C, pressure is nominal, vibration detected at sensor 4.",
    "questions": {
      "is_cooling_active": { "type": "noul" },
      "system_status": { "type": "choice", "choices": ["normal", "warning", "critical"] },
      "estimated_minutes_left": { "type": "score" }
    }
  }'