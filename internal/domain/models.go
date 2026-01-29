package domain

import "time"

type Situation struct {
	ID        int
	Answer    string
	IsUsed    bool
	CreatedAt time.Time
}

type Photo struct {
	ID          int
	SituationID int
	FileID      string
	OrderNum    int
	SortOrder   int
	CreatedAt   time.Time
}

type SituationWithPhotos struct {
	Situation Situation
	Photos    []Photo
}

type Player struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Order int     `json:"order"`
}

type GameSession struct {
	ID              string    `json:"id"`
	Players         []Player  `json:"players"`
	CurrentPlayerID string    `json:"currentPlayerId"`
	CurrentRound    int       `json:"currentRound"`
	IsActive        bool      `json:"isActive"`
	IsFinished      bool      `json:"isFinished"`
	CreatedAt       time.Time `json:"createdAt"`
}

type PlayerScore struct {
	Name            string  `json:"name"`
	Score           float64 `json:"score"`
	IsCurrentPlayer bool    `json:"isCurrentPlayer"`
}