// main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	Source     string
}

type HistoryItem struct {
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Timestamp time.Time `json:"timestamp"`
	Saved     bool      `json:"saved"` // true if it came from cache
	Source    string    `json:"source,omitempty"`
	Model     string    `json:"model,omitempty"`
	Tokens    int       `json:"tokensSaved,omitempty"`
	EnergyWh  float64   `json:"energySavedWh,omitempty"`
	CO2g      float64   `json:"co2SavedG,omitempty"`
}

type CacheEntryView struct {
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
}

type CacheUseView struct {
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Tokens    int       `json:"tokensSaved"`
	EnergyWh  float64   `json:"energySavedWh"`
	CO2g      float64   `json:"co2SavedG"`
}

type EnvironmentalStats struct {
	CacheHits            int     `json:"cacheHits"`
	LocalCacheHits       int     `json:"localCacheHits"`
	S3CacheHits          int     `json:"s3CacheHits"`
	EstimatedTokensSaved int     `json:"estimatedTokensSaved"`
	EnergySavedWh        float64 `json:"energySavedWh"`
	CO2SavedG            float64 `json:"co2SavedG"`
}

type EnergyConstants struct {
	DefaultKWhPer1KTokens float64            `json:"defaultKWhPer1KTokens"`
	GridCO2gPerKWh        float64            `json:"gridCO2gPerKWh"`
	ModelKWhPer1KTokens   map[string]float64 `json:"modelKWhPer1KTokens"`
}

type CacheStatsResponse struct {
	Uploading      bool               `json:"uploading"`
	LastUploadAt   *time.Time         `json:"lastUploadAt,omitempty"`
	LastDownloadAt *time.Time         `json:"lastDownloadAt,omitempty"`
	Metrics        EnvironmentalStats `json:"metrics"`
	Constants      EnergyConstants    `json:"constants"`
	LocalRamCache  []CacheEntryView   `json:"localRamCache"`
	S3CacheUsed    []CacheUseView     `json:"s3CacheUsed"`
}

var (
	MockVectorDB []VectorEntry
	ChatHistory  []HistoryItem
	dbMutex      sync.RWMutex
	s3Client     *s3.Client
	s3BucketName string

	statusMutex      sync.RWMutex
	s3Uploading      bool
	lastS3UploadAt   time.Time
	hasLastS3Upload  bool
	lastS3DownloadAt time.Time
	hasLastS3Sync    bool
)

const similarityThreshold = 0.90
const cacheObjectKey = "cache.json"
const (
	cacheSourceLocal = "LOCAL"
	cacheSourceS3    = "S3"
	// default if model not known
	estimatedKWhPer1KTokens = 0.00035
	gridCO2gPerKWh          = 475.0
)

// Per-model energy constants (kWh per 1k tokens).
var modelKWhPer1KTokens = map[string]float64{
	"gemini-2.5-flash-lite": 0.00020,
	"gemini-2.5-flash":      0.00035,
}

func estimateTokens(text string) int {
	runeCount := utf8.RuneCountInString(text)
	tokens := int(math.Ceil(float64(runeCount) / 4.0))
	if tokens < 1 {
		return 1
	}
	return tokens
}

func estimateSavings(question, answer, model string) (int, float64, float64) {
	tokens := estimateTokens(question) + estimateTokens(answer)
	kWhPer1K, ok := modelKWhPer1KTokens[model]
	if !ok || kWhPer1K <= 0 {
		kWhPer1K = estimatedKWhPer1KTokens
	}
	kWh := (float64(tokens) / 1000.0) * kWhPer1K
	energyWh := kWh * 1000.0
	co2g := kWh * gridCO2gPerKWh
	return tokens, energyWh, co2g
}

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
		Source:    cacheSourceLocal,
	})
}

func appendHistory(question, answer string, saved bool, source string, model string) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	tokens, energyWh, co2g := 0, 0.0, 0.0
	if saved {
		tokens, energyWh, co2g = estimateSavings(question, answer, model)
	}

	ChatHistory = append(ChatHistory, HistoryItem{
		Question:  question,
		Answer:    answer,
		Timestamp: time.Now(),
		Saved:     saved,
		Source:    source,
		Model:     model,
		Tokens:    tokens,
		EnergyWh:  energyWh,
		CO2g:      co2g,
	})
}

func setUploading(uploading bool) {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	s3Uploading = uploading
}

func markS3UploadCompleted() {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	lastS3UploadAt = time.Now()
	hasLastS3Upload = true
}

func markS3DownloadCompleted() {
	statusMutex.Lock()
	defer statusMutex.Unlock()
	lastS3DownloadAt = time.Now()
	hasLastS3Sync = true
}

func initS3Client() error {
	s3BucketName = strings.TrimSpace(os.Getenv("S3_BUCKET_NAME"))
	if s3BucketName == "" {
		return errors.New("S3_BUCKET_NAME is required")
	}

	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		return errors.New("AWS_REGION is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	s3Client = s3.NewFromConfig(cfg)
	return nil
}

func downloadAndMergeFromS3() {
	if s3Client == nil || s3BucketName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(cacheObjectKey),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			log.Println("S3 cache.json not found; starting with empty cache")
			return
		}
		log.Printf("S3 download failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Read S3 cache body failed: %v", err)
		return
	}

	if len(body) == 0 {
		return
	}

	var remoteEntries []VectorEntry
	if err := json.Unmarshal(body, &remoteEntries); err != nil {
		log.Printf("Decode S3 cache.json failed: %v", err)
		return
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	existingByQuestion := make(map[string]struct{}, len(MockVectorDB))
	for _, entry := range MockVectorDB {
		questionKey := strings.TrimSpace(entry.Question)
		if questionKey != "" {
			existingByQuestion[questionKey] = struct{}{}
		}
	}

	newEntries := 0
	for _, entry := range remoteEntries {
		questionKey := strings.TrimSpace(entry.Question)
		if questionKey == "" {
			continue
		}
		if _, exists := existingByQuestion[questionKey]; exists {
			continue
		}
		entry.Source = cacheSourceS3
		MockVectorDB = append(MockVectorDB, entry)
		existingByQuestion[questionKey] = struct{}{}
		newEntries++
	}

	markS3DownloadCompleted()
	log.Printf("Synced: %d new entries found.", newEntries)
}

func uploadToS3() {
	if s3Client == nil || s3BucketName == "" {
		return
	}

	setUploading(true)
	defer setUploading(false)

	dbMutex.RLock()
	payload := make([]VectorEntry, len(MockVectorDB))
	copy(payload, MockVectorDB)
	dbMutex.RUnlock()

	jsonBody, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("Marshal cache for S3 failed: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3BucketName),
		Key:         aws.String(cacheObjectKey),
		Body:        bytes.NewReader(jsonBody),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		log.Printf("S3 upload failed: %v", err)
		return
	}

	markS3UploadCompleted()
}

func startBackgroundSync() {
	ticker := time.NewTicker(300 * time.Second)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			downloadAndMergeFromS3()

			dbMutex.RLock()
			hasData := len(MockVectorDB) > 0
			dbMutex.RUnlock()

			if hasData {
				fmt.Println("Batching: Uploading memory to S3...")
				uploadToS3()
			}
		}
	}()
}

func resolveGeminiModel(requested string) string {
	requested = strings.TrimSpace(requested)
	allowed := map[string]struct{}{
		"gemini-2.5-flash-lite": {},
		"gemini-2.5-flash":      {},
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

	modelName := resolveGeminiModel(req.Model)

	if match, ok := findBestMatch(req.Vector); ok {
		fmt.Printf("Cache hit! similarity=%.4f\n", match.Similarity)
		source := match.Source
		if source == "" {
			source = cacheSourceLocal
		}
		appendHistory(req.Text, match.Answer, true, source, modelName)
		writeJSON(w, http.StatusOK, Response{
			Answer: match.Answer,
			Source: "CACHE",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	answer, err := callGemini(ctx, req.Text, modelName)
	if err != nil {
		fmt.Printf("Gemini error: %v\n", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to generate response from Gemini"})
		return
	}

	saveToMockVectorDB(req.Vector, answer, req.Text) // Save the question text along with the answer
	appendHistory(req.Text, answer, false, "CLOUD", modelName)

	writeJSON(w, http.StatusOK, Response{
		Answer: answer,
		Source: "CLOUD",
	})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	dbMutex.RLock()
	defer dbMutex.RUnlock()

	history := make([]HistoryItem, 0, len(ChatHistory))
	for i := len(ChatHistory) - 1; i >= 0; i-- {
		history = append(history, ChatHistory[i])
	}

	writeJSON(w, http.StatusOK, history)
}

func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	dbMutex.RLock()
	entries := make([]VectorEntry, len(MockVectorDB))
	copy(entries, MockVectorDB)
	history := make([]HistoryItem, len(ChatHistory))
	copy(history, ChatHistory)
	dbMutex.RUnlock()

	statusMutex.RLock()
	uploading := s3Uploading
	var lastUploadAt *time.Time
	if hasLastS3Upload {
		t := lastS3UploadAt
		lastUploadAt = &t
	}
	var lastDownloadAt *time.Time
	if hasLastS3Sync {
		t := lastS3DownloadAt
		lastDownloadAt = &t
	}
	statusMutex.RUnlock()

	localRamCache := make([]CacheEntryView, 0)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		source := entry.Source
		if source == "" {
			source = cacheSourceLocal
		}
		if source == cacheSourceLocal {
			localRamCache = append(localRamCache, CacheEntryView{
				Question:  entry.Question,
				Answer:    entry.Answer,
				Source:    source,
				CreatedAt: entry.CreatedAt,
			})
		}
	}

	s3CacheUsed := make([]CacheUseView, 0)
	metrics := EnvironmentalStats{}
	constants := EnergyConstants{
		DefaultKWhPer1KTokens: estimatedKWhPer1KTokens,
		GridCO2gPerKWh:        gridCO2gPerKWh,
		ModelKWhPer1KTokens:   modelKWhPer1KTokens,
	}

	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		if !item.Saved {
			continue
		}

		metrics.CacheHits++
		metrics.EstimatedTokensSaved += item.Tokens
		metrics.EnergySavedWh += item.EnergyWh
		metrics.CO2SavedG += item.CO2g

		source := item.Source
		if source == "" {
			source = cacheSourceLocal
		}
		if source == cacheSourceS3 {
			metrics.S3CacheHits++
			s3CacheUsed = append(s3CacheUsed, CacheUseView{
				Question:  item.Question,
				Answer:    item.Answer,
				Source:    source,
				Timestamp: item.Timestamp,
				Tokens:    item.Tokens,
				EnergyWh:  item.EnergyWh,
				CO2g:      item.CO2g,
			})
		} else {
			metrics.LocalCacheHits++
		}
	}

	writeJSON(w, http.StatusOK, CacheStatsResponse{
		Uploading:      uploading,
		LastUploadAt:   lastUploadAt,
		LastDownloadAt: lastDownloadAt,
		Metrics:        metrics,
		Constants:      constants,
		LocalRamCache:  localRamCache,
		S3CacheUsed:    s3CacheUsed,
	})
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file (ignoring if running in cloud/docker)")
	}

	if err := initS3Client(); err != nil {
		log.Printf("Warning: S3 disabled: %v", err)
	} else {
		downloadAndMergeFromS3()
		startBackgroundSync()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/chat", handleChat)
	mux.HandleFunc("/history", handleHistory)
	mux.HandleFunc("/cache-stats", handleCacheStats)

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
