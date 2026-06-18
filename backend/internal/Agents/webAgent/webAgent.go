package webAgent

import (
	"context"
	"einoproject/internal/model"
	"einoproject/internal/tools/websearch"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func CreateAgent(ctx context.Context) (adk.Agent, error) {
	browserTool, err := websearch.NewTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "webAgent",
		Description: "A specialized web browsing agent that can search, navigate pages, interact with browser elements, and extract web content.",
		Instruction: agentInstruction,
		Model:       model.NewTollCallModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{browserTool},
			},
		},
		MaxIterations: 12,
	})
}
