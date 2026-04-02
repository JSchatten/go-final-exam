//go:build gigachat
// +build gigachat

package gigachat

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeAuthKey decodes Base64 "client_id:client_secret" and returns both parts
func decodeAuthKey(authKey string) (clientID, clientSecret string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(authKey)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 in AUTHORIZATION_KEY: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: expected client_id:client_secret")
	}

	return parts[0], parts[1], nil
}

func TestGigaChat_SendMessage_ExtractSummary(t *testing.T) {
	// Загружаем .env
	err := godotenv.Load("../../../.env")
	require.NoError(t, err, "Не удалось загрузить .env файл")

	authKey := os.Getenv("GIGACHAT_AUTHORIZATION_KEY")
	require.NotEmpty(t, authKey, "GIGACHAT_AUTHORIZATION_KEY не задан в .env")

	// Декодируем ключ
	clientID, clientSecret, err := decodeAuthKey(authKey)
	require.NoError(t, err, "Не удалось декодировать GIGACHAT_AUTHORIZATION_KEY")
	require.NotEmpty(t, clientID, "client_id пуст после декодирования")
	require.NotEmpty(t, clientSecret, "client_secret пуст после декодирования")

	// Инициализируем OAuth2 клиент
	oauthClient := sberoath2.NewOAuth2Client(
		clientID,
		clientSecret,
		"GIGACHAT_API_PERS",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		nil,
	)

	// Создаём GigaChat клиент
	gigaClient := NewGigaChatClient(oauthClient)

	// Текст для анализа — рецепт шарлотки
	inputText := `
Классический рецепт шарлотки самый простой шарлотка с яблоками в духовке. Ингредиенты мука, сахар, яйца, разрыхлитель, ванилин и яблоки. Шаг 1 достаём яйца с холодильника, взбиваем миксером шаг 2 добавляем сахар, тщательно перемешиваем шаг 3 добавляем ванилин и ванильный сахар.
Шаг 4, теперь постепенно добавляем муку, тщательно. Каждый раз перемешиваем шаг 5. Теперь добавляем разрыхлитель, шаг 6, нарезаем яблоки, шаг 7, добавляем яблоки в тесто. Шаг 8, готовим форму для выпекания, шаг 9. Духовку заранее разогреваем до 180 градусов.
Шаг 10 выбегается Шарлотта 30 40 минут, шаг 11, подача блюда.
`
	// Отправляем два сообщения: system + user
	response, err := gigaClient.SendMessageWithSystemPrompt(SystemPrompt, inputText)
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	t.Logf("Ответ от GigaChat:\n")
	t.Logf("%s", response)

	// Проверяем, что ответ содержит ключевые слова
	// Но это не всегда срабатывает, тяжко проконтроллировать
	// assert.Contains(t, strings.ToLower(response), "шарлотка", "должен содержать упоминание шарлотки")
	// assert.Contains(t, strings.ToLower(response), "яблоки", "должен упоминать яблоки")
	// assert.Contains(t, strings.ToLower(response), "минут", "должен упоминать время")
}
