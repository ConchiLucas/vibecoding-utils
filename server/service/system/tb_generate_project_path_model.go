package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateProjectPathModelService struct{}

func (s *TbGenerateProjectPathModelService) CreateTbGenerateProjectPathModel(req *system.TbGenerateProjectPathModel) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPathModelService) DeleteTbGenerateProjectPathModel(req system.TbGenerateProjectPathModel) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateProjectPathModelService) UpdateTbGenerateProjectPathModel(req *system.TbGenerateProjectPathModel) error {
	return global.GVA_DB.Save(req).Error
}

func (s *TbGenerateProjectPathModelService) GetTbGenerateProjectPathModel(id string) (res system.TbGenerateProjectPathModel, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPathModelService) GetTbGenerateProjectPathModelList(pathId int) (res []system.TbGenerateProjectPathModel, err error) {
	db := global.GVA_DB.Model(&system.TbGenerateProjectPathModel{})
	if pathId > 0 {
		db = db.Where("path_id = ?", pathId)
	}
	err = db.Order("id ASC").Find(&res).Error
	return
}
