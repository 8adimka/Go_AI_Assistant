package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/8adimka/Go_AI_Assistant/internal/chat"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/assistant"
	"github.com/8adimka/Go_AI_Assistant/internal/chat/model"
	"github.com/8adimka/Go_AI_Assistant/internal/config"
	"github.com/8adimka/Go_AI_Assistant/internal/httpx"
	"github.com/8adimka/Go_AI_Assistant/internal/mongox"
	"github.com/8adimka/Go_AI_Assistant/internal/pb"
	"github.com/gorilla/mux"
	"github.com/twitchtv/twirp"
)

func main() {
	// Load configuration from .env file
	cfg := config.Load()

	// Set OpenAI API key for the assistant
	os.Setenv("OPENAI_API_KEY", cfg.OpenAIApiKey)

	// Connect to MongoDB
	mongo := mongox.MustConnect(cfg.MongoURI, "acai")

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.Logger(),
		httpx.Recovery(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Start the server
	slog.Info("Starting the server...", "port", "8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
