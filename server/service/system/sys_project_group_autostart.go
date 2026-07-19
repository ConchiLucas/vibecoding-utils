package system

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"go.uber.org/zap"
)

const (
	dockerReadyRetryCount    = 30
	dockerReadyRetryInterval = 5 * time.Second
)

type ProjectGroupAutoStartTarget struct {
	Project modelSystem.TbProject
	Route   modelSystem.TbProjectRoute
}

// ResolveAutoStartTarget 查找项目组内唯一的聚合全量部署入口。
func (s *ProjectGroupService) ResolveAutoStartTarget(groupId uint) (ProjectGroupAutoStartTarget, error) {
	var projects []modelSystem.TbProject
	if err := global.GVA_DB.Where("group_id = ?", groupId).Preload("Routes").Order("id asc").Find(&projects).Error; err != nil {
		return ProjectGroupAutoStartTarget{}, err
	}

	var bestTarget ProjectGroupAutoStartTarget
	bestScore := -1
	for _, project := range projects {
		projectName := strings.ToLower(strings.TrimSpace(project.ProjectName))
		language := strings.ToLower(strings.TrimSpace(project.ComputerLanguage))
		projectScore := 0
		if strings.Contains(language, "docker-compose") || strings.Contains(language, "docker compose") {
			projectScore = 100
		} else if strings.Contains(projectName, "compose") {
			projectScore = 80
		} else {
			continue
		}

		for _, route := range project.Routes {
			routeKey := strings.ToLower(strings.TrimSpace(route.RouteKey))
			routeName := strings.TrimSpace(route.RouteName)
			if route.ServerId != 0 || strings.Contains(routeKey, "incremental") || strings.Contains(routeName, "增量") {
				continue
			}

			routeScore := 0
			switch {
			case strings.Contains(routeKey, "frontend_backend_full"):
				routeScore = 50
			case strings.Contains(routeName, "全部项目全量"):
				routeScore = 45
			case strings.Contains(routeName, "前后端全量"):
				routeScore = 40
			case strings.Contains(routeKey, "full") || strings.Contains(routeName, "全量"):
				routeScore = 30
			default:
				continue
			}

			if score := projectScore + routeScore; score > bestScore {
				bestScore = score
				bestTarget = ProjectGroupAutoStartTarget{Project: project, Route: route}
			}
		}
	}

	if bestScore < 0 {
		return ProjectGroupAutoStartTarget{}, fmt.Errorf("该项目组缺少聚合项目的全量部署路线，无法开启启动联动")
	}
	return bestTarget, nil
}

// StartEnabledGroupsOnStartup 在每次 VibeDeploy 后端启动时执行一次已启用项目组。
func (s *ProjectGroupService) StartEnabledGroupsOnStartup() {
	go s.startEnabledGroupsOnStartup()
}

func (s *ProjectGroupService) startEnabledGroupsOnStartup() {
	var groups []modelSystem.TbProjectGroup
	if err := global.GVA_DB.Where("auto_start = ?", true).Order("id asc").Find(&groups).Error; err != nil {
		global.GVA_LOG.Error("读取随 VibeDeploy 启动的项目组失败", zap.Error(err))
		return
	}
	if len(groups) == 0 {
		return
	}

	dockerReady := false
	for _, group := range groups {
		target, err := s.ResolveAutoStartTarget(group.ID)
		if err != nil {
			global.GVA_LOG.Error("解析项目组启动联动路线失败", zap.String("group", group.GroupName), zap.Error(err))
			continue
		}
		if projectAccessURLRunning(target.Project.AccessUrl) {
			global.GVA_LOG.Info("项目组启动入口已运行，跳过启动联动", zap.String("group", group.GroupName), zap.String("project", target.Project.ProjectName), zap.String("access_url", target.Project.AccessUrl))
			continue
		}
		if !dockerReady {
			if err := waitForDockerReady(); err != nil {
				global.GVA_LOG.Error("Docker 未就绪，项目组启动联动已跳过", zap.Error(err))
				return
			}
			dockerReady = true
		}
		global.GVA_LOG.Info("开始执行项目组启动联动", zap.String("group", group.GroupName), zap.String("project", target.Project.ProjectName), zap.String("route", target.Route.RouteName))
		if err := DeployServiceApp.ProcessDeploy(target.Project.ID, strconv.FormatUint(uint64(target.Route.ID), 10)); err != nil {
			global.GVA_LOG.Error("项目组启动联动失败", zap.String("group", group.GroupName), zap.Error(err))
			continue
		}
		global.GVA_LOG.Info("项目组启动联动完成", zap.String("group", group.GroupName))
	}
}

func waitForDockerReady() error {
	for attempt := 1; attempt <= dockerReadyRetryCount; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := exec.CommandContext(ctx, "docker", "info").Run()
		cancel()
		if err == nil {
			return nil
		}
		if attempt < dockerReadyRetryCount {
			time.Sleep(dockerReadyRetryInterval)
		}
	}
	return fmt.Errorf("等待 Docker 就绪超时")
}
