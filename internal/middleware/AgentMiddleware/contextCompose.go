package AgentMiddleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

func ContextComposer(ctx context.Context, chatModel model.BaseChatModel) (adk.ChatModelAgentMiddleware, error) {
	return summarization.New(ctx, &summarization.Config{
		Model: chatModel,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 100000,
		},
	})
}
