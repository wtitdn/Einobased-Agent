package response

import "time"

type ConversationHistoryItem struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	AgentType    string    `json:"agent_type"`
	MessageCount int64     `json:"message_count"`
	LastMessage  string    `json:"last_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ConversationHistoryResponse struct {
	Conversations []ConversationHistoryItem `json:"conversations"`
}

type ConversationMessageItem struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationMessagesResponse struct {
	Messages []ConversationMessageItem `json:"messages"`
}
