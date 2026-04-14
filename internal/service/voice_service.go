package service

import (
	"fmt"
	"io"
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

	b.Logger.Debug().Msgf("Received voice message: %d sec, FileID: %s", voice.Duration, voice.FileID)

	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to get file info: %v", err)
		return c.Reply("Failed to load audio file.")
	}

	ext := strings.ToLower(filepath.Ext(file.FilePath))
	if !AllowedAudioExtensions[ext] {
		b.Logger.Warn().Msgf("Unsupported audio format: %s", ext)
		return c.Reply("Unsupported audio format.")
	}

	if voice.FileSize > MaxFileSize {
		b.Logger.Warn().Msgf("File too large. Max: %d MB.", MaxSizeMb)
		return c.Reply(fmt.Sprintf("File too large. Max: %d MB.", MaxSizeMb))
	}

	timestamp := time.Now().Unix()
	audioPath := filepath.Join(b.AudioStoragePath, fmt.Sprintf("voice_%d_%d%s", user.ID, timestamp, ext))

	if err := os.MkdirAll(b.AudioStoragePath, 0755); err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to create dir: %v", err)
		return c.Reply("Failed to save audio.")
	}

	outFile, err := os.Create(audioPath)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to create file: %v", err)
		return c.Reply("Failed to save audio.")
	}
	defer outFile.Close()

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Telebot.Token, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to download file: %v", err)
		return c.Reply("Failed to download audio.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.Logger.Error().Msgf("HTTP error on download: %d", resp.StatusCode)
		return c.Reply("Failed to download audio.")
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		b.Logger.Error().Err(err).Msgf("Failed to write file: %v", err)
		return c.Reply("Failed to save audio.")
	}

	b.Logger.Debug().Msgf("Voice saved: %s", audioPath)

	ctx := b.getCtx(c)

	// Единая логика обработки
	err = b.processAudio(ctx, user.ID, audioPath, fmt.Sprintf("Voice message %s", time.Now().Format("2006-01-02 15:04")))
	if err != nil {
		b.Logger.Error().Err(err).Msgf("processAudio failed: %v", err)
		return c.Reply("Failed to start transcription.")
	}

	return c.Reply("Сообщение принято для обработки!", &telebot.SendOptions{ParseMode: "Markdown"})
}
