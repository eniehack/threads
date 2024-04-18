package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/alexedwards/argon2id"
)

type CreateSessionRequestParams struct {
	Password string `json:"password"`
	UserId   string `json:"user_id"`
}

type CreateSessionResponseParams struct {
	Token string `json:"token"`
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	payload := new(CreateSessionRequestParams)
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hashedPasswordFetchStmt, err := h.DB.PrepareContext(
		r.Context(),
		"SELECT password FROM users WHERE alias_id = ?",
	)
	if err != nil {
		log.Printf("hashedPasswordFetchStmt create err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer hashedPasswordFetchStmt.Close()
	var password string
	err = hashedPasswordFetchStmt.QueryRow(payload.UserId).Scan(&password)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("hashedPasswordFetchStmt.QueryRow err: %v\n", err)
		return
	}
	match, err := argon2id.ComparePasswordAndHash(payload.Password, password)
	if err != nil {
		log.Printf("argon2id.ComparePasswordAndHash err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !match {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := paseto.NewToken()
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(2 * time.Hour))
	token.SetString("user_id", payload.UserId)
	encryptedToken := token.V4Encrypt(*h.Paseto.Key, nil)

	if err = json.NewEncoder(w).Encode(&CreateSessionResponseParams{
		Token: encryptedToken,
	}); err != nil {
		log.Printf("JSON Encoder err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	return
}
