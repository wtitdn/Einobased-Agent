package usecase

import (
	"context"
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

func NewConversationUsecase(conversationRepo *repo.ConversationRepo) *ConversationUsecase {
	return &ConversationUsecase{conversationRepo: conversationRepo}
}

func (u *ConversationUsecase) StreamAgent(ctx context.Context, agent adk.Agent, userID, sessionID, message string) <-chan SSEPayload {
	events := make(chan SSEPayload, 8)
	go u.streamAgentEvents(ctx, agent, userID, sessionID, message, events)
	return events
}

func (u *ConversationUsecase) FlushCachedConversation(ctx context.Context, sessionID string) error {
	return u.conversationRepo.FlushCachedConversationToMySQL(ctx, sessionID)
}

func (u *ConversationUsecase) streamAgentEvents(ctx context.Context, agent adk.Agent, userID, sessionID, message string, events chan<- SSEPayload) {
	defer close(events)

	conversation, history, err := u.prepareSession(ctx, agent, userID, sessionID, message)
	if err != nil {
		sendSSE(ctx, events, "error", entity.SSEEvent{Type: "error", Error: err.Error()})
		return
	}
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

	messages := append(history, schema.UserMessage(message))
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
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

		if !streamMessageOutput(ctx, event, conversation.ID, events, &assistantContent) {
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

func streamMessageOutput(ctx context.Context, event *adk.AgentEvent, conversationID string, events chan<- SSEPayload, assistantContent *strings.Builder) bool {
	output := event.Output.MessageOutput
	if output.IsStreaming {
		defer output.MessageStream.Close()
		for {
			msg, err := output.MessageStream.Recv()
			if err == io.EOF {
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
			if !sendMessage(ctx, event, conversationID, msg, events, assistantContent) {
				return false
			}
		}
	}

	return sendMessage(ctx, event, conversationID, output.Message, events, assistantContent)
}

func sendMessage(ctx context.Context, event *adk.AgentEvent, conversationID string, msg *schema.Message, events chan<- SSEPayload, assistantContent *strings.Builder) bool {
	if msg == nil {
		return true
	}
	if msg.Content != "" {
		assistantContent.WriteString(msg.Content)
	}

	return sendSSE(ctx, events, "message", entity.SSEEvent{
		Type:           "message",
		ConversationID: conversationID,
		AgentName:      event.AgentName,
		RunPath:        fmt.Sprint(event.RunPath),
		Content:        msg.Content,
		ToolCalls:      msg.ToolCalls,
	})
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
