package openai

import (
	"bytes"
	"context"
	"github.com/sashabaranov/go-openai"
	"text/template"
)

func (s *Service) NewThread(ctx context.Context) (openai.Thread, error) {
	return s.client.CreateThread(ctx, openai.ThreadRequest{})
}

func (s *Service) SendChatPrompt(ctx context.Context, threadID string) (openai.Message, error) {
	tmpl := template.New("test")
	tm, err := tmpl.Parse(s.ChatPrompt)
	if err != nil {
		panic(err)
	}
	var outputExecute bytes.Buffer
	err = tm.Execute(&outputExecute, s)
	if err != nil {
		panic(err)
	}

	return s.client.CreateMessage(ctx, threadID, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: outputExecute.String(),
	})
}
func (s *Service) SendMessage(ctx context.Context, threadID, content string) (openai.Message, error) {
	return s.client.CreateMessage(ctx, threadID, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	})
}

func (s *Service) RunThread(ctx context.Context, threadID string, id *string) (openai.Run, error) {
	if id == nil || len(*id) == 0 {
		return s.client.CreateRun(ctx, threadID, openai.RunRequest{
			AssistantID: s.assistant.ID,
		})
	}
	return s.client.RetrieveRun(ctx, threadID, *id)
}

func (s *Service) RetrieveRun(ctx context.Context, threadID, runID string) (openai.Run, error) {
	return s.client.RetrieveRun(ctx, threadID, runID)
}

func (s *Service) StopAllRun(ctx context.Context, threadID string) error {
	runs, err := s.client.ListRuns(ctx, threadID, openai.Pagination{})
	if err != nil {
		return err
	}

	for _, run := range runs.Runs {
		if run.Status == openai.RunStatusInProgress || run.Status == openai.RunStatusQueued {
			_, err := s.client.CancelRun(ctx, threadID, run.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ListMessages(ctx context.Context, threadID string) (openai.MessagesList, error) {
	return s.client.ListMessage(ctx, threadID, nil, nil, nil, nil)
}
