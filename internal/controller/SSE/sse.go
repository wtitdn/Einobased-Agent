package sse

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"einoproject/internal/usecase"

	"github.com/cloudwego/eino/adk"
	"github.com/gin-gonic/gin"
)

func AgentSSE(agent adk.Agent, conversationUsecase *usecase.ConversationUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := strings.TrimSpace(c.Query("message"))
		if message == "" {
			c.JSON(400, gin.H{"error": "message is required"})
			return
		}

		userID := strings.TrimSpace(c.Query("user_id"))
		if userID == "" {
			c.JSON(400, gin.H{"error": "user_id is required"})
			return
		}
		if conversationUsecase == nil {
			c.JSON(500, gin.H{"error": "conversation usecase is not initialized"})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		sessionID := strings.TrimSpace(c.Query("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(c.Query("conversation_id"))
		}

		var activeSessionID string
		defer func() {
			if activeSessionID == "" {
				return
			}
			flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := conversationUsecase.FlushCachedConversation(flushCtx, activeSessionID); err != nil {
				log.Printf("flush conversation cache failed: %v", err)
			}
		}()

		events := conversationUsecase.StreamAgent(ctx, agent, userID, sessionID, message)

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				return false
			case payload, ok := <-events:
				if !ok {
					return false
				}
				if payload.Data.SessionID != "" {
					activeSessionID = payload.Data.SessionID
				}
				c.SSEvent(payload.Event, payload.Data)
				return true
			}
		})
	}
}
