package httpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a middleware and zap to provide an "access log" like logging for each request.
func Logger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			req := c.Request()
			fields := []zapcore.Field{
				zap.String("remote_ip", c.RealIP()),
				zap.String("latency", time.Since(start).String()),
				zap.String("host", req.Host),
				zap.String("request", fmt.Sprintf("%s %s", req.Method, req.RequestURI)),
				zap.String("user_agent", req.UserAgent()),
			}
			id := req.Header.Get(echo.HeaderXRequestID)
			if id != "" {
				fields = append(fields, zap.String("request_id", id))
			}
			log.Info("Request received", fields...)
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			res := c.Response()

			// skip metric endpoints
			if strings.HasPrefix(c.Path(), "/metrics") {
				return nil
			}

			fields = append(fields,
				zap.Int("status", res.Status),
				zap.Int64("size", res.Size),
			)

			if id == "" {
				id = res.Header().Get(echo.HeaderXRequestID)
				fields = append(fields, zap.String("request_id", id))
			}

			n := res.Status
			switch {
			case n >= 500:
				log.With(zap.Error(err)).Error("Server error", fields...)
			case n >= 400:
				log.With(zap.Error(err)).Warn("Client error", fields...)
			case n >= 300:
				log.Info("Redirection", fields...)
			default:
				log.Info("Success", fields...)
			}

			return nil
		}
	}
}
