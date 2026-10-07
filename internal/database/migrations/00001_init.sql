-- +goose Up
CREATE TABLE files (
    id           UUID PRIMARY KEY,
    owner_id     UUID NOT NULL,
    name         TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    s3_key       TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE files;
