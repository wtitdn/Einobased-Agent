package Agents

import (
	"context"
	"einoproject/internal/Agents/ragSearchAgent"
	"einoproject/internal/Agents/simpleAgent"
	"einoproject/internal/Agents/toolCallingAgent"

	"github.com/cloudwego/eino/adk"
)

type Agents struct {
	SimpleAgent      adk.Agent
	toolCallingAgent adk.Agent
	RagSearchAgent   adk.Agent
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
	ragSearch, err := ragSearchAgent.CreateAgent(ctx)
	if err != nil {
		panic(err)
	}
	return Agents{SimpleAgent: simple, toolCallingAgent: toolCalling, RagSearchAgent: ragSearch}
}

func (a Agents) SSEAgents() map[string]adk.Agent {
	return map[string]adk.Agent{
		"simple":    a.SimpleAgent,
		"ragSearch": a.RagSearchAgent,
		"toolCall":  a.toolCallingAgent,
	}
}
