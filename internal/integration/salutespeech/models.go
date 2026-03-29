package salutespeech

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// UploadResponse изменён под новый формат
type UploadResponse struct {
	Status int `json:"status"`
	Result struct {
		RequestFileID string `json:"request_file_id"`
	} `json:"result"`
}

// AsyncTaskResponse — обновлён под новый ответ
type AsyncTaskResponse struct {
	Status int        `json:"status"` // HTTP-статус как число (200)
	Result TaskResult `json:"result"`
}

// TaskResult — данные созданной задачи
type TaskResult struct {
	ID string `json:"id"`
	// статус задачи: "NEW, RUNNING, CANCELED, DONE, ERROR
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ResponseFileID string    `json:"response_file_id,omitempty"` // Появляется при DONE
}

func (t *TaskResult) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
		ResponseFileID string `json:"response_file_id,omitempty"` // Появляется при DONE
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Формат времени с дробными секундами и часовым поясом
	const layout = "2006-01-02T15:04:05.999999999Z07:00"

	createdAt, err := time.Parse(layout, alias.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to parse created_at: %w", err)
	}
	t.CreatedAt = createdAt

	updatedAt, err := time.Parse(layout, alias.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to parse updated_at: %w", err)
	}
	t.UpdatedAt = updatedAt

	t.ID = alias.ID
	t.Status = alias.Status
	t.ResponseFileID = alias.ResponseFileID // ← Добавлено!

	return nil
}

// RecognitionResultResponse — структура для ответа с текстом
type RecognitionResultResponse struct {
	Text string `json:"text"` // Результат приходит как plain text в теле
}

// RecognitionResult — упрощённая модель результата распознавания
type RecognitionResult struct {
	Text           string `json:"text"`
	NormalizedText string `json:"normalized_text"`
}

// RecognitionResponse — массив результатов (один элемент = один сегмент)
type RecognitionResponse []RecognitionResult

// GetFullText возвращает объединённый обычный текст всех результатов через пробел.
func (r RecognitionResponse) GetFullText() string {
	var parts []string
	for _, res := range r {
		if trimmed := strings.TrimSpace(res.Text); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}

// GetFullNormalizedText возвращает объединённый нормализованный текст всех результатов,
// каждый сегмент — с новой строки.
func (r RecognitionResponse) GetFullNormalizedText() string {
	var parts []string
	for _, res := range r {
		if trimmed := strings.TrimSpace(res.NormalizedText); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n")
}
