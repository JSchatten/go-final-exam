package service

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

var AllowedAudioExtensions = map[string]bool{
	".ogg": true,
	".oga": true,
	".mp3": true,
	".wav": true,
}

const MaxFileSize = 500 * 1024 * 1024 // 500 МБ

// HandleVoice обрабатывает голосовые сообщения.
func (b *Bot) HandleVoice(c telebot.Context) error {
	voice := c.Message().Voice
	user := c.Sender()

	log.Printf("Получено голосовое сообщение: %d сек, FileID: %s", voice.Duration, voice.FileID)

	// 1. Получаем информацию о файле из Telegram
	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		log.Printf("Не удалось получить информацию о файле: %v", err)
		return c.Send("Не удалось загрузить аудиофайл.")
	}

	// Защита от path traversal
	filename := filepath.Base(file.FilePath)
	if filename == "." || filename == ".." {
		log.Printf("Invalid filename after Base(): %q", filename)
		return c.Send("Недопустимый файл.")
	}

	if voice.FileSize > MaxFileSize {
		log.Printf("Файл слишком большой: %d байт (> %d)", voice.FileSize, MaxFileSize)
		return c.Send(
			fmt.Sprintf(
				"Файл слишком большой: %.1f МБ.\nМаксимальный размер: %d МБ.",
				float64(voice.FileSize)/(1024*1024), MaxFileSize/(1024*1024),
			),
		)
	}

	// Проверка расширений
	ext := strings.ToLower(filepath.Ext(file.FilePath))
	if !AllowedAudioExtensions[ext] {
		var formats []string
		for ext := range AllowedAudioExtensions {
			formats = append(formats, strings.ToUpper(strings.TrimPrefix(ext, ".")))
		}
		log.Printf("Неподдерживаемое расширение файла: %s", ext)
		return c.Send(fmt.Sprintf("Формат %s не поддерживается.\nПоддерживаемые форматы: %s.", ext, strings.Join(formats, ", ")))
	}

	// 2. Подготавливаем путь для сохранения
	audioDir := b.AudioStoragePath
	timestamp := time.Now().Unix()
	audioPath := filepath.Join(audioDir, fmt.Sprintf("voice_%d_%d%s", user.ID, timestamp, filepath.Ext(filename)))

	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Printf("Не удалось создать папку %s: %v", audioDir, err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	outFile, err := os.Create(audioPath)
	if err != nil {
		log.Printf("Не удалось создать файл %s: %v", audioPath, err)
		return c.Send("Ошибка при сохранении аудио.")
	}
	defer outFile.Close()

	// 3. Скачиваем файл с сервера Telegram
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Telebot.Token, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Ошибка при скачивании файла: %v", err)
		return c.Send("Не удалось скачать аудио.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP ошибка при скачивании: %d", resp.StatusCode)
		return c.Send("Не удалось скачать аудио.")
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Ошибка при записи файла: %v", err)
		return c.Send("Ошибка при сохранении аудио.")
	}

	log.Printf("Голосовое сообщение успешно сохранено: %s", audioPath)

	// Загружаем файл в SaluteSpeech (это не дорого)
	requestFileID, err := b.SaluteSpeech.UploadFileByPath(audioPath)
	if err != nil {
		log.Printf("Failed to upload file to SaluteSpeech: %v", err)
		return c.Send("Не удалось отправить файл на распознавание.")
	}

	// Создаём задачу распознавания
	taskID, taskStatus, err := b.SaluteSpeech.CreateRecognitionTask(audioPath, requestFileID)
	if err != nil {
		log.Printf("Failed to create recognition task: %v", err)
		return c.Send("Не удалось создать задачу распознавания.")
	}

	log.Printf("Задача на распознавание создана: ID=%s, Status=%s", taskID, taskStatus)

	// Конец изменений

	return c.Send("Голосовое сообщение получено и отправлено на распознавание", &telebot.SendOptions{ParseMode: "Markdown"})
}
