package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"einoproject/internal/entity"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationRepo struct {
	db    *gorm.DB
	redis *redis.Client
}

type cachedConversation struct {
	Conversation entity.Conversation `json:"conversation"`
	Messages     []entity.Message    `json:"messages"`
}

func NewConversationRepo(db *gorm.DB, redisClient *redis.Client) *ConversationRepo {
	return &ConversationRepo{db: db, redis: redisClient}
}

func (r *ConversationRepo) FindOrCreate(ctx context.Context, userID, conversationID, firstMessage, agentType string) (*entity.Conversation, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("conversation repository is not initialized")
	}

	if conversationID != "" {
		var conversation entity.Conversation
		err := r.db.WithContext(ctx).
			Where("id = ? AND user_id = ?", conversationID, userID).
			First(&conversation).Error
		if err != nil {
			return nil, err
		}
		return &conversation, nil
	}

	conversation := &entity.Conversation{
		ID:        uuid.NewString(),
		UserID:    userID,
		Title:     buildTitle(firstMessage),
		AgentType: agentType,
	}
	return conversation, nil
}

func (r *ConversationRepo) FindByID(ctx context.Context, userID, conversationID string) (*entity.Conversation, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("conversation repository is not initialized")
	}

	var conversation entity.Conversation
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *ConversationRepo) LoadADKMessages(ctx context.Context, userID, conversationID string) ([]adk.Message, error) {
	records, err := r.LoadMessages(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	return toADKMessages(records), nil
}

func (r *ConversationRepo) LoadMessages(ctx context.Context, userID, conversationID string) ([]entity.Message, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("conversation repository is not initialized")
	}

	var records []entity.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (r *ConversationRepo) SaveMessage(ctx context.Context, conversationID, userID, role, content string) error {
	if r == nil || r.db == nil {
		return errors.New("conversation repository is not initialized")
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}

	err := r.db.WithContext(ctx).Create(&entity.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		Content:        content,
	}).Error
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).
		Model(&entity.Conversation{}).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

func (r *ConversationRepo) LoadCachedADKMessages(ctx context.Context, sessionID, userID string) (*entity.Conversation, []adk.Message, bool, error) {
	if r == nil || r.redis == nil {
		return nil, nil, false, errors.New("redis repository is not initialized")
	}

	cached, err := r.getCachedConversation(ctx, sessionID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil, false, nil
		}
		return nil, nil, false, err
	}
	if cached.Conversation.UserID != userID {
		return nil, nil, false, errors.New("cached conversation user mismatch")
	}

	messages := make([]adk.Message, 0, len(cached.Messages))
	for _, record := range cached.Messages {
		msg, ok := toADKMessage(record)
		if ok {
			messages = append(messages, msg)
		}
	}
	return &cached.Conversation, messages, true, nil
}

func (r *ConversationRepo) CacheConversation(ctx context.Context, sessionID string, conversation *entity.Conversation) error {
	if r == nil || r.redis == nil {
		return errors.New("redis repository is not initialized")
	}
	if conversation == nil {
		return errors.New("conversation is nil")
	}

	return r.setCachedConversation(ctx, sessionID, cachedConversation{
		Conversation: *conversation,
	})
}

func (r *ConversationRepo) CacheConversationWithMessages(ctx context.Context, sessionID string, conversation *entity.Conversation, messages []entity.Message) error {
	if r == nil || r.redis == nil {
		return errors.New("redis repository is not initialized")
	}
	if conversation == nil {
		return errors.New("conversation is nil")
	}

	return r.setCachedConversation(ctx, sessionID, cachedConversation{
		Conversation: *conversation,
		Messages:     messages,
	})
}

func (r *ConversationRepo) AppendCachedMessage(ctx context.Context, sessionID, userID, conversationID, role, content string) error {
	if r == nil || r.redis == nil {
		return errors.New("redis repository is not initialized")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	cached, err := r.getCachedConversation(ctx, sessionID)
	if err != nil {
		return err
	}
	if cached.Conversation.ID == "" {
		return errors.New("cached conversation not found")
	}
	if cached.Conversation.ID != conversationID {
		return errors.New("cached conversation id mismatch")
	}
	if cached.Conversation.UserID != userID {
		return errors.New("cached conversation user mismatch")
	}

	cached.Messages = append(cached.Messages, entity.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		Content:        content,
	})

	return r.setCachedConversation(ctx, sessionID, cached)
}

func (r *ConversationRepo) FlushCachedConversationToMySQL(ctx context.Context, sessionID string) error {
	if r == nil || r.db == nil {
		return errors.New("conversation repository is not initialized")
	}
	if r.redis == nil {
		return errors.New("redis repository is not initialized")
	}

	cached, err := r.getCachedConversation(ctx, sessionID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}
	if cached.Conversation.ID == "" {
		return nil
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&cached.Conversation).Error; err != nil {
			return err
		}

		if len(cached.Messages) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&cached.Messages).Error; err != nil {
				return err
			}
		}

		return tx.Model(&entity.Conversation{}).
			Where("id = ? AND user_id = ?", cached.Conversation.ID, cached.Conversation.UserID).
			Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
	})
	return err
}

func (r *ConversationRepo) getCachedConversation(ctx context.Context, sessionID string) (cachedConversation, error) {
	var cached cachedConversation
	data, err := r.redis.Get(ctx, sessionID).Bytes()
	if err != nil {
		return cached, err
	}
	if err := json.Unmarshal(data, &cached); err != nil {
		return cached, err
	}
	return cached, nil
}

func (r *ConversationRepo) setCachedConversation(ctx context.Context, sessionID string, cached cachedConversation) error {
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, sessionID, data, 24*time.Hour).Err()
}

func buildTitle(firstMessage string) string {
	title := strings.TrimSpace(firstMessage)
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[:30])
	}
	if title == "" {
		title = "New conversation"
	}
	return title
}

func toADKMessage(record entity.Message) (adk.Message, bool) {
	switch record.Role {
	case "user":
		return schema.UserMessage(record.Content), true
	case "assistant":
		return schema.AssistantMessage(record.Content, nil), true
	case "system":
		return schema.SystemMessage(record.Content), true
	default:
		return nil, false
	}
}

func toADKMessages(records []entity.Message) []adk.Message {
	messages := make([]adk.Message, 0, len(records))
	for _, record := range records {
		msg, ok := toADKMessage(record)
		if ok {
			messages = append(messages, msg)
		}
	}
	return messages
}
