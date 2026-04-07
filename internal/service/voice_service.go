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

// HandleVoice обрабатывает голосовые сообщения.
func (b *BotService) HandleVoice(c telebot.Context) error {
	voice := c.Message().Voice
	user := c.Sender()

	log.Printf("Received voice message: %d sec, FileID: %s", voice.Duration, voice.FileID)

	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return c.Reply("Failed to load audio file.")
	}

	ext := strings.ToLower(filepath.Ext(file.FilePath))
	if !AllowedAudioExtensions[ext] {
		return c.Reply("Unsupported audio format.")
	}

	if voice.FileSize > MaxFileSize {
		return c.Reply(fmt.Sprintf("File too large. Max: %d MB.", MaxSizeMb))
	}

	timestamp := time.Now().Unix()
	audioPath := filepath.Join(b.AudioStoragePath, fmt.Sprintf("voice_%d_%d%s", user.ID, timestamp, ext))

	if err := os.MkdirAll(b.AudioStoragePath, 0755); err != nil {
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

	log.Printf("Voice saved: %s", audioPath)

	ctx := b.getCtx(c)

	// Единая логика обработки
	err = b.processAudio(ctx, user.ID, audioPath, fmt.Sprintf("Voice message %s", time.Now().Format("2006-01-02 15:04")))
	if err != nil {
		log.Printf("processAudio failed: %v", err)
		return c.Reply("Failed to start transcription.")
	}

	return c.Reply("Сообщение принято для обработки!", &telebot.SendOptions{ParseMode: "Markdown"})
}
