package main

import (
	"database/sql"
	"log"
	"net/http"

	"aidanwoods.dev/go-paseto"
	"github.com/eniehack/threads/internal/handler"
	mymiddleware "github.com/eniehack/threads/internal/middleware"
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
		r.Post("/sessions/new", h.CreateSession)
		r.Get("/notes/{noteId}", h.ReadNote)
		r.Get("/notes/{noteId}/revisions", h.ReadNoteRevisions)
		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.CheckAuthzHeader(&mymiddleware.CheckAuthzConfig{
				Paseto: h.Paseto,
			}))
			r.Post("/notes/new", h.CreateNote)
			r.Put("/notes/{noteId}", h.UpdateNote)
			//r.Post("/note/{noteId}/reply", h.CreateReplyNote)
		})
	})
	http.ListenAndServe(":3000", r)
}
