package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/xid"
)

func (h *Handler) CreateReply(w http.ResponseWriter, r *http.Request) {
	referent := chi.URLParam(r, "noteId")
	userAliasId, ok := r.Context().Value("userAliasId").(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload := new(CreateNoteRequestParams)
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("failed open tx: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userUlid, err := lookupUserUlid(tx, userAliasId, r.Context())
	if err != nil {
		return
	}
	now := time.Now()
	revId := CreateRevisionId(now)
	if err = createNoteRevision(
		tx,
		r.Context(),
		revId,
		payload.Text,
		now,
	); err != nil {
		log.Printf("failed exec insert note_rev: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	noteId := xid.New().String()
	if err = createNote(
		tx,
		r.Context(),
		noteId,
		*userUlid,
		revId,
		now,
	); err != nil {
		log.Printf("failed exec insert note: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = tx.ExecContext(
		r.Context(),
		"INSERT INTO note_references(referent, referrer) VALUES (?, ?);",
		referent,
		noteId,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err = tx.Commit(); err != nil {
		log.Printf("failed commit tx: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v0/notes/%s", noteId))
	w.WriteHeader(http.StatusCreated)
	return
}

func (h *Handler) ReadChildReply(w http.ResponseWriter, r *http.Request) {
	noteId := chi.URLParam(r, "noteId")
	if noteId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	row := h.DB.QueryRowContext(
		r.Context(),
		`WITH RECURSIVE r AS (
			SELECT notes.*, note_references.*, 1 AS num
			FROM notes
			JOIN note_references ON note_references.referent = notes.id
			WHERE notes.id = ?
			
		  UNION ALL
			
		  SELECT notes.*, note_references.*, num + 1
			FROM notes, r
			JOIN note_references ON note_references.referent = r.id
			WHERE note_references.referrer = notes.id
		)
		SELECT r.id AS note_id, U.alias_id, r.created_at, r.updated_at, NREV.content, NREV.created_at
		FROM r
		JOIN note_revisions AS NREV ON NREV.id = r.rev_id
		JOIN users AS U ON U.id = r.user_id;`,
		noteId,
	)
	payload := new(NotePayload)
	if err := row.Scan(
		&payload.UserId,
		&payload.Revision.Id,
		&payload.Content,
		&payload.Revision.CreatedAt,
		&payload.CreatedAt,
		&payload.UpdatedAt,
	); err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("sql.scan: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	payload.Id = noteId
	payload.Revision.Content = payload.Content

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encoder: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	return
}
