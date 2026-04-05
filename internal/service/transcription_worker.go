package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JSchatten/go-final-exam/internal/integration/salutespeech"
	"github.com/JSchatten/go-final-exam/internal/models"
	"github.com/google/uuid"
)

func (b *Bot) runTranscription(ctx context.Context, meeting *models.Meeting) error {
	audioPath := *meeting.AudioFilePath

	fileID, err := b.SaluteSpeech.UploadFileByPath(audioPath)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	taskID, _, err := b.SaluteSpeech.CreateRecognitionTask(audioPath, fileID)
	if err != nil {
		return fmt.Errorf("create task failed: %w", err)
	}

	var taskResult *salutespeech.TaskResult
	for {
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}

		taskResult, err = b.SaluteSpeech.CheckTaskStatus(taskID)
		if err != nil {
			return err
		}

		switch taskResult.Status {
		case "DONE":
			break
		case "ERROR", "CANCELED":
			return fmt.Errorf("task failed: %s", taskResult.Status)
		default:
			continue
		}
		break
	}

	recognition, err := b.SaluteSpeech.GetRecognitionResult(taskResult.ResponseFileID)
	if err != nil {
		return fmt.Errorf("get result failed: %w", err)
	}

	text := recognition.GetFullText()
	transID, err := b.TranscriptionRepo.Create(meeting.ID, text, "")
	if err != nil {
		return err
	}

	return b.MeetingRepo.UpdateTranscription(meeting.ID, transID)
}

func (b *Bot) summaryWorker(ctx context.Context, meetingID uuid.UUID) {
	const maxRetries = 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := b.runSummary(ctx, meetingID)
		if err == nil {
			_ = b.MeetingRepo.UpdateStatus(meetingID, models.StatusCompleted)
			// Получаем встречу для уведомления
			meeting, err := b.MeetingRepo.GetByMeetingID(meetingID)
			if err != nil || meeting == nil {
				log.Printf("Failed to load meeting %s after successful summary", meetingID)
				return
			}
			b.notifyUserOfSuccess(meeting.UserID, meeting.Title)
			return
		}

		log.Printf("Summary attempt %d failed for meeting %s: %v", attempt, meetingID, err)

		if attempt == maxRetries {
			_ = b.MeetingRepo.UpdateStatus(meetingID, models.StatusFailed)

			// Загружаем встречу, чтобы получить UserID и Title
			meeting, err := b.MeetingRepo.GetByMeetingID(meetingID)
			if err != nil || meeting == nil {
				log.Printf("Failed to load meeting %s for failure notification", meetingID)
				return
			}

			b.notifyUserOfFailure(meeting.UserID, meeting.Title)
			return
		}

		select {
		case <-time.After(time.Second * time.Duration(attempt*3)):
		case <-ctx.Done():
			return
		}
	}
}

func (b *Bot) runSummary(ctx context.Context, meetingID uuid.UUID) error {
	// Ждём, пока будет транскрипция
	transcription, err := b.TranscriptionRepo.GetByMeetingID(meetingID)
	if err != nil || transcription.FullText == "" {
		return fmt.Errorf("transcription not ready")
	}

	summary, err := b.GigaChat.Transcribe(transcription.FullText)
	if err != nil {
		return err
	}

	sumID, err := b.SummaryRepo.Create(meetingID, summary)
	if err != nil {
		return err
	}

	return b.MeetingRepo.UpdateSummary(meetingID, sumID)
}

func (b *Bot) transcriptionWorker(ctx context.Context, meeting *models.Meeting) {
	const maxRetries = 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := b.runTranscription(ctx, meeting)
		if err == nil {
			// Успех → переходим к следующему шагу
			_ = b.MeetingRepo.UpdateStatus(meeting.ID, models.StatusSummarizing)
			go b.summaryWorker(context.Background(), meeting.ID)
			return
		}

		log.Printf("Transcription attempt %d failed for meeting %s: %v", attempt, meeting.ID, err)

		if attempt == maxRetries {
			_ = b.MeetingRepo.UpdateStatus(meeting.ID, models.StatusFailed)
			_ = b.MeetingRepo.UpdateError(meeting.ID, err.Error())
			b.notifyUserOfFailure(meeting.UserID, meeting.Title)
			return
		}

		select {
		case <-time.After(time.Second * time.Duration(attempt*5)):
		case <-ctx.Done():
			return
		}
	}
}
