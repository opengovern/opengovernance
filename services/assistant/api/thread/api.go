package thread

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/assistant/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/assistant/model"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/labstack/echo/v4"
	openai2 "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type API struct {
	tracer trace.Tracer
	logger *zap.Logger
	oc     *openai.Service
	db     repository.Run
}

func New(logger *zap.Logger, oc *openai.Service, db repository.Run) API {
	return API{
		tracer: otel.GetTracerProvider().Tracer("assistant.http.sources"),
		logger: logger.Named("source"),
		oc:     oc,
		db:     db,
	}
}

// ListMessages godoc
//
//	@Summary		List messages of a thread
//	@Description	List messages of a thread
//	@Security		BearerToken
//	@Tags			assistant
//	@Produce		json
//	@Success		200			{object}	entity.ListMessagesResponse
//	@Param			thread_id	path		string	true	"Thread ID"
//	@Param			run_id		query		string	false	"Run ID"
//	@Router			/assistant/api/v1/thread/{thread_id} [get]
func (s API) ListMessages(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	threadID := c.Param("thread_id")
	runID := c.QueryParam("run_id")

	msgs, err := s.oc.ListMessages(threadID)
	if err != nil {
		s.logger.Error("failed to read msgs from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var respMsgs []entity.Message
	for _, msg := range msgs.Messages {
		contentString := ""
		for _, content := range msg.Content {
			if content.Text != nil {
				contentString += content.Text.Value + "\n"
			}
		}
		respMsgs = append(respMsgs, entity.Message{Content: contentString, Role: msg.Role})
	}

	var status openai2.RunStatus
	if len(runID) > 0 {
		run, err := s.oc.RetrieveRun(threadID, runID)
		if err != nil {
			return err
		}
		status = run.Status
	}
	return c.JSON(http.StatusOK, entity.ListMessagesResponse{Messages: respMsgs, Status: string(status)})
}

// SendMessage godoc
//
//	@Summary		Send a message [standalone]
//	@Description	Send a message [standalone]
//	@Security		BearerToken
//	@Tags			assistant
//	@Produce		json
//	@Param			request	body		entity.SendMessageRequest	true	"Request"
//	@Success		200		{object}	entity.SendMessageResponse
//	@Router			/assistant/api/v1/thread [post]
func (s API) SendMessage(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := s.tracer.Start(ctx, "send.message")
	defer span.End()

	var req entity.SendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var threadID string
	if req.ThreadID == nil || len(*req.ThreadID) == 0 {
		th, err := s.oc.NewThread()
		if err != nil {
			return fmt.Errorf("newThread failed due to %v", err)
		}
		threadID = th.ID

		_, err = s.oc.SendChatPrompt(threadID)
		if err != nil {
			return fmt.Errorf("SendChatPrompt failed due to %v", err)
		}
	} else {
		threadID = *req.ThreadID
	}

	_, err := s.oc.SendMessage(threadID, req.Content)
	if err != nil {
		return fmt.Errorf("SendMessage failed due to %v", err)
	}

	run, err := s.oc.RunThread(threadID, req.RunID)
	if err != nil {
		return fmt.Errorf("RunThread failed due to %v", err)
	}

	err = s.db.Create(context.Background(), model.Run{ID: run.ID, ThreadID: threadID, Status: openai2.RunStatusQueued, UpdatedAt: time.Now()})
	if err != nil {
		return fmt.Errorf("db.Create failed due to %v", err)
	}

	return c.JSON(http.StatusOK, entity.SendMessageResponse{
		RunID:    run.ID,
		ThreadID: threadID,
	})
}

func (s API) Register(g *echo.Group) {
	g.GET("/:thread_id", httpserver.AuthorizeHandler(s.ListMessages, api.EditorRole))
	g.POST("", httpserver.AuthorizeHandler(s.SendMessage, api.EditorRole))
}
