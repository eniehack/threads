package handler

import (
	"database/sql"
	"encoding/json"
)

type NullString struct {
	sql.NullString
}

func (s *NullString) MarshalJSON() ([]byte, error) {
	if s.Valid {
		return json.Marshal(s.String)
	} else {
		return json.Marshal(nil)
	}
}

type UserPayload struct {
	Id   string `json:"id"`
	Ulid string `json:"ulid"`
}

type NoteRevisionPayload struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
}

type ReplyPayload struct {
	Ancestors   []NotePayload `json:"ancestors"`   // parent
	Descendants []NotePayload `json:"descendants"` // children
}

type NotePayload struct {
	Id        string              `json:"id"`
	User      UserPayload         `json:"author"`
	Content   string              `json:"content"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
	Revision  NoteRevisionPayload `json:"revision"`
	InReplyTo NullString          `json:"in_reply_to"`
}
