package http

import (
	agents "einoproject/internal/Agents"
	ssehandler "einoproject/internal/controller/SSE"

	"github.com/gin-gonic/gin"
)

func SetRouter(registeredAgents agents.Agents) *gin.Engine {
	r := gin.Default()

	sseGroup := r.Group("/sse")
	for name, agent := range registeredAgents.SSEAgents() {
		sseGroup.GET("/"+name, ssehandler.AgentSSE(agent))
	}

	return r
}
