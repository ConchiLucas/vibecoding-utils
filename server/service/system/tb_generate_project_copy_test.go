package system

import (
	"strings"
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupGenerateProjectCopyTestDB(t *testing.T) *gorm.DB {
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
		&modelSystem.TbGenerateProject{},
		&modelSystem.TbGenerateProjectInstance{},
		&modelSystem.TbGenerateProjectPathGroup{},
		&modelSystem.TbGenerateProjectPath{},
		&modelSystem.TbGenerateProjectPathModel{},
		&modelSystem.TbGenerateDbTemplateType{},
		&modelSystem.TbGenerateDbTemplateScript{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestCopyProjectCopiesCardInstancesPathsModelsAndDbTemplates(t *testing.T) {
	db := setupGenerateProjectCopyTestDB(t)
	svc := TbGenerateProjectService{}

	sourceProject := modelSystem.TbGenerateProject{
		ProjectConfigId: 7,
		BusinessType:    "源业务",
		ProjectType:     "backend",
		ProjectName:     "源卡片",
		DiskPath:        "/tmp/source",
		Remark:          "源描述",
		UserName:        "tester",
	}
	if err := db.Create(&sourceProject).Error; err != nil {
		t.Fatalf("create source project: %v", err)
	}

	templateGroup := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         int(sourceProject.ID),
		ProjectInstanceId: 0,
		PathSet:           0,
		PathSetName:       "模板配置",
		BasePath:          "template/service/src",
		Sort:              1,
	}
	if err := db.Create(&templateGroup).Error; err != nil {
		t.Fatalf("create template group: %v", err)
	}
	templatePath := modelSystem.TbGenerateProjectPath{
		ProjectId:           int(sourceProject.ID),
		ProjectInstanceId:   0,
		PathSet:             0,
		PathSetName:         "模板配置",
		PathGroupId:         int(templateGroup.ID),
		FileUrl:             "template/service/src/{{module}}",
		FileName:            "{{TableName}}Item.java",
		DynamicPlaceholders: `[{"key":"tenantCode","description":"租户编码","value":"pzh"}]`,
		Enabled:             1,
		Incremented:         1,
	}
	if err := db.Create(&templatePath).Error; err != nil {
		t.Fatalf("create template path: %v", err)
	}
	if err := db.Create(&modelSystem.TbGenerateProjectPathModel{
		PathId:  int(templatePath.ID),
		Content: "template content {{module}}",
		Prompt:  "template prompt",
	}).Error; err != nil {
		t.Fatalf("create template model: %v", err)
	}

	sourceInstance := modelSystem.TbGenerateProjectInstance{
		TemplateProjectId:       int(sourceProject.ID),
		ProjectName:             "源项目实例",
		DiskPath:                "/tmp/source-instance",
		Remark:                  "实例描述",
		UserName:                "tester",
		SelectedPathSetIdentity: "path-set-2",
	}
	if err := db.Create(&sourceInstance).Error; err != nil {
		t.Fatalf("create source instance: %v", err)
	}
	if err := db.Model(&sourceProject).Update("selected_project_instance_id", int(sourceInstance.ID)).Error; err != nil {
		t.Fatalf("update selected instance: %v", err)
	}

	instanceGroup := modelSystem.TbGenerateProjectPathGroup{
		ProjectId:         int(sourceInstance.ID),
		ProjectInstanceId: int(sourceInstance.ID),
		PathSet:           2,
		PathSetName:       "实例配置",
		BasePath:          "instance/service/src",
		Sort:              3,
	}
	if err := db.Create(&instanceGroup).Error; err != nil {
		t.Fatalf("create instance group: %v", err)
	}
	instancePath := modelSystem.TbGenerateProjectPath{
		ProjectId:           int(sourceInstance.ID),
		ProjectInstanceId:   int(sourceInstance.ID),
		PathSet:             2,
		PathSetName:         "实例配置",
		PathGroupId:         int(instanceGroup.ID),
		FileUrl:             "instance/service/src/{{module}}",
		FileName:            "{{TableName}}Controller.java",
		DynamicPlaceholders: `[{"key":"${menuCode}","description":"菜单编码","value":"btWaybill"}]`,
		Enabled:             1,
		Incremented:         0,
	}
	if err := db.Create(&instancePath).Error; err != nil {
		t.Fatalf("create instance path: %v", err)
	}
	if err := db.Create(&modelSystem.TbGenerateProjectPathModel{
		PathId:  int(instancePath.ID),
		Content: "instance content {{TableName}}",
		Prompt:  "instance prompt",
	}).Error; err != nil {
		t.Fatalf("create instance model: %v", err)
	}

	templateType := modelSystem.TbGenerateDbTemplateType{
		ProjectId:           int(sourceProject.ID),
		TypeName:            "建表",
		Prompt:              "SQL 提示词",
		DynamicPlaceholders: `[{"key":"companyId","description":"公司 ID","value":"-1"}]`,
		Sort:                5,
	}
	if err := db.Create(&templateType).Error; err != nil {
		t.Fatalf("create db template type: %v", err)
	}
	if err := db.Create(&modelSystem.TbGenerateDbTemplateScript{
		ProjectId:  int(sourceProject.ID),
		TypeId:     int(templateType.ID),
		ScriptName: "station.sql",
		ScriptKind: "ddl",
		Content:    "CREATE TABLE station(id bigint);",
		Sort:       6,
	}).Error; err != nil {
		t.Fatalf("create db template script: %v", err)
	}

	copiedProject, err := svc.CopyProject(systemReq.CopyGenerateProjectReq{
		SourceProjectId: int(sourceProject.ID),
		ProjectName:     "新卡片",
		BusinessType:    "新业务",
		ProjectType:     "frontend",
		Remark:          "新描述",
		UserName:        "copy-user",
	})
	if err != nil {
		t.Fatalf("copy project: %v", err)
	}
	if copiedProject.ID == sourceProject.ID {
		t.Fatalf("copied project ID = source ID %d", sourceProject.ID)
	}
	if copiedProject.ProjectName != "新卡片" || copiedProject.BusinessType != "新业务" || copiedProject.ProjectType != "frontend" || copiedProject.Remark != "新描述" {
		t.Fatalf("copied project fields = %#v", copiedProject)
	}

	var copiedTemplatePath modelSystem.TbGenerateProjectPath
	if err := db.Where("project_id = ? AND project_instance_id = 0 AND file_name = ?", copiedProject.ID, "{{TableName}}Item.java").First(&copiedTemplatePath).Error; err != nil {
		t.Fatalf("find copied template path: %v", err)
	}
	var copiedTemplateGroup modelSystem.TbGenerateProjectPathGroup
	if err := db.First(&copiedTemplateGroup, copiedTemplatePath.PathGroupId).Error; err != nil {
		t.Fatalf("find copied template group: %v", err)
	}
	if copiedTemplateGroup.ProjectId != int(copiedProject.ID) || copiedTemplateGroup.BasePath != "template/service/src" {
		t.Fatalf("copied template group = %#v", copiedTemplateGroup)
	}
	if copiedTemplatePath.DynamicPlaceholders != templatePath.DynamicPlaceholders {
		t.Fatalf("copied template path dynamic placeholders = %q, want %q", copiedTemplatePath.DynamicPlaceholders, templatePath.DynamicPlaceholders)
	}
	var copiedTemplateModel modelSystem.TbGenerateProjectPathModel
	if err := db.Where("path_id = ?", copiedTemplatePath.ID).First(&copiedTemplateModel).Error; err != nil {
		t.Fatalf("find copied template model: %v", err)
	}
	if copiedTemplateModel.Content != "template content {{module}}" || copiedTemplateModel.Prompt != "template prompt" {
		t.Fatalf("copied template model = %#v", copiedTemplateModel)
	}

	var copiedInstances []modelSystem.TbGenerateProjectInstance
	if err := db.Where("template_project_id = ?", copiedProject.ID).Find(&copiedInstances).Error; err != nil {
		t.Fatalf("find copied instances: %v", err)
	}
	if len(copiedInstances) != 1 {
		t.Fatalf("len(copiedInstances) = %d, want 1", len(copiedInstances))
	}
	copiedInstance := copiedInstances[0]
	if copiedInstance.ProjectName != "源项目实例" || copiedInstance.SelectedPathSetIdentity != "path-set-2" {
		t.Fatalf("copied instance = %#v", copiedInstance)
	}
	if copiedProject.SelectedProjectInstanceId != int(copiedInstance.ID) {
		t.Fatalf("copied selected instance = %d, want %d", copiedProject.SelectedProjectInstanceId, copiedInstance.ID)
	}

	var copiedInstancePath modelSystem.TbGenerateProjectPath
	if err := db.Where("project_instance_id = ? AND file_name = ?", copiedInstance.ID, "{{TableName}}Controller.java").First(&copiedInstancePath).Error; err != nil {
		t.Fatalf("find copied instance path: %v", err)
	}
	var copiedInstanceGroup modelSystem.TbGenerateProjectPathGroup
	if err := db.First(&copiedInstanceGroup, copiedInstancePath.PathGroupId).Error; err != nil {
		t.Fatalf("find copied instance group: %v", err)
	}
	if copiedInstanceGroup.ProjectId != int(copiedInstance.ID) || copiedInstanceGroup.ProjectInstanceId != int(copiedInstance.ID) || copiedInstanceGroup.BasePath != "instance/service/src" {
		t.Fatalf("copied instance group = %#v", copiedInstanceGroup)
	}
	if copiedInstancePath.DynamicPlaceholders != instancePath.DynamicPlaceholders {
		t.Fatalf("copied instance path dynamic placeholders = %q, want %q", copiedInstancePath.DynamicPlaceholders, instancePath.DynamicPlaceholders)
	}
	var copiedInstanceModel modelSystem.TbGenerateProjectPathModel
	if err := db.Where("path_id = ?", copiedInstancePath.ID).First(&copiedInstanceModel).Error; err != nil {
		t.Fatalf("find copied instance model: %v", err)
	}
	if copiedInstanceModel.Content != "instance content {{TableName}}" || copiedInstanceModel.Prompt != "instance prompt" {
		t.Fatalf("copied instance model = %#v", copiedInstanceModel)
	}

	var copiedTemplateType modelSystem.TbGenerateDbTemplateType
	if err := db.Where("project_id = ? AND type_name = ?", copiedProject.ID, "建表").First(&copiedTemplateType).Error; err != nil {
		t.Fatalf("find copied db template type: %v", err)
	}
	if copiedTemplateType.Prompt != "SQL 提示词" {
		t.Fatalf("copied db template prompt = %q", copiedTemplateType.Prompt)
	}
	if copiedTemplateType.DynamicPlaceholders != `[{"key":"companyId","description":"公司 ID","value":"-1"}]` {
		t.Fatalf("copied db template placeholders = %q", copiedTemplateType.DynamicPlaceholders)
	}
	var copiedScript modelSystem.TbGenerateDbTemplateScript
	if err := db.Where("project_id = ? AND type_id = ?", copiedProject.ID, copiedTemplateType.ID).First(&copiedScript).Error; err != nil {
		t.Fatalf("find copied db template script: %v", err)
	}
	if copiedScript.Content != "CREATE TABLE station(id bigint);" {
		t.Fatalf("copied script = %#v", copiedScript)
	}
}
