// internal/integration/gigachat/models.go
package gigachat

// Message represents a message in the chat
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // message text
}

// ChatRequest represents a request to GigaChat API
type ChatRequest struct {
	Model             string    `json:"model"`              // e.g., "GigaChat"
	Messages          []Message `json:"messages"`           // conversation history
	Stream            bool      `json:"stream"`             // must be false for sync
	RepetitionPenalty float64   `json:"repetition_penalty"` // recommended: 1.0
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
