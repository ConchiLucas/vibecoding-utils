package system

import (
	"errors"
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupGeneratePathGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	oldDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() {
		global.GVA_DB = oldDB
	})

	if err := db.AutoMigrate(
		&modelSystem.TbGenerateProjectPathGroup{},
		&modelSystem.TbGenerateProjectPath{},
		&modelSystem.TbGenerateProjectPathModel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestCopyPathSetPreservesStoredPathGroups(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	sourceGroup := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         10,
		ProjectInstanceId: 10,
		PathSet:           0,
		PathSetName:       "默认配置",
		BasePath:          "service/api/src",
		Sort:              1,
	}
	if err := db.Create(&sourceGroup).Error; err != nil {
		t.Fatalf("create source group: %v", err)
	}

	sourcePath := modelSystem.TbGenerateProjectPath{
		ProjectId:         10,
		ProjectInstanceId: 10,
		PathSet:           0,
		PathSetName:       "默认配置",
		PathGroupId:       int(sourceGroup.ID),
		FileUrl:           "service/api/src/main/java/{{module}}",
		FileName:          "{{TableName}}.java",
		Enabled:           1,
		Incremented:       0,
	}
	if err := db.Create(&sourcePath).Error; err != nil {
		t.Fatalf("create source path: %v", err)
	}
	if err := db.Create(&modelSystem.TbGenerateProjectPathModel{
		PathId:  int(sourcePath.ID),
		Content: "package {{module}};",
		Prompt:  "生成 Java 文件",
	}).Error; err != nil {
		t.Fatalf("create source model: %v", err)
	}

	nextPathSet, err := svc.CopyPathSet(systemReq.CopyGenerateProjectPathSetReq{
		ProjectId:         10,
		ProjectInstanceId: 10,
		PathSet:           0,
		PathIds:           []uint{sourcePath.ID},
		GroupIds:          []uint{sourceGroup.ID},
	})
	if err != nil {
		t.Fatalf("copy path set: %v", err)
	}
	if nextPathSet != 1 {
		t.Fatalf("nextPathSet = %d, want 1", nextPathSet)
	}

	var copiedGroup modelSystem.TbGenerateProjectPathGroup
	if err := db.Where("project_instance_id = ? AND path_set = ? AND base_path = ?", 10, nextPathSet, "service/api/src").First(&copiedGroup).Error; err != nil {
		t.Fatalf("find copied group: %v", err)
	}
	if copiedGroup.PathSetName != "默认配置" {
		t.Fatalf("copiedGroup.PathSetName = %q", copiedGroup.PathSetName)
	}

	var copiedPath modelSystem.TbGenerateProjectPath
	if err := db.Where("project_instance_id = ? AND path_set = ? AND file_name = ?", 10, nextPathSet, "{{TableName}}.java").First(&copiedPath).Error; err != nil {
		t.Fatalf("find copied path: %v", err)
	}
	if copiedPath.PathGroupId != int(copiedGroup.ID) {
		t.Fatalf("copiedPath.PathGroupId = %d, want %d", copiedPath.PathGroupId, copiedGroup.ID)
	}

	var copiedModel modelSystem.TbGenerateProjectPathModel
	if err := db.Where("path_id = ?", copiedPath.ID).First(&copiedModel).Error; err != nil {
		t.Fatalf("find copied model: %v", err)
	}
	if copiedModel.Content != "package {{module}};" || copiedModel.Prompt != "生成 Java 文件" {
		t.Fatalf("copied model = %#v", copiedModel)
	}
}

func TestDeletePathSetRemovesPathsModelsAndGroups(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	sourceGroup := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         21,
		ProjectInstanceId: 21,
		PathSet:           2,
		PathSetName:       "可删除配置",
		BasePath:          "service/api/src",
		Sort:              1,
	}
	if err := db.Create(&sourceGroup).Error; err != nil {
		t.Fatalf("create source group: %v", err)
	}

	sourcePath := modelSystem.TbGenerateProjectPath{
		ProjectId:         21,
		ProjectInstanceId: 21,
		PathSet:           2,
		PathSetName:       "可删除配置",
		PathGroupId:       int(sourceGroup.ID),
		FileUrl:           "service/api/src/main/java",
		FileName:          "{{TableName}}.java",
		Enabled:           1,
		Incremented:       0,
	}
	if err := db.Create(&sourcePath).Error; err != nil {
		t.Fatalf("create source path: %v", err)
	}
	if err := db.Create(&modelSystem.TbGenerateProjectPathModel{
		PathId:  int(sourcePath.ID),
		Content: "template content",
		Prompt:  "template prompt",
	}).Error; err != nil {
		t.Fatalf("create source model: %v", err)
	}

	deletedCount, err := svc.DeletePathSet(systemReq.DeleteGenerateProjectPathSetReq{
		ProjectId:         21,
		ProjectInstanceId: 21,
		PathSet:           2,
		GroupIds:          []uint{sourceGroup.ID},
	})
	if err != nil {
		t.Fatalf("delete path set: %v", err)
	}
	if deletedCount == 0 {
		t.Fatal("deletedCount = 0, want at least one deleted row")
	}

	var pathCount int64
	if err := db.Model(&modelSystem.TbGenerateProjectPath{}).Where("id = ?", sourcePath.ID).Count(&pathCount).Error; err != nil {
		t.Fatalf("count path: %v", err)
	}
	if pathCount != 0 {
		t.Fatalf("pathCount = %d, want 0", pathCount)
	}

	var modelCount int64
	if err := db.Model(&modelSystem.TbGenerateProjectPathModel{}).Where("path_id = ?", sourcePath.ID).Count(&modelCount).Error; err != nil {
		t.Fatalf("count model: %v", err)
	}
	if modelCount != 0 {
		t.Fatalf("modelCount = %d, want 0", modelCount)
	}

	var groupCount int64
	if err := db.Model(&modelSystem.TbGenerateProjectPathGroup{}).Where("id = ?", sourceGroup.ID).Count(&groupCount).Error; err != nil {
		t.Fatalf("count group: %v", err)
	}
	if groupCount != 0 {
		t.Fatalf("groupCount = %d, want 0", groupCount)
	}
}

func TestUpdatePathGroupMovesChildPathPrefixes(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	group := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         11,
		ProjectInstanceId: 11,
		PathSet:           0,
		BasePath:          "service/api/src",
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	pathObj := modelSystem.TbGenerateProjectPath{
		ProjectId:         11,
		ProjectInstanceId: 11,
		PathSet:           0,
		PathGroupId:       int(group.ID),
		FileUrl:           "service/api/src/main/java",
		FileName:          "Demo.java",
		Enabled:           1,
	}
	if err := db.Create(&pathObj).Error; err != nil {
		t.Fatalf("create path: %v", err)
	}

	group.BasePath = "service/basic-api/src"
	if err := svc.UpdatePathGroup(&group); err != nil {
		t.Fatalf("update group: %v", err)
	}

	var updatedPath modelSystem.TbGenerateProjectPath
	if err := db.First(&updatedPath, pathObj.ID).Error; err != nil {
		t.Fatalf("find path: %v", err)
	}
	if updatedPath.FileUrl != "service/basic-api/src/main/java" {
		t.Fatalf("updatedPath.FileUrl = %q", updatedPath.FileUrl)
	}
}

func TestGetPathGroupListRepairsLegacyServiceSrcBasePath(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	pathObj := modelSystem.TbGenerateProjectPath{
		ProjectId:         13,
		ProjectInstanceId: 13,
		PathSet:           0,
		FileUrl:           "c12-mtp-basic-service/c12-mtp-basic-api/src/main/java/com/chinaservices/dsly/basic/module/btStation/domain",
		FileName:          "{{TableName}}.java",
		Enabled:           1,
	}
	if err := db.Create(&pathObj).Error; err != nil {
		t.Fatalf("create legacy path: %v", err)
	}

	groups, err := svc.GetPathGroupList(0, 13)
	if err != nil {
		t.Fatalf("get path groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].BasePath != "c12-mtp-basic-service/c12-mtp-basic-api/src" {
		t.Fatalf("group basePath = %q", groups[0].BasePath)
	}

	var updatedPath modelSystem.TbGenerateProjectPath
	if err := db.First(&updatedPath, pathObj.ID).Error; err != nil {
		t.Fatalf("find updated path: %v", err)
	}
	if updatedPath.PathGroupId != int(groups[0].ID) {
		t.Fatalf("updatedPath.PathGroupId = %d, want %d", updatedPath.PathGroupId, groups[0].ID)
	}
}

func TestGetPathGroupListRepairsLegacySingleServiceSrcBasePath(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	pathObj := modelSystem.TbGenerateProjectPath{
		ProjectId:         14,
		ProjectInstanceId: 14,
		PathSet:           0,
		FileUrl:           "c12-mtp-web-service/src/main/java/com/chinaservices/mtp/web/module/btwaybill/controller",
		FileName:          "{{TableName}}Controller.java",
		Enabled:           1,
	}
	if err := db.Create(&pathObj).Error; err != nil {
		t.Fatalf("create legacy path: %v", err)
	}

	groups, err := svc.GetPathGroupList(0, 14)
	if err != nil {
		t.Fatalf("get path groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].BasePath != "c12-mtp-web-service/src/main/java/com/chinaservices/mtp/web/module/btwaybill" {
		t.Fatalf("group basePath = %q", groups[0].BasePath)
	}

	var updatedPath modelSystem.TbGenerateProjectPath
	if err := db.First(&updatedPath, pathObj.ID).Error; err != nil {
		t.Fatalf("find updated path: %v", err)
	}
	if updatedPath.PathGroupId != int(groups[0].ID) {
		t.Fatalf("updatedPath.PathGroupId = %d, want %d", updatedPath.PathGroupId, groups[0].ID)
	}
}

func TestGetPathGroupListRepairsLegacySingleServiceSqlExtBasePath(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	pathObj := modelSystem.TbGenerateProjectPath{
		ProjectId:         15,
		ProjectInstanceId: 15,
		PathSet:           0,
		FileUrl:           "c12-mtp-web-service/src/main/resources/sql-ext/bttransport",
		FileName:          "btWaybill_query_getPageList.sql",
		Enabled:           1,
	}
	if err := db.Create(&pathObj).Error; err != nil {
		t.Fatalf("create legacy path: %v", err)
	}

	groups, err := svc.GetPathGroupList(0, 15)
	if err != nil {
		t.Fatalf("get path groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].BasePath != "c12-mtp-web-service/src/main/resources/sql-ext/bttransport" {
		t.Fatalf("group basePath = %q", groups[0].BasePath)
	}

	var updatedPath modelSystem.TbGenerateProjectPath
	if err := db.First(&updatedPath, pathObj.ID).Error; err != nil {
		t.Fatalf("find updated path: %v", err)
	}
	if updatedPath.PathGroupId != int(groups[0].ID) {
		t.Fatalf("updatedPath.PathGroupId = %d, want %d", updatedPath.PathGroupId, groups[0].ID)
	}
}

func TestDeletePathGroupRejectsExistingPathData(t *testing.T) {
	db := setupGeneratePathGroupTestDB(t)
	svc := TbGenerateProjectPathService{}

	group := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         12,
		ProjectInstanceId: 12,
		PathSet:           0,
		BasePath:          "service/web/src",
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	pathObj := modelSystem.TbGenerateProjectPath{
		ProjectId:         12,
		ProjectInstanceId: 12,
		PathSet:           0,
		PathGroupId:       int(group.ID),
		FileUrl:           "service/web/src/views",
		FileName:          "index.tsx",
		Enabled:           1,
	}
	if err := db.Create(&pathObj).Error; err != nil {
		t.Fatalf("create path: %v", err)
	}

	if err := svc.DeletePathGroup(group); err == nil {
		t.Fatal("DeletePathGroup error = nil, want existing data error")
	}

	if err := db.Delete(&pathObj).Error; err != nil {
		t.Fatalf("delete path: %v", err)
	}
	if err := svc.DeletePathGroup(group); err != nil {
		t.Fatalf("delete empty group: %v", err)
	}
	var deletedGroup modelSystem.TbGenerateProjectPathGroup
	err := db.First(&deletedGroup, group.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted group lookup error = %v, want ErrRecordNotFound", err)
	}
}
