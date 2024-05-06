package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"aidanwoods.dev/go-paseto"
	"github.com/BurntSushi/toml"
	"github.com/eniehack/threads/internal/handler"
	mymiddleware "github.com/eniehack/threads/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"
)

type ConfigRoot struct {
	Database string `toml:"database"`
	Port     uint   `toml:"port"`
}

func main() {
	configFile := flag.String("f", "config.toml", "path of config file")
	config := new(ConfigRoot)
	_, err := toml.DecodeFile(*configFile, config)
	if err != nil {
		log.Fatalf("toml decode err: %v", err)
	}
	db, err := sql.Open("sqlite", config.Database)
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
		r.Get("/notes/{noteId}/reply", h.ReadChildReply)
		r.Group(func(r chi.Router) {
			r.Use(mymiddleware.CheckAuthzHeader(&mymiddleware.CheckAuthzConfig{
				Paseto: h.Paseto,
			}))
			r.Post("/notes/new", h.CreateNote)
			r.Put("/notes/{noteId}", h.UpdateNote)
			r.Post("/notes/{noteId}/reply", h.CreateReply)
		})
	})
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), r)
}
