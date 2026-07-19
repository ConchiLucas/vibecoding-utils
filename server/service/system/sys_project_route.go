package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

type ProjectRouteService struct{}

var ProjectRouteServiceApp = new(ProjectRouteService)

func (s *ProjectRouteService) SaveOrUpdateRoute(route system.TbProjectRoute) error {
	if route.ID != 0 {
		return global.GVA_DB.Model(&system.TbProjectRoute{}).Where("id = ?", route.ID).Updates(map[string]interface{}{
			"route_name":             route.RouteName,
			"route_key":              route.RouteKey,
			"server_id":              route.ServerId,
			"local_project_path":     route.LocalProjectPath,
			"local_script_path":      route.LocalScriptPath,
			"server_project_path":    route.ServerProjectPath,
			"local_execute_command":  route.LocalExecuteCommand,
			"local_stop_command":     route.LocalStopCommand,
			"local_start_command":    route.LocalStartCommand,
			"server_execute_command": route.ServerExecuteCommand,
			"color":                  route.Color,
			"icon":                   route.Icon,
			"build_type":             route.BuildType,
			"file_name":              route.FileName,
		}).Error
	} else {
		return global.GVA_DB.Create(&route).Error
	}
}

func (s *ProjectRouteService) DeleteRoute(id uint) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("route_id = ?", id).Unscoped().Delete(&system.TbProjectScript{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Unscoped().Delete(&system.TbProjectRoute{}).Error
	})
}

func (s *ProjectRouteService) GetRouteById(id uint) (route system.TbProjectRoute, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&route).Error
	return
}
