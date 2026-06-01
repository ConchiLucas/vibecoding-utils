package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbGenerateRecordService struct{}

func (s *TbGenerateRecordService) CreateTbGenerateRecord(req *system.TbGenerateRecord) error {
	return global.GVA_DB.Create(req).Error
}

func (s *TbGenerateRecordService) DeleteTbGenerateRecord(req system.TbGenerateRecord) error {
	return global.GVA_DB.Unscoped().Delete(&req).Error
}

func (s *TbGenerateRecordService) UpdateTbGenerateRecord(req *system.TbGenerateRecord) error {
	return global.GVA_DB.Updates(req).Error
}

func (s *TbGenerateRecordService) GetTbGenerateRecord(id string) (res system.TbGenerateRecord, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&res).Error
	return
}

func (s *TbGenerateRecordService) GetTbGenerateRecordList() (res []system.TbGenerateRecord, err error) {
	err = global.GVA_DB.Order("id DESC").Find(&res).Error
	return
}

func (s *TbGenerateRecordService) GetTbGenerateRecordByUserName(userName string) (res system.TbGenerateRecord, err error) {
	err = global.GVA_DB.Where("user_name = ?", userName).Order("id DESC").First(&res).Error
	return
}

func (s *TbGenerateRecordService) GetLatestGenerateRecord(userName string, projectId int) (res system.TbGenerateRecord, err error) {
	db := global.GVA_DB.Where("user_name = ?", userName)
	if projectId > 0 {
		db = db.Where("project_id = ?", projectId)
	}
	err = db.Order("id DESC").First(&res).Error
	return
}
