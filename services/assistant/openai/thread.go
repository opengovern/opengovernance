package openai

import (
	"context"
	"github.com/sashabaranov/go-openai"
)

func (s *Service) NewThread() (openai.Thread, error) {
	return s.client.CreateThread(context.Background(), openai.ThreadRequest{
		Messages: nil,
		Metadata: nil,
	})
}

func (s *Service) SendMessage(threadID, content string) (openai.Message, error) {
	return s.client.CreateMessage(context.Background(), threadID, openai.MessageRequest{
		Role:     openai.ChatMessageRoleUser,
		Content:  content,
		FileIds:  nil,
		Metadata: nil,
	})
}

func (s *Service) StopAllRun(threadID string) error {
	runs, err := s.client.ListRuns(context.Background(), threadID, openai.Pagination{
		Limit:  nil,
		Order:  nil,
		After:  nil,
		Before: nil,
	})
	if err != nil {
		return err
	}

	for _, run := range runs.Runs {
		if run.Status == openai.RunStatusInProgress || run.Status == openai.RunStatusQueued {
			_, err := s.client.CancelRun(context.Background(), threadID, run.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ListMessages(threadID string) (openai.MessagesList, error) {
	return s.client.ListMessage(context.Background(), threadID, nil, nil, nil, nil)
}
