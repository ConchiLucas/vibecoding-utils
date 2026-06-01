package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateProjectPathService struct{}

func (s *TbGenerateProjectPathService) CreateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPathService) DeleteTbGenerateProjectPath(req system.TbGenerateProjectPath) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateProjectPathService) UpdateTbGenerateProjectPath(req *system.TbGenerateProjectPath) error {
	return global.GVA_DB.Updates(req).Error
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPath(id string) (res system.TbGenerateProjectPath, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPathService) GetTbGenerateProjectPathList() (res []system.TbGenerateProjectPath, err error) {
	err = global.GVA_DB.Find(&res).Error
	return
}

func (s *TbGenerateProjectPathService) UpdateEnabled(id uint, enabled int) error {
	return global.GVA_DB.Model(&system.TbGenerateProjectPath{}).Where("id = ?", id).Update("enabled", enabled).Error
}
