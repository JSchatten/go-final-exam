//go:build gigachat
// +build gigachat

package gigachat

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestClient загружает конфиг из .env и создаёт GigaChatClient
func setupTestClient(t *testing.T) *GigaChatClient {
	t.Helper()

	// Загружаем .env
	err := godotenv.Load("../../.env")
	require.NoError(t, err, "Не удалось загрузить .env файл")

	// Читаем переменные окружения
	clientID := os.Getenv("GIGACHAT_CLIENT_ID")
	authKey := os.Getenv("GIGACHAT_AUTH_KEY")
	scope := os.Getenv("GIGACHAT_SCOPE")
	if scope == "" {
		scope = "GIGACHAT_API_PERS" // значение по умолчанию
	}

	require.NotEmpty(t, clientID, "GIGACHAT_CLIENT_ID не задан в .env")
	require.NotEmpty(t, authKey, "GIGACHAT_AUTH_KEY не задан в .env")

	// Создаём конфиг
	cfg := &Config{
		ClientID: clientID,
		AuthKey:  authKey,
		Scope:    scope,
	}

	// Создаём клиент
	client, err := NewGigaChatClient(cfg)
	require.NoError(t, err, "Не удалось создать GigaChatClient")
	require.NotNil(t, client)

	return client
}

func TestGigaChat_SendMessage_ExtractSummary(t *testing.T) {
	// Настройка клиента
	client := setupTestClient(t)

	// Текст для анализа — рецепт шарлотки
	inputText := `
Классический рецепт шарлотки самый простой шарлотка с яблоками в духовке. Ингредиенты мука, сахар, яйца, разрыхлитель, ванилин и яблоки. Шаг 1 достаём яйца с холодильника, взбиваем миксером шаг 2 добавляем сахар, тщательно перемешиваем шаг 3 добавляем ванилин и ванильный сахар.
Шаг 4, теперь постепенно добавляем муку, тщательно. Каждый раз перемешиваем шаг 5 добавляем разрыхлитель, шаг 6, нарезаем яблоки, шаг 7, добавляем яблоки в тесто. Шаг 8, готовим форму для выпекания, шаг 9. Духовку заранее разогреваем до 180 градусов.
Шаг 10 выбегается Шарлотта 30 40 минут, шаг 11, подача блюда.
`

	// Отправляем запрос с системным промптом
	response, err := client.SendMessageWithSystemPrompt(SystemPrompt, inputText)
	require.NoError(t, err, "SendMessageWithSystemPrompt вернул ошибку")
	assert.NotEmpty(t, response, "Ответ от GigaChat пуст")

	t.Logf("Ответ от GigaChat:\n%s", response)

	// Проверяем, что ответ содержит ключевые элементы
	lowerResp := strings.ToLower(response)
	assert.Contains(t, lowerResp, "шарлотка", "ответ должен содержать упоминание 'шарлотка'")
	assert.Contains(t, lowerResp, "яблоки", "ответ должен содержать упоминание 'яблоки'")
	assert.Contains(t, lowerResp, "минут", "ответ должен содержать упоминание времени приготовления")
}

func TestGigaChat_Transcribe(t *testing.T) {
	// Настройка клиента
	client := setupTestClient(t)

	// Текст встречи
	meetingText := `
Сегодня обсуждали запуск нового продукта. Присутствовали: Иван (продукт), Мария (маркетинг), Алексей (разработка).
Решено: запуск 15 октября. Мария готовит кампанию к 10 октября. Алексей завершает фронтенд к 12 октября.
Вопросы: не решён вопрос по бюджету.
`

	// Вызываем метод Transcribe
	summary, err := client.Transcribe(meetingText)
	require.NoError(t, err, "Transcribe вернул ошибку")
	assert.NotEmpty(t, summary, "выжимка пустая")

	t.Logf("Выжимка:\n%s", summary)

	// Проверяем наличие ключевых элементов
	lowerSummary := strings.ToLower(summary)
	assert.Contains(t, lowerSummary, "запуск", "должен упоминать запуск")
	assert.Contains(t, lowerSummary, "мария", "должен упоминать участников")
	assert.Contains(t, lowerSummary, "октябрь", "должен упоминать даты")
	assert.Contains(t, lowerSummary, "бюджет", "должен упоминать открытые вопросы")
}

func TestGigaChat_GetToken_Retry(t *testing.T) {
	client := setupTestClient(t)

	// Получаем токен первый раз
	token1, err := client.oauthClient.GetToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// Ждём, чтобы проверить обновление (не обязательно, но можно)
	time.Sleep(100 * time.Millisecond)

	// Получаем второй раз — должен быть тот же или обновлён
	token2, err := client.oauthClient.GetToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	// В реальности может отличаться, поэтому просто логируем
	t.Logf("Token 1: %s", token1)
	t.Logf("Token 2: %s", token2)
}
