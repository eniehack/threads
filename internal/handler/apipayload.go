package handler

type NoteRevisionPayload struct {
	Id        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
}

type NotePayload struct {
	Id        string              `json:"id"`
	UserId    string              `json:"author_id"`
	Content   string              `json:"content"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
	Revision  NoteRevisionPayload `json:"revision"`
}
