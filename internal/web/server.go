package web

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
)

type Server struct {
	httpServer *http.Server
	handlers   *Handlers
	botAPI     *tgbotapi.BotAPI
}

func NewServer(addr string, repo *postgres.SituationRepository, botToken string) (*Server, error) {
	botAPI, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API for web: %w", err)
	}

	handlers := NewHandlers(repo)

	mux := http.NewServeMux()

	s := &Server{
		handlers: handlers,
		botAPI:   botAPI,
	}

	// API routes
	mux.HandleFunc("/api/start", s.methodPost(handlers.StartGame))
	mux.HandleFunc("/api/next-photo", s.methodPost(handlers.NextPhoto))
	mux.HandleFunc("/api/answer", s.methodPost(handlers.ShowAnswer))
	mux.HandleFunc("/api/next-round", s.methodPost(handlers.NextRound))
	mux.HandleFunc("/api/stats", s.methodGet(handlers.Stats))
	mux.HandleFunc("/api/photo/", s.servePhoto)
	mux.Handle("/", http.FileServer(http.Dir("internal/web/static")))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s, nil
}

func (s *Server) Run() error {
	log.Printf("Web server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) servePhoto(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/photo/")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := s.botAPI.GetFile(fileConfig)
	if err != nil {
		log.Printf("Error getting file from Telegram: %v", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	fileURL := file.Link(s.botAPI.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Error downloading file: %v", err)
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, resp.Body)
}

func (s *Server) methodPost(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

func (s *Server) methodGet(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}