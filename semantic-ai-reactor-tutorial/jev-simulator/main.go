package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Estructuras para imitar la especificación de Jev (TypeSafe AI)
type QuestionDef struct {
	Type    string   `json:"type"`              // "noul", "choice", "score"
	Choices []string `json:"choices,omitempty"` // Requerido si es "choice"
}

type JevRequest struct {
	State     string                 `json:"state"`
	Questions map[string]QuestionDef `json:"questions"`
}

type JevResponse struct {
	Answers map[string]interface{} `json:"answers"`
}

func main() {
	// Configuración por flags (CLI)
	port := flag.String("port", "8080", "Puerto para escuchar")
	expectedToken := flag.String("token", "YOUR_TYPESAFE_API_KEY", "Token de autorización esperado")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	// Definir el manejador de la ruta
	http.HandleFunc("/v1/systemone", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("\n--- [NUEVA PETICIÓN HTTP: %s %s] ---", r.Method, r.URL.Path)

		// 1. Validar Método
		if r.Method != http.MethodPost {
			http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
			log.Printf("❌ Rechazado: Método %s no permitido", r.Method)
			return
		}

		// 2. Validar Autenticación (Bearer Token)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Missing or invalid Authorization header"}`, http.StatusUnauthorized)
			log.Printf("❌ Rechazado: Falta header de autorización")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != *expectedToken {
			http.Error(w, `{"error": "Unauthorized: Invalid API Key"}`, http.StatusUnauthorized)
			log.Printf("❌ Rechazado: Token inválido ('%s')", token)
			return
		}
		log.Printf("✅ Autenticación exitosa (Token aceptado)")

		// 3. Decodificar el Payload
		var req JevRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
			log.Printf("❌ Rechazado: Error al parsear JSON: %v", err)
			return
		}

		log.Printf("📝 State Recibido: %d bytes (Snippet: '%s...')", len(req.State), snippet(req.State, 40))
		log.Printf("❓ Preguntas recibidas: %d", len(req.Questions))

		// 4. Generar Respuestas Simuladas (Mocking)
		answers := make(map[string]interface{})
		for key, q := range req.Questions {
			log.Printf("   -> Procesando pregunta [%s] de tipo: %s", key, q.Type)

			switch q.Type {
			case "noul":
				// Simular una probabilidad alta para validación
				answers[key] = map[string]interface{}{
					"type": "noul",
					"noul": 0.85 + (rand.Float64() * 0.14), // Ej: 0.92
				}
			case "choice":
				selected := "unknown"
				if len(q.Choices) > 0 {
					selected = q.Choices[rand.Intn(len(q.Choices))]
				}
				answers[key] = map[string]interface{}{
					"type":       "choice",
					"choice":     selected,
					"confidence": 0.90 + (rand.Float64() * 0.09),
				}
			case "score":
				answers[key] = map[string]interface{}{
					"type":       "score",
					"score":      rand.Intn(100), // Puntuación aleatoria 0-100
					"confidence": 0.95,
				}
			default:
				answers[key] = map[string]interface{}{
					"error": "unknown type",
				}
			}
		}

		// 5. Enviar Respuesta
		resp := JevResponse{Answers: answers}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		log.Printf("📤 Respuesta enviada exitosamente")
		fmt.Println("--------------------------------------------------")
	})

	addr := ":" + *port
	fmt.Printf("🚀 Jev Simulator (TypeSafe AI Mock) iniciado en http://localhost%s\n", addr)
	fmt.Printf("🔐 Esperando Authorization: Bearer %s\n", *expectedToken)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// Función auxiliar para truncar el log del State
func snippet(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
