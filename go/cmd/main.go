// main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/google/generative-ai-go/genai"
	"github.com/rs/cors"
	"google.golang.org/api/option"
)

type Request struct {
	Text   string    `json:"text"`
	Vector []float32 `json:"vector"`
	Model  string    `json:"model,omitempty"`
}

type Response struct {
	Answer string `json:"answer"`
	Source string `json:"source"`
}

type VectorEntry struct {
	Vector     []float32
	Answer     string
	Question   string
	CreatedAt  time.Time
	Similarity float64
}

type HistoryItem struct {
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Timestamp time.Time `json:"timestamp"`
	Saved     bool      `json:"saved"` // true if it came from cache
}

var (
	MockVectorDB []VectorEntry
	dbMutex      sync.RWMutex
)

const similarityThreshold = 0.90

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		av := float64(a[i])
		bv := float64(b[i])
		dotProduct += av * bv
		normA += av * av
		normB += bv * bv
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func findBestMatch(vector []float32) (VectorEntry, bool) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	bestScore := 0.0
	var best VectorEntry

	for _, entry := range MockVectorDB {
		score := cosineSimilarity(vector, entry.Vector)
		if score > bestScore {
			bestScore = score
			best = entry
		}
	}

	if bestScore >= similarityThreshold {
		best.Similarity = bestScore
		return best, true
	}

	return VectorEntry{}, false
}

func saveToMockVectorDB(vector []float32, answer string, question string) {
	copyVector := make([]float32, len(vector))
	copy(copyVector, vector)

	dbMutex.Lock()
	defer dbMutex.Unlock()

	MockVectorDB = append(MockVectorDB, VectorEntry{
		Vector:    copyVector,
		Answer:    answer,
		Question:  question, // In a real app, this would be passed in from the request
		CreatedAt: time.Now(),
	})
}

func resolveGeminiModel(requested string) string {
	requested = strings.TrimSpace(requested)
	allowed := map[string]struct{}{
		"gemini-2.5-flash-lite": {},
		"gemini-2.5-flash":      {},
		"gemini-1.5-flash":      {},
	}
	if _, ok := allowed[requested]; ok {
		return requested
	}
	return "gemini-2.5-flash-lite"
}

func callGemini(ctx context.Context, prompt string, modelName string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", errors.New("GEMINI_API_KEY is not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("create Gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("Gemini generate content: %w", err)
	}

	var answerBuilder strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if textPart, ok := part.(genai.Text); ok {
				answerBuilder.WriteString(string(textPart))
			}
		}
	}

	answer := strings.TrimSpace(answerBuilder.String())
	if answer == "" {
		return "", errors.New("Gemini returned empty response")
	}

	return answer, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}

	if strings.TrimSpace(req.Text) == "" || len(req.Vector) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text and vector are required"})
		return
	}

	fmt.Printf("Received Vector from Browser! Length: %d\n", len(req.Vector))

	if match, ok := findBestMatch(req.Vector); ok {
		fmt.Printf("Cache hit! similarity=%.4f\n", match.Similarity)
		writeJSON(w, http.StatusOK, Response{
			Answer: match.Answer,
			Source: "CACHE",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	modelName := resolveGeminiModel(req.Model)
	answer, err := callGemini(ctx, req.Text, modelName)
	if err != nil {
		fmt.Printf("Gemini error: %v\n", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to generate response from Gemini"})
		return
	}

	saveToMockVectorDB(req.Vector, answer, req.Text) // Save the question text along with the answer

	writeJSON(w, http.StatusOK, Response{
		Answer: answer,
		Source: "CLOUD",
	})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	// Convert VectorEntry to simple HistoryItem
	// We iterate backwards to show newest first
	history := []HistoryItem{}
	for i := len(MockVectorDB) - 1; i >= 0; i-- {
		entry := MockVectorDB[i]
		history = append(history, HistoryItem{
			Question:  entry.Question, // Now we store the question text in VectorEntry
			Answer:    entry.Answer,
			Timestamp: entry.CreatedAt,
		})
	}

	// Write JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file (ignoring if running in cloud/docker)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/history", handleHistory)

	// Enable CORS so SvelteKit on port 5173 can call this API
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}).Handler(mux)

	fmt.Println("Echo backend listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
