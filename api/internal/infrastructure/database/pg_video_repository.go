package database

import (
	"context"
	"database/sql"
	"video-api/internal/domain/video"
)

type VideoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) video.Repository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) Create(ctx context.Context, v *video.Video) error {
	query := `
        INSERT INTO videos (title, url, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`

	return r.db.QueryRowContext(ctx, query,
		v.Title, v.URL, v.Status, v.CreatedAt, v.UpdatedAt).Scan(&v.ID)
}

func (r *VideoRepository) GetAll(ctx context.Context) ([]*video.Video, error) {
	query := `SELECT id, title, url, status, created_at, updated_at FROM videos`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*video.Video
	for rows.Next() {
		v := &video.Video{}
		err := rows.Scan(&v.ID, &v.Title, &v.URL, &v.Status, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}

	return videos, nil
}

func (r *VideoRepository) GetByID(ctx context.Context, id uint) (*video.Video, error) {
	query := `SELECT id, title, url, status, created_at, updated_at FROM videos WHERE id = $1`

	var v video.Video
	err := r.db.QueryRowContext(ctx, query, id).Scan(&v.ID, &v.Title, &v.URL, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) Update(ctx context.Context, v *video.Video) error {
	query := `UPDATE videos SET title = $1, url = $2, status = $3, updated_at = $4 WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, v.Title, v.URL, v.Status, v.UpdatedAt, v.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *VideoRepository) Delete(ctx context.Context, id uint) error {
	query := `DELETE FROM videos WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}
