package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	service service
	logger  *slog.Logger
}

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Post("/api/shorten", app.shorten)
	r.Delete("/{alias}", app.delete)
	r.Get("/{alias}", app.redirect)

	return r
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var port int
	var dsn string

	flag.IntVar(&port, "port", 9000, "HTTP server port")
	flag.StringVar(&dsn, "dsn", os.Getenv("DB_DSN"), "PostgreSQL data source name")
	flag.Parse()

	db, err := openDB(dsn)
	if err != nil {
		logger.Error("failed to establish database connection pool", "error", err.Error())
		os.Exit(1)
	}
	defer db.Close() //nolint:errcheck
	logger.Info("database connection pool established")
	err = migrateDB(db)
	if err != nil {
		logger.Error("failed to migrate database", "err", err.Error())
	}

	app := &application{
		service: service{db: db},
		logger:  logger,
	}

	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     app.routes(),
		ReadTimeout: 5 * time.Second,
	}

	logger.Info("HTTP server is listening on %s", "addr", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("failed to start HTTP server", "error", err.Error())
		os.Exit(1)
	}
}
