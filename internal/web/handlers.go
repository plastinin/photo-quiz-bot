package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/plastinin/photo-quiz-bot/internal/domain"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
	"github.com/plastinin/photo-quiz-bot/internal/service"
)

type Handlers struct {
	game    *service.GameService
	repo    *postgres.SituationRepository
	session *SessionManager
}

func NewHandlers(repo *postgres.SituationRepository, session *SessionManager) *Handlers {
	return &Handlers{
		game:    service.NewGameService(repo),
		repo:    repo,
		session: session,
	}
}

type GameResponse struct {
	Success       bool                  `json:"success"`
	PhotoURL      string                `json:"photoUrl,omitempty"`
	CurrentPhoto  int                   `json:"currentPhoto,omitempty"`
	TotalPhotos   int                   `json:"totalPhotos,omitempty"`
	Answer        string                `json:"answer,omitempty"`
	Message       string                `json:"message,omitempty"`
	HasMore       bool                  `json:"hasMore"`
	GameOver      bool                  `json:"gameOver"`
	CurrentPlayer *domain.Player        `json:"currentPlayer,omitempty"`
	Scoreboard    []domain.PlayerScore  `json:"scoreboard,omitempty"`
	NeedScore     bool                  `json:"needScore,omitempty"`
}

type StatsResponse struct {
	Total     int `json:"total"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

type SessionResponse struct {
	Success       bool                  `json:"success"`
	Message       string                `json:"message,omitempty"`
	Session       *domain.GameSession   `json:"session,omitempty"`
	CurrentPlayer *domain.Player        `json:"currentPlayer,omitempty"`
	Scoreboard    []domain.PlayerScore  `json:"scoreboard,omitempty"`
}

type CreateSessionRequest struct {
	Players []string `json:"players"`
}

func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if len(req.Players) < 1 {
		h.errorResponse(w, "Нужен хотя бы один игрок", http.StatusBadRequest)
		return
	}

	if len(req.Players) > 10 {
		h.errorResponse(w, "Максимум 10 игроков", http.StatusBadRequest)
		return
	}

	for _, name := range req.Players {
		if name == "" {
			h.errorResponse(w, "Имя игрока не может быть пустым", http.StatusBadRequest)
			return
		}
	}

	session := h.session.CreateSession(req.Players)
	currentPlayer := h.session.GetCurrentPlayer()

	h.jsonResponse(w, SessionResponse{
		Success:       true,
		Session:       session,
		CurrentPlayer: currentPlayer,
		Scoreboard:    h.session.GetScoreboard(),
	})
}

func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	session := h.session.GetSession()
	if session == nil {
		h.jsonResponse(w, SessionResponse{
			Success: false,
			Message: "Сессия не создана",
		})
		return
	}

	h.jsonResponse(w, SessionResponse{
		Success:       true,
		Session:       session,
		CurrentPlayer: h.session.GetCurrentPlayer(),
		Scoreboard:    h.session.GetScoreboard(),
	})
}

func (h *Handlers) EndSession(w http.ResponseWriter, r *http.Request) {
	scoreboard := h.session.FinishGame()
	if scoreboard == nil {
		h.errorResponse(w, "Нет активной сессии", http.StatusBadRequest)
		return
	}

	h.jsonResponse(w, SessionResponse{
		Success:    true,
		Message:    "Игра завершена",
		Scoreboard: scoreboard,
	})
}

func (h *Handlers) StartGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.session.HasActiveSession() {
		h.jsonResponse(w, GameResponse{
			Success: false,
			Message: "Сначала создайте сессию с игроками",
		})
		return
	}

	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			scoreboard := h.session.FinishGame()
			h.jsonResponse(w, GameResponse{
				Success:    false,
				Message:    "Нет доступных ситуаций",
				GameOver:   true,
				Scoreboard: scoreboard,
			})
			return
		}
		h.errorResponse(w, "Ошибка запуска игры", http.StatusInternalServerError)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	h.jsonResponse(w, GameResponse{
		Success:       true,
		PhotoURL:      h.getPhotoURL(ctx, photo.FileID),
		CurrentPhoto:  current,
		TotalPhotos:   total,
		HasMore:       current < total,
		CurrentPlayer: h.session.GetCurrentPlayer(),
		Scoreboard:    h.session.GetScoreboard(),
	})
}

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

	h.session.NotifyTurnEnd()

	h.jsonResponse(w, GameResponse{
		Success:       true,
		Answer:        answer,
		NeedScore:     true,
		CurrentPlayer: h.session.GetCurrentPlayer(),
	})
}

func (h *Handlers) NextRound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	nextPlayer := h.session.NextPlayer()

	_ = h.game.FinishRound(ctx)

	photo, err := h.game.StartNewRound(ctx)
	if err != nil {
		if errors.Is(err, service.ErrNoSituations) {
			scoreboard := h.session.FinishGame()
			h.jsonResponse(w, GameResponse{
				Success:    false,
				Message:    "Все ситуации сыграны! 🎉",
				GameOver:   true,
				Scoreboard: scoreboard,
			})
			return
		}
		h.errorResponse(w, "Ошибка", http.StatusInternalServerError)
		return
	}

	current, total, _ := h.game.GetCurrentPhotoInfo()

	h.jsonResponse(w, GameResponse{
		Success:       true,
		PhotoURL:      h.getPhotoURL(ctx, photo.FileID),
		CurrentPhoto:  current,
		TotalPhotos:   total,
		HasMore:       current < total,
		CurrentPlayer: nextPlayer,
		Scoreboard:    h.session.GetScoreboard(),
	})
}

func (h *Handlers) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	scoreboard := h.session.GetScoreboard()
	h.jsonResponse(w, SessionResponse{
		Success:    true,
		Scoreboard: scoreboard,
	})
}

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