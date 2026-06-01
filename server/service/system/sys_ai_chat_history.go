package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIChatHistoryService struct{}

type AIChatHistoryMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type SaveAIChatHistoryRequest struct {
	ChatID   string                 `json:"chatId"`
	Title    string                 `json:"title"`
	Provider string                 `json:"provider"`
	Messages []AIChatHistoryMessage `json:"messages"`
}

type AIChatHistoryItem struct {
	ID        uint                   `json:"ID"`
	ChatID    string                 `json:"chatId"`
	Title     string                 `json:"title"`
	Provider  string                 `json:"provider"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
	Messages  []AIChatHistoryMessage `json:"messages,omitempty"`
}

func (s *AIChatHistoryService) SaveOrUpdateChatHistory(req SaveAIChatHistoryRequest, userID uint) (AIChatHistoryItem, error) {
	if userID == 0 {
		return AIChatHistoryItem{}, errors.New("用户未登录")
	}
	if len(req.Messages) == 0 {
		return AIChatHistoryItem{}, errors.New("消息不能为空")
	}
	if strings.TrimSpace(req.ChatID) == "" {
		req.ChatID = uuid.NewString()
	}
	req.Title = normalizeAIChatHistoryTitle(req.Title, req.Messages)

	if err := s.ensureChatHistoryTable(); err != nil {
		return AIChatHistoryItem{}, err
	}

	messagesJSON, err := json.Marshal(req.Messages)
	if err != nil {
		return AIChatHistoryItem{}, err
	}

	var record modelSystem.TbAIChatHistory
	err = global.GVA_DB.Where("user_id = ? AND chat_id = ?", userID, req.ChatID).First(&record).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AIChatHistoryItem{}, err
	}

	record.ChatID = req.ChatID
	record.UserID = userID
	record.Title = req.Title
	record.Provider = req.Provider
	record.Messages = string(messagesJSON)

	if record.ID == 0 {
		err = global.GVA_DB.Create(&record).Error
	} else {
		err = global.GVA_DB.Save(&record).Error
	}
	if err != nil {
		return AIChatHistoryItem{}, err
	}

	return toAIChatHistoryItem(record, true)
}

func (s *AIChatHistoryService) GetChatHistoryList(userID uint, limit int) ([]AIChatHistoryItem, int64, error) {
	if userID == 0 {
		return nil, 0, errors.New("用户未登录")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if err := s.ensureChatHistoryTable(); err != nil {
		return nil, 0, err
	}

	var total int64
	query := global.GVA_DB.Model(&modelSystem.TbAIChatHistory{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []modelSystem.TbAIChatHistory
	if err := query.Order("updated_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	list := make([]AIChatHistoryItem, 0, len(records))
	for _, record := range records {
		item, err := toAIChatHistoryItem(record, true)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, total, nil
}

func (s *AIChatHistoryService) GetChatHistoryByChatID(userID uint, chatID string) (AIChatHistoryItem, error) {
	if userID == 0 {
		return AIChatHistoryItem{}, errors.New("用户未登录")
	}
	if strings.TrimSpace(chatID) == "" {
		return AIChatHistoryItem{}, errors.New("对话ID不能为空")
	}
	if err := s.ensureChatHistoryTable(); err != nil {
		return AIChatHistoryItem{}, err
	}

	var record modelSystem.TbAIChatHistory
	if err := global.GVA_DB.Where("user_id = ? AND chat_id = ?", userID, chatID).First(&record).Error; err != nil {
		return AIChatHistoryItem{}, err
	}
	return toAIChatHistoryItem(record, true)
}

func (s *AIChatHistoryService) ensureChatHistoryTable() error {
	if global.GVA_DB == nil {
		return errors.New("数据库未初始化")
	}
	if global.GVA_DB.Migrator().HasTable(&modelSystem.TbAIChatHistory{}) {
		return nil
	}
	if err := global.GVA_DB.AutoMigrate(&modelSystem.TbAIChatHistory{}); err != nil {
		return fmt.Errorf("创建 AI 对话历史表失败: %w", err)
	}
	return nil
}

func normalizeAIChatHistoryTitle(title string, messages []AIChatHistoryMessage) string {
	title = strings.TrimSpace(title)
	if title == "" {
		for _, message := range messages {
			if message.Role == "user" && strings.TrimSpace(message.Content) != "" {
				title = strings.TrimSpace(message.Content)
				break
			}
		}
	}
	if title == "" {
		title = "新的对话"
	}
	title = strings.Join(strings.Fields(title), " ")
	if len([]rune(title)) > 64 {
		title = string([]rune(title)[:64]) + "..."
	}
	return title
}

func toAIChatHistoryItem(record modelSystem.TbAIChatHistory, includeMessages bool) (AIChatHistoryItem, error) {
	item := AIChatHistoryItem{
		ID:        record.ID,
		ChatID:    record.ChatID,
		Title:     record.Title,
		Provider:  record.Provider,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05.000Z07:00"),
	}
	if includeMessages && record.Messages != "" {
		if err := json.Unmarshal([]byte(record.Messages), &item.Messages); err != nil {
			return item, err
		}
	}
	return item, nil
}
