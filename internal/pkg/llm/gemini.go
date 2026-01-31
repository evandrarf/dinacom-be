package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type GeminiClient struct {
	APIKey  string
	BaseURL string
	Model   string
	client  *openai.Client
}

func NewGeminiClient(apiKey string, model string, baseURL string) *GeminiClient {
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &GeminiClient{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		client:  openai.NewClientWithConfig(config),
	}
}

func (c *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("client not initialized")
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.3,
			TopP:        0.95,
			MaxTokens:   2048 * 4,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("openai generate error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	text := resp.Choices[0].Message.Content
	if text == "" {
		return "", fmt.Errorf("openai returned empty response")
	}

	return text, nil
}

// GenerateChatResponse generates plain text response for chatbot (no JSON formatting)
func (c *GeminiClient) GenerateChatResponse(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("client not initialized")
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.Model,
			Messages:    messages,
			Temperature: 0.7,
			TopP:        0.95,
			MaxTokens:   2048,
			// No ResponseFormat - allow plain text response
		},
	)
	if err != nil {
		return "", fmt.Errorf("openai chat error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	text := resp.Choices[0].Message.Content
	if text == "" {
		return "", fmt.Errorf("openai returned empty response")
	}

	return text, nil
}
