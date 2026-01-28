package service

import (
	"context"
	"errors"
	"sync"

	"github.com/plastinin/photo-quiz-bot/internal/domain"
	"github.com/plastinin/photo-quiz-bot/internal/repository/postgres"
)

var (
	ErrNoSituations   = errors.New("нет доступных задач")
	ErrNoMorePhotos   = errors.New("больше нет фотографий")
	ErrGameNotStarted = errors.New("игра не начата, используйте /start")
)

type GameService struct {
	repo  *postgres.SituationRepository
	state *GameState
	mu    sync.RWMutex
}

type GameState struct {
	CurrentSituation *domain.SituationWithPhotos
	CurrentPhotoIdx  int
}

func NewGameService(repo *postgres.SituationRepository) *GameService {
	return &GameService{
		repo:  repo,
		state: &GameState{},
	}
}

func (s *GameService) StartNewRound(ctx context.Context) (*domain.Photo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	situation, err := s.repo.GetRandomUnused(ctx)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNoSituations
		}
		return nil, err
	}

	if len(situation.Photos) == 0 {
		// Если у ситуации нет фото, помечаем её использованной и пробуем снова
		_ = s.repo.MarkAsUsed(ctx, situation.Situation.ID)
		s.mu.Unlock()
		return s.StartNewRound(ctx)
	}

	s.state.CurrentSituation = situation
	s.state.CurrentPhotoIdx = 0

	return &situation.Photos[0], nil
}

func (s *GameService) NextPhoto(ctx context.Context) (*domain.Photo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.CurrentSituation == nil {
		return nil, ErrGameNotStarted
	}

	nextIdx := s.state.CurrentPhotoIdx + 1
	if nextIdx >= len(s.state.CurrentSituation.Photos) {
		return nil, ErrNoMorePhotos
	}

	s.state.CurrentPhotoIdx = nextIdx
	return &s.state.CurrentSituation.Photos[nextIdx], nil
}

func (s *GameService) GetAnswer(ctx context.Context) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.CurrentSituation == nil {
		return "", ErrGameNotStarted
	}

	return s.state.CurrentSituation.Situation.Answer, nil
}

func (s *GameService) FinishRound(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.CurrentSituation == nil {
		return ErrGameNotStarted
	}

	err := s.repo.MarkAsUsed(ctx, s.state.CurrentSituation.Situation.ID)
	if err != nil {
		return err
	}

	s.state.CurrentSituation = nil
	s.state.CurrentPhotoIdx = 0

	return nil
}

func (s *GameService) ResetGame(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.CurrentSituation = nil
	s.state.CurrentPhotoIdx = 0

	return s.repo.ResetAllUsed(ctx)
}

func (s *GameService) GetStats(ctx context.Context) (total, used, remaining int, err error) {
	total, used, err = s.repo.GetStats(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	remaining = total - used
	return total, used, remaining, nil
}

func (s *GameService) GetCurrentPhotoInfo() (current, total int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.CurrentSituation == nil {
		return 0, 0, ErrGameNotStarted
	}

	return s.state.CurrentPhotoIdx + 1, len(s.state.CurrentSituation.Photos), nil
}