package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbClientService struct{}

func (s *TbClientService) CreateTbClient(client system.TbClient) (err error) {
	err = global.GVA_DB.Create(&client).Error
	return err
}

func (s *TbClientService) DeleteTbClient(client system.TbClient) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&client).Error
	return err
}

func (s *TbClientService) DeleteTbClientByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbClient{}, "id in ?", ids).Error
	return err
}

func (s *TbClientService) UpdateTbClient(client *system.TbClient) (err error) {
	err = global.GVA_DB.Updates(client).Error
	return err
}

func (s *TbClientService) GetTbClient(id uint) (client system.TbClient, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&client).Error
	return
}

func (s *TbClientService) GetTbClientInfoList(info systemReq.ClientSearch) (list []system.TbClient, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbClient{})
	if info.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+info.NickName+"%")
	}
	if info.LoginName != "" {
		db = db.Where("login_name = ?", info.LoginName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}
