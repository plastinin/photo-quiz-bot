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
	SortOrder   int
	CreatedAt   time.Time
}

type SituationWithPhotos struct {
	Situation Situation
	Photos    []Photo
}

type GameState struct {
	ChatID           int64
	CurrentSituation *SituationWithPhotos
	CurrentPhotoIdx  int // индекс текущей фотографии (0-4)
}