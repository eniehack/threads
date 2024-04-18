package handler

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rs/xid"
)

type CreateNoteRequestParams struct {
	Text     string  `json:"text"`
	Referent *string `json:"referent"`
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	authzHeader := r.Header.Get("Authorization")
	if len(authzHeader) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if strings.HasPrefix(authzHeader, "Bearer ") == false {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := strings.TrimPrefix(authzHeader, "Bearer ")
	parsedToken, err := h.Paseto.Parser.ParseV4Local(*h.Paseto.Key, token, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userAliasId, ok := parsedToken.Claims()["user_id"].(string)
	if !ok || userAliasId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	payload := new(CreateNoteRequestParams)
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("failed open tx: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var userId string
	if err := tx.QueryRowContext(
		r.Context(),
		"SELECT id FROM users WHERE alias_id = ?",
		userAliasId,
	).Scan(&userId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	noteId := xid.New().String()
	if _, err = tx.ExecContext(
		r.Context(),
		"INSERT INTO notes(id, user_id) VALUES (?, ?);",
		noteId,
		userId,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()
	revId, _ := ulid.New(ulid.Timestamp(now), entropy)
	if _, err = tx.ExecContext(
		r.Context(),
		"INSERT INTO note_revisions (id, note_id, content, created_at) VALUES (?, ?, ?, ?)",
		revId.String(),
		noteId,
		payload.Text,
		now.Format(time.RFC3339),
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	return
}
