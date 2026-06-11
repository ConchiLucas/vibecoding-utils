package system

import (
	"fmt"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbDevelopmentPrepareService struct{}

const (
	developmentPrepareTypeScript    = "script"
	developmentPrepareTypeCode      = "code"
	developmentPrepareTypeChecklist = "checklist"
)

func (s *TbDevelopmentPrepareService) GetList(info systemReq.DevelopmentPrepareSearch, userID uint) (list []modelSystem.TbDevelopmentPrepare, total int64, err error) {
	if err = ensureDevelopmentPrepareTable(); err != nil {
		return
	}

	db := global.GVA_DB.Model(&modelSystem.TbDevelopmentPrepare{})
	if userID != 0 {
		db = db.Where("user_id = ?", userID)
	}
	if info.ProjectConfigId != 0 {
		db = db.Where("project_config_id = ?", info.ProjectConfigId)
	} else if strings.TrimSpace(info.ProjectConfigName) != "" {
		db = db.Where("project_config_name = ?", strings.TrimSpace(info.ProjectConfigName))
	}
	if itemType := normalizeDevelopmentPrepareType(info.ItemType); itemType != "" {
		db = db.Where("item_type = ?", itemType)
	}
	if businessGroup := strings.TrimSpace(info.BusinessGroup); businessGroup != "" {
		db = db.Where("business_group = ?", businessGroup)
	}
	if keyword := strings.TrimSpace(info.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("business_group LIKE ? OR title LIKE ? OR summary LIKE ? OR tags LIKE ? OR content LIKE ?", like, like, like, like, like)
	}

	if err = db.Count(&total).Error; err != nil {
		return
	}
	err = db.Scopes(info.Paginate()).
		Order("is_pinned desc, sort asc, updated_at desc, id desc").
		Find(&list).Error
	return
}

func (s *TbDevelopmentPrepareService) GetByID(id uint, userID uint) (item modelSystem.TbDevelopmentPrepare, err error) {
	if err = ensureDevelopmentPrepareTable(); err != nil {
		return
	}
	db := global.GVA_DB.Where("id = ?", id)
	if userID != 0 {
		db = db.Where("user_id = ?", userID)
	}
	err = db.First(&item).Error
	return
}

func (s *TbDevelopmentPrepareService) SaveOrUpdate(item modelSystem.TbDevelopmentPrepare, userID uint) error {
	if err := ensureDevelopmentPrepareTable(); err != nil {
		return err
	}
	item.Title = strings.TrimSpace(item.Title)
	item.ProjectConfigName = strings.TrimSpace(item.ProjectConfigName)
	item.BusinessGroup = strings.TrimSpace(item.BusinessGroup)
	item.ItemType = normalizeDevelopmentPrepareType(item.ItemType)
	item.Language = strings.TrimSpace(item.Language)
	item.Tags = strings.TrimSpace(item.Tags)
	item.Summary = strings.TrimSpace(item.Summary)

	if item.Title == "" {
		return fmt.Errorf("标题不能为空")
	}
	if item.ProjectConfigId == 0 && item.ProjectConfigName == "" {
		return fmt.Errorf("请选择所属项目")
	}
	if item.ItemType == "" {
		item.ItemType = developmentPrepareTypeScript
	}

	if item.ID != 0 {
		db := global.GVA_DB.Model(&modelSystem.TbDevelopmentPrepare{}).Where("id = ?", item.ID)
		if userID != 0 {
			db = db.Where("user_id = ?", userID)
		}
		return db.Updates(map[string]interface{}{
			"project_config_id":   item.ProjectConfigId,
			"project_config_name": item.ProjectConfigName,
			"business_group":      item.BusinessGroup,
			"title":               item.Title,
			"item_type":           item.ItemType,
			"language":            item.Language,
			"tags":                item.Tags,
			"summary":             item.Summary,
			"content":             item.Content,
			"is_pinned":           item.IsPinned,
			"sort":                item.Sort,
		}).Error
	}

	item.UserId = userID
	return global.GVA_DB.Create(&item).Error
}

func (s *TbDevelopmentPrepareService) Delete(ids []int, userID uint) error {
	if err := ensureDevelopmentPrepareTable(); err != nil {
		return err
	}
	db := global.GVA_DB.Unscoped().Where("id in ?", ids)
	if userID != 0 {
		db = db.Where("user_id = ?", userID)
	}
	return db.Delete(&modelSystem.TbDevelopmentPrepare{}).Error
}

func ensureDevelopmentPrepareTable() error {
	if global.GVA_DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	return global.GVA_DB.AutoMigrate(&modelSystem.TbDevelopmentPrepare{})
}

func normalizeDevelopmentPrepareType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case developmentPrepareTypeScript:
		return developmentPrepareTypeScript
	case developmentPrepareTypeCode:
		return developmentPrepareTypeCode
	case developmentPrepareTypeChecklist:
		return developmentPrepareTypeChecklist
	default:
		return ""
	}
}
