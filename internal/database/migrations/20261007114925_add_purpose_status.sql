-- +goose Up
ALTER TABLE files
ADD COLUMN purpose SMALLINT NOT NULL DEFAULT 0,
ADD COLUMN status SMALLINT NOT NULL DEFAULT 1;

CREATE INDEX files_owner_id_idx ON files (owner_id);

-- +goose Down
ALTER TABLE files
    DROP COLUMN purpose,
    DROP COLUMN status;

DROP INDEX files_owner_id_idx;
