package system

import (
	"fmt"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupProjectGroupAutoStartTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&modelSystem.TbProjectGroup{}, &modelSystem.TbProject{}, &modelSystem.TbProjectRoute{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	oldDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = oldDB })
	return db
}

func TestResolveAutoStartTargetPrefersAggregateFullRoute(t *testing.T) {
	db := setupProjectGroupAutoStartTestDB(t)
	group := modelSystem.TbProjectGroup{GroupName: "AI数据库", UserId: 1}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}

	backend := modelSystem.TbProject{GroupId: group.ID, ProjectName: "backend", ComputerLanguage: "go"}
	aggregate := modelSystem.TbProject{GroupId: group.ID, ProjectName: "ai-compose", ComputerLanguage: "前后端 docker-compose"}
	if err := db.Create(&backend).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	routes := []modelSystem.TbProjectRoute{
		{ProjectId: int(backend.ID), RouteKey: "local_full", RouteName: "本地全量部署"},
		{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_incremental", RouteName: "前后端增量部署"},
		{ProjectId: int(aggregate.ID), RouteKey: "frontend_backend_full", RouteName: "前后端全量部署"},
	}
	if err := db.Create(&routes).Error; err != nil {
		t.Fatal(err)
	}

	target, err := ProjectGroupServiceApp.ResolveAutoStartTarget(group.ID)
	if err != nil {
		t.Fatalf("ResolveAutoStartTarget() error = %v", err)
	}
	if target.Project.ID != aggregate.ID || target.Route.RouteKey != "frontend_backend_full" {
		t.Fatalf("unexpected target: project=%s route=%s", target.Project.ProjectName, target.Route.RouteKey)
	}
}

func TestUpdateAutoStartRequiresAggregateFullRoute(t *testing.T) {
	db := setupProjectGroupAutoStartTestDB(t)
	group := modelSystem.TbProjectGroup{GroupName: "无聚合路线", UserId: 7}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	project := modelSystem.TbProject{GroupId: group.ID, ProjectName: "backend", ComputerLanguage: "go"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}

	if err := ProjectGroupServiceApp.UpdateAutoStart(group.UserId, group.ID, true); err == nil {
		t.Fatal("UpdateAutoStart() unexpectedly enabled a group without an aggregate full route")
	}

	var reloaded modelSystem.TbProjectGroup
	if err := db.First(&reloaded, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.AutoStart {
		t.Fatal("auto_start was persisted after validation failure")
	}
}
