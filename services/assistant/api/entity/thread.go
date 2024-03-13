package entity

type SendMessageRequest struct {
	ThreadID *string `json:"thread_id"`
	Content  string  `json:"content"`
}

type SendMessageResponse struct {
	ThreadID string `json:"thread_id"`
	RunID    string `json:"run_id"`
}

type Message struct {
	Content string `json:"content"`
}

type ListMessagesResponse struct {
	Messages []Message `json:"messages"`
	Status   string    `json:"status"`
}
