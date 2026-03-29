//go:build salutespeech
// +build salutespeech

package salutespeech

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeAuthKey(authKey string) (string, string, error) {
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

func loadTestConfig(t *testing.T) (string, string, string) {
	err := godotenv.Load("../../.env") // Путь от internal/integration к корню
	if err != nil {
		t.Skip(".env file not found, skipping integration test")
	}

	authKey := os.Getenv("SALUTESPEECH_AUTHORIZATION_KEY")
	scope := os.Getenv("SALUTESPEECH_SCOPE")

	if authKey == "" {
		t.Skip("SALUTESPEECH_AUTHORIZATION_KEY not set in .env")
	}
	if scope == "" {
		t.Skip("SALUTESPEECH_SCOPE not set in .env")
	}

	clientID, clientSecret, err := decodeAuthKey(authKey)
	require.NoError(t, err, "Failed to decode SALUTESPEECH_AUTHORIZATION_KEY")

	return clientID, clientSecret, scope
}

type TaskStatusResponse struct {
	Status string `json:"status"`
}

// === Тест: Полный цикл распознавания через прямую загрузку ===

func TestSaluteSpeech_FullRecognitionFlow_DirectUpload(t *testing.T) {
	clientID, clientSecret, scope := loadTestConfig(t)

	// 1. Читаем локальный файл
	filePath := "../../test/шарлотка.wav"
	audioData, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read audio file")
	require.Greater(t, len(audioData), 400, "Audio file must be larger than 400 bytes")

	// 2. Получаем токен
	oauthClient := sberoath2.NewOAuth2Client(
		clientID,
		clientSecret,
		scope,
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		nil,
	)
	token, err := oauthClient.GetToken()
	require.NoError(t, err)
	require.NotEmpty(t, token)

	t.Log("Token obtained")

	// --- Шаг 1: Прямая загрузка файла ---
	url := "https://smartspeech.sber.ru/rest/v1/data:upload"
	req, err := http.NewRequest("POST", url, bytes.NewReader(audioData))
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "audio/wav") // или "audio/mpeg" — в зависимости от файла
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Upload response: %s", string(body))

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK from upload")

	var uploadResp UploadResponse
	err = json.Unmarshal(body, &uploadResp)
	require.NoError(t, err)
	requestFileID := uploadResp.Result.RequestFileID
	require.NotEmpty(t, requestFileID)

	t.Logf("File uploaded. request_file_id: %s", requestFileID)

	// --- Шаг 2: Создание задачи на распознавание ---
	taskURL := "https://smartspeech.sber.ru/rest/v1/speech:async_recognize"
	taskBody := map[string]interface{}{
		"options": map[string]interface{}{
			"model":                   "general",
			"audio_encoding":          "PCM_S16LE",
			"sample_rate":             16000,
			"language":                "ru-RU",
			"enable_profanity_filter": true,
			"hypotheses_count":        1,
			// "no_speech_timeout":       2,
			// "max_speech_timeout":      2,
			"channels_count": 1,
			"speaker_separation_options": map[string]interface{}{
				"enable":                   false,
				"enable_only_main_speaker": false,
				"count":                    1,
			},
		},
		"request_file_id": requestFileID,
	}

	jsonTaskBody, _ := json.Marshal(taskBody)
	req, err = http.NewRequest("POST", taskURL, bytes.NewReader(jsonTaskBody))
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	t.Logf("Task creation response: %s", string(body))

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var taskResp AsyncTaskResponse
	err = json.Unmarshal(body, &taskResp)
	require.NoError(t, err)

	// Проверка HTTP-статуса
	if taskResp.Status != 200 {
		t.Fatalf("API returned status: %d", taskResp.Status)
	}

	// Используем данные задачи
	taskID := taskResp.Result.ID
	taskStatus := taskResp.Result.Status
	createdAt := taskResp.Result.CreatedAt

	t.Logf("Task created: ID=%s, Status=%s, Created=%v", taskID, taskStatus, createdAt)

	// --- Шаг 3: Ожидание завершения задачи с ограничением по количеству попыток ---
	t.Logf("Waiting for task %s to complete (polling every 5 seconds, max 5 attempts)", taskID)

	maxAttempts := 5
	finalStatus := ""
	var taskStatusResp AsyncTaskResponse

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Ждём 5 секунд перед каждой попыткой (кроме первой)
		if attempt > 0 {
			time.Sleep(5 * time.Second)
		}

		statusURL := fmt.Sprintf("https://smartspeech.sber.ru/rest/v1/task:get?id=%s", taskID)
		req, err := http.NewRequest("GET", statusURL, nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Attempt %d: failed to get task status: %v", attempt+1, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		t.Logf("Attempt %d: status response: %s", attempt+1, string(body))

		if err := json.Unmarshal(body, &taskStatusResp); err != nil {
			t.Logf("Attempt %d: failed to parse task status response: %v", attempt+1, err)
			continue
		}

		if taskStatusResp.Status != 200 {
			t.Logf("Attempt %d: API returned status: %d", attempt+1, taskStatusResp.Status)
			continue
		}

		finalStatus = taskStatusResp.Result.Status
		t.Logf("Attempt %d: current status = %s", attempt+1, finalStatus)

		if finalStatus == "DONE" {
			break
		}

		if finalStatus == "ERROR" || finalStatus == "CANCELED" {
			t.Fatalf("Task failed with status: %s", finalStatus)
		}
	}

	if finalStatus != "DONE" {
		t.Fatalf("Task did not reach 'DONE' status after %d attempts. Last status: %q", maxAttempts, finalStatus)
	}

	t.Logf("Task completed successfully with status: %s", finalStatus)

	// --- Шаг 4: Получение результата ---
	resultURL := fmt.Sprintf("https://smartspeech.sber.ru/rest/v1/data:download?response_file_id=%s", taskStatusResp.Result.ResponseFileID)
	req, err = http.NewRequest("GET", resultURL, nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK from download")

	// Читаем тело
	resultBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Логируем сырой ответ для отладки
	t.Logf("Raw recognition response: %s", string(resultBytes))

	// Парсим в промежуточную структуру (из-за вложенности results)
	var segments []struct {
		Results []RecognitionResult `json:"results"`
	}
	err = json.Unmarshal(resultBytes, &segments)
	require.NoError(t, err, "Failed to parse recognition JSON")

	// Преобразуем в плоский RecognitionResponse
	var recognitionResults RecognitionResponse
	for _, seg := range segments {
		for _, res := range seg.Results {
			if res.Text != "" || res.NormalizedText != "" {
				recognitionResults = append(recognitionResults, res)
			}
		}
	}

	// Проверяем, что результат не пустой
	assert.NotEmpty(t, recognitionResults, "Expected non-empty recognition results")

	// Получаем полные тексты
	fullText := recognitionResults.GetFullText()
	fullNormalizedText := recognitionResults.GetFullNormalizedText()

	t.Logf("Full text:\n%s", fullText)
	t.Logf("Full normalized text:\n%s", fullNormalizedText)

	// Проверяем, что хотя бы один из текстов заполнен
	assert.NotEmpty(t, fullText, "Full text should not be empty")
	assert.NotEmpty(t, fullNormalizedText, "Full normalized text should not be empty")
}
