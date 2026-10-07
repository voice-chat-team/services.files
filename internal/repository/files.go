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
		return File{}, fmt.Errorf("File not found: %s", id)
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
		return nil, fmt.Errorf("Query files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, fmt.Errorf("Scan file: %w", err)
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	return files, err
}
