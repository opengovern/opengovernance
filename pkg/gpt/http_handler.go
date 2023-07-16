package gpt

import (
	"fmt"
	"go.uber.org/zap"
)

type HttpHandler struct {
	logger *zap.Logger
}

func InitializeHttpHandler(
	logger *zap.Logger,
) (h *HttpHandler, err error) {

	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	h.logger = logger
	return h, nil
}
