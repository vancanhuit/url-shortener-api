package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"

	"192.168.1.100/homelab/url-shortener/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type model struct {
	OriginalURL string `json:"original_url"`
	Alias       string `json:"alias"`
}

type service struct {
	db *sql.DB
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateDB(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, "."); err != nil {
		return err
	}
	return nil
}

func (s service) createAlias(url string) (string, error) {
	hash := sha256.Sum256([]byte(url))
	alias := base64.URLEncoding.EncodeToString(hash[:])[:11]

	query := `
	INSERT INTO urls (original_url, alias)
	VALUES ($1, $2)`

	_, err := s.db.Exec(query, url, alias)
	if err != nil {
		return "", nil
	}

	return alias, nil
}

func (s service) getURL(alias string) (string, error) {
	query := `
	SELECT original_url FROM urls
	WHERE alias = $1`

	var url string
	err := s.db.QueryRow(query, alias).Scan(&url)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	return url, nil
}
