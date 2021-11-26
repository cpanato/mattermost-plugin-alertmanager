package main

// ActionContext passed from action buttons
type ActionContext struct {
	SilenceID string `json:"silence_id"`
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
}

// Action type for decoding action buttons
type Action struct {
	Context *ActionContext `json:"context"`
	UserID  string         `json:"user_id"`
	PostID  string         `json:"post_id"`
}
