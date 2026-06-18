package toolCallingAgent

import (
	"context"
	"einoproject/internal/Agents/ragSearchAgent"
	"einoproject/internal/Agents/webAgent"
	"einoproject/internal/middleware/AgentMiddleware"
	"einoproject/internal/model"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func CreateAgent(ctx context.Context) (adk.Agent, error) {
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}

	chatModel := model.NewChatModel()
	//中间件注册
	middleware, err := AgentMiddleware.Register(ctx)
	if err != nil {
		return nil, err
	}
	//RAG subagent注册
	ragTool, err := ragSearchAgent.CreateTool(ctx)
	if err != nil {
		return nil, err
	}
	//webSearch subagent注册
	web, err := webAgent.CreateAgent(ctx)
	if err != nil {
		return nil, err
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:        "toolCallingAgent",
		Description: "ChatWithDoc agent with filesystem access via LocalBackend.",
		ChatModel:   chatModel,
		Instruction: agentInstruction,
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{ragTool},
			},
		},
		SubAgents:      []adk.Agent{web},
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers:       middleware.List,
	})
	return agent, err
}
