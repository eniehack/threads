package handler

import (
	"database/sql"

	"aidanwoods.dev/go-paseto"
)

type Handler struct {
	DB     *sql.DB
	Paseto struct {
		Key    *paseto.V4SymmetricKey
		Parser *paseto.Parser
	}
}
