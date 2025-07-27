package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GenerateResult struct {
	Text             string
	InputTokenCount  int
	OutputTokenCount int
}

func ListModels(apiKey string) ([]string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	var models []string
	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate models: %w", err)
		}
		if slices.Contains(m.SupportedGenerationMethods, "generateContent") {
			models = append(models, m.Name)
		}
	}
	return models, nil
}

func GenerateText(apiKey, modelName, combinedPrompt string) (*GenerateResult, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	var resp *genai.GenerateContentResponse
	var lastErr error

	for i := 0; i < 2; i++ {
		resp, err = model.GenerateContent(ctx, genai.Text(combinedPrompt))
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		log.Printf("API call failed (attempt %d/2): %v. Retrying in 1 second...", i+1, err)
		time.Sleep(1 * time.Second)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("API request failed after retries: %w", lastErr)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from API")
	}

	resultText, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type")
	}

	usage := resp.UsageMetadata
	inputTokens := 0
	outputTokens := 0
	if usage != nil {
		inputTokens = int(usage.PromptTokenCount)
		outputTokens = int(usage.CandidatesTokenCount)
	}

	return &GenerateResult{
		Text:             string(resultText),
		InputTokenCount:  inputTokens,
		OutputTokenCount: outputTokens,
	}, nil
}
