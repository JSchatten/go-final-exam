package salutespeech

// Config holds configuration specific to GigaChat API
type Config struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
	AuthKey  string `json:"auth_key"`
}
