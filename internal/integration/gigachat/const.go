// internal/integration/gigachat/const.go
package gigachat

import "time"

const (
	// BaseURL - базовый URL GigaChat API
	BaseURL = "https://gigachat.devices.sberbank.ru/api/v1"

	// Model - модель по умолчанию
	Model = "GigaChat"

	// EndpointChatCompletions - эндпоинт для генерации ответа
	EndpointChatCompletions = "/chat/completions"

	// HTTPTimeout - таймаут HTTP-запросов
	HTTPTimeout = 15 * time.Second

	// DefaultRepetitionPenalty - значение по умолчанию
	DefaultRepetitionPenalty = 1.0

	// StreamDisabled - режим синхронной генерации
	StreamDisabled = false

	SystemPrompt = `
		Ты — опытный ассистент. Проанализируй текст и предоставь краткую выжимку в виде пунктов:
		- Основная тема сообщения
		- Ключевые тезисы
		- Вывод по возможности (например, если это перегоовры, то к чему пришли стороны)

		Формат: маркированный список на русском языке.
`
)
