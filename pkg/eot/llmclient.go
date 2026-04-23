package eot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatMessage is a single OpenAI-style chat message.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMClient talks to any OpenAI-compatible /v1/chat/completions endpoint.
type LLMClient struct {
	Config *LLMConfig
	HTTP   *http.Client
}

// NewLLMClient constructs a client. Pass nil to auto-load config from env/files.
func NewLLMClient(cfg *LLMConfig) (*LLMClient, error) {
	if cfg == nil {
		c, err := LoadConfig()
		if err != nil {
			return nil, err
		}
		cfg = c
	}
	return &LLMClient{
		Config: cfg,
		HTTP:   &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// ChatOptions tweaks a single chat call.
type ChatOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

type chatReq struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResp struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat sends messages and returns the assistant text.
func (c *LLMClient) Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (string, error) {
	if opts.Model == "" {
		opts.Model = c.Config.DefaultModel
	}
	if opts.Temperature == 0 {
		opts.Temperature = 0.7
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 1024
	}

	body, err := json.Marshal(chatReq{
		Model:       opts.Model,
		Messages:    messages,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.Config.APIBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Config.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w, raw=%s", err, string(raw))
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("LLM error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices in response: %s", string(raw))
	}
	return parsed.Choices[0].Message.Content, nil
}
