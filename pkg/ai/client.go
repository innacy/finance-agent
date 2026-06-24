package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNoAPIKey = fmt.Errorf("AI API key not configured")

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

type Client struct {
	cfg    Config
	http   *http.Client
}

type CategorizeRequest struct {
	Merchant    string
	Description string
	Amount      float64
	Type        string
	Channel     string
	Categories  []string
}

type CategorizeResult struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

type CompletionResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: timeout},
	}
}

type AIClient interface {
	CategorizeTransaction(ctx context.Context, req CategorizeRequest) (*CategorizeResult, error)
}

func (c *Client) CategorizeTransaction(ctx context.Context, req CategorizeRequest) (*CategorizeResult, error) {
	if c.cfg.APIKey == "" {
		return nil, ErrNoAPIKey
	}

	prompt := buildPrompt(req)

	body := completionRequest{
		Model: c.cfg.Model,
		Messages: []Message{
			{Role: "system", Content: "You are a financial transaction categorizer. Respond ONLY with valid JSON."},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := c.cfg.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling AI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var completion CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, fmt.Errorf("decoding AI response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("AI returned no choices")
	}

	content := completion.Choices[0].Message.Content
	content = extractJSON(content)

	var result CategorizeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parsing AI category response: %w", err)
	}

	return &result, nil
}

func buildPrompt(req CategorizeRequest) string {
	var sb strings.Builder
	sb.WriteString("Categorize this financial transaction:\n\n")
	sb.WriteString(fmt.Sprintf("Merchant: %s\n", req.Merchant))
	if req.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", req.Description))
	}
	sb.WriteString(fmt.Sprintf("Amount: %.2f\n", req.Amount))
	if req.Type != "" {
		sb.WriteString(fmt.Sprintf("Type: %s\n", req.Type))
	}
	if req.Channel != "" {
		sb.WriteString(fmt.Sprintf("Channel: %s\n", req.Channel))
	}
	if len(req.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("\nAvailable categories: %s\n", strings.Join(req.Categories, ", ")))
	}
	sb.WriteString("\nRespond with JSON: {\"category\": \"<category>\", \"confidence\": <0.0-1.0>, \"reasoning\": \"<brief explanation>\"}")
	return sb.String()
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
