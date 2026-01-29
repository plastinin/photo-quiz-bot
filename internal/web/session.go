package web

import (
	"math/rand"
	"sync"
	"time"

	"github.com/plastinin/photo-quiz-bot/internal/domain"
)

type SessionManager struct {
	session *domain.GameSession
	mu      sync.RWMutex

	TurnEndChan chan TurnEndEvent
}

type TurnEndEvent struct {
	PlayerName string
	SessionID  string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		TurnEndChan: make(chan TurnEndEvent, 10),
	}
}

func (sm *SessionManager) CreateSession(playerNames []string) *domain.GameSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	players := make([]domain.Player, len(playerNames))
	for i, name := range playerNames {
		players[i] = domain.Player{
			ID:    generateID(),
			Name:  name,
			Score: 0,
			Order: i,
		}
	}

	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
		players[i].Order = i
		players[j].Order = j
	})

	sm.session = &domain.GameSession{
		ID:              generateID(),
		Players:         players,
		CurrentPlayerID: players[0].ID,
		CurrentRound:    1,
		IsActive:        true,
		IsFinished:      false,
		CreatedAt:       time.Now(),
	}

	return sm.session
}

func (sm *SessionManager) GetSession() *domain.GameSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.session
}

func (sm *SessionManager) GetCurrentPlayer() *domain.Player {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.session == nil {
		return nil
	}

	for i := range sm.session.Players {
		if sm.session.Players[i].ID == sm.session.CurrentPlayerID {
			return &sm.session.Players[i]
		}
	}
	return nil
}

func (sm *SessionManager) NextPlayer() *domain.Player {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.session == nil || len(sm.session.Players) == 0 {
		return nil
	}

	currentIdx := 0
	for i, p := range sm.session.Players {
		if p.ID == sm.session.CurrentPlayerID {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(sm.session.Players)
	sm.session.CurrentPlayerID = sm.session.Players[nextIdx].ID
	sm.session.CurrentRound++

	return &sm.session.Players[nextIdx]
}

func (sm *SessionManager) AddScore(playerName string, score float64) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.session == nil {
		return false
	}

	for i := range sm.session.Players {
		if sm.session.Players[i].Name == playerName {
			sm.session.Players[i].Score += score
			return true
		}
	}
	return false
}

func (sm *SessionManager) AddScoreToCurrentPlayer(score float64) *domain.Player {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.session == nil {
		return nil
	}

	for i := range sm.session.Players {
		if sm.session.Players[i].ID == sm.session.CurrentPlayerID {
			sm.session.Players[i].Score += score
			return &sm.session.Players[i]
		}
	}
	return nil
}

func (sm *SessionManager) GetScoreboard() []domain.PlayerScore {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.session == nil {
		return nil
	}

	scores := make([]domain.PlayerScore, len(sm.session.Players))
	for i, p := range sm.session.Players {
		scores[i] = domain.PlayerScore{
			Name:            p.Name,
			Score:           p.Score,
			IsCurrentPlayer: p.ID == sm.session.CurrentPlayerID,
		}
	}

	// Сортировка по убыванию очков
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores
}

func (sm *SessionManager) FinishGame() []domain.PlayerScore {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.session == nil {
		return nil
	}

	sm.session.IsActive = false
	sm.session.IsFinished = true

	scores := make([]domain.PlayerScore, len(sm.session.Players))
	for i, p := range sm.session.Players {
		scores[i] = domain.PlayerScore{
			Name:  p.Name,
			Score: p.Score,
		}
	}

	// Сортировка по убыванию очков
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores
}

func (sm *SessionManager) ResetSession() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.session = nil
}

func (sm *SessionManager) HasActiveSession() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.session != nil && sm.session.IsActive
}

func (sm *SessionManager) NotifyTurnEnd() {
	player := sm.GetCurrentPlayer()
	if player != nil {
		sm.mu.RLock()
		sessionID := ""
		if sm.session != nil {
			sessionID = sm.session.ID
		}
		sm.mu.RUnlock()

		select {
		case sm.TurnEndChan <- TurnEndEvent{
			PlayerName: player.Name,
			SessionID:  sessionID,
		}:
		default:
		}
	}
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}