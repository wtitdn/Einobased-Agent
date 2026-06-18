package handler

import (
	"errors"
	"net/http"
	"strings"

	response "einoproject/internal/controller/DTO/response"
	"einoproject/internal/usecase"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConversationHandler struct {
	conversationUsecase *usecase.ConversationUsecase
}

func NewConversationHandler(conversationUsecase *usecase.ConversationUsecase) *ConversationHandler {
	return &ConversationHandler{conversationUsecase: conversationUsecase}
}

func (h *ConversationHandler) ListHistory(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	histories, err := h.conversationUsecase.ListHistoryByUserID(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidConversationInput):
			c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: err.Error()})
		}
		return
	}

	conversations := make([]response.ConversationHistoryItem, 0, len(histories))
	for _, history := range histories {
		conversations = append(conversations, response.ConversationHistoryItem{
			ID:           history.ID,
			UserID:       history.UserID,
			Title:        history.Title,
			AgentType:    history.AgentType,
			MessageCount: history.MessageCount,
			LastMessage:  history.LastMessage,
			CreatedAt:    history.CreatedAt,
			UpdatedAt:    history.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, response.ConversationHistoryResponse{
		Conversations: conversations,
	})
}

func (h *ConversationHandler) ListMessages(c *gin.Context) {
	userID := strings.TrimSpace(c.Query("user_id"))
	conversationID := strings.TrimSpace(c.Query("conversation_id"))
	messages, err := h.conversationUsecase.ListMessages(c.Request.Context(), userID, conversationID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidConversationInput), errors.Is(err, usecase.ErrInvalidConversationID):
			c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: "conversation not found"})
		default:
			c.JSON(http.StatusInternalServerError, response.ErrorResponse{Error: err.Error()})
		}
		return
	}

	items := make([]response.ConversationMessageItem, 0, len(messages))
	for _, message := range messages {
		items = append(items, response.ConversationMessageItem{
			ID:             message.ID,
			ConversationID: message.ConversationID,
			UserID:         message.UserID,
			Role:           message.Role,
			Content:        message.Content,
			CreatedAt:      message.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response.ConversationMessagesResponse{
		Messages: items,
	})
}
