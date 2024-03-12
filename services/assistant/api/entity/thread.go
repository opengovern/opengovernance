package entity

type SendMessageRequest struct {
	ThreadID *string `json:"thread_id"`
	Content  string  `json:"content"`
}

type SendMessageResponse struct {
	ThreadID string `json:"thread_id"`
}

type Message struct {
	Content string `json:"content"`
}

type ListMessagesResponse struct {
	Messages []Message `json:"messages"`
}
