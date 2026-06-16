package http

import (
	agents "einoproject/internal/Agents"
	ssehandler "einoproject/internal/controller/SSE"
	"einoproject/internal/repo"
	"einoproject/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func SetRouter(registeredAgents agents.Agents, db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	r := gin.Default()
	conversationRepo := repo.NewConversationRepo(db, redisClient)
	conversationUsecase := usecase.NewConversationUsecase(conversationRepo)

	sseGroup := r.Group("/sse")
	for name, agent := range registeredAgents.SSEAgents() {
		sseGroup.GET("/"+name, ssehandler.AgentSSE(agent, conversationUsecase))
	}

	return r
}
