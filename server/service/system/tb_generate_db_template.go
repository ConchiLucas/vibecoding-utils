package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateDbTemplateTypeService struct{}
type TbGenerateDbTemplateScriptService struct{}

func (s *TbGenerateDbTemplateTypeService) Create(req *system.TbGenerateDbTemplateType) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateDbTemplateTypeService) Delete(req system.TbGenerateDbTemplateType) error {
	tx := global.GVA_DB.Begin()
	if err := tx.Where("type_id = ?", req.ID).Unscoped().Delete(&system.TbGenerateDbTemplateScript{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Unscoped().Delete(&req).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *TbGenerateDbTemplateTypeService) Update(req *system.TbGenerateDbTemplateType) error {
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateDbTemplateTypeService) Get(id string) (res system.TbGenerateDbTemplateType, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateDbTemplateTypeService) List(projectId int) (res []system.TbGenerateDbTemplateType, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateDbTemplateType{})
	if projectId > 0 {
		db = db.Where("project_id = ?", projectId)
	}
	err = db.Order("sort ASC, id ASC").Find(&res).Error
	return
}

func (s *TbGenerateDbTemplateScriptService) Create(req *system.TbGenerateDbTemplateScript) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateDbTemplateScriptService) Delete(req system.TbGenerateDbTemplateScript) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateDbTemplateScriptService) Update(req *system.TbGenerateDbTemplateScript) error {
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateDbTemplateScriptService) Get(id string) (res system.TbGenerateDbTemplateScript, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateDbTemplateScriptService) List(projectId, typeId int) (res []system.TbGenerateDbTemplateScript, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateDbTemplateScript{})
	if projectId > 0 {
		db = db.Where("project_id = ?", projectId)
	}
	if typeId > 0 {
		db = db.Where("type_id = ?", typeId)
	}
	err = db.Order("sort ASC, id ASC").Find(&res).Error
	return
}
