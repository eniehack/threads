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
		WHERE N.is_deleted = FALSE;`,
		noteId,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
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
		WHERE N.is_deleted = FALSE;
		`,
		noteId,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
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
	return
}
