package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
)

type TbInterfaceProjectService struct{}

func (s *TbInterfaceProjectService) CreateTbInterfaceProject(project system.TbInterfaceProject) (err error) {
	err = global.GVA_DB.Create(&project).Error
	return err
}

func (s *TbInterfaceProjectService) DeleteTbInterfaceProject(project system.TbInterfaceProject) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&project).Error
	return err
}

func (s *TbInterfaceProjectService) GetTbInterfaceProjectList() (list []system.TbInterfaceProject, err error) {
	err = global.GVA_DB.Order("id asc").Find(&list).Error
	return list, err
}

func (s *TbInterfaceProjectService) UpdateTbInterfaceProject(project system.TbInterfaceProject) (err error) {
	err = global.GVA_DB.Model(&project).Select("project_name", "project_desc").Updates(project).Error
	return err
}
