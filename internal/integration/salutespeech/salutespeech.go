// internal/integration/salutespeech/salutespeech.go

package salutespeech

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
)

// SaluteSpeechClient управляет взаимодействием с Sber SaluteSpeech API
type SaluteSpeechClient struct {
	oauthClient *sberoath2.OAuth2Client
	BaseURL     string
	HTTPClient  *http.Client
}

// NewSaluteSpeechClient создаёт новый клиент
func NewSaluteSpeechClient(cfg *Config) (*SaluteSpeechClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("salutespeech config is required")
	}

	oauthClient, err := sberoath2.NewOAuth2Client(
		cfg.ClientID,
		cfg.AuthKey,
		cfg.Scope,
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		nil, // использовать дефолтный RqUID генератор
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 client for SaluteSpeech: %w", err)
	}

	return &SaluteSpeechClient{
		oauthClient: oauthClient,
		BaseURL:     "https://smartspeech.sber.ru/rest/v1",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// UploadFileByPath загружает локальный аудиофайл напрямую на сервер SaluteSpeech
// и возвращает request_file_id - идентификатор, который можно использовать
// для создания задачи распознавания.
//
// Параметры:
//   - filePath: путь к локальному аудиофайлу (.wav, .mp3, .ogg и др.)
//
// Возвращает:
//   - request_file_id: строковый идентификатор файла в системе SaluteSpeech
//   - ошибка, если загрузка не удалась
//
// Пример использования:
//
//	fileID, err := client.UploadFileByPath("./audio/test.ogg")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *SaluteSpeechClient) UploadFileByPath(filePath string) (string, error) {
	audioData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/data:upload", bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "audio/wav") // Предполагаем WAV; можно параметризовать
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send upload request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read upload response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal upload response: %w", err)
	}

	return uploadResp.Result.RequestFileID, nil
}

// UploadFileByUrl загружает аудиофайл по внешнему URL (например, из Telegram или S3)
// и возвращает request_file_id - идентификатор, пригодный для создания задачи распознавания.
//
// Параметры:
//   - fileURL: публичный URL аудиофайла (должен быть доступен для скачивания)
//
// Возвращает:
//   - request_file_id: строковый идентификатор файла в системе SaluteSpeech
//   - ошибка, если загрузка не удалась
//
// Важно: файл должен быть доступен по GET без авторизации.
// Поддерживаются те же форматы, что и в UploadFileByPath.
func (c *SaluteSpeechClient) UploadFileByUrl(fileURL string) (string, error) {
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	data := url.Values{}
	data.Set("url", fileURL)

	req, err := http.NewRequest("POST", c.BaseURL+"/data:upload", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp UploadResponse
	err = json.Unmarshal(body, &uploadResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal upload response: %w", err)
	}

	return uploadResp.Result.RequestFileID, nil
}

// CreateRecognitionTask создаёт асинхронную задачу на распознавание речи
// по уже загруженному аудиофайлу.
//
// Параметры:
//   - audioPath: путь к исходному аудиофайлу (локальному), используемый
//     для определения формата (аудиокодека: OPUS, PCM_S16LE и т.п.)
//   - fileID: request_file_id, полученный ранее через UploadFileByPath
//     или UploadFileByUrl
//
// Возвращает:
//   - taskID: идентификатор задачи
//   - status: начальный статус задачи (например, "NEW")
//   - ошибка, если создание задачи не удалось
func (c *SaluteSpeechClient) CreateRecognitionTask(audioPath, fileID string) (string, string, error) {
	if audioPath == "" {
		return "", "", fmt.Errorf("audioPath is required to determine encoding")
	}
	if fileID == "" {
		return "", "", fmt.Errorf("fileID is required")
	}

	// Определяем кодировку по расширению файла
	encoding, err := getAudioOptions(audioPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get audio encoding from path %q: %w", audioPath, err)
	}

	// Формируем тело запроса
	body := map[string]interface{}{
		"options": map[string]interface{}{
			"model":                   "general",
			"audio_encoding":          encoding,
			"sample_rate":             16000,
			"language":                "ru-RU",
			"enable_profanity_filter": true,
			"hypotheses_count":        1,
			"channels_count":          1,
			"speaker_separation_options": map[string]interface{}{
				"enable":                   false,
				"enable_only_main_speaker": false,
				"count":                    1,
			},
		},
		"request_file_id": fileID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal task body: %w", err)
	}

	// Получаем токен
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Создаём запрос
	req, err := http.NewRequest("POST", c.BaseURL+"/speech:async_recognize", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выполняем запрос
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("task creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp AsyncTaskResponse
	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal task response: %w", err)
	}

	if taskResp.Status != 200 {
		return "", "", fmt.Errorf("API returned error status: %d", taskResp.Status)
	}

	// Возвращаем ID и статус задачи
	return taskResp.Result.ID, taskResp.Result.Status, nil
}

// CheckTaskStatus проверяет статус задачи и возвращает полный объект TaskResult
func (c *SaluteSpeechClient) CheckTaskStatus(taskId string) (*TaskResult, error) {
	url := fmt.Sprintf("%s/task:get?id=%s", c.BaseURL, taskId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var statusResp AsyncTaskResponse
	err = json.Unmarshal(body, &statusResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	if statusResp.Status != 200 {
		return nil, fmt.Errorf("API returned error status: %d", statusResp.Status)
	}

	return &statusResp.Result, nil
}

// GetRecognitionResult получает результат по ResponseFileID
func (c *SaluteSpeechClient) GetRecognitionResult(responseFileID string) (RecognitionResponse, error) {
	url := fmt.Sprintf("%s/data:download?response_file_id=%s", c.BaseURL, responseFileID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get result failed with status %d: %s", resp.StatusCode, string(body))
	}

	resultBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read result body: %w", err)
	}

	// Результат - массив сегментов, каждый содержит `results`
	var segments []struct {
		Results []RecognitionResult `json:"results"`
	}
	if err := json.Unmarshal(resultBytes, &segments); err != nil {
		return nil, fmt.Errorf("failed to parse recognition JSON: %w", err)
	}

	var recognitionResults RecognitionResponse
	for _, seg := range segments {
		for _, res := range seg.Results {
			if res.Text != "" || res.NormalizedText != "" {
				recognitionResults = append(recognitionResults, res)
			}
		}
	}

	if len(recognitionResults) == 0 {
		return nil, fmt.Errorf("recognition result is empty")
	}

	return recognitionResults, nil
}

// getAudioOptions returns appropriate audio encoding, sample rate, and channels
// based on file extension. It uses defaults if metadata is not available.
func getAudioOptions(filePath string) (encoding string, err error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".wav":
		// Предполагаем стандартный PCM 16-bit LE
		return "PCM_S16LE", nil

	case ".mp3":
		// MP3 поддерживается с разной частотой и каналами
		return "MP3", nil

	case ".ogg":
		// Telegram отправляет .ogg с Opus - используем OGG_OPUS
		// Требует 16000 или 48000 Hz, только моно
		return "OPUS", nil

	default:
		return "", fmt.Errorf("unsupported audio format: %s", ext)
	}
}
