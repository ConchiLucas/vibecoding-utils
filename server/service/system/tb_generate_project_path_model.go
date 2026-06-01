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
	return global.GVA_DB.Updates(req).Error
}

func (s *TbGenerateProjectPathModelService) GetTbGenerateProjectPathModel(id string) (res system.TbGenerateProjectPathModel, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPathModelService) GetTbGenerateProjectPathModelList() (res []system.TbGenerateProjectPathModel, err error) {
	err = global.GVA_DB.Find(&res).Error
	return
}
