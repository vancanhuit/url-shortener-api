-- +goose Up
-- +goose StatementBegin
CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    original_url VARCHAR(500) NOT NULL UNIQUE,
    alias VARCHAR(11) NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE urls;
-- +goose StatementEnd
