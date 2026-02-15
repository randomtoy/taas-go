package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/randomtoy/taas-go/internal/adapters/decks"
	httpadapter "github.com/randomtoy/taas-go/internal/adapters/http"
	"github.com/randomtoy/taas-go/internal/adapters/llm/openrouter"
	"github.com/randomtoy/taas-go/internal/app"
	"github.com/randomtoy/taas-go/internal/config"
)

// stdRNG delegates to math/rand/v2 (auto-seeded).
type stdRNG struct{}

func (stdRNG) Intn(n int) int { return rand.IntN(n) }

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	deckStore := decks.NewEmbeddedStore()

	llmClient := openrouter.NewClient(
		&http.Client{Timeout: cfg.LLMTimeout},
		cfg.OpenRouterAPIKey,
		cfg.OpenRouterBaseURL,
		cfg.LLMModel,
		cfg.LLMFallbackModels,
		logger,
	)

	svc := app.NewTarotService(deckStore, llmClient, stdRNG{}, cfg.LLMModel)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(httpadapter.RequestIDMiddleware())
	e.Use(httpadapter.LoggingMiddleware(logger))

	handler := httpadapter.NewHandler(svc)
	handler.Register(e)

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		logger.Info("starting server", "addr", cfg.HTTPAddr)
		if err := e.Start(cfg.HTTPAddr); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
