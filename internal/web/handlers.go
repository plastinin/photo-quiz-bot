package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/plastinin/photo-quiz-bot/internal/domain"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
)

type Handlers struct {
	game *service.GameService
	repo *postgres.SituationRepository

	// Отдельное состояние для веба
	webState *WebGameState
	mu       sync.RWMutex
}

type WebGameState struct {
	CurrentSituation *domain.SituationWithPhotos
	CurrentPhotoIdx  int
}

func NewHandlers(repo *postgres.SituationRepository) *Handlers {
	return &Handlers{
		game:     service.NewGameService(repo),
		repo:     repo,
		webState: &WebGameState{},
	}
}

// Response structures
type GameResponse struct {
	Success      bool   `json:"success"`
	PhotoURL     string `json:"photoUrl,omitempty"`
	CurrentPhoto int    `json:"currentPhoto,omitempty"`
	TotalPhotos  int    `json:"totalPhotos,omitempty"`
	Answer       string `json:"answer,omitempty"`
	Message      string `json:"message,omitempty"`
	HasMore      bool   `json:"hasMore"`
	GameOver     bool   `json:"gameOver"`
}

type StatsResponse struct {
	Total     int `json:"total"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

// POST /api/start
func (h *Handlers) StartGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			h.jsonResponse(w, GameResponse{
				Success:  false,
				Message:  "Нет доступных ситуаций",
				GameOver: true,
			})
			return
		}
		h.errorResponse(w, "Ошибка запуска игры", http.StatusInternalServerError)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	h.jsonResponse(w, GameResponse{
		Success:      true,
		PhotoURL:     h.getPhotoURL(ctx, photo.FileID),
		CurrentPhoto: current,
		TotalPhotos:  total,
		HasMore:      current < total,
	})
}

// POST /api/next-photo
func (h *Handlers) NextPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photo, err := h.game.NextPhoto(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoMorePhotos) {
			h.jsonResponse(w, GameResponse{
				Success: false,
				Message: "Больше нет фотографий",
				HasMore: false,
			})
			return
		}
		if errors.Is(err, service.ErrGameNotStarted) {
			h.jsonResponse(w, GameResponse{
				Success: false,
				Message: "Сначала начните игру",
			})
			return
		}
		h.errorResponse(w, "Ошибка", http.StatusInternalServerError)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	h.jsonResponse(w, GameResponse{
		Success:      true,
		PhotoURL:     h.getPhotoURL(ctx, photo.FileID),
		CurrentPhoto: current,
		TotalPhotos:  total,
		HasMore:      current < total,
	})
}

// POST /api/answer
func (h *Handlers) ShowAnswer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	answer, err := h.game.GetAnswer(ctx)
	if err != nil {
		if errors.Is(err, service.ErrGameNotStarted) {
			h.jsonResponse(w, GameResponse{
				Success: false,
				Message: "Сначала начните игру",
			})
			return
		}
		h.errorResponse(w, "Ошибка", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, GameResponse{
		Success: true,
		Answer:  answer,
	})
}

// POST /api/next-round
func (h *Handlers) NextRound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Завершаем текущий раунд
	_ = h.game.FinishRound(ctx)

	// Начинаем новый
	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			h.jsonResponse(w, GameResponse{
				Success:  false,
				Message:  "Все ситуации сыграны! 🎉",
				GameOver: true,
			})
			return
		}
		h.errorResponse(w, "Ошибка", http.StatusInternalServerError)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	h.jsonResponse(w, GameResponse{
		Success:      true,
		PhotoURL:     h.getPhotoURL(ctx, photo.FileID),
		CurrentPhoto: current,
		TotalPhotos:  total,
		HasMore:      current < total,
	})
}

// GET /api/stats
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	total, used, remaining, err := h.game.GetStats(ctx)
	if err != nil {
		h.errorResponse(w, "Ошибка получения статистики", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, StatsResponse{
		Total:     total,
		Used:      used,
		Remaining: remaining,
	})
}

// getPhotoURL возвращает URL для получения фото из Telegram
func (h *Handlers) getPhotoURL(ctx context.Context, fileID string) string {
	return "/api/photo/" + fileID
}

func (h *Handlers) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) errorResponse(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(GameResponse{
		Success: false,
		Message: message,
	})
}