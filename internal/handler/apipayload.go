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

func (s *NullString) UnmarshalJSON(data []byte) error {
	var str string
	json.Unmarshal(data, &str)
	s.String = str
	s.Valid = (str != "")
	return nil
}

func NewNullString(s string) NullString {
	return NullString{sql.NullString{String: s, Valid: s != ""}}
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
