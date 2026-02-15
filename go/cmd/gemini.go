package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

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
