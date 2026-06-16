package toolCallingAgent

import (
	"context"
	"einoproject/internal/model"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
)

func CreateAgent(ctx context.Context) (adk.Agent, error) {
	// 创建 LocalBackend
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		return nil, err
	}
	// 创建 DeepAgent,自动注册文件系统工具
	agent, err := deep.New(ctx, &deep.Config{
		Name:           "toolCallingAgent",
		Description:    "ChatWithDoc agent with filesystem access via LocalBackend.",
		ChatModel:      model.NewTollCallModel(),
		Instruction:    agentInstruction,
		Backend:        backend, // 提供文件系统操作能力
		StreamingShell: backend, // 提供命令执行能力
		MaxIteration:   50,
	})
	return agent, err
}
