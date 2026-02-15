package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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

	saveToMockVectorDB(req.Vector, answer, req.Text)
	appendHistory(req.Text, answer, false, "CLOUD", modelName)

	writeJSON(w, http.StatusOK, Response{
		Answer: answer,
		Source: "CLOUD",
	})
}
