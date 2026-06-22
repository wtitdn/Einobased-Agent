package handler

import (
	response "einoproject/internal/controller/DTO/response"
	"einoproject/internal/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	agentService *usecase.AgentService
}

func NewAgentHandler(agentService *usecase.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

func (h *AgentHandler) ListAgents(c *gin.Context) {
	agents := h.agentService.AgentList()
	items := make([]response.AgentItem, 0, len(agents))
	for _, agent := range agents {
		items = append(items, response.AgentItem{
			Name:  agent.Name,
			Label: agent.Label,
			Path:  agent.Path,
		})
	}

	c.JSON(http.StatusOK, response.AgentsResponse{
		Agents: items,
	})
}
