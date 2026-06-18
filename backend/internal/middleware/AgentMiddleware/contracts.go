package AgentMiddleware

import (
	"context"
	"einoproject/internal/model"

	"github.com/cloudwego/eino/adk"
)

type MiddleWareList struct {
	List []adk.ChatModelAgentMiddleware
}

func Register(ctx context.Context) (MiddleWareList, error) {
	cm := model.NewChatModel()
	cxtcomposer, err := ContextComposer(ctx, cm)
	if err != nil {
		return MiddleWareList{}, err
	}
	toolReduct, err := NewToolReduction(ctx)
	if err != nil {
		return MiddleWareList{}, err
	}

	return MiddleWareList{
		List: []adk.ChatModelAgentMiddleware{
			cxtcomposer,
			toolReduct,
		},
	}, nil
}
