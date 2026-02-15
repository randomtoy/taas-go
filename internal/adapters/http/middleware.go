package http

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

const headerRequestID = "X-Request-Id"

// RequestIDMiddleware ensures every request has a unique X-Request-Id.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(headerRequestID)
			if id == "" {
				id = generateID()
			}
			c.Response().Header().Set(headerRequestID, id)
			c.Set("request_id", id)
			return next(c)
		}
	}
}

// LoggingMiddleware logs each request with structured fields.
func LoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			logger.Info("request",
				"request_id", c.Get("request_id"),
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", c.Response().Status,
				"latency_ms", time.Since(start).Milliseconds(),
			)
			return err
		}
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
