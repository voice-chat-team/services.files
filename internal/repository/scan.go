package repository

const fileColumns = `id::text, owner_id::text, purpose, status, name, content_type, size_bytes, s3_key, created_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanFile(s scanner) (File, error) {
	var f File
	err := s.Scan(&f.ID, &f.OwnerID, &f.Purpose, &f.Status, &f.Name,
		&f.ContentType, &f.SizeBytes, &f.S3Key, &f.CreatedAt)
	return f, err
}
