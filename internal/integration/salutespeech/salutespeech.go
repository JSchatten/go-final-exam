package salutespeech

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JSchatten/go-final-exam/internal/sberoath2"
)

// internal/integration/salutespeech.go

type SaluteSpeechClient struct {
	oauthClient *sberoath2.OAuth2Client
	BaseURL     string
}

func NewSaluteSpeechClient(oauthClient *sberoath2.OAuth2Client) *SaluteSpeechClient {
	return &SaluteSpeechClient{
		oauthClient: oauthClient,
		BaseURL:     "https://smartspeech.sber.ru/rest/v1",
	}
}

func (c *SaluteSpeechClient) UploadFile(fileURL string) (string, error) {
	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	body := url.Values{}
	body.Set("url", fileURL)

	req, err := http.NewRequest("POST", c.BaseURL+"/data:upload", strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var uploadResp UploadResponse
	err = json.Unmarshal(respBody, &uploadResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal upload response: %w", err)
	}

	return uploadResp.Result.RequestFileID, nil
}

func (c *SaluteSpeechClient) CreateRecognitionTask(audioId string) (string, error) {
	body := map[string]interface{}{
		"options": map[string]interface{}{
			"model":                   "general",
			"audio_encoding":          "PCM_S16LE",
			"sample_rate":             16000,
			"language":                "ru-RU",
			"enable_profanity_filter": true,
			"hypotheses_count":        1,
			"no_speech_timeout":       2,
			"max_speech_timeout":      2,
			"channels_count":          1,
			"speaker_separation_options": map[string]interface{}{
				"enable":                   false,
				"enable_only_main_speaker": false,
				"count":                    1,
			},
		},
		"request_file_id": audioId,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/speech:async_recognize", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp AsyncTaskResponse
	err = json.Unmarshal(respBody, &taskResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal task response: %w", err)
	}

	return taskResp.Result.ID, nil
}

func (c *SaluteSpeechClient) CheckTaskStatus(taskId string) (string, error) {
	// Новый endpoint: /task:get с параметром id
	url := fmt.Sprintf("%s/task:get?id=%s", c.BaseURL, taskId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status check failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var statusResp map[string]string
	err = json.Unmarshal(respBody, &statusResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	return statusResp["status"], nil
}

func (c *SaluteSpeechClient) GetRecognitionResult(taskId string) (string, error) {
	url := fmt.Sprintf("%s/data:download?response_file_id=%s", c.BaseURL, taskId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	accessToken, err := c.oauthClient.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/octet-stream")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get result failed with status %d: %s", resp.StatusCode, string(body))
	}

	resultBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read result body: %w", err)
	}

	return string(resultBytes), nil
}

// RecognizeSpeech performs full speech recognition workflow
func (c *SaluteSpeechClient) RecognizeSpeech(fileURL string) (string, error) {
	// Step 1: Upload file
	audioId, err := c.UploadFile(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Step 2: Create recognition task
	taskId, err := c.CreateRecognitionTask(audioId)
	if err != nil {
		return "", fmt.Errorf("failed to create recognition task: %w", err)
	}

	// Step 3: Poll for task completion
	pollInterval := 5 * time.Second
	for {
		status, err := c.CheckTaskStatus(taskId)
		if err != nil {
			return "", fmt.Errorf("failed to check task status: %w", err)
		}

		if status == "DONE" {
			break
		}

		if status == "ERROR" {
			return "", fmt.Errorf("recognition task failed")
		}

		time.Sleep(pollInterval)
	}

	// Step 4: Get recognition result
	result, err := c.GetRecognitionResult(taskId)
	if err != nil {
		return "", fmt.Errorf("failed to get recognition result: %w", err)
	}

	return result, nil
}
