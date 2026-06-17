package AgentMiddleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
)

func NewToolReduction(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	fsBackend := filesystem.NewInMemoryBackend()
	mw, err := reduction.New(ctx, &reduction.Config{
		Backend:           fsBackend,
		MaxLengthForTrunc: 30000,
		MaxTokensForClear: 50000,
	})
	return mw, err
}
