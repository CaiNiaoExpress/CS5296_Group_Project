package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const DefaultOpenRouterModel = "gpt-4o-mini"

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	siteURL    string
	siteName   string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenRouterClientFromEnv() *Client {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseURL := getEnv("OPENROUTER_BASE_URL", "https://api.chatanywhere.tech/v1")
	model := getEnv("OPENROUTER_MODEL", DefaultOpenRouterModel)
	timeout := time.Duration(getEnvInt("OPENROUTER_TIMEOUT_SEC", 45)) * time.Second
	siteURL := getEnv("OPENROUTER_SITE_URL", "http://localhost:8080")
	siteName := getEnv("OPENROUTER_SITE_NAME", "car-mall-intelligent-agent")

	// Normalize base URL (remove trailing slash, handle API endpoint correctly)
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.Contains(baseURL, "/chat/completions") {
		// If baseURL already includes full path, extract just the base
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
		log.Printf("Warning: OPENROUTER_BASE_URL contains /chat/completions - automatically normalized to: %s", baseURL)
	}

	log.Printf("OpenRouter Client Configuration:")
	log.Printf("  API Key: %s", maskAPIKey(apiKey))
	log.Printf("  Base URL: %s", baseURL)
	log.Printf("  Model: %s", model)
	log.Printf("  Timeout: %v", timeout)
	log.Printf("  Site URL: %s", siteURL)
	log.Printf("  Site Name: %s", siteName)

	if apiKey == "" {
		log.Println("Warning: OPENROUTER_API_KEY is not set - OpenRouter integration will be disabled")
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
		siteURL:    siteURL,
		siteName:   siteName,
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != ""
}

func (c *Client) Model() string {
	if c == nil {
		return DefaultOpenRouterModel
	}
	return c.model
}

func (c *Client) GenerateSalesReply(
	ctx context.Context,
	userMessage string,
	intentName string,
	lastIntent string,
	lastReply string,
	candidateCars string,
) (string, error) {
	if !c.Enabled() {
		return "", errors.New("openrouter is not configured")
	}

	systemPrompt := strings.Join([]string{
		"You are an intelligent sales consultant for the CarMall automotive e-commerce platform.",
		"Please respond to users in concise and professional English, keeping it to 2-4 sentences.",
		"If users are asking about prices, provide suggestions around discounts, quotes, and model configurations.",
		"If users are asking about loans, provide suggestions around down payments, monthly payments, and interest rates.",
		"If users are preparing to purchase, encourage test drives, ordering, and delivery inquiries.",
		"Do not invent precise inventory, financial policies, or delivery dates that the platform does not provide.",
	}, "\n")

	userPrompt := fmt.Sprintf(
		"Current recognized intent: %s\nPrevious round intent: %s\nPrevious round reply: %s\nCandidate models: %s\nUser message: %s",
		intentName,
		emptyFallback(lastIntent, "none"),
		emptyFallback(lastReply, "none"),
		emptyFallback(candidateCars, "no candidate models"),
		userMessage,
	)

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Build full URL by adding /chat/completions to base URL
	fullURL := c.baseURL + "/chat/completions"
	log.Printf("Making request to: %s", fullURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		if out.Error != nil && out.Error.Message != "" {
			return "", errors.New(out.Error.Message)
		}
		return "", fmt.Errorf("openrouter request failed: status=%d", resp.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openrouter returned empty choices")
	}

	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("openrouter returned empty content")
	}
	return content, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
