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

// HandleAudio обрабатывает аудиофайлы (MP3, WAV и т.д.)
func (b *BotService) HandleAudio(c telebot.Context) error {
	audio := c.Message().Audio
	user := c.Sender()

	log.Printf("AUDIO MEASSEG:\n%+v\n", audio)

	log.Printf("Received audio file: %s, Duration: %d sec, FileID: %s", audio.Title, audio.Duration, audio.FileID)

	file, err := b.Telebot.FileByID(audio.FileID)
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return c.Reply("Failed to load audio file.")
	}

	ext := strings.ToLower(filepath.Ext(file.FilePath))
	if !AllowedAudioExtensions[ext] {
		return c.Reply("Unsupported audio format.")
	}

	if audio.FileSize > MaxFileSize {
		return c.Reply(fmt.Sprintf("File too large. Max: %d MB.", MaxSizeMb))
	}

	audioDir := b.AudioStoragePath
	timestamp := time.Now().Unix()
	audioPath := filepath.Join(audioDir, fmt.Sprintf("audio_%d_%d%s", user.ID, timestamp, ext))

	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Printf("Failed to create dir: %v", err)
		return c.Reply("Failed to save audio.")
	}

	outFile, err := os.Create(audioPath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return c.Reply("Failed to save audio.")
	}
	defer outFile.Close()

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Telebot.Token, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return c.Reply("Failed to download audio.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP error on download: %d", resp.StatusCode)
		return c.Reply("Failed to download audio.")
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Failed to write file: %v", err)
		return c.Reply("Failed to save audio.")
	}

	log.Printf("Audio saved: %s", audioPath)

	ctx := b.getCtx(c)

	title := formatAudioTitle(audio)

	// Единая логика обработки
	err = b.processAudio(ctx, user.ID, audioPath, title)
	if err != nil {
		log.Printf("processAudio failed: %v", err)
		return c.Reply("Failed to start transcription.")
	}

	return c.Reply("Аудиофайл принят для обработки!", &telebot.SendOptions{ParseMode: "Markdown"})
}

// formatAudioTitle формирует название встречи на основе доступных данных
func formatAudioTitle(audio *telebot.Audio) string {
	title := audio.Title
	performer := audio.Performer

	// Что-то такое, если есть мета
	switch {
	case title != "" && performer != "":
		return fmt.Sprintf("%s - %s", performer, title)
	case title != "":
		return title
	case performer != "":
		return performer
	default:
		// Можно было бы сделать meeting, но пусть он будет для войсов
		// return fmt.Sprintf("Meeting %s", time.Now().Format("2006-01-02 15:04"))
		return audio.FileName
	}
}
