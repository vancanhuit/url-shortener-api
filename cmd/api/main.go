package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	service service
}

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.NotFound(notFoundResponse)
	r.MethodNotAllowed(methodNotAllowedResponse)

	r.Post("/api/shorten", app.shorten)
	r.Delete("/{alias}", app.delete)
	r.Get("/{alias}", app.redirect)

	return r
}

func main() {
	var port int
	var dsn string

	flag.IntVar(&port, "port", 9000, "HTTP server port")
	flag.StringVar(&dsn, "dsn", os.Getenv("DB_DSN"), "PostgreSQL data source name")
	flag.Parse()

	db, err := openDB(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Database connection pool established")
	err = migrateDB(db)
	if err != nil {
		log.Fatal(err)
	}

	app := &application{service: service{db: db}}

	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		Handler:     app.routes(),
		ReadTimeout: 5 * time.Second,
	}

	log.Printf("HTTP server is listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
