package Agents

import (
	"context"
	"einoproject/internal/Agents/simpleAgent"
	"einoproject/internal/Agents/toolCallingAgent"

	"github.com/cloudwego/eino/adk"
)

type Agents struct {
	SimpleAgent      adk.Agent
	toolCallingAgent adk.Agent
}

func RegisterAgents(ctx context.Context) Agents {
	simple, err := simpleAgent.CreateAgent(ctx)
	if err != nil {
		panic(err)
	}
	toolCalling, err := toolCallingAgent.CreateAgent(ctx)
	if err != nil {
		panic(err)
	}
	return Agents{SimpleAgent: simple, toolCallingAgent: toolCalling}
}

func (a Agents) SSEAgents() map[string]adk.Agent {
	return map[string]adk.Agent{
		"simple": a.SimpleAgent,
	}
}
