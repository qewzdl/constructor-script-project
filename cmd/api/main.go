package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"constructor-script-backend/internal/app"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/validator"
)

func main() {
	logger.Init()
	logger.Info("Starting Blog Backend API", nil)

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables", nil)
	}

	cfg := config.New()
	validator.Init()

	application, err := app.New(cfg, app.Options{TemplatesDir: "./templates"})
	if err != nil {
		logger.Error(err, "Failed to initialize application", nil)
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := application.Run(); err != nil && err != http.ErrServerClosed {
			logger.Error(err, "Failed to start server", nil)
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	logger.Info("Shutting down server...", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Error(err, "Server forced to shutdown", nil)
		log.Fatal(err)
	}

	logger.Info("Server exited gracefully", nil)
}
