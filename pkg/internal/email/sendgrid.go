package email

import (
	"context"
	"fmt"

	sendgridgo "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.uber.org/zap"
)

//go:generate mockery --name Service
type Service interface {
	SendEmail(ctx context.Context, email, password, workspace string) error
}

type sendGridClient struct {
	client     *sendgridgo.Client
	sender     string
	senderName string
	logger     *zap.Logger
}

func NewSendGripClient(apiKey, sender, senderName string, logger *zap.Logger) Service {
	return sendGridClient{
		client:     sendgridgo.NewSendClient(apiKey),
		sender:     sender,
		senderName: senderName,
		logger:     logger,
	}
}

func (c sendGridClient) SendEmail(ctx context.Context, email, password, workspace string) error {
	var msg string

	if password != "" {
		msg = fmt.Sprintf(`You have been added to a "Service".
			A one-time password has been generated for you to enter the system.
			Use it, after which you will be prompted to come up with a new password.
			Your first login password :%s`, password)
	} else {
		msg = fmt.Sprintf(`You have been invited to a new workspace : %s .
			To log in, use the login and password of an existing account in the service.`, workspace)
	}

	from := mail.NewEmail(c.senderName, c.sender)
	subject := "Invite to a Service"
	to := mail.NewEmail(email, email)

	message := mail.NewSingleEmail(from, subject, to, msg, "")
	resp, err := c.client.Send(message)
	if err != nil {
		c.logger.Error("send email error",
			zap.Error(err))
		return err
	}

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		c.logger.Error("send email error",
			zap.String("recipient email:", email),
			zap.String("reponse", resp.Body),
		)
		return err
	}

	c.logger.Info("Letter sent",
		zap.String("user email:", email))
	return nil
}
