package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/chat"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/health"
	"github.com/8adimka/Go_AI_Assistant/internal/httpx"
	"github.com/8adimka/Go_AI_Assistant/internal/mongox"
	"github.com/8adimka/Go_AI_Assistant/internal/otel"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/gorilla/mux"
	"github.com/twitchtv/twirp"
)

func main() {
	ctx := context.Background()

	// Load configuration from .env file
	cfg := config.Load()

	// Initialize OpenTelemetry
	shutdown, err := otel.InitOpenTelemetry(ctx, "go-ai-assistant")
	if err != nil {
		slog.Error("Failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	// Set OpenAI API key for the assistant
	os.Setenv("OPENAI_API_KEY", cfg.OpenAIApiKey)

	// Connect to MongoDB
	mongo := mongox.MustConnect(cfg.MongoURI, "acai")

	// Initialize metrics - temporarily disabled due to type issues
	// meter := otel.GetMeter()
	// appMetrics, err := metrics.NewMetrics(meter)
	// if err != nil {
	// 	slog.Error("Failed to initialize metrics", "error", err)
	// 	os.Exit(1)
	// }

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.OTelMiddleware(),
		httpx.Logger(),
		httpx.Recovery(),
	)

	// Health checks
	healthChecker := health.NewHealthChecker(mongo.Client())
	handler.HandleFunc("/health", healthChecker.HealthHandler)
	handler.HandleFunc("/ready", healthChecker.ReadyHandler)

	// Metrics endpoint
	handler.Handle("/metrics", http.DefaultServeMux)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Start the server with graceful shutdown
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Starting the server...", "port", "8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
