package ragSearchAgent

import (
	"context"
	"einoproject/internal/model"
	"einoproject/internal/tools/ragSearch"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func CreateAgent(ctx context.Context) (adk.Agent, error) {
	chromaTool, err := ragSearch.NewTool()
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ragSearchAgent",
		Description: "An agent specialized in querying the local persisted Chroma database.",
		Instruction: agentInstruction,
		Model:       model.NewTollCallModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{chromaTool},
			},
			ReturnDirectly: map[string]bool{
				ragSearch.ToolName: false,
			},
		},
		MaxIterations: 6,
	})
}

func CreateTool(ctx context.Context) (tool.BaseTool, error) {
	agent, err := CreateAgent(ctx)
	if err != nil {
		return nil, err
	}
	return adk.NewAgentTool(ctx, agent), nil
}
