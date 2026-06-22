package usecase

import (
	"context"
	store2 "einoproject/internal/store"
	"errors"
	"fmt"
	"io"
	"strings"

	"einoproject/internal/entity"
	"einoproject/internal/repo"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type ConversationUsecase struct {
	conversationRepo *repo.ConversationRepo
}

type SSEPayload struct {
	Event string
	Data  entity.SSEEvent
}

type toolCallAccumulator struct {
	order []string
	calls map[string]schema.ToolCall
}

type sessionContextKey struct{}

var (
	ErrInvalidConversationInput = errors.New("user_id is required")
	ErrInvalidConversationID    = errors.New("conversation_id is required")
)

func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sessionID)
}

func SessionIDFromContext(ctx context.Context) string {
	sessionID, _ := ctx.Value(sessionContextKey{}).(string)
	return sessionID
}

func NewConversationUsecase(conversationRepo *repo.ConversationRepo) *ConversationUsecase {
	return &ConversationUsecase{conversationRepo: conversationRepo}
}

func (u *ConversationUsecase) StreamAgent(ctx context.Context, agent adk.Agent, userID, sessionID, message string) <-chan SSEPayload {
	events := make(chan SSEPayload, 8)

	conversation, history, err := u.prepareSession(ctx, agent, userID, sessionID, message)
	if err != nil {
		go func() {
			defer close(events)
			sendSSE(ctx, events, "error", entity.SSEEvent{
				Type:  "error",
				Error: err.Error(),
			})
		}()
		return events
	}

	ctx = ContextWithSessionID(ctx, conversation.ID)

	go u.streamAgentEvents(ctx, agent, userID, message, conversation, history, events)
	return events
}

func (u *ConversationUsecase) FlushCachedConversation(ctx context.Context, sessionID string) error {
	return u.conversationRepo.FlushCachedConversationToMySQL(ctx, sessionID)
}

func (u *ConversationUsecase) ListHistoryByUserID(ctx context.Context, userID string) ([]repo.ConversationHistoryRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidConversationInput
	}
	return u.conversationRepo.ListHistoryByUserID(ctx, userID)
}

func (u *ConversationUsecase) ListMessages(ctx context.Context, userID, conversationID string) ([]entity.Message, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidConversationInput
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, ErrInvalidConversationID
	}

	if _, err := u.conversationRepo.FindByID(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return u.conversationRepo.LoadMessages(ctx, userID, conversationID)
}

func (u *ConversationUsecase) streamAgentEvents(ctx context.Context, agent adk.Agent, userID, message string, conversation *entity.Conversation, history []adk.Message, events chan<- SSEPayload) {
	defer close(events)
	rootAgentName := agent.Name(ctx)

	if !sendSSE(ctx, events, "session", entity.SSEEvent{
		Type:           "session",
		SessionID:      conversation.ID,
		ConversationID: conversation.ID,
	}) {
		return
	}

	if err := u.conversationRepo.AppendCachedMessage(ctx, conversation.ID, userID, conversation.ID, "user", message); err != nil {
		sendSSE(ctx, events, "error", entity.SSEEvent{Type: "error", ConversationID: conversation.ID, Error: err.Error()})
		return
	}
	store := store2.NewStore()
	messages := append(history, schema.UserMessage(message))
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: store,
	})
	iter := runner.Run(ctx, messages)
	var assistantContent strings.Builder

	for {
		event, ok := iter.Next()
		if !ok {
			if content := strings.TrimSpace(assistantContent.String()); content != "" {
				if err := u.conversationRepo.AppendCachedMessage(ctx, conversation.ID, userID, conversation.ID, "assistant", content); err != nil {
					sendSSE(ctx, events, "error", entity.SSEEvent{Type: "error", ConversationID: conversation.ID, Error: err.Error()})
					return
				}
			}
			sendSSE(ctx, events, "done", entity.SSEEvent{Type: "done", ConversationID: conversation.ID})
			return
		}

		if event.Err != nil {
			sendSSE(ctx, events, "error", entity.SSEEvent{
				Type:           "error",
				ConversationID: conversation.ID,
				AgentName:      event.AgentName,
				RunPath:        fmt.Sprint(event.RunPath),
				Error:          event.Err.Error(),
			})
			return
		}

		if event.Action != nil {
			if !sendSSE(ctx, events, "action", entity.SSEEvent{
				Type:           "action",
				ConversationID: conversation.ID,
				AgentName:      event.AgentName,
				RunPath:        fmt.Sprint(event.RunPath),
				ActionType:     actionType(event.Action),
			}) {
				return
			}
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		if !streamMessageOutput(ctx, event, conversation.ID, rootAgentName, events, &assistantContent) {
			return
		}
	}
}

func (u *ConversationUsecase) prepareSession(ctx context.Context, agent adk.Agent, userID, sessionID, message string) (*entity.Conversation, []adk.Message, error) {
	if sessionID != "" {
		conversation, history, ok, err := u.conversationRepo.LoadCachedADKMessages(ctx, sessionID, userID)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return conversation, history, nil
		}

		conversation, err = u.conversationRepo.FindByID(ctx, userID, sessionID)
		if err != nil {
			return nil, nil, err
		}
		messageRecords, err := u.conversationRepo.LoadMessages(ctx, userID, conversation.ID)
		if err != nil {
			return nil, nil, err
		}
		if err := u.conversationRepo.CacheConversationWithMessages(ctx, conversation.ID, conversation, messageRecords); err != nil {
			return nil, nil, err
		}
		return conversation, toADKMessages(messageRecords), nil
	}

	conversation, err := u.conversationRepo.FindOrCreate(ctx, userID, "", message, agent.Name(ctx))
	if err != nil {
		return nil, nil, err
	}
	if err := u.conversationRepo.CacheConversation(ctx, conversation.ID, conversation); err != nil {
		return nil, nil, err
	}
	return conversation, nil, nil
}

func toADKMessages(records []entity.Message) []adk.Message {
	messages := make([]adk.Message, 0, len(records))
	for _, record := range records {
		switch record.Role {
		case "user":
			messages = append(messages, schema.UserMessage(record.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(record.Content, nil))
		case "system":
			messages = append(messages, schema.SystemMessage(record.Content))
		}
	}
	return messages
}

func streamMessageOutput(ctx context.Context, event *adk.AgentEvent, conversationID, rootAgentName string, events chan<- SSEPayload, assistantContent *strings.Builder) bool {
	output := event.Output.MessageOutput
	toolCalls := newToolCallAccumulator()
	var subagentContent strings.Builder
	if output.IsStreaming {
		defer output.MessageStream.Close()
		for {
			msg, err := output.MessageStream.Recv()
			if err == io.EOF {
				if !flushSubagentContent(ctx, event, output, conversationID, &subagentContent, events) {
					return false
				}
				if !toolCalls.flush(ctx, event, output, conversationID, events) {
					return false
				}
				return true
			}
			if err != nil {
				return sendSSE(ctx, events, "error", entity.SSEEvent{
					Type:           "error",
					ConversationID: conversationID,
					AgentName:      event.AgentName,
					RunPath:        fmt.Sprint(event.RunPath),
					Error:          err.Error(),
				})
			}
			if !sendMessage(ctx, event, output, conversationID, rootAgentName, msg, events, assistantContent, toolCalls, &subagentContent) {
				return false
			}
		}
	}

	if !sendMessage(ctx, event, output, conversationID, rootAgentName, output.Message, events, assistantContent, toolCalls, &subagentContent) {
		return false
	}
	if !flushSubagentContent(ctx, event, output, conversationID, &subagentContent, events) {
		return false
	}
	return toolCalls.flush(ctx, event, output, conversationID, events)
}

func sendMessage(ctx context.Context, event *adk.AgentEvent, output *adk.MessageVariant, conversationID, rootAgentName string, msg *schema.Message, events chan<- SSEPayload, assistantContent *strings.Builder, toolCalls *toolCallAccumulator, subagentContent *strings.Builder) bool {
	if msg == nil {
		return true
	}
	toolCalls.add(msg.ToolCalls)

	content := msg.Content
	if strings.TrimSpace(content) == "" {
		return true
	}
	if isSubagentOutput(event, rootAgentName) {
		subagentContent.WriteString(content)
		return true
	}
	if content != "" && shouldPersistAssistantContent(event, output, rootAgentName) {
		assistantContent.WriteString(content)
	}

	return sendSSE(ctx, events, "message", entity.SSEEvent{
		Type:           "message",
		ConversationID: conversationID,
		AgentName:      event.AgentName,
		RunPath:        fmt.Sprint(event.RunPath),
		Role:           string(output.Role),
		ToolName:       output.ToolName,
		Content:        content,
	})
}

func flushSubagentContent(ctx context.Context, event *adk.AgentEvent, output *adk.MessageVariant, conversationID string, content *strings.Builder, events chan<- SSEPayload) bool {
	if strings.TrimSpace(content.String()) == "" {
		return true
	}

	return sendSSE(ctx, events, "message", entity.SSEEvent{
		Type:           "message",
		ConversationID: conversationID,
		AgentName:      event.AgentName,
		RunPath:        fmt.Sprint(event.RunPath),
		Role:           string(output.Role),
		ToolName:       output.ToolName,
		Content:        content.String(),
	})
}

func newToolCallAccumulator() *toolCallAccumulator {
	return &toolCallAccumulator{
		calls: make(map[string]schema.ToolCall),
	}
}

func (a *toolCallAccumulator) add(calls []schema.ToolCall) {
	for i, call := range calls {
		key := toolCallKey(call, i)
		current, ok := a.calls[key]
		if !ok {
			a.order = append(a.order, key)
			current = schema.ToolCall{
				Index: call.Index,
				ID:    call.ID,
				Type:  call.Type,
				Extra: call.Extra,
			}
		}

		if current.ID == "" {
			current.ID = call.ID
		}
		if current.Type == "" {
			current.Type = call.Type
		}
		if current.Index == nil {
			current.Index = call.Index
		}
		if call.Function.Name != "" {
			current.Function.Name = call.Function.Name
		}
		current.Function.Arguments += call.Function.Arguments
		if len(call.Extra) > 0 {
			if current.Extra == nil {
				current.Extra = make(map[string]any, len(call.Extra))
			}
			for k, v := range call.Extra {
				current.Extra[k] = v
			}
		}

		a.calls[key] = current
	}
}

func (a *toolCallAccumulator) flush(ctx context.Context, event *adk.AgentEvent, output *adk.MessageVariant, conversationID string, events chan<- SSEPayload) bool {
	if len(a.order) == 0 {
		return true
	}

	calls := make([]schema.ToolCall, 0, len(a.order))
	for _, key := range a.order {
		call := a.calls[key]
		if call.Type == "" {
			call.Type = "function"
		}
		calls = append(calls, call)
	}

	a.order = nil
	a.calls = make(map[string]schema.ToolCall)

	return sendSSE(ctx, events, "message", entity.SSEEvent{
		Type:           "message",
		ConversationID: conversationID,
		AgentName:      event.AgentName,
		RunPath:        fmt.Sprint(event.RunPath),
		Role:           string(output.Role),
		ToolName:       output.ToolName,
		ToolCalls:      calls,
	})
}

func toolCallKey(call schema.ToolCall, fallback int) string {
	if call.Index != nil {
		return fmt.Sprintf("index:%d", *call.Index)
	}
	if call.ID != "" {
		return "id:" + call.ID
	}
	if call.Function.Name != "" {
		return "name:" + call.Function.Name
	}
	return fmt.Sprintf("fallback:%d", fallback)
}

func shouldPersistAssistantContent(event *adk.AgentEvent, output *adk.MessageVariant, rootAgentName string) bool {
	if isSubagentOutput(event, rootAgentName) {
		return false
	}
	return output.Role != schema.Tool && output.ToolName == ""
}

func isSubagentOutput(event *adk.AgentEvent, rootAgentName string) bool {
	return event.AgentName != "" && event.AgentName != rootAgentName
}

func sendSSE(ctx context.Context, events chan<- SSEPayload, event string, data entity.SSEEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case events <- SSEPayload{Event: event, Data: data}:
		return true
	}
}

func actionType(action *adk.AgentAction) string {
	switch {
	case action.Exit:
		return "exit"
	case action.Interrupted != nil:
		return "interrupted"
	case action.TransferToAgent != nil:
		return "transfer_to_agent"
	case action.BreakLoop != nil:
		return "break_loop"
	case action.CustomizedAction != nil:
		return "customized"
	default:
		return "unknown"
	}
}
