package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/plastinin/photo-quiz-bot/internal/domain"
)

var ErrNotFound = errors.New("not found")

type SituationRepository struct {
	db *DB
}

func NewSituationRepository(db *DB) *SituationRepository {
	return &SituationRepository{db: db}
}

func (r *SituationRepository) CreateSituation(ctx context.Context, answer string) (int, error) {
	var id int
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO situations (answer) VALUES ($1) RETURNING id`,
		answer,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create situation: %w", err)
	}
	return id, nil
}

func (r *SituationRepository) AddPhoto(ctx context.Context, situationID int, fileID string) error {
	
	var sortOrder int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM photos WHERE situation_id = $1`,
		situationID,
	).Scan(&sortOrder)
	if err != nil {
		return fmt.Errorf("get sort order: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx,
		`INSERT INTO photos (situation_id, file_id, sort_order) VALUES ($1, $2, $3)`,
		situationID, fileID, sortOrder,
	)
	if err != nil {
		return fmt.Errorf("add photo: %w", err)
	}
	return nil
}

func (r *SituationRepository) Create(ctx context.Context, answer string, photoFileIDs []string) error {
	// Создаём ситуацию
	situationID, err := r.CreateSituation(ctx, answer)
	if err != nil {
		return err
	}

	// Добавляем фотографии
	for _, fileID := range photoFileIDs {
		if err := r.AddPhoto(ctx, situationID, fileID); err != nil {
			return err
		}
	}

	return nil
}

func (r *SituationRepository) GetRandomUnused(ctx context.Context) (*domain.SituationWithPhotos, error) {
	
	var s domain.Situation
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, answer, is_used, created_at 
		 FROM situations 
		 WHERE is_used = FALSE 
		 ORDER BY RANDOM() 
		 LIMIT 1`,
	).Scan(&s.ID, &s.Answer, &s.IsUsed, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get random situation: %w", err)
	}

	photos, err := r.getPhotosBySituationID(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	return &domain.SituationWithPhotos{
		Situation: s,
		Photos:    photos,
	}, nil
}

func (r *SituationRepository) MarkAsUsed(ctx context.Context, situationID int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE situations SET is_used = TRUE WHERE id = $1`,
		situationID,
	)
	if err != nil {
		return fmt.Errorf("mark as used: %w", err)
	}
	return nil
}

func (r *SituationRepository) ResetAllUsed(ctx context.Context) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE situations SET is_used = FALSE`)
	if err != nil {
		return fmt.Errorf("reset all used: %w", err)
	}
	return nil
}

func (r *SituationRepository) GetByID(ctx context.Context, id int) (*domain.SituationWithPhotos, error) {
	var s domain.Situation
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, answer, is_used, created_at FROM situations WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Answer, &s.IsUsed, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get situation by id: %w", err)
	}

	photos, err := r.getPhotosBySituationID(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	return &domain.SituationWithPhotos{
		Situation: s,
		Photos:    photos,
	}, nil
}

func (r *SituationRepository) CountPhotos(ctx context.Context, situationID int) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM photos WHERE situation_id = $1`,
		situationID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count photos: %w", err)
	}
	return count, nil
}

func (r *SituationRepository) GetStats(ctx context.Context) (total, used int, err error) {
	err = r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_used = TRUE) FROM situations`,
	).Scan(&total, &used)
	if err != nil {
		return 0, 0, fmt.Errorf("get stats: %w", err)
	}
	return total, used, nil
}

func (r *SituationRepository) getPhotosBySituationID(ctx context.Context, situationID int) ([]domain.Photo, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, situation_id, file_id, sort_order, created_at 
		 FROM photos 
		 WHERE situation_id = $1 
		 ORDER BY sort_order`,
		situationID,
	)
	if err != nil {
		return nil, fmt.Errorf("get photos: %w", err)
	}
	defer rows.Close()

	var photos []domain.Photo
	for rows.Next() {
		var p domain.Photo
		if err := rows.Scan(&p.ID, &p.SituationID, &p.FileID, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan photo: %w", err)
		}
		p.OrderNum = p.SortOrder // для совместимости
		photos = append(photos, p)
	}

	return photos, rows.Err()
}

func (r *SituationRepository) DeleteAll(ctx context.Context) (int, error) {

	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM situations`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count situations: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx, `DELETE FROM situations`)
	if err != nil {
		return 0, fmt.Errorf("delete all: %w", err)
	}

	return count, nil
}