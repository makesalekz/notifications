package messages

type EmailDetails struct {
	Language string            `json:"language,omitempty"`
	Type     string            `json:"type,omitempty"`
	Email    string            `json:"email,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}
