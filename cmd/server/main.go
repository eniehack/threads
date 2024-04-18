package main

import (
	"database/sql"
	"log"
	"net/http"

	"aidanwoods.dev/go-paseto"
	"github.com/eniehack/threads/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./test.sqlite")
	if err != nil {
		log.Fatalf("cannot open sqlite file: %v", err)
		return
	}
	defer db.Close()
	key := paseto.NewV4SymmetricKey()
	h := new(handler.Handler)
	h.DB = db
	h.Paseto.Key = &key
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())
	h.Paseto.Parser = &parser

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v0", func(r chi.Router) {
		r.Post("/session/new", h.CreateSession)
		r.Post("/note/new", h.CreateNote)
	})
	http.ListenAndServe(":3000", r)
}
