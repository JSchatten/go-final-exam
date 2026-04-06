package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
	"gopkg.in/telebot.v3"
)

var AllowedAudioExtensions = map[string]bool{
	".ogg": true,
	".oga": true,
	".mp3": true,
	".wav": true,
}

const MaxFileSize = 500 * 1024 * 1024 // 500 МБ

func (b *Bot) HandleVoice(c telebot.Context) error {
	voice := c.Message().Voice
	user := c.Sender()

	log.Printf("Received voice message: %d sec, FileID: %s", voice.Duration, voice.FileID)

	file, err := b.Telebot.FileByID(voice.FileID)
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return c.Send("Failed to load audio file.")
	}

	filename := filepath.Base(file.FilePath)
	if filename == "." || filename == ".." {
		log.Printf("Invalid filename: %q", filename)
		return c.Send("Invalid file.")
	}

	if voice.FileSize > MaxFileSize {
		return c.Send(fmt.Sprintf(
			"File too large: %.1f MB. Max: 500 MB.",
			float64(voice.FileSize)/(1024*1024),
		))
	}

	ext := strings.ToLower(filepath.Ext(file.FilePath))
	if !AllowedAudioExtensions[ext] {
		var formats []string
		for ext := range AllowedAudioExtensions {
			formats = append(formats, strings.ToUpper(strings.TrimPrefix(ext, ".")))
		}
		return c.Send(fmt.Sprintf("Format %s not supported. Supported: %s.", ext, strings.Join(formats, ", ")))
	}

	audioDir := b.AudioStoragePath
	timestamp := time.Now().Unix()
	audioPath := filepath.Join(audioDir, fmt.Sprintf("voice_%d_%d%s", user.ID, timestamp, filepath.Ext(filename)))

	if err := os.MkdirAll(audioDir, 0755); err != nil {
		log.Printf("Failed to create dir %s: %v", audioDir, err)
		return c.Send("Failed to save audio.")
	}

	outFile, err := os.Create(audioPath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", audioPath, err)
		return c.Send("Failed to save audio.")
	}
	defer outFile.Close()

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.Telebot.Token, file.FilePath)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return c.Send("Failed to download audio.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("HTTP error on download: %d", resp.StatusCode)
		return c.Send("Failed to download audio.")
	}

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		log.Printf("Failed to write file: %v", err)
		return c.Send("Failed to save audio.")
	}

	log.Printf("Audio saved: %s", audioPath)

	meeting := &models.Meeting{
		ID:            uuid.New(),
		UserID:        user.ID,
		Title:         fmt.Sprintf("Meeting %s", time.Now().Format("2006-01-02 15:04")),
		AudioFilePath: &audioPath,
		Status:        models.StatusProcessing,
		CreatedAt:     time.Now().UTC(),
	}

	err = b.MeetingRepo.Create(context.Background(), meeting)
	if err != nil {
		log.Printf("Failed to create meeting: %v", err)
		return c.Send("Failed to save meeting.")
	}

	requestFileID, err := b.SaluteSpeech.UploadFileByPath(audioPath)
	if err != nil {
		log.Printf("Failed to upload file to SaluteSpeech: %v", err)
		setMeetingError(b, context.Background(), meeting, "upload failed")
		return c.Send("Failed to send file for transcription.")
	}

	taskID, taskStatus, err := b.SaluteSpeech.CreateRecognitionTask(audioPath, requestFileID)
	if err != nil {
		log.Printf("Failed to create recognition task: %v", err)
		setMeetingError(b, context.Background(), meeting, "task creation failed")
		return c.Send("Failed to create transcription task.")
	}

	log.Printf("Transcription task created: ID=%s, Status=%s", taskID, taskStatus)

	_, err = b.TranscriptionRepo.Create(context.Background(), meeting.ID, taskID, taskStatus)
	if err != nil {
		log.Printf("Failed to create transcription: %v", err)
		setMeetingError(b, context.Background(), meeting, "transcription failed")
		return c.Send("Не удалось сохранить задачу распознавания.")
	}

	// Обновляем только статус встречи
	meeting.Status = models.StatusProcessing
	err = b.MeetingRepo.UpdateMeeting(context.Background(), meeting)
	if err != nil {
		log.Printf("Failed to update meeting status: %v", err)
		// Не фатально
	}

	message := "Сообщение принято для обработки!"

	return c.Send(message, &telebot.SendOptions{ParseMode: "Markdown"})
}

func setMeetingError(b *Bot, ctx context.Context, meeting *models.Meeting, msg string) {
	log.Println("Error:", msg)
	meeting.Status = models.StatusFailed
	meeting.ErrorMessage.Valid = true
	meeting.ErrorMessage.String = msg
	_ = b.MeetingRepo.UpdateMeeting(ctx, meeting)
}
