//go:build salutespeech
// +build salutespeech

package salutespeech

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupClient(t *testing.T) *SaluteSpeechClient {
	t.Helper()

	err := godotenv.Load("../../../.env")
	require.NoError(t, err, "Не удалось загрузить .env файл")

	clientID := os.Getenv("SALUTESPEECH_CLIENT_ID")
	authKey := os.Getenv("SALUTESPEECH_AUTHORIZATION_KEY")
	scope := os.Getenv("SALUTESPEECH_SCOPE")
	if scope == "" {
		scope = "SALUTE_SPEECH_PERS"
	}

	require.NotEmpty(t, clientID, "SALUTESPEECH_CLIENT_ID не задан в .env")
	require.NotEmpty(t, authKey, "SALUTESPEECH_AUTHORIZATION_KEY не задан в .env")

	cfg := &Config{
		ClientID: clientID,
		AuthKey:  authKey,
		Scope:    scope,
	}

	client, err := NewSaluteSpeechClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}

func getTestAudioPath() string {
	paths := []string{
		"../../../test/шарлотка.ogg", // короткая инструкция
		// "../../../test/шарлотка.wav", // полная версия
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestSaluteSpeech_FullRecognitionFlow(t *testing.T) {
	client := setupClient(t)

	audioPath := getTestAudioPath()
	if audioPath == "" {
		t.Skip("Файл ./test/шарлотка.ogg не найден — пропускаем интеграционный тест")
	}

	// Проверим, что файл существует и не пустой
	info, err := os.Stat(audioPath)
	require.NoError(t, err, "Не удалось получить информацию о файле")
	require.Greater(t, info.Size(), int64(100), "Аудиофайл слишком мал для распознавания")

	t.Logf("Используем аудиофайл: %s", audioPath)

	// Шаг 1: Загружаем аудиофайл
	t.Log("Загружаем аудиофайл...")
	fileID, err := client.UploadFileByPath(audioPath)
	require.NoError(t, err)
	require.NotEmpty(t, fileID)
	t.Logf("Файл загружен: request_file_id = %s", fileID)

	// Шаг 2: Создаём задачу распознавания с указанием пути к файлу (для правильного audio_encoding)
	t.Log("Создаём задачу распознавания...")
	taskID, err := client.CreateRecognitionTask(audioPath, fileID)
	require.NoError(t, err)
	require.NotEmpty(t, taskID)
	t.Logf("Задача создана: id = %s", taskID)

	// Шаг 3: Поллинг статуса до DONE
	t.Log("Ожидаем завершения распознавания...")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var finalTask *TaskResult
	done := false

	for !done {
		select {
		case <-ctx.Done():
			t.Fatalf("Таймаут ожидания завершения задачи: %v", ctx.Err())
		case <-time.After(3 * time.Second):
			task, err := client.CheckTaskStatus(taskID)
			require.NoError(t, err)
			t.Logf("Текущий статус задачи: %s", task.Status)

			switch task.Status {
			case "DONE":
				finalTask = task
				done = true
			case "ERROR", "CANCELED":
				t.Fatalf("Задача завершилась с ошибкой: %s", task.Status)
			default:
				t.Log("Продолжаем опрос...")
			}
		}
	}

	// После цикла — проверки
	require.NotNil(t, finalTask, "Финальная задача должна быть получена")
	require.Equal(t, "DONE", finalTask.Status, "Статус задачи должен быть 'DONE'")
	require.NotEmpty(t, finalTask.ResponseFileID, "ResponseFileID должен быть заполнен")

	t.Logf("Распознавание завершено. Результат в файле: %s", finalTask.ResponseFileID)
	t.Logf("Полный ответ от API: %+v", *finalTask)

	// Шаг 4: Получаем результат распознавания
	t.Log("Получаем текст распознавания...")
	result, err := client.GetRecognitionResult(finalTask.ResponseFileID)
	require.NoError(t, err)
	require.NotEmpty(t, result, "Результат распознавания не должен быть пустым")

	fullText := result.GetFullText()
	normalizedText := result.GetFullNormalizedText()

	t.Logf("Распознанный текст:\n%s", fullText)
	if normalizedText != "" {
		t.Logf("Нормализованный текст:\n%s", normalizedText)
	}

	// Проверки результата
	assert.NotEmpty(t, fullText, "Распознанный текст должен быть непустым")

	// Опционально: проверить наличие ключевых слов
	expectedWords := []string{"шарлотка", "яблоко", "сахар", "мука"}
	found := false
	lowerText := strings.ToLower(fullText)
	for _, word := range expectedWords {
		if strings.Contains(lowerText, word) {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Внимание: ни одно из ожидаемых слов (%v) не найдено в тексте", expectedWords)
		// Не падаем — качество распознавания может варьироваться
	}
}

// Example использования (не тест)
// func ExampleSaluteSpeechClient_FullFlow() {
// 	log.SetOutput(nil) // подавляем логи в примере
// 	defer log.SetOutput(os.Stderr)

// 	client := setupClient(&testing.T{})
// 	audioPath := getTestAudioPath()
// 	if audioPath == "" {
// 		fmt.Println("Файл ./audio/test.ogg не найден")
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
// 	defer cancel()

// 	// 1. Загрузка
// 	fileID, err := client.UploadFileByPath(audioPath)
// 	if err != nil {
// 		fmt.Printf("Upload failed: %v\n", err)
// 		return
// 	}
// 	fmt.Printf("Uploaded: %s\n", fileID)

// 	// 2. Создание задачи
// 	taskID, err := client.CreateRecognitionTask(fileID)
// 	if err != nil {
// 		fmt.Printf("Create task failed: %v\n", err)
// 		return
// 	}
// 	fmt.Printf("Task created: %s\n", taskID)

// 	// 3. Ожидание
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			fmt.Println("Timeout")
// 			return
// 		case <-time.After(3 * time.Second):
// 			task, err := client.CheckTaskStatus(taskID)
// 			if err != nil {
// 				fmt.Printf("Status check failed: %v\n", err)
// 				return
// 			}
// 			fmt.Printf("Status: %s\n", task.Status)

// 			if task.Status == "DONE" {
// 				// 4. Получение результата
// 				result, err := client.GetRecognitionResult(task.ResponseFileID)
// 				if err != nil {
// 					fmt.Printf("Get result failed: %v\n", err)
// 					return
// 				}
// 				fmt.Printf("Text: %s\n", result.GetFullText())
// 				return
// 			}
// 			if task.Status == "ERROR" || task.Status == "CANCELED" {
// 				fmt.Printf("Task failed: %s\n", task.Status)
// 				return
// 			}
// 		}
// 	}
// }
