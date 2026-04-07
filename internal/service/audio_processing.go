// internal/service/audio_processing.go
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
)

// processAudio - унифицированная логика обработки аудиофайла.
// Вызывается из HandleVoice и HandleAudio.
func (b *BotService) processAudio(ctx context.Context, userID int64, audioPath, title string) error {
	meeting := &models.Meeting{
		ID:            uuid.New(),
		UserID:        userID,
		Title:         title,
		AudioFilePath: &audioPath,
		Status:        models.MeetingStatusUploaded,
		CreatedAt:     time.Now().UTC(),
	}

	err := b.MeetingRepo.Create(ctx, meeting)
	if err != nil {
		return fmt.Errorf("failed to create meeting: %w", err)
	}

	// Загружаем файл в SaluteSpeech
	requestFileID, err := b.SaluteSpeech.UploadFileByPath(audioPath)
	if err != nil {
		setMeetingError(b, ctx, meeting, "upload failed")
		return fmt.Errorf("failed to upload file to SaluteSpeech: %w", err)
	}

	// Создаём задачу распознавания
	taskID, taskStatus, err := b.SaluteSpeech.CreateRecognitionTask(audioPath, requestFileID)
	if err != nil {
		setMeetingError(b, ctx, meeting, "task creation failed")
		return fmt.Errorf("failed to create recognition task: %w", err)
	}

	log.Printf("Transcription task created: meeting_id=%s, task_id=%s, status=%s", meeting.ID, taskID, taskStatus)

	// Сохраняем транскрипцию
	_, err = b.TranscriptionRepo.Create(ctx, meeting.ID, taskID, taskStatus)
	if err != nil {
		setMeetingError(b, ctx, meeting, "transcription save failed")
		return fmt.Errorf("failed to save transcription: %w", err)
	}

	// Обновляем статус встречи
	meeting.Status = models.MeetingStatusProcessing
	err = b.MeetingRepo.UpdateMeeting(ctx, meeting)
	if err != nil {
		log.Printf("Warning: failed to update meeting status: %v", err)
	}

	return nil
}

func setMeetingError(b *BotService, ctx context.Context, meeting *models.Meeting, msg string) {
	log.Println("Error:", msg)
	meeting.Status = models.MeetingStatusFailed
	meeting.ErrorMessage.Valid = true
	meeting.ErrorMessage.String = msg
	_ = b.MeetingRepo.UpdateMeeting(ctx, meeting)
}
