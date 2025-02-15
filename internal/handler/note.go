package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/eniehack/threads/pkg/nullstring"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/rs/xid"
)

type CreateNoteRequestParams struct {
	Text      string  `json:"text"`
	InReplyTo *string `json:"in_reply_to"`
}

func lookupUserUlid(tx *sql.Tx, userAliasId string, ctx context.Context) (*string, error) {
	var userId string
	if err := tx.QueryRowContext(
		ctx,
		"SELECT id FROM users WHERE alias_id = ?",
		userAliasId,
	).Scan(&userId); err != nil {
		return nil, err
	}
	return &userId, nil
}

func createNoteRevision(tx *sql.Tx, ctx context.Context, revId string, content string, now time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO note_revisions (id, content, created_at) VALUES (?, ?, ?)",
		revId,
		content,
		now.Format(time.RFC3339),
	)
	return err
}

func createNote(tx *sql.Tx, ctx context.Context, noteId string, userId string, revId string, now time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		"INSERT INTO notes(id, user_id, rev_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?);",
		noteId,
		userId,
		revId,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	return err
}

func updateNote(tx *sql.Tx, ctx context.Context, revId string, now time.Time, noteId string) error {
	_, err := tx.ExecContext(
		ctx,
		`UPDATE notes
		 SET rev_id = ?, updated_at = ?
		 WHERE id = ?;`,
		revId,
		now.Format(time.RFC3339),
		noteId,
	)
	return err
}

func CreateRevisionId(now time.Time) string {
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	revId, _ := ulid.New(ulid.Timestamp(now), entropy)
	return revId.String()
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
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
	if err := createNoteRevision(
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
	inReplyTo := nullstring.New(payload.InReplyTo)
	if err = createNoteReference(tx, noteId, &inReplyTo); err != nil {
		log.Printf("failed exec insert note reference: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("failed commit tx: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v0/notes/%s", noteId))
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	noteId := chi.URLParam(r, "noteId")
	if noteId == "" {
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
		log.Printf("failed open tx: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	now := time.Now()
	revId := CreateRevisionId(now)

	if err := createNoteRevision(
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
	if err = updateNote(
		tx,
		r.Context(),
		revId,
		now,
		noteId,
	); err != nil {
		log.Printf("failed exec update note: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("failed commit tx: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ReadNote(w http.ResponseWriter, r *http.Request) {
	noteId := chi.URLParam(r, "noteId")
	if noteId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	row := tx.QueryRowContext(
		r.Context(),
		`SELECT N.id, U.alias_id, U.id, N.rev_id, NREV.content, NREF.ancestor, NREV.created_at, N.created_at, N.updated_at
			FROM notes AS N
			LEFT JOIN note_references AS NREF ON N.id = NREF.id
			JOIN note_revisions AS NREV ON N.rev_id = NREV.id
			JOIN users AS U ON U.id = N.user_id
			WHERE N.id = ? AND N.is_deleted = 0;`,
		noteId,
	)
	payload := new(NotePayload)
	if err := row.Scan(
		&payload.Id,
		&payload.User.Id,
		&payload.User.Ulid,
		&payload.Revision.Id,
		&payload.Content,
		&payload.InReplyTo,
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
}

func (h *Handler) ReadNoteRevisions(w http.ResponseWriter, r *http.Request) {
	noteId := chi.URLParam(r, "noteId")
	if noteId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rows, err := h.DB.QueryContext(
		r.Context(),
		`SELECT NR.id, NR.content, NR.created_at
			FROM note_revisions AS NR
			JOIN notes AS N ON N.rev_id = NR.id
			WHERE N.id = ? AND N.is_deleted = 0;`,
		noteId,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		return
	}
	payload := make([]NoteRevisionPayload, 0, 5)
	for rows.Next() {
		elem := new(NoteRevisionPayload)
		if err := rows.Scan(
			&elem.Id,
			&elem.Content,
			&elem.CreatedAt,
		); err != nil {
			log.Printf("sql.scan: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		payload = append(payload, *elem)
	}
	if len(payload) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&payload); err != nil {
		log.Printf("json encoder: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
