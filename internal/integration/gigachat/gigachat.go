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

// NewGigaChatClient creates a new GigaChat client using Config
func NewGigaChatClient(cfg *Config) (*GigaChatClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("gigachat config is required")
	}

	oauthClient, err := sberoath2.NewOAuth2Client(
		cfg.ClientID,
		cfg.AuthKey,
		"GIGACHAT_API_PERS",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth client: %w", err)
	}

	return &GigaChatClient{
		oauthClient: oauthClient,
		BaseURL:     BaseURL,
		HTTPClient: &http.Client{
			Timeout: HTTPTimeout,
		},
	}, nil
}

// sendChatRequest выполняет HTTP-запрос к GigaChat и возвращает текст ответа
func (c *GigaChatClient) sendChatRequest(request ChatRequest) (string, error) {
	// Получаем токен
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Маршалим тело запроса
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Создаём HTTP-запрос
	req, err := http.NewRequest("POST", c.BaseURL+EndpointChatCompletions, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выполняем запрос
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Проверяем статус
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		_ = json.Unmarshal(respBody, &errResp)
		errMsg, _ := errResp["error"]
		return "", fmt.Errorf("gigachat request failed with status %d: %v", resp.StatusCode, errMsg)
	}

	// Парсим ответ
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Проверяем наличие ответа
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from GigaChat")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *GigaChatClient) newChatRequest(systemPrompt, userContent string) ChatRequest {
	return ChatRequest{
		Model: Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Stream:            StreamDisabled,
		RepetitionPenalty: DefaultRepetitionPenalty,
	}
}

// SendChatMessage отправляет сообщение пользователю и возвращает ответ ассистента
func (c *GigaChatClient) SendChatMessage(content string) (string, error) {
	request := c.newChatRequest(SystemPromptChat, content)
	return c.sendChatRequest(request)
}

// SendMessageWithSystemPrompt отправляет системный промпт + сообщение пользователя
func (c *GigaChatClient) SendMessageWithSystemPrompt(systemPrompt, userContent string) (string, error) {
	request := c.newChatRequest(systemPrompt, userContent)
	return c.sendChatRequest(request)
}

// Transcribe анализирует текст встречи и возвращает структурированную выжимку
func (c *GigaChatClient) Transcribe(speechText string) (string, error) {
	response, err := c.SendMessageWithSystemPrompt(SystemPromptSummary, speechText)
	if err != nil {
		return "", fmt.Errorf("failed to get transcription from GigaChat: %w", err)
	}
	return response, nil
}
