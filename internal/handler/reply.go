package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/eniehack/threads/pkg/nullstring"
	"github.com/go-chi/chi/v5"
)

func createNoteReference(tx *sql.Tx, id string, ancestor *nullstring.NullString) error {
	if _, err := tx.Exec(
		"INSERT INTO note_references(ancestor, id) VALUES (?, ?);",
		ancestor,
		id,
	); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ReadChildReply(w http.ResponseWriter, r *http.Request) {
	noteId := chi.URLParam(r, "noteId")
	if noteId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ancestors, err := h.DB.QueryContext(
		r.Context(),
		`WITH RECURSIVE r AS (
				SELECT NREF.*, 0 AS depth
					FROM note_references AS NREF
					WHERE NREF.id = ?
		    UNION ALL
		    SELECT NREF.*, depth + 1
					FROM note_references AS NREF
					INNER JOIN r
					WHERE NREF.id = r.ancestor AND r.depth <= 100
		)
		SELECT
			r.ancestor,
			N.id,
			U.alias_id,
			U.id,
			NREV.content,
			NREV.id,
			N.created_at,
			N.updated_at,
			NREV.created_at
		FROM r
		JOIN notes AS N ON N.id = r.id
		JOIN note_revisions AS NREV ON NREV.id = N.rev_id
		JOIN users AS U ON U.id = N.user_id
		WHERE N.is_deleted = FALSE
			AND N.id != ?;`,
		noteId,
		noteId,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Println("ancestors query err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload := new(ReplyPayload)
	for ancestors.Next() {
		note := new(NotePayload)
		if err := ancestors.Scan(
			&note.InReplyTo,
			&note.Id,
			&note.User.Id,
			&note.User.Ulid,
			&note.Content,
			&note.Revision.Id,
			&note.CreatedAt,
			&note.UpdatedAt,
			&note.Revision.CreatedAt,
		); err != nil {
			log.Printf("sql.scan: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		note.Revision.Content = note.Content
		payload.Ancestors = append(payload.Ancestors, *note)
	}

	descendants, err := h.DB.QueryContext(
		r.Context(),
		`
		WITH RECURSIVE r AS (
				SELECT NREF.*, 0 AS depth
					FROM note_references AS NREF
					WHERE NREF.ancestor = ?
				UNION ALL
				SELECT NREF.*, depth + 1
					FROM note_references AS NREF
					INNER JOIN r
					WHERE NREF.ancestor = r.id AND r.depth <= 100
		)
		SELECT
			N.id,
			r.ancestor,
			U.alias_id,
			U.id,
			NREV.content,
			NREV.id,
			N.created_at,
			N.updated_at,
			NREV.created_at
		FROM r
		JOIN notes AS N
			ON N.id = r.id
		JOIN note_revisions AS NREV 
			ON NREV.id = N.rev_id
		JOIN users AS U 
			ON U.id = N.user_id
		WHERE N.is_deleted = FALSE
			AND N.id != ?;`,
		noteId,
		noteId,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Println("descendants query err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for descendants.Next() {
		note := new(NotePayload)
		if err := descendants.Scan(
			&note.Id,
			&note.InReplyTo,
			&note.User.Id,
			&note.User.Ulid,
			&note.Content,
			&note.Revision.Id,
			&note.CreatedAt,
			&note.UpdatedAt,
			&note.Revision.CreatedAt,
		); err != nil {
			log.Printf("sql.scan: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		note.Revision.Content = note.Content
		payload.Descendants = append(payload.Descendants, *note)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encoder: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
