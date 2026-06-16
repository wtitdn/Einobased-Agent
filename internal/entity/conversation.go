package entity

import "time"

type Conversation struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"index;size:64;not null" json:"user_id"`
	Title     string    `gorm:"size:255" json:"title"`
	Messages  []Message `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"messages"`
	AgentType string    `gorm:"size:64;index" json:"agent_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	ConversationID string    `gorm:"index;size:64;not null" json:"conversation_id"`
	UserID         string    `gorm:"index;size:64;not null" json:"user_id"`
	Role           string    `gorm:"size:32;not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}
