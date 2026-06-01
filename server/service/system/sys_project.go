package system

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"gorm.io/gorm"
)

type ProjectService struct{}

var ProjectServiceApp = new(ProjectService)
var isDeployPortAvailable = defaultDeployPortAvailable

const deployPortStep = 10

type NextDeployPortResult struct {
	Type     string `json:"type"`
	BasePort int    `json:"basePort"`
	MaxPort  int    `json:"maxPort"`
	NextPort int    `json:"nextPort"`
}

// GetProjectPage 分页获取项目列表
func (s *ProjectService) GetProjectPage(req request.TbProjectSearch) (list []system.TbProject, total int64, err error) {
	db := global.GVA_DB.Model(&system.TbProject{})
	if req.ProjectName != "" {
		db = db.Where("project_name LIKE ?", "%"+req.ProjectName+"%")
	}
	if req.ComputerLanguage != "" {
		db = db.Where("computer_language = ?", req.ComputerLanguage)
	}
	if req.UserId != 0 {
		db = db.Where("user_id = ?", req.UserId)
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Scopes(req.Paginate()).Order("id desc").Preload("Routes").Find(&list).Error
	return
}

// GetProjectList 获取项目列表（不分页）
func (s *ProjectService) GetProjectList(project system.TbProject) (list []system.TbProject, err error) {
	db := global.GVA_DB.Model(&system.TbProject{})
	if project.ProjectName != "" {
		db = db.Where("project_name LIKE ?", "%"+project.ProjectName+"%")
	}
	if project.UserId != 0 {
		db = db.Where("user_id = ?", project.UserId)
	}
	err = db.Preload("Routes").Find(&list).Error
	return
}

// GetProjectById 根据ID获取项目
func (s *ProjectService) GetProjectById(id uint) (project system.TbProject, err error) {
	err = global.GVA_DB.Preload("Routes").Where("id = ?", id).First(&project).Error
	return
}

// GetNextDeployPort 根据项目访问地址计算下一次部署建议端口。
func (s *ProjectService) GetNextDeployPort(portType string) (NextDeployPortResult, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(portType))
	basePort := 0
	switch normalizedType {
	case "frontend":
		basePort = 6001
	case "backend":
		basePort = 10001
	default:
		return NextDeployPortResult{}, fmt.Errorf("不支持的端口类型: %s", portType)
	}

	var accessURLs []string
	if err := global.GVA_DB.Model(&system.TbProject{}).
		Where("access_url IS NOT NULL AND access_url <> ?", "").
		Pluck("access_url", &accessURLs).Error; err != nil {
		return NextDeployPortResult{}, err
	}

	maxPort := 0
	for _, accessURL := range accessURLs {
		port, ok := extractAccessURLPort(accessURL)
		if !ok || !portMatchesDeployType(port, normalizedType) {
			continue
		}
		if port > maxPort {
			maxPort = port
		}
	}

	nextPort := basePort
	if maxPort > 0 {
		nextPort = nextDeployPortSlot(maxPort, basePort)
	}
	nextPort, err := nextAvailableDeployPort(normalizedType, nextPort)
	if err != nil {
		return NextDeployPortResult{}, err
	}

	return NextDeployPortResult{
		Type:     normalizedType,
		BasePort: basePort,
		MaxPort:  maxPort,
		NextPort: nextPort,
	}, nil
}

func nextAvailableDeployPort(portType string, startPort int) (int, error) {
	minPort, maxPort, err := deployPortRange(portType)
	if err != nil {
		return 0, err
	}
	if startPort < minPort || startPort > maxPort {
		startPort = minPort
	}
	return startPort, nil
}

func nextDeployPortSlot(maxPort int, basePort int) int {
	if maxPort < basePort {
		return basePort
	}
	return basePort + ((maxPort-basePort)/deployPortStep+1)*deployPortStep
}

func deployPortRange(portType string) (int, int, error) {
	switch portType {
	case "frontend":
		return 6001, 6999, nil
	case "backend":
		return 10001, 10999, nil
	default:
		return 0, 0, fmt.Errorf("不支持的端口类型: %s", portType)
	}
}

func defaultDeployPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func extractAccessURLPort(accessURL string) (int, bool) {
	raw := strings.TrimSpace(accessURL)
	if raw == "" {
		return 0, false
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return 0, false
	}

	portValue := parsed.Port()
	if portValue == "" {
		return 0, false
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || port <= 0 || port > 65535 {
		return 0, false
	}
	return port, true
}

func portMatchesDeployType(port int, portType string) bool {
	switch portType {
	case "frontend":
		return port >= 6001 && port <= 6999
	case "backend":
		return port >= 10001 && port < 11000
	default:
		return false
	}
}

// SaveOrUpdateProject 新增或修改项目
func (s *ProjectService) SaveOrUpdateProject(project system.TbProject) (err error) {
	if project.ID != 0 {
		err = global.GVA_DB.Model(&system.TbProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
			"group_id":           project.GroupId,
			"computer_language":  project.ComputerLanguage,
			"project_name":       project.ProjectName,
			"description":        project.Description,
			"access_url":         project.AccessUrl,
			"local_project_path": project.LocalProjectPath,
			"user_id":            project.UserId,
		}).Error
	} else {
		err = global.GVA_DB.Create(&project).Error
	}
	return
}

// DeleteProject 批量删除项目（含关联脚本）
func (s *ProjectService) DeleteProject(ids []int) (err error) {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			// 1. 删除脚本数据库记录
			if err := tx.Where("project_id = ?", id).Unscoped().Delete(&system.TbProjectScript{}).Error; err != nil {
				return err
			}
			// 1.5 删除路由配置记录
			if err := tx.Where("project_id = ?", id).Unscoped().Delete(&system.TbProjectRoute{}).Error; err != nil {
				return err
			}
			// 2. 删除项目记录
			if err := tx.Where("id = ?", id).Unscoped().Delete(&system.TbProject{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
