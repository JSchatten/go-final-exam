package gigachat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
)

// GigaChatClient represents a client for GigaChat API
type GigaChatClient struct {
	oauthClient *sberoath2.OAuth2Client
	BaseURL     string
}

// NewGigaChatClient creates a new GigaChat client
func NewGigaChatClient(oauthClient *sberoath2.OAuth2Client) *GigaChatClient {
	return &GigaChatClient{
		oauthClient: oauthClient,
		BaseURL:     "https://gigachat.devices.sberbank.ru/api/v1",
	}
}

// ChatRequest represents a request to GigaChat API
type ChatRequest struct {
	Model             string    `json:"model"`              // e.g., "GigaChat"
	Messages          []Message `json:"messages"`           // conversation history
	Stream            bool      `json:"stream"`             // must be false for sync
	RepetitionPenalty float64   `json:"repetition_penalty"` // recommended: 1.0
}

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // message text
}

// ChatResponse represents full response from GigaChat
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Object  string   `json:"object"`
	Usage   struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		SystemTokens     int `json:"system_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Choice represents one generated message
type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"` // "stop", "length"
	Message      Message `json:"message"`
}

// SendMessage sends a message to GigaChat and returns the assistant's reply
func (c *GigaChatClient) SendMessage(content string) (string, error) {
	// Get fresh access token
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Prepare request body
	request := ChatRequest{
		Model: "GigaChat", // согласно документации
		Messages: []Message{
			{Role: "user", Content: content},
		},
		Stream:            false, // синхронный режим
		RepetitionPenalty: 1.0,   // как в примере
	}

	// Marshal to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		// Пытаемся извлечь детали ошибки
		var errResp map[string]interface{}
		_ = json.Unmarshal(respBody, &errResp)
		errMsg, _ := errResp["error"]
		return "", fmt.Errorf("gigachat request failed with status %d: %v", resp.StatusCode, errMsg)
	}

	// Unmarshal response
	var chatResp ChatResponse
	err = json.Unmarshal(respBody, &chatResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Логируем использование токенов (опционально, можно удалить)
	// fmt.Printf("Tokens used: %d total\n", chatResp.Usage.TotalTokens)

	// Проверяем наличие ответа
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from GigaChat")
	}

	// Возвращаем текст ассистента
	return chatResp.Choices[0].Message.Content, nil
}
