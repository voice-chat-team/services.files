package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusPending int16 = 1
	StatusReady   int16 = 2
	StatusDeleted int16 = 3
)

type File struct {
	ID          string
	OwnerID     string
	Purpose     int16
	Status      int16
	Name        string
	ContentType string
	SizeBytes   int64
	S3Key       string
	CreatedAt   time.Time
}

var ErrNotFound = errors.New("file not found")

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (r *FileRepository) Create(ctx context.Context, file File) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (
			id, owner_id,
			purpose,
			status,
			name,
			content_type,
			size_bytes,
			s3_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		file.ID,
		file.OwnerID,
		file.Purpose,
		file.Status,
		file.Name,
		file.ContentType,
		file.SizeBytes,
		file.S3Key,
	)

	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	return nil
}

func (r *FileRepository) GetById(ctx context.Context, id string) (File, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+fileColumns+` FROM files WHERE id = $1 AND status <> $2`,
		id, StatusDeleted)

	file, err := scanFile(row)

	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, ErrNotFound
	}

	if err != nil {
		return File{}, fmt.Errorf("get file: %w", err)
	}

	return file, nil
}

func (r *FileRepository) GetByIds(ctx context.Context, ids []string) ([]File, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+fileColumns+` FROM files WHERE id = ANY($1::uuid[]) AND status <> $2`,
		ids, StatusDeleted)

	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	return files, nil
}

func (r *FileRepository) MarkReady(ctx context.Context, id string, size int64, ownerId string) (File, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE files SET status = $1, size_bytes = $2
		WHERE id = $3 AND owner_id = $4 AND status = $5
		RETURNING `+fileColumns, StatusReady, size, id, ownerId, StatusPending)

	file, err := scanFile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, ErrNotFound
	}

	if err != nil {
		return File{}, fmt.Errorf("mark ready: %w", err)
	}

	return file, nil
}

func (r *FileRepository) MarkDelete(ctx context.Context, id string, ownerId string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE files SET status = $1
		WHERE id = $2 AND owner_id = $3 AND status <> $1`,
		StatusDeleted, id, ownerId)

	if err != nil {
		return fmt.Errorf("mark delete: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
