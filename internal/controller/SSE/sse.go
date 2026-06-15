package sse

import (
	"context"
	"fmt"
	"io"
	"strings"

	"einoproject/internal/entity"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

type ssePayload struct {
	event string
	data  entity.SSEEvent
}

func AgentSSE(agent adk.Agent) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := strings.TrimSpace(c.Query("message"))
		if message == "" {
			c.JSON(400, gin.H{"error": "message is required"})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		events := make(chan ssePayload, 8)
		go streamAgentEvents(ctx, agent, message, events)

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				return false
			case payload, ok := <-events:
				if !ok {
					return false
				}
				c.SSEvent(payload.event, payload.data)
				return true
			}
		})
	}
}

func streamAgentEvents(ctx context.Context, agent adk.Agent, message string, events chan<- ssePayload) {
	defer close(events)

	iter := agent.Run(ctx, &adk.AgentInput{
		Messages:        []adk.Message{schema.UserMessage(message)},
		EnableStreaming: true,
	})

	for {
		event, ok := iter.Next()
		if !ok {
			sendSSE(ctx, events, "done", entity.SSEEvent{Type: "done"})
			return
		}

		if event.Err != nil {
			sendSSE(ctx, events, "error", entity.SSEEvent{
				Type:      "error",
				AgentName: event.AgentName,
				RunPath:   fmt.Sprint(event.RunPath),
				Error:     event.Err.Error(),
			})
			return
		}

		if event.Action != nil {
			if !sendSSE(ctx, events, "action", entity.SSEEvent{
				Type:       "action",
				AgentName:  event.AgentName,
				RunPath:    fmt.Sprint(event.RunPath),
				ActionType: actionType(event.Action),
			}) {
				return
			}
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		if !streamMessageOutput(ctx, event, events) {
			return
		}
	}
}

func streamMessageOutput(ctx context.Context, event *adk.AgentEvent, events chan<- ssePayload) bool {
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
					Type:      "error",
					AgentName: event.AgentName,
					RunPath:   fmt.Sprint(event.RunPath),
					Error:     err.Error(),
				})
			}
			if !sendMessage(ctx, event, msg, events) {
				return false
			}
		}
	}

	return sendMessage(ctx, event, output.Message, events)
}

func sendMessage(ctx context.Context, event *adk.AgentEvent, msg *schema.Message, events chan<- ssePayload) bool {
	if msg == nil {
		return true
	}

	return sendSSE(ctx, events, "message", entity.SSEEvent{
		Type:      "message",
		AgentName: event.AgentName,
		RunPath:   fmt.Sprint(event.RunPath),
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	})
}

func sendSSE(ctx context.Context, events chan<- ssePayload, event string, data entity.SSEEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case events <- ssePayload{event: event, data: data}:
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
