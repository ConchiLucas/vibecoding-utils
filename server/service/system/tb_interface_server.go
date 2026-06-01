package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"

	"gorm.io/gorm"
)

type TbInterfaceServerService struct{}

func (s *TbInterfaceServerService) CreateTbInterfaceServer(server system.TbInterfaceServer) (err error) {
	err = global.GVA_DB.Create(&server).Error
	return err
}

// DeleteTbInterfaceServer deletes a server and cascades to interfaces, entities, columns
// (mirrors Python delete_server_services)
func (s *TbInterfaceServerService) DeleteTbInterfaceServer(server system.TbInterfaceServer) (err error) {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var srv system.TbInterfaceServer
		if e := tx.Where("id = ?", server.ID).First(&srv).Error; e != nil {
			return e
		}
		if e := tx.Where("server_name = ? AND user_name = ?", srv.ServerName, srv.UserName).
			Unscoped().Delete(&system.TbInterface{}).Error; e != nil {
			return e
		}
		if e := tx.Where("server_name = ? AND user_name = ?", srv.ServerName, srv.UserName).
			Unscoped().Delete(&system.TbEntity{}).Error; e != nil {
			return e
		}
		if e := tx.Where("server_name = ? AND user_name = ?", srv.ServerName, srv.UserName).
			Unscoped().Delete(&system.TbColumn{}).Error; e != nil {
			return e
		}
		return tx.Unscoped().Delete(&srv).Error
	})
}

func (s *TbInterfaceServerService) DeleteTbInterfaceServerByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbInterfaceServer{}, "id in ?", ids).Error
	return err
}

func (s *TbInterfaceServerService) UpdateTbInterfaceServer(server *system.TbInterfaceServer) (err error) {
	err = global.GVA_DB.Updates(server).Error
	return err
}

// RenameServer renames a server node and cascades the new name to all
// tb_interface / tb_entity / tb_column rows that reference the old serverName,
// keeping the tree filter consistent after rename.
func (s *TbInterfaceServerService) RenameServer(id uint, newName string) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		var srv system.TbInterfaceServer
		if e := tx.Where("id = ?", id).First(&srv).Error; e != nil {
			return e
		}
		oldName := srv.ServerName
		if oldName == newName {
			return nil
		}
		// Update the server record itself
		if e := tx.Model(&srv).Update("server_name", newName).Error; e != nil {
			return e
		}
		// Cascade to tb_interface
		if e := tx.Model(&system.TbInterface{}).
			Where("server_name = ? AND user_name = ?", oldName, srv.UserName).
			Update("server_name", newName).Error; e != nil {
			return e
		}
		// Cascade to tb_entity
		if e := tx.Model(&system.TbEntity{}).
			Where("server_name = ? AND user_name = ?", oldName, srv.UserName).
			Update("server_name", newName).Error; e != nil {
			return e
		}
		// Cascade to tb_column
		if e := tx.Model(&system.TbColumn{}).
			Where("server_name = ? AND user_name = ?", oldName, srv.UserName).
			Update("server_name", newName).Error; e != nil {
			return e
		}
		return nil
	})
}

func (s *TbInterfaceServerService) GetTbInterfaceServer(id uint) (server system.TbInterfaceServer, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&server).Error
	return
}

func (s *TbInterfaceServerService) GetTbInterfaceServerInfoList(info systemReq.ServerSearch) (list []system.TbInterfaceServer, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbInterfaceServer{})
	if info.ProjectName != "" {
		db = db.Where("project_name LIKE ?", "%"+info.ProjectName+"%")
	}
	if info.ServerName != "" {
		db = db.Where("server_name LIKE ?", "%"+info.ServerName+"%")
	}
	if info.UserName != "" {
		db = db.Where("user_name = ?", info.UserName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}
