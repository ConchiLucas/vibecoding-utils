package system

import (
	"errors"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

type ServerService struct{}

var ServerServiceApp = new(ServerService)

// GetServerPage 分页获取服务器列表
func (s *ServerService) GetServerPage(req request.TbServerSearch) (list []system.TbServer, total int64, err error) {
	db := global.GVA_DB.Model(&system.TbServer{})
	if req.ServerName != "" {
		db = db.Where("server_name LIKE ?", "%"+req.ServerName+"%")
	}
	if req.ServerIp != "" {
		db = db.Where("server_ip LIKE ?", "%"+req.ServerIp+"%")
	}
	if req.UserId != 0 {
		db = db.Where("user_id = ?", req.UserId)
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Scopes(req.Paginate()).Order("id desc").Find(&list).Error
	return
}

// GetServerList 获取服务器列表（不分页）
func (s *ServerService) GetServerList(server system.TbServer) (list []system.TbServer, err error) {
	db := global.GVA_DB.Model(&system.TbServer{})
	if server.ServerName != "" {
		db = db.Where("server_name LIKE ?", "%"+server.ServerName+"%")
	}
	if server.UserId != 0 {
		db = db.Where("user_id = ?", server.UserId)
	}
	err = db.Find(&list).Error
	return
}

// GetServerById 根据ID获取服务器
func (s *ServerService) GetServerById(id uint) (server system.TbServer, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&server).Error
	return
}

// SaveOrUpdateServer 新增或修改服务器
func (s *ServerService) SaveOrUpdateServer(server system.TbServer) (err error) {
	if server.ID != 0 {
		// 更新
		err = global.GVA_DB.Model(&system.TbServer{}).Where("id = ?", server.ID).Updates(map[string]interface{}{
			"server_name":           server.ServerName,
			"server_ip":             server.ServerIp,
			"server_internal_ip":    server.ServerInternalIp,
			"server_login_name":     server.ServerLoginName,
			"server_login_password": server.ServerLoginPassword,
			"server_login_port":     server.ServerLoginPort,
			"user_id":               server.UserId,
			"extend_params":         server.ExtendParams,
			"remark":                server.Remark,
		}).Error
	} else {
		// 新增
		err = global.GVA_DB.Create(&server).Error
	}
	return
}

// DeleteServer 批量删除服务器
func (s *ServerService) DeleteServer(ids []int) (err error) {
	// 检查是否有路由配置关联这些服务器
	var count int64
	global.GVA_DB.Model(&system.TbProjectRoute{}).Where("server_id IN ?", ids).Count(&count)
	if count > 0 {
		return errors.New("存在关联部署路由，无法删除")
	}
	err = global.GVA_DB.Where("id IN ?", ids).Unscoped().Delete(&system.TbServer{}).Error
	return
}

// GetServerByInfo 根据条件查询单个服务器
func (s *ServerService) GetServerByInfo(server system.TbServer) (result system.TbServer, err error) {
	db := global.GVA_DB.Model(&system.TbServer{})
	if server.ServerName != "" {
		db = db.Where("server_name = ?", server.ServerName)
	}
	if server.ServerIp != "" {
		db = db.Where("server_ip = ?", server.ServerIp)
	}
	err = db.First(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	return
}
