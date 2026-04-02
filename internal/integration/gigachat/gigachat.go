package gigachat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
)

// GigaChatClient represents a client for GigaChat API
type GigaChatClient struct {
	oauthClient *sberoath2.OAuth2Client
	BaseURL     string
	HTTPClient  *http.Client
}

// NewGigaChatClient creates a new GigaChat client
func NewGigaChatClient(oauthClient *sberoath2.OAuth2Client) *GigaChatClient {
	return &GigaChatClient{
		oauthClient: oauthClient,
		BaseURL:     BaseURL,
		HTTPClient: &http.Client{
			Timeout: HTTPTimeout,
		},
	}
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
		Model: Model,
		Messages: []Message{
			{Role: "user", Content: content},
		},
		Stream:            StreamDisabled,
		RepetitionPenalty: DefaultRepetitionPenalty,
	}

	// Marshal to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+EndpointChatCompletions, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.HTTPClient.Do(req)
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

	// Check for empty response
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from GigaChat")
	}

	// Return assistant's message
	return chatResp.Choices[0].Message.Content, nil
}

// SendMessageWithSystemPrompt sends a message with a system prompt and user content
func (c *GigaChatClient) SendMessageWithSystemPrompt(systemPrompt, userContent string) (string, error) {
	// Get fresh access token
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Prepare request body
	request := ChatRequest{
		Model: Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Stream:            StreamDisabled,
		RepetitionPenalty: DefaultRepetitionPenalty,
	}

	// Marshal to JSON
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.BaseURL+EndpointChatCompletions, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request using existing HTTPClient
	resp, err := c.HTTPClient.Do(req)
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

	// Check for empty response
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from GigaChat")
	}

	// Return assistant's message
	return chatResp.Choices[0].Message.Content, nil
}

// Transcribe анализирует текст встречи и возвращает структурированную выжимку
func (c *GigaChatClient) Transcribe(speechText string) (string, error) {
	// Шаг 1: Отправляем запрос с системным промптом
	response, err := c.SendMessageWithSystemPrompt(SystemPrompt, speechText)
	if err != nil {
		return "", fmt.Errorf("failed to get transcription from GigaChat: %w", err)
	}

	return response, nil
}
