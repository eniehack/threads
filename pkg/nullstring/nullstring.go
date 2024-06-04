package nullstring

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

func New(s *string) NullString {
	if s == nil {
		return NullString{sql.NullString{String: "", Valid: false}}
	}
	return NullString{sql.NullString{String: *s, Valid: true}}
}
