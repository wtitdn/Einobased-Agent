package simpleAgent

import (
	"context"
	"einoproject/internal/model"

	"github.com/cloudwego/eino/adk"
)

func CreateAgent(ctx context.Context) (adk.Agent, error) {
	// add sub-agents if you want to.
	// for demonstration purpose we use a simple ChatModelAgent
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SSEAgent",
		Description: "An agent that responds via Server-Sent Events",
		Instruction: `You are a helpful assistant. Provide clear and concise responses to user queries.`,
		Model:       model.NewChatModel(),
		// add tools if you want to
	})
}
