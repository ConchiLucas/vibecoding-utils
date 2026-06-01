package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGeneratePlaceHolderService struct{}

func (s *TbGeneratePlaceHolderService) CreateTbGeneratePlaceHolder(req *system.TbGeneratePlaceHolder) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGeneratePlaceHolderService) DeleteTbGeneratePlaceHolder(req system.TbGeneratePlaceHolder) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGeneratePlaceHolderService) UpdateTbGeneratePlaceHolder(req *system.TbGeneratePlaceHolder) error {
	return global.GVA_DB.Updates(req).Error
}

func (s *TbGeneratePlaceHolderService) GetTbGeneratePlaceHolder(id string) (res system.TbGeneratePlaceHolder, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGeneratePlaceHolderService) GetTbGeneratePlaceHolderList() (res []system.TbGeneratePlaceHolder, err error) {
	err = global.GVA_DB.Find(&res).Error
	return
}
