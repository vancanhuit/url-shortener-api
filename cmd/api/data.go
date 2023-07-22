package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"

	"192.168.1.100/homelab/url-shortener/migrations"
	"github.com/asaskevich/govalidator"
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

func validateURL(v *validator, url string) {
	v.check(url != "", "url", "must be provided")
	v.check(len(url) <= 500, "url", "must not be more than 500 bytes long")
	v.check(govalidator.IsURL(url), "url", "must be a valid URL")
}

func validateAlias(v *validator, alias string) {
	v.check(alias != "", "alias", "must be provided")
	v.check(len(alias) <= 11, "alias", "must not be more than 11 bytes long")
}

func (s service) createAlias(url string) (string, error) {
	var alias string

	query := `SELECT alias FROM urls
	WHERE original_url = $1`

	err := s.db.QueryRow(query, url).Scan(&alias)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if err == nil {
		return alias, nil
	}

	hash := sha256.Sum256([]byte(url))
	alias = base64.URLEncoding.EncodeToString(hash[:])[:11]

	query = `INSERT INTO urls (original_url, alias)
	VALUES ($1, $2)`

	_, err = s.db.Exec(query, url, alias)
	if err != nil {
		return "", err
	}

	return alias, nil
}

func (s service) getURL(alias string) (string, error) {
	query := `SELECT original_url FROM urls
	WHERE alias = $1`

	var url string
	err := s.db.QueryRow(query, alias).Scan(&url)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	return url, nil
}
