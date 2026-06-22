package entity

import "github.com/cloudwego/eino/schema"

type SSEEvent struct {
	Type           string            `json:"type"`
	SessionID      string            `json:"session_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	AgentName      string            `json:"agent_name,omitempty"`
	RunPath        string            `json:"run_path,omitempty"`
	Role           string            `json:"role,omitempty"`
	ToolName       string            `json:"tool_name,omitempty"`
	Content        string            `json:"content,omitempty"`
	ToolCalls      []schema.ToolCall `json:"tool_calls,omitempty"`
	ActionType     string            `json:"action_type,omitempty"`
	Error          string            `json:"error,omitempty"`
}
