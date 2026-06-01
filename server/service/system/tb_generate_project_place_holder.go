package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateProjectPlaceHolderService struct{}

func (s *TbGenerateProjectPlaceHolderService) CreateTbGenerateProjectPlaceHolder(req *system.TbGenerateProjectPlaceHolder) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateProjectPlaceHolderService) DeleteTbGenerateProjectPlaceHolder(req system.TbGenerateProjectPlaceHolder) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateProjectPlaceHolderService) UpdateTbGenerateProjectPlaceHolder(req *system.TbGenerateProjectPlaceHolder) error {
	return global.GVA_DB.Updates(req).Error
}

func (s *TbGenerateProjectPlaceHolderService) GetTbGenerateProjectPlaceHolder(id string) (res system.TbGenerateProjectPlaceHolder, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateProjectPlaceHolderService) GetTbGenerateProjectPlaceHolderList() (res []system.TbGenerateProjectPlaceHolder, err error) {
	err = global.GVA_DB.Find(&res).Error
	return
}
