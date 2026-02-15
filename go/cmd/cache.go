package main

import (
	"encoding/json"
	"math"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"
)

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
	Saved     bool      `json:"saved"`
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
	statusMutex  sync.RWMutex
)

const similarityThreshold = 0.90
const (
	cacheSourceLocal = "LOCAL"
	cacheSourceS3    = "S3"

	estimatedKWhPer1KTokens = 0.00035
	gridCO2gPerKWh          = 475.0
)

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
		Question:  question,
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
