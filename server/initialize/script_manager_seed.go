package initialize

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sysModel "github.com/flipped-aurora/easy-deploy/server/model/system"
	"gorm.io/gorm"
)

const (
	defaultScriptCategoryName            = "数据库运维"
	postgresScriptCategoryName           = "postgresql"
	clickHouseScriptCategoryName         = "clickHouse"
	mysqlScriptCategoryName              = "mysql"
	defaultScriptWorkflowName            = "PostgreSQL 数据库导出到目标服务器"
	legacyDefaultScriptWorkflowName      = "easy-deploy 数据库导出到目标服务器"
	defaultTableScriptWorkflowName       = "PostgreSQL 单表导出到目标服务器"
	reverseScriptWorkflowName            = "PostgreSQL 数据库从目标服务器导出到本地"
	reverseTableScriptWorkflowName       = "PostgreSQL 单表从目标服务器导出到本地"
	defaultMySQLScriptWorkflowName       = "MySQL 数据库导出到目标服务器"
	defaultMySQLTableWorkflowName        = "MySQL 单表导出到目标服务器"
	reverseMySQLScriptWorkflowName       = "MySQL 数据库从目标服务器导出到本地"
	reverseMySQLTableWorkflowName        = "MySQL 单表从目标服务器导出到本地"
	defaultClickHouseScriptWorkflowName  = "ClickHouse 数据库从 mac mini 导出到本机服务器"
	defaultClickHouseTableWorkflowName   = "ClickHouse 单表从 mac mini 导出到本机服务器"
	reverseClickHouseScriptWorkflowName  = "ClickHouse 数据库从本机导出到 mac mini 服务器"
	reverseClickHouseTableWorkflowName   = "ClickHouse 单表从本机导出到 mac mini 服务器"
	defaultExportStepName                = "导出本地 PostgreSQL 数据库"
	reverseExportStepName                = "导出目标服务器 PostgreSQL 数据库"
	legacyReverseUploadStepName          = "通过 Tailscale 上传 PostgreSQL 导出文件到本地"
	defaultClickHouseExportStepName      = "导出 mac mini ClickHouse 数据库"
	reverseClickHouseExportStepName      = "导出本机 ClickHouse 数据库"
	legacyDefaultExportStepName          = "导出 easy-deploy 本地数据库"
	defaultTableExportStepName           = "导出本地 PostgreSQL 单表"
	reverseTableExportStepName           = "导出目标服务器 PostgreSQL 单表"
	defaultMySQLExportStepName           = "导出本地 MySQL 数据库"
	reverseMySQLExportStepName           = "导出目标服务器 MySQL 数据库"
	defaultMySQLTableExportStepName      = "导出本地 MySQL 单表"
	reverseMySQLTableExportStepName      = "导出目标服务器 MySQL 单表"
	defaultClickHouseTableExportStepName = "导出 mac mini ClickHouse 单表"
	reverseClickHouseTableExportStepName = "导出本机 ClickHouse 单表"
	defaultServerResourceCategoryName    = "服务器配置"
	defaultDatabaseResourceCategoryName  = "数据库配置"
	oracleDatabaseResourceCategoryName   = "Oracle 数据库配置"
	pgsqlDatabaseResourceCategoryName    = "PostgreSQL 数据库配置"
	clickHouseResourceCategoryName       = "ClickHouse 数据库配置"
	mysqlDatabaseResourceCategoryName    = "MySQL 数据库配置"
	defaultDeployResourceCategoryName    = "部署脚本配置"
	defaultDynamicParamsCategoryName     = "数据库导出执行参数"
	defaultTableDynamicParamsConfigName  = "PostgreSQL 单表导出执行参数"
	defaultPathConstantsCategoryName     = "数据库迁移路径常量"
	legacyPathConstantsCategoryName      = "数据库导出路径常量"
	defaultPathConstantsConfigName       = "数据库迁移路径"
	legacyPathConstantsConfigName        = "PostgreSQL 数据库导出路径"
	defaultDeployResourceConfigName      = "PostgreSQL 数据库导出流程"
	legacyDeployResourceConfigName       = "easy-deploy 数据库导出流程"
	defaultMacBookProDatabaseConfigName  = "macbookPro"
	defaultMacMiniDatabaseConfigName     = "macMini"
	defaultMacMiniDatabaseHost           = "192.168.0.141"
	defaultTencentServerConfigName       = "tencentServer"
	defaultTencentServerHost             = "1.15.62.252"
)

type databaseResourceCategorySpec struct {
	DBType       string
	CategoryName string
}

var databaseResourceCategorySpecs = []databaseResourceCategorySpec{
	{DBType: "oracle", CategoryName: oracleDatabaseResourceCategoryName},
	{DBType: "postgresql", CategoryName: pgsqlDatabaseResourceCategoryName},
	{DBType: "clickhouse", CategoryName: clickHouseResourceCategoryName},
	{DBType: "mysql", CategoryName: mysqlDatabaseResourceCategoryName},
}

var pathConstantResourceRowNames = []string{
	"EXPORT_ROOT",
	"LOCAL_ENV",
	"TARGET_MANIFEST",
	"REMOTE_INBOX",
	"REMOTE_WORKDIR",
	"REMOTE_MANIFEST",
	"RESTORE_ENV",
}

func seedDefaultScriptManagerData(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&sysModel.TbUser{}) {
		return nil
	}

	var users []sysModel.TbUser
	if err := db.Select("id").Order("id").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if user.ID == 0 {
			continue
		}
		if err := seedScriptResourceConfigs(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultPostgresTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReversePostgresDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReversePostgresTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultMySQLDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultMySQLTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReverseMySQLDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReverseMySQLTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultClickHouseDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedDefaultClickHouseTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReverseClickHouseDatabaseExportWorkflow(db, user.ID); err != nil {
			return err
		}
		if err := seedReverseClickHouseTableExportWorkflow(db, user.ID); err != nil {
			return err
		}
	}
	return nil
}

func defaultScriptWorkflowExists(db *gorm.DB) bool {
	var count int64
	var workflow sysModel.TbScriptWorkflow
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("workflow_name IN ?", []string{defaultScriptWorkflowName, legacyDefaultScriptWorkflowName}).
		First(&workflow).Error; err != nil {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, defaultTableScriptWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, reverseScriptWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, reverseTableScriptWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	for _, workflowName := range []string{
		defaultMySQLScriptWorkflowName,
		defaultMySQLTableWorkflowName,
		reverseMySQLScriptWorkflowName,
		reverseMySQLTableWorkflowName,
	} {
		if err := db.Model(&sysModel.TbScriptWorkflow{}).
			Where("user_id = ? AND workflow_name = ?", workflow.UserId, workflowName).
			Count(&count).Error; err != nil || count == 0 {
			return false
		}
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, defaultClickHouseScriptWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, defaultClickHouseTableWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, reverseClickHouseScriptWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if err := db.Model(&sysModel.TbScriptWorkflow{}).
		Where("user_id = ? AND workflow_name = ?", workflow.UserId, reverseClickHouseTableWorkflowName).
		Count(&count).Error; err != nil || count == 0 {
		return false
	}
	if !db.Migrator().HasTable(&sysModel.TbScriptResourceCategory{}) {
		return false
	}
	if err := db.Model(&sysModel.TbScriptResourceCategory{}).
		Where("user_id = ? AND category_name IN ?", workflow.UserId, []string{
			defaultServerResourceCategoryName,
			oracleDatabaseResourceCategoryName,
			pgsqlDatabaseResourceCategoryName,
			clickHouseResourceCategoryName,
			mysqlDatabaseResourceCategoryName,
			defaultDynamicParamsCategoryName,
		}).
		Count(&count).Error; err != nil {
		return false
	}
	if count < 6 {
		return false
	}
	var dynamicCategory sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", workflow.UserId, defaultDynamicParamsCategoryName).First(&dynamicCategory).Error; err != nil {
		return false
	}
	if err := db.Model(&sysModel.TbScriptResourceConfig{}).
		Where("user_id = ? AND category_id = ? AND workflow_id = ?", workflow.UserId, dynamicCategory.ID, workflow.ID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func seedDefaultDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensurePostgresScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name IN ?", userID, []string{defaultScriptWorkflowName, legacyDefaultScriptWorkflowName}).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: defaultScriptWorkflowName,
			Description:  "通用 PostgreSQL 迁移流程：本地导出指定数据库，通过 Tailscale 传输文件，目标机校验后清空并导入到指定数据库。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if workflow.WorkflowName == legacyDefaultScriptWorkflowName {
		if err := db.Model(&workflow).Updates(map[string]interface{}{
			"workflow_name": defaultScriptWorkflowName,
			"description":   "通用 PostgreSQL 迁移流程：本地导出指定数据库，通过 Tailscale 传输文件，目标机校验后清空并导入到指定数据库。",
			"category_id":   category.ID,
		}).Error; err != nil {
			return err
		}
		workflow.WorkflowName = defaultScriptWorkflowName
		workflow.CategoryId = category.ID
	} else if workflow.CategoryId != category.ID {
		if err := db.Model(&workflow).Update("category_id", category.ID).Error; err != nil {
			return err
		}
		workflow.CategoryId = category.ID
	}
	_ = db.Model(&sysModel.TbScriptStep{}).
		Where("workflow_id = ? AND step_name = ?", workflow.ID, legacyDefaultExportStepName).
		Update("step_name", defaultExportStepName).Error

	serverID := firstServerIDForUser(db, userID)
	connectionID := firstConnectionID(db)
	serverResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultTencentServerConfigName,
		defaultMacMiniDatabaseConfigName,
		defaultMacBookProDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultSourceDatabasePlaceholderKey(),
	)
	if sourceDatabaseResourceID == 0 {
		sourceDatabaseResourceID = firstDatabaseResourceConfigID(db, userID, "postgresql")
	}
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultTargetDatabasePlaceholderKey(),
		defaultSourceDatabasePlaceholderKey(),
	)
	if targetDatabaseResourceID == 0 {
		targetDatabaseResourceID = sourceDatabaseResourceID
	}
	pathConstantsCategory, err := ensurePathConstantsResourceCategory(db, userID)
	if err != nil {
		return err
	}
	if err := seedPathConstantsResourceConfig(db, userID, pathConstantsCategory.ID); err != nil {
		return err
	}
	pathConstantsResourceID := firstResourceConfigID(db, userID, defaultPathConstantsCategoryName)
	projectNameRow := deployProjectNameSeedRow(db, userID)
	dynamicParamsCategory, err := ensureScriptResourceCategory(db, userID, defaultDynamicParamsCategoryName, "dynamic")
	if err != nil {
		return err
	}
	if err := seedDynamicParamsResourceConfigs(db, userID, dynamicParamsCategory.ID, workflow.ID, projectNameRow); err != nil {
		return err
	}
	dynamicParamsCategoryID := dynamicParamsCategory.ID
	dynamicParamsResourceID := firstResourceConfigIDForWorkflow(db, userID, defaultDynamicParamsCategoryName, workflow.ID)
	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      defaultExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultEasyDeployExportScript,
			Placeholders:  defaultExportPlaceholders(sourceDatabaseResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传导出文件",
			StepType:      "local_upload",
			ScriptContent: defaultEasyDeployUploadScript,
			Placeholders:  defaultUploadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, serverID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "目标服务器整理并校验文件",
			StepType:      "target_download",
			ScriptContent: defaultEasyDeployTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, serverID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "目标服务器导入数据库",
			StepType:      "target_exec",
			ScriptContent: defaultEasyDeployTargetExecScript,
			Placeholders:  defaultTargetExecPlaceholders(targetDatabaseResourceID, serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, serverID),
		},
	}

	for _, step := range steps {
		var existing sysModel.TbScriptStep
		err := db.Where("workflow_id = ? AND step_name = ?", workflow.ID, step.StepName).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&step).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := syncDefaultScriptStep(db, existing, step); err != nil {
			return err
		}
	}
	return nil
}

func seedDefaultPostgresTableExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensurePostgresScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, defaultTableScriptWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: defaultTableScriptWorkflowName,
			Description:  "通用 PostgreSQL 单表迁移流程：本地导出指定表，通过 Tailscale 传输文件，目标机校验后导入同名表。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if workflow.CategoryId != category.ID {
		if err := db.Model(&workflow).Update("category_id", category.ID).Error; err != nil {
			return err
		}
		workflow.CategoryId = category.ID
	}

	serverID := firstServerIDForUser(db, userID)
	connectionID := firstConnectionID(db)
	serverResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultTencentServerConfigName,
		defaultMacMiniDatabaseConfigName,
		defaultMacBookProDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultSourceDatabasePlaceholderKey(),
	)
	if sourceDatabaseResourceID == 0 {
		sourceDatabaseResourceID = firstDatabaseResourceConfigID(db, userID, "postgresql")
	}
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultTargetDatabasePlaceholderKey(),
		defaultSourceDatabasePlaceholderKey(),
	)
	if targetDatabaseResourceID == 0 {
		targetDatabaseResourceID = sourceDatabaseResourceID
	}
	pathConstantsCategory, err := ensurePathConstantsResourceCategory(db, userID)
	if err != nil {
		return err
	}
	if err := seedPathConstantsResourceConfig(db, userID, pathConstantsCategory.ID); err != nil {
		return err
	}
	pathConstantsResourceID := firstResourceConfigID(db, userID, defaultPathConstantsCategoryName)
	dynamicParamsCategory, err := ensureScriptResourceCategory(db, userID, defaultDynamicParamsCategoryName, "dynamic")
	if err != nil {
		return err
	}
	if err := seedTableDynamicParamsResourceConfigs(db, userID, dynamicParamsCategory.ID, workflow.ID); err != nil {
		return err
	}
	dynamicParamsCategoryID := dynamicParamsCategory.ID
	dynamicParamsResourceID := firstResourceConfigIDForWorkflow(db, userID, defaultDynamicParamsCategoryName, workflow.ID)

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      defaultTableExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultPostgresTableExportScript,
			Placeholders:  defaultTableExportPlaceholders(sourceDatabaseResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传单表导出文件",
			StepType:      "local_upload",
			ScriptContent: defaultPostgresTableUploadScript,
			Placeholders:  defaultUploadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, serverID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "目标服务器整理并校验单表文件",
			StepType:      "target_download",
			ScriptContent: defaultPostgresTableTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, serverID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "目标服务器导入 PostgreSQL 单表",
			StepType:      "target_exec",
			ScriptContent: defaultPostgresTableTargetExecScript,
			Placeholders:  defaultTableTargetExecPlaceholders(targetDatabaseResourceID, serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, serverID),
		},
	}

	for _, step := range steps {
		var existing sysModel.TbScriptStep
		err := db.Where("workflow_id = ? AND step_name = ?", workflow.ID, step.StepName).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&step).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := syncDefaultScriptStep(db, existing, step); err != nil {
			return err
		}
	}
	return nil
}

func seedReversePostgresDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensurePostgresScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, reverseScriptWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: reverseScriptWorkflowName,
			Description:  "通用 PostgreSQL 反向迁移流程：从目标服务器导出指定数据库，通过 Tailscale 传输文件，本机校验后清空并导入到指定数据库。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if workflow.CategoryId != category.ID {
		if err := db.Model(&workflow).Update("category_id", category.ID).Error; err != nil {
			return err
		}
		workflow.CategoryId = category.ID
	}

	localServerID := firstServerIDForUser(db, userID)
	connectionID := firstConnectionID(db)
	localServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProDatabaseConfigName,
		defaultMacMiniDatabaseConfigName,
		defaultTencentServerConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultTargetDatabasePlaceholderKey(),
		defaultSourceDatabasePlaceholderKey(),
	)
	if sourceDatabaseResourceID == 0 {
		sourceDatabaseResourceID = firstDatabaseResourceConfigID(db, userID, "postgresql")
	}
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultSourceDatabasePlaceholderKey(),
		defaultTargetDatabasePlaceholderKey(),
	)
	if targetDatabaseResourceID == 0 {
		targetDatabaseResourceID = sourceDatabaseResourceID
	}
	sourceDatabaseHost := firstText(databaseResourceHostForConfigID(db, userID, sourceDatabaseResourceID), defaultMacMiniDatabaseHost)
	sourceServerResourceID := firstServerResourceConfigIDForHost(db, userID, sourceDatabaseHost)
	if sourceServerResourceID == 0 {
		sourceServerResourceID = firstResourceConfigIDByPlaceholderKeys(
			db,
			userID,
			defaultServerResourceCategoryName,
			defaultMacMiniServerPlaceholderKey(),
			defaultMacMiniDatabaseConfigName,
			defaultTencentServerConfigName,
		)
	}
	sourceServerID := firstServerIDForHost(db, userID, sourceDatabaseHost)
	if sourceServerID == 0 {
		sourceServerID = firstServerIDForUser(db, userID)
	}
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensurePostgresWorkflowResources(db, userID, workflow.ID, reverseDeployProjectNameSeedRow(), seedDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}
	_ = db.Model(&sysModel.TbScriptStep{}).
		Where("workflow_id = ? AND step_name = ?", workflow.ID, "通过 Tailscale 拉取 PostgreSQL 导出文件到本地").
		Update("step_name", legacyReverseUploadStepName).Error

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      reverseExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultPostgresRemoteDatabaseExportScript,
			Placeholders:  defaultRemoteExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, sourceServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      legacyReverseUploadStepName,
			StepType:      "local_upload",
			ScriptContent: defaultEasyDeployUploadScript,
			Placeholders:  defaultUploadPlaceholders(localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, localServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本地服务器整理并校验 PostgreSQL 文件",
			StepType:      "target_download",
			ScriptContent: defaultEasyDeployTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, localServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本地服务器导入 PostgreSQL 数据库",
			StepType:      "target_exec",
			ScriptContent: defaultEasyDeployTargetExecScript,
			Placeholders:  defaultTargetExecPlaceholders(targetDatabaseResourceID, localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, localServerID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedReversePostgresTableExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensurePostgresScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, reverseTableScriptWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: reverseTableScriptWorkflowName,
			Description:  "通用 PostgreSQL 反向单表迁移流程：从目标服务器导出指定表，通过 Tailscale 传输文件，本机校验后导入同名表。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if workflow.CategoryId != category.ID {
		if err := db.Model(&workflow).Update("category_id", category.ID).Error; err != nil {
			return err
		}
		workflow.CategoryId = category.ID
	}

	localServerID := firstServerIDForUser(db, userID)
	connectionID := firstConnectionID(db)
	localServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProDatabaseConfigName,
		defaultMacMiniDatabaseConfigName,
		defaultTencentServerConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultTargetDatabasePlaceholderKey(),
		defaultSourceDatabasePlaceholderKey(),
	)
	if sourceDatabaseResourceID == 0 {
		sourceDatabaseResourceID = firstDatabaseResourceConfigID(db, userID, "postgresql")
	}
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		pgsqlDatabaseResourceCategoryName,
		defaultSourceDatabasePlaceholderKey(),
		defaultTargetDatabasePlaceholderKey(),
	)
	if targetDatabaseResourceID == 0 {
		targetDatabaseResourceID = sourceDatabaseResourceID
	}
	sourceDatabaseHost := firstText(databaseResourceHostForConfigID(db, userID, sourceDatabaseResourceID), defaultMacMiniDatabaseHost)
	sourceServerResourceID := firstServerResourceConfigIDForHost(db, userID, sourceDatabaseHost)
	if sourceServerResourceID == 0 {
		sourceServerResourceID = firstResourceConfigIDByPlaceholderKeys(
			db,
			userID,
			defaultServerResourceCategoryName,
			defaultMacMiniServerPlaceholderKey(),
			defaultMacMiniDatabaseConfigName,
			defaultTencentServerConfigName,
		)
	}
	sourceServerID := firstServerIDForHost(db, userID, sourceDatabaseHost)
	if sourceServerID == 0 {
		sourceServerID = firstServerIDForUser(db, userID)
	}
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensurePostgresWorkflowResources(db, userID, workflow.ID, reverseTableProjectNameSeedRow(), seedReverseTableDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      reverseTableExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultPostgresRemoteTableExportScript,
			Placeholders:  defaultRemoteTableExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, sourceServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传 PostgreSQL 单表导出文件到本地",
			StepType:      "local_upload",
			ScriptContent: defaultPostgresTableUploadScript,
			Placeholders:  defaultUploadPlaceholders(localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, localServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本地服务器整理并校验 PostgreSQL 单表文件",
			StepType:      "target_download",
			ScriptContent: defaultPostgresTableTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, localServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本地服务器导入 PostgreSQL 单表",
			StepType:      "target_exec",
			ScriptContent: defaultPostgresTableTargetExecScript,
			Placeholders:  defaultTableTargetExecPlaceholders(targetDatabaseResourceID, localServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, connectionID, localServerID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedDefaultMySQLDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	return seedMySQLDatabaseExportWorkflow(
		db,
		userID,
		defaultMySQLScriptWorkflowName,
		"通用 MySQL 迁移流程：本地导出指定数据库，通过 Tailscale 传输文件，目标机校验后清空并导入到指定数据库。",
		defaultMySQLExportStepName,
		"通过 Tailscale 上传 MySQL 导出文件",
		"目标服务器整理并校验 MySQL 文件",
		"目标服务器导入 MySQL 数据库",
		mysqlProjectNameSeedRow(),
		false,
	)
}

func seedReverseMySQLDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	return seedMySQLDatabaseExportWorkflow(
		db,
		userID,
		reverseMySQLScriptWorkflowName,
		"通用 MySQL 反向迁移流程：从目标服务器导出指定数据库，通过 Tailscale 传输文件，本机校验后清空并导入到指定数据库。",
		reverseMySQLExportStepName,
		"通过 Tailscale 上传 MySQL 导出文件到本地",
		"本地服务器整理并校验 MySQL 文件",
		"本地服务器导入 MySQL 数据库",
		mysqlReverseProjectNameSeedRow(),
		true,
	)
}

func seedMySQLDatabaseExportWorkflow(db *gorm.DB, userID uint, workflowName string, description string, exportStepName string, uploadStepName string, downloadStepName string, importStepName string, projectNameRow scriptResourceSeedRow, reverse bool) error {
	category, err := ensureMySQLScriptCategory(db, userID)
	if err != nil {
		return err
	}
	workflow, err := ensureScriptWorkflowInCategory(db, userID, category.ID, workflowName, description)
	if err != nil {
		return err
	}
	sourceDatabaseResourceID, targetDatabaseResourceID, targetServerResourceID, targetServerID := mysqlWorkflowResourceIDs(db, userID, reverse)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensurePostgresWorkflowResources(db, userID, workflow.ID, projectNameRow, seedMySQLDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}
	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      exportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultEasyDeployExportScript,
			Placeholders:  defaultExportPlaceholders(sourceDatabaseResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      uploadStepName,
			StepType:      "local_upload",
			ScriptContent: defaultEasyDeployUploadScript,
			Placeholders:  defaultUploadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, targetServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      downloadStepName,
			StepType:      "target_download",
			ScriptContent: defaultEasyDeployTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, targetServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      importStepName,
			StepType:      "target_exec",
			ScriptContent: defaultEasyDeployTargetExecScript,
			Placeholders:  defaultTargetExecPlaceholders(targetDatabaseResourceID, targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0, targetServerID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedDefaultMySQLTableExportWorkflow(db *gorm.DB, userID uint) error {
	return seedMySQLTableExportWorkflow(
		db,
		userID,
		defaultMySQLTableWorkflowName,
		"通用 MySQL 单表迁移流程：本地导出指定表，通过 Tailscale 传输文件，目标机校验后导入同名表。",
		defaultMySQLTableExportStepName,
		"通过 Tailscale 上传 MySQL 单表导出文件",
		"目标服务器整理并校验 MySQL 单表文件",
		"目标服务器导入 MySQL 单表",
		mysqlTableProjectNameSeedRow(),
		false,
	)
}

func seedReverseMySQLTableExportWorkflow(db *gorm.DB, userID uint) error {
	return seedMySQLTableExportWorkflow(
		db,
		userID,
		reverseMySQLTableWorkflowName,
		"通用 MySQL 反向单表迁移流程：从目标服务器导出指定表，通过 Tailscale 传输文件，本机校验后导入同名表。",
		reverseMySQLTableExportStepName,
		"通过 Tailscale 上传 MySQL 单表导出文件到本地",
		"本地服务器整理并校验 MySQL 单表文件",
		"本地服务器导入 MySQL 单表",
		mysqlReverseTableProjectNameSeedRow(),
		true,
	)
}

func seedMySQLTableExportWorkflow(db *gorm.DB, userID uint, workflowName string, description string, exportStepName string, uploadStepName string, downloadStepName string, importStepName string, projectNameRow scriptResourceSeedRow, reverse bool) error {
	category, err := ensureMySQLScriptCategory(db, userID)
	if err != nil {
		return err
	}
	workflow, err := ensureScriptWorkflowInCategory(db, userID, category.ID, workflowName, description)
	if err != nil {
		return err
	}
	sourceDatabaseResourceID, targetDatabaseResourceID, targetServerResourceID, targetServerID := mysqlWorkflowResourceIDs(db, userID, reverse)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensurePostgresWorkflowResources(db, userID, workflow.ID, projectNameRow, seedMySQLTableDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}
	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      exportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultMySQLTableExportScript,
			Placeholders:  defaultTableExportPlaceholders(sourceDatabaseResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      uploadStepName,
			StepType:      "local_upload",
			ScriptContent: defaultPostgresTableUploadScript,
			Placeholders:  defaultUploadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, targetServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      downloadStepName,
			StepType:      "target_download",
			ScriptContent: defaultPostgresTableTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, targetServerID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      importStepName,
			StepType:      "target_exec",
			ScriptContent: defaultMySQLTableTargetExecScript,
			Placeholders:  defaultTableTargetExecPlaceholders(targetDatabaseResourceID, targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0, targetServerID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func mysqlWorkflowResourceIDs(db *gorm.DB, userID uint, reverse bool) (uint, uint, uint, uint) {
	localDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		mysqlDatabaseResourceCategoryName,
		databaseResourcePlaceholderKey(defaultMacBookProDatabaseConfigName, "mysql"),
	)
	remoteDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		mysqlDatabaseResourceCategoryName,
		databaseResourcePlaceholderKey(defaultTencentServerConfigName, "mysql"),
	)
	if remoteDatabaseResourceID == 0 {
		remoteDatabaseResourceID = firstDatabaseResourceConfigID(db, userID, "mysql")
	}
	if localDatabaseResourceID == 0 {
		localDatabaseResourceID = remoteDatabaseResourceID
	}
	localServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProServerPlaceholderKey(),
		defaultMacBookProDatabaseConfigName,
	)
	remoteServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		databaseResourcePlaceholderKey(defaultTencentServerConfigName, "server"),
		defaultTencentServerConfigName,
	)
	localServerID := firstServerIDForUser(db, userID)
	remoteServerID := firstServerIDForHost(db, userID, defaultTencentServerHost)
	if reverse {
		return remoteDatabaseResourceID, localDatabaseResourceID, localServerResourceID, localServerID
	}
	return localDatabaseResourceID, remoteDatabaseResourceID, remoteServerResourceID, remoteServerID
}

func seedDefaultClickHouseDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensureClickHouseScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, defaultClickHouseScriptWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: defaultClickHouseScriptWorkflowName,
			Description:  "ClickHouse 迁移流程：从 mac mini 只读导出指定数据库，通过 Tailscale 传输文件，在本机目标库按需清表后导入。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	serverResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProServerPlaceholderKey(),
		defaultMacBookProDatabaseConfigName,
	)
	sourceServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacMiniServerPlaceholderKey(),
		defaultMacMiniDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseSourceDatabasePlaceholderKey(),
	)
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseTargetDatabasePlaceholderKey(),
	)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensureClickHouseWorkflowResources(db, userID, workflow.ID, clickHouseProjectNameSeedRow(), seedClickHouseDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      defaultClickHouseExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultClickHouseExportScript,
			Placeholders:  defaultClickHouseExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传 ClickHouse 导出文件",
			StepType:      "local_upload",
			ScriptContent: defaultClickHouseUploadScript,
			Placeholders:  defaultUploadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本机服务器整理并校验 ClickHouse 文件",
			StepType:      "target_download",
			ScriptContent: defaultClickHouseTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本机服务器导入 ClickHouse 数据库",
			StepType:      "target_exec",
			ScriptContent: defaultClickHouseTargetExecScript,
			Placeholders:  defaultClickHouseTargetExecPlaceholders(targetDatabaseResourceID, serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedDefaultClickHouseTableExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensureClickHouseScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, defaultClickHouseTableWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: defaultClickHouseTableWorkflowName,
			Description:  "ClickHouse 单表迁移流程：从 mac mini 只读导出指定表，通过 Tailscale 传输文件，在本机目标库按需清表后导入。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	serverResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProServerPlaceholderKey(),
		defaultMacBookProDatabaseConfigName,
	)
	sourceServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacMiniServerPlaceholderKey(),
		defaultMacMiniDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseSourceDatabasePlaceholderKey(),
	)
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseTargetDatabasePlaceholderKey(),
	)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensureClickHouseWorkflowResources(db, userID, workflow.ID, clickHouseTableProjectNameSeedRow(), seedClickHouseTableDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      defaultClickHouseTableExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultClickHouseTableExportScript,
			Placeholders:  defaultClickHouseExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传 ClickHouse 单表导出文件",
			StepType:      "local_upload",
			ScriptContent: defaultClickHouseTableUploadScript,
			Placeholders:  defaultUploadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本机服务器整理并校验 ClickHouse 单表文件",
			StepType:      "target_download",
			ScriptContent: defaultClickHouseTableTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "本机服务器导入 ClickHouse 单表",
			StepType:      "target_exec",
			ScriptContent: defaultClickHouseTableTargetExecScript,
			Placeholders:  defaultClickHouseTargetExecPlaceholders(targetDatabaseResourceID, serverResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedReverseClickHouseDatabaseExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensureClickHouseScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, reverseClickHouseScriptWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: reverseClickHouseScriptWorkflowName,
			Description:  "ClickHouse 反向迁移流程：从本机只读导出指定数据库，通过 Tailscale 传输文件，在 mac mini 目标库按需清表后导入。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	sourceServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProServerPlaceholderKey(),
		defaultMacBookProDatabaseConfigName,
	)
	targetServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacMiniServerPlaceholderKey(),
		defaultMacMiniDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseTargetDatabasePlaceholderKey(),
	)
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseSourceDatabasePlaceholderKey(),
	)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensureClickHouseWorkflowResources(db, userID, workflow.ID, clickHouseReverseProjectNameSeedRow(), seedClickHouseDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      reverseClickHouseExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultClickHouseExportScript,
			Placeholders:  defaultClickHouseExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传 ClickHouse 导出文件到 mac mini",
			StepType:      "local_upload",
			ScriptContent: defaultClickHouseUploadScript,
			Placeholders:  defaultUploadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "mac mini 服务器整理并校验 ClickHouse 文件",
			StepType:      "target_download",
			ScriptContent: defaultClickHouseTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "mac mini 服务器导入 ClickHouse 数据库",
			StepType:      "target_exec",
			ScriptContent: defaultClickHouseTargetExecScript,
			Placeholders:  defaultClickHouseTargetExecPlaceholders(targetDatabaseResourceID, targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

func seedReverseClickHouseTableExportWorkflow(db *gorm.DB, userID uint) error {
	category, err := ensureClickHouseScriptCategory(db, userID)
	if err != nil {
		return err
	}

	var workflow sysModel.TbScriptWorkflow
	err = db.Where("user_id = ? AND workflow_name = ?", userID, reverseClickHouseTableWorkflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   category.ID,
			WorkflowName: reverseClickHouseTableWorkflowName,
			Description:  "ClickHouse 反向单表迁移流程：从本机只读导出指定表，通过 Tailscale 传输文件，在 mac mini 目标库按需清表后导入。",
		}
		if err := db.Create(&workflow).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	sourceServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacBookProServerPlaceholderKey(),
		defaultMacBookProDatabaseConfigName,
	)
	targetServerResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		defaultServerResourceCategoryName,
		defaultMacMiniServerPlaceholderKey(),
		defaultMacMiniDatabaseConfigName,
	)
	sourceDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseTargetDatabasePlaceholderKey(),
	)
	targetDatabaseResourceID := firstResourceConfigIDByPlaceholderKeys(
		db,
		userID,
		clickHouseResourceCategoryName,
		defaultClickHouseSourceDatabasePlaceholderKey(),
	)
	pathConstantsResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, err := ensureClickHouseWorkflowResources(db, userID, workflow.ID, clickHouseReverseTableProjectNameSeedRow(), seedClickHouseTableDynamicParamsResourceConfigs)
	if err != nil {
		return err
	}

	steps := []sysModel.TbScriptStep{
		{
			WorkflowId:    workflow.ID,
			StepName:      reverseClickHouseTableExportStepName,
			StepType:      "local_exec",
			ScriptContent: defaultClickHouseTableExportScript,
			Placeholders:  defaultClickHouseExportPlaceholders(sourceDatabaseResourceID, sourceServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "通过 Tailscale 上传 ClickHouse 单表导出文件到 mac mini",
			StepType:      "local_upload",
			ScriptContent: defaultClickHouseTableUploadScript,
			Placeholders:  defaultUploadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "mac mini 服务器整理并校验 ClickHouse 单表文件",
			StepType:      "target_download",
			ScriptContent: defaultClickHouseTableTargetDownloadScript,
			Placeholders:  defaultTargetDownloadPlaceholders(targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID, 0),
		},
		{
			WorkflowId:    workflow.ID,
			StepName:      "mac mini 服务器导入 ClickHouse 单表",
			StepType:      "target_exec",
			ScriptContent: defaultClickHouseTableTargetExecScript,
			Placeholders:  defaultClickHouseTargetExecPlaceholders(targetDatabaseResourceID, targetServerResourceID, dynamicParamsCategoryID, dynamicParamsResourceID, pathConstantsResourceID),
		},
	}
	return syncDefaultScriptSteps(db, steps)
}

type clickHouseDynamicParamsSeeder func(*gorm.DB, uint, uint, uint, scriptResourceSeedRow) error

type postgresDynamicParamsSeeder func(*gorm.DB, uint, uint, uint, scriptResourceSeedRow) error

func ensurePostgresWorkflowResources(db *gorm.DB, userID uint, workflowID uint, projectNameRow scriptResourceSeedRow, seedDynamicParams postgresDynamicParamsSeeder) (uint, uint, uint, error) {
	pathConstantsCategory, err := ensurePathConstantsResourceCategory(db, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	if err := seedPathConstantsResourceConfig(db, userID, pathConstantsCategory.ID); err != nil {
		return 0, 0, 0, err
	}
	pathConstantsResourceID := firstResourceConfigID(db, userID, defaultPathConstantsCategoryName)
	dynamicParamsCategory, err := ensureScriptResourceCategory(db, userID, defaultDynamicParamsCategoryName, "dynamic")
	if err != nil {
		return 0, 0, 0, err
	}
	if err := seedDynamicParams(db, userID, dynamicParamsCategory.ID, workflowID, projectNameRow); err != nil {
		return 0, 0, 0, err
	}
	return pathConstantsResourceID, dynamicParamsCategory.ID, firstResourceConfigIDForWorkflow(db, userID, defaultDynamicParamsCategoryName, workflowID), nil
}

func seedReverseTableDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标数据库名称", Value: "easy_deploy"},
		{Name: "TABLE_SCHEMA", Placeholder: "表 Schema", Value: "public"},
		{Name: "TABLE_NAME", Placeholder: "表名", Value: ""},
		{Name: "TARGET_TABLE_DROP_BEFORE_IMPORT", Placeholder: "导入前删除目标表", Value: "true"},
		{Name: "SOURCE_PG_DUMP_CMD", Placeholder: "源库 pg_dump 命令", Value: "/usr/local/bin/docker exec postgres16 pg_dump"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_PSQL_CMD", Placeholder: "目标库 psql 命令", Value: "/usr/local/bin/psql"},
		{Name: "TARGET_PG_RESTORE_CMD", Placeholder: "目标库 pg_restore 命令", Value: "/usr/local/bin/pg_restore"},
	}
	configName := defaultTableDynamicParamsConfigName
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
		"TARGET_DB_DROP_TABLES_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

func seedMySQLDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DROP_TABLES_BEFORE_IMPORT", Placeholder: "导入前删除目标数据库所有表", Value: "true"},
		{Name: "SOURCE_MYSQLDUMP_CMD", Placeholder: "源库 mysqldump 命令", Value: "/usr/local/bin/mysqldump"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_MYSQL_CMD", Placeholder: "目标库 mysql 命令", Value: "/usr/local/bin/mysql"},
	}
	configName := "MySQL 数据库导出执行参数"
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

func seedMySQLTableDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标数据库名称", Value: "easy_deploy"},
		{Name: "TABLE_NAME", Placeholder: "表名称", Value: ""},
		{Name: "TARGET_TABLE_DROP_BEFORE_IMPORT", Placeholder: "导入前删除目标表", Value: "true"},
		{Name: "SOURCE_MYSQLDUMP_CMD", Placeholder: "源库 mysqldump 命令", Value: "/usr/local/bin/mysqldump"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_MYSQL_CMD", Placeholder: "目标库 mysql 命令", Value: "/usr/local/bin/mysql"},
	}
	configName := "MySQL 单表导出执行参数"
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
		"TARGET_DB_DROP_TABLES_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

func ensureClickHouseWorkflowResources(db *gorm.DB, userID uint, workflowID uint, projectNameRow scriptResourceSeedRow, seedDynamicParams clickHouseDynamicParamsSeeder) (uint, uint, uint, error) {
	pathConstantsCategory, err := ensurePathConstantsResourceCategory(db, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	if err := seedPathConstantsResourceConfig(db, userID, pathConstantsCategory.ID); err != nil {
		return 0, 0, 0, err
	}
	pathConstantsResourceID := firstResourceConfigID(db, userID, defaultPathConstantsCategoryName)
	dynamicParamsCategory, err := ensureScriptResourceCategory(db, userID, defaultDynamicParamsCategoryName, "dynamic")
	if err != nil {
		return 0, 0, 0, err
	}
	if err := seedDynamicParams(db, userID, dynamicParamsCategory.ID, workflowID, projectNameRow); err != nil {
		return 0, 0, 0, err
	}
	dynamicParamsResourceID := firstResourceConfigIDForWorkflow(db, userID, defaultDynamicParamsCategoryName, workflowID)
	return pathConstantsResourceID, dynamicParamsCategory.ID, dynamicParamsResourceID, nil
}

func syncDefaultScriptSteps(db *gorm.DB, steps []sysModel.TbScriptStep) error {
	for _, step := range steps {
		var existing sysModel.TbScriptStep
		err := db.Where("workflow_id = ? AND step_name = ?", step.WorkflowId, step.StepName).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&step).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := syncDefaultScriptStep(db, existing, step); err != nil {
			return err
		}
	}
	return nil
}

func syncDefaultScriptStep(db *gorm.DB, existing sysModel.TbScriptStep, desired sysModel.TbScriptStep) error {
	updates := map[string]any{}
	if existing.Placeholders == "" && desired.Placeholders != "" {
		updates["placeholders"] = desired.Placeholders
	} else if merged, ok := mergeSeedPlaceholders(existing.Placeholders, desired.Placeholders); ok {
		updates["placeholders"] = merged
	}
	if shouldRefreshDefaultStepScript(existing.StepName, existing.ScriptContent) {
		updates["script_content"] = desired.ScriptContent
	}
	if shouldRefreshDefaultStepType(existing.StepName, existing.StepType, desired.StepType) {
		updates["step_type"] = desired.StepType
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&existing).Updates(updates).Error
}

func ensureDefaultScriptCategory(db *gorm.DB, userID uint) (sysModel.TbScriptCategory, error) {
	return ensureScriptCategoryByName(db, userID, defaultScriptCategoryName, "数据库导出、迁移、备份与恢复脚本流程")
}

func ensurePostgresScriptCategory(db *gorm.DB, userID uint) (sysModel.TbScriptCategory, error) {
	return ensureScriptCategoryByName(db, userID, postgresScriptCategoryName, "PostgreSQL 数据库导出、迁移、备份与恢复脚本流程")
}

func ensureMySQLScriptCategory(db *gorm.DB, userID uint) (sysModel.TbScriptCategory, error) {
	return ensureScriptCategoryByName(db, userID, mysqlScriptCategoryName, "MySQL 数据库导出、迁移、备份与恢复脚本流程")
}

func ensureClickHouseScriptCategory(db *gorm.DB, userID uint) (sysModel.TbScriptCategory, error) {
	return ensureScriptCategoryByName(db, userID, clickHouseScriptCategoryName, "ClickHouse 数据库导出、迁移、备份与恢复脚本流程")
}

func ensureScriptWorkflowInCategory(db *gorm.DB, userID uint, categoryID uint, workflowName string, description string) (sysModel.TbScriptWorkflow, error) {
	var workflow sysModel.TbScriptWorkflow
	err := db.Where("user_id = ? AND workflow_name = ?", userID, workflowName).First(&workflow).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		workflow = sysModel.TbScriptWorkflow{
			UserId:       userID,
			CategoryId:   categoryID,
			WorkflowName: workflowName,
			Description:  description,
		}
		return workflow, db.Create(&workflow).Error
	}
	if err != nil {
		return workflow, err
	}
	updates := map[string]any{}
	if workflow.CategoryId != categoryID {
		updates["category_id"] = categoryID
	}
	if strings.TrimSpace(workflow.Description) != strings.TrimSpace(description) {
		updates["description"] = description
	}
	if len(updates) > 0 {
		if err := db.Model(&workflow).Updates(updates).Error; err != nil {
			return workflow, err
		}
		workflow.CategoryId = categoryID
		workflow.Description = description
	}
	return workflow, nil
}

func ensureScriptCategoryByName(db *gorm.DB, userID uint, categoryName string, description string) (sysModel.TbScriptCategory, error) {
	var category sysModel.TbScriptCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		category = sysModel.TbScriptCategory{
			UserId:       userID,
			CategoryName: categoryName,
			Description:  description,
		}
		err = db.Create(&category).Error
	}
	return category, err
}

func firstServerIDForUser(db *gorm.DB, userID uint) uint {
	var server sysModel.TbServer
	if err := db.Where("user_id = ?", userID).Order("id").First(&server).Error; err == nil {
		return server.ID
	}
	if err := db.Order("id").First(&server).Error; err == nil {
		return server.ID
	}
	return 0
}

func firstServerIDForHost(db *gorm.DB, userID uint, host string) uint {
	normalizedHost := normalizeDatabaseResourceHost(host)
	if normalizedHost == "" {
		return 0
	}
	var server sysModel.TbServer
	if err := db.Where("(user_id = ? OR user_id = 0) AND (server_ip = ? OR server_internal_ip = ?)", userID, normalizedHost, normalizedHost).Order("user_id DESC, id").First(&server).Error; err == nil {
		return server.ID
	}
	return 0
}

func firstServerResourceConfigIDForHost(db *gorm.DB, userID uint, host string) uint {
	identifier := resourceIdentifierForHost(host)
	preferredKeys := []string{}
	if identifier != "" {
		preferredKeys = append(preferredKeys, databaseResourcePlaceholderKey(identifier, "server"), identifier)
	}
	if normalizeDatabaseResourceHost(host) == defaultMacMiniDatabaseHost {
		preferredKeys = append(preferredKeys, defaultMacMiniServerPlaceholderKey(), defaultMacMiniDatabaseConfigName)
	}
	if normalizeDatabaseResourceHost(host) == defaultTencentServerHost {
		preferredKeys = append(preferredKeys, databaseResourcePlaceholderKey(defaultTencentServerConfigName, "server"), defaultTencentServerConfigName)
	}
	if len(preferredKeys) == 0 {
		return 0
	}
	var category sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", userID, defaultServerResourceCategoryName).First(&category).Error; err != nil {
		return 0
	}
	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND (workflow_id = 0 OR workflow_id IS NULL)", userID, category.ID).Order("id").Find(&configs).Error; err != nil {
		return 0
	}
	for _, preferredKey := range preferredKeys {
		preferredKey = normalizeSeedPlaceholderName(preferredKey)
		if preferredKey == "" {
			continue
		}
		for _, config := range configs {
			if normalizeSeedPlaceholderName(config.PlaceholderKey) == preferredKey {
				return config.ID
			}
		}
	}
	return 0
}

func firstConnectionID(db *gorm.DB) uint {
	var connection sysModel.TbConnection
	if err := db.Order("id").First(&connection).Error; err == nil {
		return connection.ID
	}
	return 0
}

func firstResourceConfigID(db *gorm.DB, userID uint, categoryName string) uint {
	var category sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error; err != nil {
		return 0
	}
	var config sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND (workflow_id = 0 OR workflow_id IS NULL)", userID, category.ID).Order("id").First(&config).Error; err == nil {
		return config.ID
	}
	return 0
}

func firstResourceConfigIDForWorkflow(db *gorm.DB, userID uint, categoryName string, workflowID uint) uint {
	var category sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error; err != nil {
		return 0
	}
	var config sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND workflow_id = ?", userID, category.ID, workflowID).Order("id").First(&config).Error; err == nil {
		return config.ID
	}
	return 0
}

func firstDatabaseResourceConfigID(db *gorm.DB, userID uint, preferredTypes ...string) uint {
	for _, dbType := range preferredTypes {
		categoryName, ok := databaseResourceCategoryNameForType(dbType)
		if !ok {
			continue
		}
		if configID := firstResourceConfigID(db, userID, categoryName); configID != 0 {
			return configID
		}
	}
	for _, spec := range databaseResourceCategorySpecs {
		if configID := firstResourceConfigID(db, userID, spec.CategoryName); configID != 0 {
			return configID
		}
	}
	return firstResourceConfigID(db, userID, defaultDatabaseResourceCategoryName)
}

func firstResourceConfigIDByPlaceholderKeys(db *gorm.DB, userID uint, categoryName string, preferredKeys ...string) uint {
	config, ok := firstResourceConfigByPlaceholderKeys(db, userID, categoryName, preferredKeys...)
	if !ok {
		return 0
	}
	return config.ID
}

func firstResourceConfigByPlaceholderKeys(db *gorm.DB, userID uint, categoryName string, preferredKeys ...string) (sysModel.TbScriptResourceConfig, bool) {
	var category sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error; err != nil {
		return sysModel.TbScriptResourceConfig{}, false
	}
	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND (workflow_id = 0 OR workflow_id IS NULL)", userID, category.ID).Order("id").Find(&configs).Error; err != nil {
		return sysModel.TbScriptResourceConfig{}, false
	}
	for _, preferredKey := range preferredKeys {
		preferredKey = normalizeSeedPlaceholderName(preferredKey)
		if preferredKey == "" {
			continue
		}
		for _, config := range configs {
			if normalizeSeedPlaceholderName(config.PlaceholderKey) == preferredKey {
				return config, true
			}
		}
	}
	if len(configs) == 0 {
		return sysModel.TbScriptResourceConfig{}, false
	}
	return configs[0], true
}

func databaseResourceHostForConfigID(db *gorm.DB, userID uint, configID uint) string {
	if configID == 0 {
		return ""
	}
	var config sysModel.TbScriptResourceConfig
	if err := db.Where("id = ? AND user_id = ?", configID, userID).First(&config).Error; err != nil {
		return ""
	}
	rows, err := scriptResourceSeedRowsFromJSON(config.Rows)
	if err != nil {
		return ""
	}
	return databaseResourceHostFromRows(rows)
}

func firstResourceCategoryID(db *gorm.DB, userID uint, categoryName string) uint {
	var category sysModel.TbScriptResourceCategory
	if err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error; err == nil {
		return category.ID
	}
	return 0
}

func seedScriptResourceConfigs(db *gorm.DB, userID uint) error {
	if !db.Migrator().HasTable(&sysModel.TbScriptResourceCategory{}) || !db.Migrator().HasTable(&sysModel.TbScriptResourceConfig{}) {
		return nil
	}
	serverCategory, err := ensureScriptResourceCategory(db, userID, defaultServerResourceCategoryName, "fixed")
	if err != nil {
		return err
	}
	if err := seedServerResourceConfigs(db, userID, serverCategory.ID); err != nil {
		return err
	}
	databaseCategoryIDs, err := ensureDatabaseResourceCategories(db, userID)
	if err != nil {
		return err
	}
	if err := migrateLegacyDatabaseResourceConfigs(db, userID, databaseCategoryIDs); err != nil {
		return err
	}
	if err := seedDatabaseResourceConfigs(db, userID, databaseCategoryIDs); err != nil {
		return err
	}
	if err := seedMacMiniDatabaseResourceConfigs(db, userID, databaseCategoryIDs); err != nil {
		return err
	}
	if err := seedLocalMySQLDatabaseResourceConfig(db, userID, databaseCategoryIDs); err != nil {
		return err
	}
	if err := dedupeDatabaseResourceConfigs(db, userID, databaseCategoryIDs); err != nil {
		return err
	}
	pathConstantsCategory, err := ensurePathConstantsResourceCategory(db, userID)
	if err != nil {
		return err
	}
	if err := seedPathConstantsResourceConfig(db, userID, pathConstantsCategory.ID); err != nil {
		return err
	}
	dynamicParamsCategory, err := ensureScriptResourceCategory(db, userID, defaultDynamicParamsCategoryName, "dynamic")
	if err != nil {
		return err
	}
	if err := removeResourceConfigRowsByName(db, userID, dynamicParamsCategory.ID, 0, "数据库导出执行参数", pathConstantResourceRowNames...); err != nil {
		return err
	}
	return removeDefaultDeployResourceConfigs(db, userID)
}

func ensureScriptResourceCategory(db *gorm.DB, userID uint, categoryName string, categoryType string) (sysModel.TbScriptResourceCategory, error) {
	var category sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, categoryName).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		category = sysModel.TbScriptResourceCategory{
			UserId:       userID,
			CategoryName: categoryName,
			CategoryType: categoryType,
		}
		err = db.Create(&category).Error
	}
	if err == nil && strings.TrimSpace(category.CategoryType) == "" && strings.TrimSpace(categoryType) != "" {
		err = db.Model(&category).Update("category_type", categoryType).Error
		category.CategoryType = categoryType
	}
	return category, err
}

func ensurePathConstantsResourceCategory(db *gorm.DB, userID uint) (sysModel.TbScriptResourceCategory, error) {
	var category sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, defaultPathConstantsCategoryName).First(&category).Error
	if err == nil {
		if category.CategoryType != "constant" {
			if updateErr := db.Model(&category).Update("category_type", "constant").Error; updateErr != nil {
				return category, updateErr
			}
			category.CategoryType = "constant"
		}
		return category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return category, err
	}

	err = db.Where("user_id = ? AND category_name = ?", userID, legacyPathConstantsCategoryName).First(&category).Error
	if err == nil {
		if updateErr := db.Model(&category).Updates(map[string]any{
			"category_name": defaultPathConstantsCategoryName,
			"category_type": "constant",
		}).Error; updateErr != nil {
			return category, updateErr
		}
		category.CategoryName = defaultPathConstantsCategoryName
		category.CategoryType = "constant"
		return category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return category, err
	}
	return ensureScriptResourceCategory(db, userID, defaultPathConstantsCategoryName, "constant")
}

func ensureDatabaseResourceCategories(db *gorm.DB, userID uint) (map[string]uint, error) {
	categoryIDs := make(map[string]uint, len(databaseResourceCategorySpecs))
	for _, spec := range databaseResourceCategorySpecs {
		category, err := ensureScriptResourceCategory(db, userID, spec.CategoryName, "fixed")
		if err != nil {
			return nil, err
		}
		categoryIDs[spec.DBType] = category.ID
	}
	return categoryIDs, nil
}

func seedServerResourceConfigs(db *gorm.DB, userID uint, categoryID uint) error {
	var servers []sysModel.TbServer
	if err := db.Where("user_id = ? OR user_id = 0", userID).Order("id").Find(&servers).Error; err != nil {
		return err
	}
	for _, server := range servers {
		placeholderKey := serverResourcePlaceholderKey(server)
		configName := resourceDisplayName(placeholderKey, server.ServerIp, fmt.Sprintf("服务器-%d", server.ID))
		rows := []scriptResourceSeedRow{
			{Name: "NAME", Placeholder: "服务器名称", Value: server.ServerName},
			{Name: "IP", Placeholder: "服务器 IP", Value: server.ServerIp},
			{Name: "INTERNAL_IP", Placeholder: "服务器内网 IP", Value: server.ServerInternalIp},
			{Name: "USER", Placeholder: "服务器登录用户", Value: server.ServerLoginName},
			{Name: "PASSWORD", Placeholder: "服务器登录密码", Value: server.ServerLoginPassword},
			{Name: "PORT", Placeholder: "服务器登录端口", Value: fmt.Sprintf("%d", server.ServerLoginPort)},
			{Name: "TARGET_TAILSCALE_IP", Placeholder: "文件传输 Tailscale IP", Value: ""},
			{Name: "TARGET_TAILSCALE_PORT", Placeholder: "文件传输 Tailscale SSH 端口", Value: ""},
		}
		if err := ensureScriptResourceConfigWithPlaceholderKey(db, userID, categoryID, configName, placeholderKey, rows); err != nil {
			return err
		}
	}
	return nil
}

func serverResourcePlaceholderKey(server sysModel.TbServer) string {
	host := firstText(server.ServerIp, server.ServerInternalIp)
	if identifier := resourceIdentifierForHost(host); identifier != "" && !strings.HasPrefix(identifier, "ip_") {
		return databaseResourcePlaceholderKey(identifier, "server")
	}
	if name := strings.TrimSpace(server.ServerName); name != "" {
		normalizedName := normalizeSeedPlaceholderName(name)
		switch {
		case strings.Contains(normalizedName, "MACBOOK"):
			return defaultMacBookProServerPlaceholderKey()
		case strings.Contains(normalizedName, "MACMINI") || strings.Contains(normalizedName, "MAC_MINI"):
			return defaultMacMiniServerPlaceholderKey()
		}
		return name
	}
	return resourceIdentifierForHost(host)
}

func defaultMacBookProServerPlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacBookProDatabaseConfigName, "server")
}

func defaultMacMiniServerPlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacMiniDatabaseConfigName, "server")
}

func seedDatabaseResourceConfigs(db *gorm.DB, userID uint, categoryIDs map[string]uint) error {
	var connections []sysModel.TbConnection
	if err := db.Order("id").Find(&connections).Error; err != nil {
		return err
	}
	seeded := map[string]struct{}{}
	for _, connection := range connections {
		categoryID, err := databaseResourceCategoryID(db, userID, categoryIDs, connection.ConnectionType)
		if err != nil {
			return err
		}
		if categoryID == 0 {
			continue
		}
		key := databaseResourceDedupKey(connection.ConnectionType, connection.ConnectionUrl, fmt.Sprintf("connection:%d", connection.ID))
		if _, ok := seeded[key]; ok {
			continue
		}
		seeded[key] = struct{}{}
		configName := databaseResourceConfigName(connection)
		if err := ensureDatabaseResourceConfig(db, userID, categoryID, 0, configName, databaseResourceRows(connection, configName)); err != nil {
			return err
		}
	}
	return removeLegacyDatabaseResourceCategoryIfEmpty(db, userID)
}

func seedMacMiniDatabaseResourceConfigs(db *gorm.DB, userID uint, categoryIDs map[string]uint) error {
	for _, dbType := range []string{"postgresql", "clickhouse"} {
		categoryID := categoryIDs[dbType]
		if categoryID == 0 {
			continue
		}
		sourceRows, ok, err := macBookProDatabaseResourceRows(db, userID, categoryID, dbType)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		sourceLabel := databaseResourceConfigLabel(
			databaseResourceValueFromRows(sourceRows, "NAME"),
			databaseResourceHostFromRows(sourceRows),
		)
		configName := databaseResourceDisplayName(defaultMacMiniDatabaseConfigName, sourceLabel, defaultMacMiniDatabaseHost, "")
		rows := cloneDatabaseResourceRowsForHost(sourceRows, configName, defaultMacMiniDatabaseHost)
		rows = ensureDatabaseResourceRow(rows, "REMOTE_HOST", "远端执行数据库地址", "127.0.0.1")
		if err := ensureDatabaseResourceConfig(db, userID, categoryID, 0, configName, rows); err != nil {
			return err
		}
	}
	return nil
}

func seedLocalMySQLDatabaseResourceConfig(db *gorm.DB, userID uint, categoryIDs map[string]uint) error {
	categoryID := categoryIDs["mysql"]
	if categoryID == 0 {
		return nil
	}
	configName := databaseResourceDisplayName(defaultMacBookProDatabaseConfigName, "", "127.0.0.1", "")
	rows := []scriptResourceSeedRow{
		{Name: "NAME", Placeholder: "数据库配置名称", Value: configName},
		{Name: "TYPE", Placeholder: "数据库类型", Value: "mysql"},
		{Name: "HOST", Placeholder: "数据库地址", Value: "127.0.0.1"},
		{Name: "PORT", Placeholder: "数据库端口", Value: "3306"},
		{Name: "USER", Placeholder: "数据库用户", Value: "root"},
		{Name: "PASSWORD", Placeholder: "数据库密码", Value: "root123456"},
		{Name: "GROUP", Placeholder: "数据库分组", Value: "本地环境"},
		{Name: "ENV", Placeholder: "数据库环境", Value: "本地环境"},
	}
	placeholderKey := databaseResourcePlaceholderKey(defaultMacBookProDatabaseConfigName, "mysql")
	rows = pruneDatabaseResourceRows(rows, configName)
	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	var existing sysModel.TbScriptResourceConfig
	err = db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND placeholder_key = ?", userID, categoryID, 0, placeholderKey).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, 0, configName).First(&existing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&sysModel.TbScriptResourceConfig{
			UserId:         userID,
			CategoryId:     categoryID,
			WorkflowId:     0,
			ConfigName:     configName,
			PlaceholderKey: placeholderKey,
			Rows:           string(rowsJSON),
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&existing).Updates(map[string]any{
		"config_name":     configName,
		"placeholder_key": placeholderKey,
		"rows":            string(rowsJSON),
	}).Error
}

func macBookProDatabaseResourceRows(db *gorm.DB, userID uint, categoryID uint, dbType string) ([]scriptResourceSeedRow, bool, error) {
	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ?", userID, categoryID).Order("id").Find(&configs).Error; err != nil {
		return nil, false, err
	}
	for _, config := range configs {
		rows, err := scriptResourceSeedRowsFromJSON(config.Rows)
		if err != nil {
			continue
		}
		if normalizeDatabaseResourceHost(databaseResourceHostFromRows(rows)) != "127.0.0.1" {
			continue
		}
		rowDBType := normalizeDatabaseResourceType(databaseResourceValueFromRows(rows, "TYPE"))
		if rowDBType != "" && rowDBType != dbType {
			continue
		}
		return rows, true, nil
	}

	var connections []sysModel.TbConnection
	if err := db.Order("id").Find(&connections).Error; err != nil {
		return nil, false, err
	}
	for _, connection := range connections {
		if normalizeDatabaseResourceType(connection.ConnectionType) != dbType {
			continue
		}
		if normalizeDatabaseResourceHost(connection.ConnectionUrl) != "127.0.0.1" {
			continue
		}
		configName := databaseResourceConfigName(connection)
		return databaseResourceRows(connection, configName), true, nil
	}
	return nil, false, nil
}

func cloneDatabaseResourceRowsForHost(sourceRows []scriptResourceSeedRow, configName string, host string) []scriptResourceSeedRow {
	rows := pruneDatabaseResourceRows(sourceRows, configName)
	hasName := false
	hasHost := false
	for index, row := range rows {
		switch strings.ToUpper(strings.TrimSpace(row.Name)) {
		case "NAME":
			rows[index].Value = configName
			hasName = true
		case "HOST":
			rows[index].Value = host
			hasHost = true
		}
	}
	if !hasName {
		rows = append([]scriptResourceSeedRow{{Name: "NAME", Placeholder: "数据库配置名称", Value: configName}}, rows...)
	}
	if !hasHost {
		rows = append(rows, scriptResourceSeedRow{Name: "HOST", Placeholder: "数据库地址", Value: host})
	}
	return rows
}

func ensureDatabaseResourceRow(rows []scriptResourceSeedRow, name string, placeholder string, value string) []scriptResourceSeedRow {
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Name), name) {
			return rows
		}
	}
	return append(rows, scriptResourceSeedRow{Name: name, Placeholder: placeholder, Value: value})
}

func databaseResourceCategoryID(db *gorm.DB, userID uint, categoryIDs map[string]uint, connectionType string) (uint, error) {
	dbType := normalizeDatabaseResourceType(connectionType)
	if dbType != "" {
		if categoryID := categoryIDs[dbType]; categoryID != 0 {
			return categoryID, nil
		}
	}
	category, err := ensureScriptResourceCategory(db, userID, defaultDatabaseResourceCategoryName, "fixed")
	if err != nil {
		return 0, err
	}
	return category.ID, nil
}

func migrateLegacyDatabaseResourceConfigs(db *gorm.DB, userID uint, categoryIDs map[string]uint) error {
	var legacyCategory sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, defaultDatabaseResourceCategoryName).First(&legacyCategory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ?", userID, legacyCategory.ID).Order("id").Find(&configs).Error; err != nil {
		return err
	}
	for _, config := range configs {
		dbType := databaseResourceTypeFromRows(config.Rows)
		targetCategoryID := categoryIDs[dbType]
		if targetCategoryID == 0 {
			continue
		}
		if err := moveDatabaseResourceConfigToCategory(db, userID, config, targetCategoryID); err != nil {
			return err
		}
	}
	return removeLegacyDatabaseResourceCategoryIfEmpty(db, userID)
}

func moveDatabaseResourceConfigToCategory(db *gorm.DB, userID uint, config sysModel.TbScriptResourceConfig, targetCategoryID uint) error {
	configName := databaseResourceConfigNameFromRows(config.Rows, config.ConfigName)
	cleanedRows, err := normalizeDatabaseResourceRows(config.Rows, configName)
	if err != nil {
		cleanedRows = nil
	}
	placeholderKey := databaseResourcePlaceholderKeyFromRows(cleanedRows)
	var duplicate sysModel.TbScriptResourceConfig
	err = db.Where(
		"user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ? AND id <> ?",
		userID,
		targetCategoryID,
		config.WorkflowId,
		configName,
		config.ID,
	).First(&duplicate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		updates := map[string]any{
			"category_id": targetCategoryID,
			"config_name": configName,
		}
		if shouldUpdateDatabasePlaceholderKey(config.PlaceholderKey, placeholderKey, cleanedRows) {
			updates["placeholder_key"] = placeholderKey
		}
		if cleanedRows != nil {
			rowsJSON, marshalErr := json.Marshal(cleanedRows)
			if marshalErr != nil {
				return marshalErr
			}
			updates["rows"] = string(rowsJSON)
		}
		return db.Model(&config).Updates(updates).Error
	}
	if err != nil {
		return err
	}
	if cleanedRows != nil {
		if err := mergeDatabaseResourceConfigRows(db, duplicate, cleanedRows, configName, databaseResourcePlaceholderKeyFromRows(cleanedRows)); err != nil {
			return err
		}
	}
	if err := replaceResourceConfigReferences(db, userID, config.ID, duplicate.ID, targetCategoryID); err != nil {
		return err
	}
	return db.Delete(&config).Error
}

func dedupeDatabaseResourceConfigs(db *gorm.DB, userID uint, categoryIDs map[string]uint) error {
	for dbType, categoryID := range categoryIDs {
		if categoryID == 0 {
			continue
		}
		var configs []sysModel.TbScriptResourceConfig
		if err := db.Where("user_id = ? AND category_id = ?", userID, categoryID).Order("id").Find(&configs).Error; err != nil {
			return err
		}
		seen := map[string]sysModel.TbScriptResourceConfig{}
		for _, config := range configs {
			configName := databaseResourceConfigNameFromRows(config.Rows, config.ConfigName)
			cleanedRows, err := normalizeDatabaseResourceRows(config.Rows, configName)
			if err != nil {
				continue
			}
			key := databaseResourceDedupKey(dbType, databaseResourceHostFromRows(cleanedRows), fmt.Sprintf("config:%d", config.ID))
			if existing, ok := seen[key]; ok {
				if err := mergeDatabaseResourceConfigRows(db, existing, cleanedRows, configName, databaseResourcePlaceholderKeyFromRows(cleanedRows)); err != nil {
					return err
				}
				if err := replaceResourceConfigReferences(db, userID, config.ID, existing.ID, categoryID); err != nil {
					return err
				}
				if err := db.Delete(&config).Error; err != nil {
					return err
				}
				continue
			}
			seen[key] = config
			updates := map[string]any{}
			if config.ConfigName != configName {
				updates["config_name"] = configName
			}
			rowsJSON, err := json.Marshal(cleanedRows)
			if err != nil {
				return err
			}
			if string(rowsJSON) != strings.TrimSpace(config.Rows) {
				updates["rows"] = string(rowsJSON)
			}
			if len(updates) > 0 {
				if err := db.Model(&config).Updates(updates).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ensureDatabaseResourceConfig(db *gorm.DB, userID uint, categoryID uint, workflowID uint, configName string, rows []scriptResourceSeedRow) error {
	placeholderKey := databaseResourcePlaceholderKeyFromRows(rows)
	rows = pruneDatabaseResourceRows(rows, configName)
	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	var existing sysModel.TbScriptResourceConfig
	err = db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, workflowID, configName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&sysModel.TbScriptResourceConfig{
			UserId:         userID,
			CategoryId:     categoryID,
			WorkflowId:     workflowID,
			ConfigName:     configName,
			PlaceholderKey: placeholderKey,
			Rows:           string(rowsJSON),
		}).Error
	}
	if err != nil {
		return err
	}
	return mergeDatabaseResourceConfigRows(db, existing, rows, configName, placeholderKey)
}

func mergeDatabaseResourceConfigRows(db *gorm.DB, config sysModel.TbScriptResourceConfig, desiredRows []scriptResourceSeedRow, configName string, placeholderKey string) error {
	existingRows, err := scriptResourceSeedRowsFromJSON(config.Rows)
	if err != nil {
		existingRows = nil
	}
	if placeholderKey == "" {
		placeholderKey = databaseResourcePlaceholderKeyFromRows(existingRows)
	}
	mergedRows := mergeDatabaseResourceRows(existingRows, desiredRows, configName)
	mergedRowsJSON, err := json.Marshal(mergedRows)
	if err != nil {
		return err
	}
	updates := map[string]any{}
	if string(mergedRowsJSON) != strings.TrimSpace(config.Rows) {
		updates["rows"] = string(mergedRowsJSON)
	}
	if strings.TrimSpace(config.ConfigName) != strings.TrimSpace(configName) {
		updates["config_name"] = configName
	}
	if shouldUpdateDatabasePlaceholderKey(config.PlaceholderKey, placeholderKey, mergedRows) {
		updates["placeholder_key"] = strings.TrimSpace(placeholderKey)
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&config).Updates(updates).Error
}

func shouldUpdateDatabasePlaceholderKey(existing string, desired string, rows []scriptResourceSeedRow) bool {
	existing = strings.TrimSpace(existing)
	desired = strings.TrimSpace(desired)
	if desired == "" || existing == desired {
		return false
	}
	if existing == "" {
		return true
	}
	host := databaseResourceHostFromRows(rows)
	if existing == databaseResourceIdentifierForHost(host) {
		return true
	}
	parentKey := databaseResourceValueFromRows(rows, "PARENT_KEY")
	childKey := databaseResourceValueFromRows(rows, "CHILD_KEY")
	if parentKey != "" && childKey != "" && existing == databaseResourcePlaceholderKey(parentKey, childKey) {
		return true
	}
	return false
}

func mergeDatabaseResourceRows(existingRows []scriptResourceSeedRow, desiredRows []scriptResourceSeedRow, configName string) []scriptResourceSeedRow {
	mergedRows := pruneDatabaseResourceRows(existingRows, configName)
	known := make(map[string]int, len(mergedRows))
	for index, row := range mergedRows {
		if name := strings.ToUpper(strings.TrimSpace(row.Name)); name != "" {
			known[name] = index
		}
	}
	for _, row := range pruneDatabaseResourceRows(desiredRows, configName) {
		name := strings.ToUpper(strings.TrimSpace(row.Name))
		if name == "" {
			continue
		}
		if index, ok := known[name]; ok {
			if name == "NAME" {
				mergedRows[index].Value = configName
			}
			continue
		}
		mergedRows = append(mergedRows, row)
		known[name] = len(mergedRows) - 1
	}
	return mergedRows
}

func pruneDatabaseResourceRows(rows []scriptResourceSeedRow, configName string) []scriptResourceSeedRow {
	cleaned := make([]scriptResourceSeedRow, 0, len(rows))
	for _, row := range rows {
		name := strings.ToUpper(strings.TrimSpace(row.Name))
		if name == "DATABASE" || name == "PARENT_KEY" || name == "CHILD_KEY" ||
			name == "ROLE" ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "数据库名称") ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "父标识") ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "子标识") ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "配置用途") {
			continue
		}
		if name == "NAME" {
			row.Value = configName
		}
		cleaned = append(cleaned, row)
	}
	return cleaned
}

func normalizeDatabaseResourceRows(rawRows string, configName string) ([]scriptResourceSeedRow, error) {
	rows, err := scriptResourceSeedRowsFromJSON(rawRows)
	if err != nil {
		return nil, err
	}
	return pruneDatabaseResourceRows(rows, configName), nil
}

func replaceResourceConfigReferences(db *gorm.DB, userID uint, fromConfigID uint, toConfigID uint, toCategoryID uint) error {
	if fromConfigID == 0 || toConfigID == 0 || fromConfigID == toConfigID {
		return nil
	}
	var workflows []sysModel.TbScriptWorkflow
	if err := db.Select("id").Where("user_id = ?", userID).Find(&workflows).Error; err != nil {
		return err
	}
	workflowIDs := make([]uint, 0, len(workflows))
	for _, workflow := range workflows {
		workflowIDs = append(workflowIDs, workflow.ID)
	}
	if len(workflowIDs) == 0 {
		return nil
	}
	var steps []sysModel.TbScriptStep
	if err := db.Where("workflow_id IN ? AND placeholders LIKE ?", workflowIDs, "%"+uintToString(fromConfigID)+"%").Find(&steps).Error; err != nil {
		return err
	}
	for _, step := range steps {
		var placeholders []scriptSeedPlaceholder
		if err := json.Unmarshal([]byte(step.Placeholders), &placeholders); err != nil {
			continue
		}
		changed := false
		for index, placeholder := range placeholders {
			if strings.TrimSpace(placeholder.ValueKind) != "resource" {
				continue
			}
			if placeholder.ResourceConfigId != fromConfigID && strings.TrimSpace(placeholder.Value) != uintToString(fromConfigID) {
				continue
			}
			placeholders[index].ResourceCategoryId = toCategoryID
			placeholders[index].ResourceConfigId = toConfigID
			placeholders[index].Value = uintToString(toConfigID)
			changed = true
		}
		if !changed {
			continue
		}
		data, err := json.Marshal(placeholders)
		if err != nil {
			return err
		}
		if err := db.Model(&step).Update("placeholders", string(data)).Error; err != nil {
			return err
		}
	}
	return nil
}

func removeLegacyDatabaseResourceCategoryIfEmpty(db *gorm.DB, userID uint) error {
	var category sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, defaultDatabaseResourceCategoryName).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var count int64
	if err := db.Model(&sysModel.TbScriptResourceConfig{}).
		Where("user_id = ? AND category_id = ?", userID, category.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Delete(&category).Error
}

func databaseResourceTypeFromRows(rawRows string) string {
	rows, err := scriptResourceSeedRowsFromJSON(rawRows)
	if err != nil {
		return ""
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Name), "TYPE") ||
			strings.EqualFold(strings.TrimSpace(row.Placeholder), "数据库类型") {
			return normalizeDatabaseResourceType(row.Value)
		}
	}
	return ""
}

func databaseResourceRows(connection sysModel.TbConnection, configName string) []scriptResourceSeedRow {
	return []scriptResourceSeedRow{
		{Name: "NAME", Placeholder: "数据库配置名称", Value: configName},
		{Name: "TYPE", Placeholder: "数据库类型", Value: connection.ConnectionType},
		{Name: "HOST", Placeholder: "数据库地址", Value: connection.ConnectionUrl},
		{Name: "PORT", Placeholder: "数据库端口", Value: fmt.Sprintf("%d", connection.Port)},
		{Name: "USER", Placeholder: "数据库用户", Value: connection.DbLoginName},
		{Name: "PASSWORD", Placeholder: "数据库密码", Value: connection.DbLoginPassword},
		{Name: "GROUP", Placeholder: "数据库分组", Value: connection.ConnectionGroup},
		{Name: "ENV", Placeholder: "数据库环境", Value: connection.EnvName},
	}
}

func databaseResourceChildIdentifier(connectionType string) string {
	dbType := normalizeDatabaseResourceType(connectionType)
	if dbType != "" {
		return dbType
	}
	return strings.ToLower(strings.TrimSpace(connectionType))
}

func databaseResourcePlaceholderKeyFromRows(rows []scriptResourceSeedRow) string {
	parentKey := databaseResourceValueFromRows(rows, "PARENT_KEY")
	childKey := databaseResourceValueFromRows(rows, "CHILD_KEY")
	if strings.TrimSpace(parentKey) != "" && strings.TrimSpace(childKey) != "" {
		return databaseResourcePlaceholderKey(parentKey, childKey)
	}
	host := databaseResourceHostFromRows(rows)
	dbType := databaseResourceValueFromRows(rows, "TYPE")
	return databaseResourcePlaceholderKey(databaseResourceIdentifierForHost(host), databaseResourceChildIdentifier(dbType))
}

func databaseResourcePlaceholderKey(parentKey string, childKey string) string {
	parentKey = strings.TrimSpace(parentKey)
	childKey = strings.TrimSpace(childKey)
	if parentKey == "" {
		return ""
	}
	if childKey == "" {
		return parentKey
	}
	return parentKey + "_" + childKey
}

func defaultSourceDatabasePlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacBookProDatabaseConfigName, "postgresql")
}

func defaultTargetDatabasePlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacMiniDatabaseConfigName, "postgresql")
}

func defaultClickHouseSourceDatabasePlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacMiniDatabaseConfigName, "clickhouse")
}

func defaultClickHouseTargetDatabasePlaceholderKey() string {
	return databaseResourcePlaceholderKey(defaultMacBookProDatabaseConfigName, "clickhouse")
}

func databaseResourceConfigName(connection sysModel.TbConnection) string {
	return databaseResourceDisplayName(
		databaseResourceIdentifierForHost(connection.ConnectionUrl),
		connection.ConnectionName,
		connection.ConnectionUrl,
		displayName(connection.ConnectionName, "", fmt.Sprintf("数据库-%d", connection.ID)),
	)
}

func databaseResourceConfigNameFromRows(rawRows string, fallback string) string {
	rows, err := scriptResourceSeedRowsFromJSON(rawRows)
	if err != nil {
		return fallback
	}
	host := databaseResourceHostFromRows(rows)
	identifier := firstText(databaseResourceValueFromRows(rows, "PARENT_KEY"), databaseResourceIdentifierForHost(host))
	return databaseResourceDisplayName(
		identifier,
		databaseResourceValueFromRows(rows, "NAME"),
		host,
		fallback,
	)
}

func databaseResourceDisplayName(identifier string, label string, host string, fallback string) string {
	identifier = strings.TrimSpace(identifier)
	host = strings.TrimSpace(host)
	label = databaseResourceConfigLabel(label, host, identifier)
	if identifier != "" && host != "" {
		return identifier + " / " + host
	}
	if identifier != "" {
		return identifier
	}
	if host != "" && label != "" {
		return label + " / " + host
	}
	if host != "" {
		return host
	}
	if label != "" {
		return label
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "数据库配置"
}

func databaseResourceIdentifierForHost(host string) string {
	return resourceIdentifierForHost(host)
}

func resourceIdentifierForHost(host string) string {
	normalizedHost := normalizeDatabaseResourceHost(host)
	switch normalizeDatabaseResourceHost(host) {
	case "127.0.0.1":
		return defaultMacBookProDatabaseConfigName
	case defaultMacMiniDatabaseHost:
		return defaultMacMiniDatabaseConfigName
	case defaultTencentServerHost:
		return defaultTencentServerConfigName
	default:
		if normalizedHost != "" {
			return "ip_" + strings.ReplaceAll(normalizedHost, ".", "_")
		}
		return ""
	}
}

func normalizeSeedPlaceholderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func resourceDisplayName(identifier string, host string, fallback string) string {
	identifier = strings.TrimSpace(identifier)
	host = strings.TrimSpace(host)
	if identifier != "" && host != "" {
		return identifier + " / " + host
	}
	if identifier != "" {
		return identifier
	}
	if host != "" {
		return host
	}
	return strings.TrimSpace(fallback)
}

func databaseResourceConfigLabel(label string, host string, identifiers ...string) string {
	label = trimDatabaseResourceHostSuffix(strings.TrimSpace(label), strings.TrimSpace(host))
	return trimDatabaseResourceIdentifierPrefix(label, identifiers...)
}

func trimDatabaseResourceIdentifierPrefix(label string, identifiers ...string) string {
	for _, identifier := range append(identifiers, defaultMacBookProDatabaseConfigName, defaultMacMiniDatabaseConfigName) {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			continue
		}
		prefix := identifier + " / "
		if strings.HasPrefix(label, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(label, prefix))
		}
	}
	return label
}

func trimDatabaseResourceHostSuffix(label string, host string) string {
	if label == "" || host == "" {
		return label
	}
	for _, sep := range []string{" / ", "/", " - ", "-"} {
		suffix := sep + host
		if strings.HasSuffix(label, suffix) {
			return strings.TrimSpace(strings.TrimSuffix(label, suffix))
		}
	}
	return label
}

func databaseResourceHostFromRows(rows []scriptResourceSeedRow) string {
	return databaseResourceValueFromRows(rows, "HOST")
}

func databaseResourceValueFromRows(rows []scriptResourceSeedRow, name string) string {
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Name), name) {
			return strings.TrimSpace(row.Value)
		}
	}
	return ""
}

func databaseResourceDedupKey(connectionType string, host string, fallback string) string {
	dbType := normalizeDatabaseResourceType(connectionType)
	if dbType == "" {
		dbType = strings.ToLower(strings.TrimSpace(connectionType))
	}
	host = normalizeDatabaseResourceHost(host)
	if host == "" {
		host = strings.ToLower(strings.TrimSpace(fallback))
	}
	return dbType + "|" + host
}

func normalizeDatabaseResourceHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if slashIndex := strings.Index(host, "/"); slashIndex >= 0 {
		host = host[:slashIndex]
	}
	if strings.Count(host, ":") == 1 {
		if colonIndex := strings.LastIndex(host, ":"); colonIndex > 0 {
			host = host[:colonIndex]
		}
	}
	switch host {
	case "localhost", "::1", "0.0.0.0":
		return "127.0.0.1"
	default:
		return host
	}
}

func scriptResourceSeedRowsFromJSON(rawRows string) ([]scriptResourceSeedRow, error) {
	if strings.TrimSpace(rawRows) == "" {
		return nil, nil
	}
	var rows []scriptResourceSeedRow
	if err := json.Unmarshal([]byte(rawRows), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func databaseResourceCategoryNameForType(connectionType string) (string, bool) {
	dbType := normalizeDatabaseResourceType(connectionType)
	for _, spec := range databaseResourceCategorySpecs {
		if spec.DBType == dbType {
			return spec.CategoryName, true
		}
	}
	return "", false
}

func normalizeDatabaseResourceType(connectionType string) string {
	switch strings.ToLower(strings.TrimSpace(connectionType)) {
	case "oracle", "orcl":
		return "oracle"
	case "pgsql", "postgres", "postgresql", "pg":
		return "postgresql"
	case "clickhouse", "click-house", "click_house", "ch":
		return "clickhouse"
	case "mysql", "mariadb":
		return "mysql"
	default:
		return ""
	}
}

func seedPathConstantsResourceConfig(db *gorm.DB, userID uint, categoryID uint) error {
	rows := defaultPathConstantRows()
	if err := db.Model(&sysModel.TbScriptResourceConfig{}).
		Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, 0, legacyPathConstantsConfigName).
		Update("config_name", defaultPathConstantsConfigName).Error; err != nil {
		return err
	}
	if err := ensureScriptResourceConfig(db, userID, categoryID, defaultPathConstantsConfigName, rows); err != nil {
		return err
	}
	return migratePathConstantDefaultRows(db, userID, categoryID, rows)
}

func defaultPathConstantRows() []scriptResourceSeedRow {
	return []scriptResourceSeedRow{
		{Name: "EXPORT_ROOT", Placeholder: "本地导出目录", Value: "/tmp/db-migrate/export"},
		{Name: "LOCAL_ENV", Placeholder: "本地导出流程清单", Value: "/tmp/db-migrate/export/latest.env"},
		{Name: "TARGET_MANIFEST", Placeholder: "本地上传清单路径", Value: "/tmp/db-migrate/export/target_latest.env"},
		{Name: "REMOTE_INBOX", Placeholder: "目标服务器接收目录", Value: "/tmp/db-migrate/inbox"},
		{Name: "REMOTE_WORKDIR", Placeholder: "目标服务器导入工作目录", Value: "/tmp/db-migrate/restore"},
		{Name: "REMOTE_MANIFEST", Placeholder: "目标服务器上传清单路径", Value: "/tmp/db-migrate/inbox/target_latest.env"},
		{Name: "RESTORE_ENV", Placeholder: "目标服务器导入流程清单", Value: "/tmp/db-migrate/restore/latest.env"},
	}
}

func migratePathConstantDefaultRows(db *gorm.DB, userID uint, categoryID uint, desiredRows []scriptResourceSeedRow) error {
	var config sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, 0, defaultPathConstantsConfigName).First(&config).Error; err != nil {
		return err
	}
	var existingRows []scriptResourceSeedRow
	if err := json.Unmarshal([]byte(config.Rows), &existingRows); err != nil {
		return nil
	}
	desiredByName := make(map[string]scriptResourceSeedRow, len(desiredRows))
	for _, row := range desiredRows {
		desiredByName[strings.ToUpper(strings.TrimSpace(row.Name))] = row
	}
	changed := false
	for index, row := range existingRows {
		name := strings.ToUpper(strings.TrimSpace(row.Name))
		desired, ok := desiredByName[name]
		if !ok {
			continue
		}
		if row.Value == "" || row.Value == legacyPathConstantDefaultValue(name) {
			existingRows[index].Value = desired.Value
			changed = true
		}
		if strings.TrimSpace(row.Placeholder) == "" {
			existingRows[index].Placeholder = desired.Placeholder
			changed = true
		}
	}
	if !changed {
		return nil
	}
	data, err := json.Marshal(existingRows)
	if err != nil {
		return err
	}
	return db.Model(&config).Update("rows", string(data)).Error
}

func legacyPathConstantDefaultValue(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "EXPORT_ROOT":
		return "/tmp/easy-deploy-db-export"
	case "LOCAL_ENV":
		return "/tmp/easy-deploy-db-export/latest.env"
	case "TARGET_MANIFEST":
		return "/tmp/easy-deploy-db-export/target_latest.env"
	case "REMOTE_INBOX":
		return "/tmp/easy-deploy-db-inbox"
	case "REMOTE_WORKDIR":
		return "/tmp/easy-deploy-db-restore"
	case "REMOTE_MANIFEST":
		return "/tmp/easy-deploy-db-inbox/target_latest.env"
	case "RESTORE_ENV":
		return "/tmp/easy-deploy-db-restore/latest.env"
	default:
		return ""
	}
}

func seedDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DROP_TABLES_BEFORE_IMPORT", Placeholder: "导入前删除目标数据库所有表", Value: "true"},
		{Name: "SOURCE_PG_DUMP_CMD", Placeholder: "源库 pg_dump 命令", Value: "/usr/local/bin/docker exec postgres16 pg_dump"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_PSQL_CMD", Placeholder: "目标服务器 psql 命令", Value: "/usr/local/bin/docker exec -i postgres16 psql"},
		{Name: "TARGET_PG_RESTORE_CMD", Placeholder: "目标服务器 pg_restore 命令", Value: "/usr/local/bin/docker exec -i postgres16 pg_restore"},
	}
	configName := "数据库导出执行参数"
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

func seedTableDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint) error {
	rows := []scriptResourceSeedRow{
		{Name: "PROJECT_NAME", Placeholder: "单表导出任务名称", Value: "postgres-table-export"},
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源数据库名称", Value: "easy_deploy"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标数据库名称", Value: "easy_deploy"},
		{Name: "TABLE_SCHEMA", Placeholder: "表 Schema", Value: "public"},
		{Name: "TABLE_NAME", Placeholder: "表名称", Value: ""},
		{Name: "TARGET_TABLE_DROP_BEFORE_IMPORT", Placeholder: "导入前删除目标表", Value: "true"},
		{Name: "SOURCE_PG_DUMP_CMD", Placeholder: "源库 pg_dump 命令", Value: "/usr/local/bin/docker exec postgres16 pg_dump"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_PSQL_CMD", Placeholder: "目标服务器 psql 命令", Value: "/usr/local/bin/docker exec -i postgres16 psql"},
		{Name: "TARGET_PG_RESTORE_CMD", Placeholder: "目标服务器 pg_restore 命令", Value: "/usr/local/bin/docker exec -i postgres16 pg_restore"},
	}
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, defaultTableDynamicParamsConfigName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
		"TARGET_DB_DROP_TABLES_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		defaultTableDynamicParamsConfigName,
		removeNames...,
	)
}

func seedClickHouseDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源 ClickHouse 数据库名称", Value: "default"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标 ClickHouse 数据库名称", Value: "default"},
		{Name: "TARGET_DB_DROP_TABLES_BEFORE_IMPORT", Placeholder: "导入前删除目标 ClickHouse 数据库所有表", Value: "true"},
		{Name: "SOURCE_CLICKHOUSE_HTTP_CMD", Placeholder: "源库 ClickHouse HTTP 命令", Value: "/usr/bin/curl"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_CLICKHOUSE_HTTP_CMD", Placeholder: "目标服务器 ClickHouse HTTP 命令", Value: "/usr/bin/curl"},
	}
	configName := "ClickHouse 数据库导出执行参数"
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

func seedClickHouseTableDynamicParamsResourceConfigs(db *gorm.DB, userID uint, categoryID uint, workflowID uint, projectNameRow scriptResourceSeedRow) error {
	rows := []scriptResourceSeedRow{
		projectNameRow,
		{Name: "SOURCE_DB_DATABASE", Placeholder: "源 ClickHouse 数据库名称", Value: "default"},
		{Name: "TARGET_DB_DATABASE", Placeholder: "目标 ClickHouse 数据库名称", Value: "default"},
		{Name: "TABLE_NAME", Placeholder: "ClickHouse 表名称", Value: ""},
		{Name: "TARGET_TABLE_DROP_BEFORE_IMPORT", Placeholder: "导入前删除目标 ClickHouse 表", Value: "true"},
		{Name: "SOURCE_CLICKHOUSE_HTTP_CMD", Placeholder: "源库 ClickHouse HTTP 命令", Value: "/usr/bin/curl"},
		{Name: "LOCAL_EXPECT_CMD", Placeholder: "本地 expect 命令", Value: "/usr/bin/expect"},
		{Name: "TARGET_CLICKHOUSE_HTTP_CMD", Placeholder: "目标服务器 ClickHouse HTTP 命令", Value: "/usr/bin/curl"},
	}
	configName := "ClickHouse 单表导出执行参数"
	if err := ensureScriptResourceConfigForWorkflow(db, userID, categoryID, workflowID, configName, "", rows); err != nil {
		return err
	}
	removeNames := append([]string{
		"SOURCE_DB_KEY",
		"TARGET_DB_KEY",
		"TARGET_SERVER_KEY",
		"TARGET_DB_RESET_BEFORE_IMPORT",
		"TARGET_DB_DROP_TABLES_BEFORE_IMPORT",
	}, pathConstantResourceRowNames...)
	return removeResourceConfigRowsByName(
		db,
		userID,
		categoryID,
		workflowID,
		configName,
		removeNames...,
	)
}

type scriptResourceSeedRow struct {
	Name        string `json:"name"`
	Placeholder string `json:"placeholder"`
	Value       string `json:"value"`
}

func ensureScriptResourceConfig(db *gorm.DB, userID uint, categoryID uint, configName string, rows []scriptResourceSeedRow) error {
	return ensureScriptResourceConfigForWorkflow(db, userID, categoryID, 0, configName, "", rows)
}

func ensureScriptResourceConfigWithPlaceholderKey(db *gorm.DB, userID uint, categoryID uint, configName string, placeholderKey string, rows []scriptResourceSeedRow) error {
	return ensureScriptResourceConfigForWorkflow(db, userID, categoryID, 0, configName, placeholderKey, rows)
}

func ensureScriptResourceConfigForWorkflow(db *gorm.DB, userID uint, categoryID uint, workflowID uint, configName string, placeholderKey string, rows []scriptResourceSeedRow) error {
	placeholderKey = strings.TrimSpace(placeholderKey)
	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	var existing sysModel.TbScriptResourceConfig
	err = db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, workflowID, configName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && placeholderKey != "" {
		err = db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND placeholder_key = ?", userID, categoryID, workflowID, placeholderKey).First(&existing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		existingConfig, ok, lookupErr := findResourceConfigByHost(db, userID, categoryID, workflowID, rows)
		if lookupErr != nil {
			return lookupErr
		}
		if ok {
			existing = existingConfig
			err = nil
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&sysModel.TbScriptResourceConfig{
			UserId:         userID,
			CategoryId:     categoryID,
			WorkflowId:     workflowID,
			ConfigName:     configName,
			PlaceholderKey: placeholderKey,
			Rows:           string(rowsJSON),
		}).Error
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(existing.Rows) == "" {
		updates := map[string]any{
			"workflow_id": workflowID,
			"config_name": configName,
			"rows":        string(rowsJSON),
		}
		if placeholderKey != "" {
			updates["placeholder_key"] = placeholderKey
		}
		return db.Model(&existing).Updates(updates).Error
	}
	var existingRows []scriptResourceSeedRow
	if err := json.Unmarshal([]byte(existing.Rows), &existingRows); err != nil {
		return nil
	}
	removedRoleRow := false
	if workflowID == 0 || placeholderKey != "" {
		cleanedExistingRows := make([]scriptResourceSeedRow, 0, len(existingRows))
		for _, row := range existingRows {
			if strings.EqualFold(strings.TrimSpace(row.Name), "ROLE") ||
				strings.EqualFold(strings.TrimSpace(row.Placeholder), "配置用途") {
				removedRoleRow = true
				continue
			}
			cleanedExistingRows = append(cleanedExistingRows, row)
		}
		existingRows = cleanedExistingRows
	}
	known := make(map[string]int, len(existingRows))
	for index, row := range existingRows {
		if name := strings.ToUpper(strings.TrimSpace(row.Name)); name != "" {
			known[name] = index
		}
	}
	changed := removedRoleRow
	for _, row := range rows {
		name := strings.ToUpper(strings.TrimSpace(row.Name))
		if name == "" {
			continue
		}
		if name == "ROLE" {
			changed = true
			continue
		}
		if _, ok := known[name]; ok {
			continue
		}
		existingRows = append(existingRows, row)
		known[name] = len(existingRows) - 1
		changed = true
	}
	if !changed {
		updates := map[string]any{}
		if strings.TrimSpace(existing.ConfigName) != strings.TrimSpace(configName) {
			updates["config_name"] = configName
		}
		if strings.TrimSpace(existing.PlaceholderKey) == "" && placeholderKey != "" {
			updates["placeholder_key"] = placeholderKey
		}
		if len(updates) == 0 {
			return nil
		}
		return db.Model(&existing).Updates(updates).Error
	}
	mergedRowsJSON, err := json.Marshal(existingRows)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"workflow_id": workflowID,
		"config_name": configName,
		"rows":        string(mergedRowsJSON),
	}
	if strings.TrimSpace(existing.PlaceholderKey) == "" && placeholderKey != "" {
		updates["placeholder_key"] = placeholderKey
	}
	return db.Model(&existing).Updates(updates).Error
}

func removeResourceConfigRowsByName(db *gorm.DB, userID uint, categoryID uint, workflowID uint, configName string, names ...string) error {
	if len(names) == 0 {
		return nil
	}
	removeNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		if normalized := strings.ToUpper(strings.TrimSpace(name)); normalized != "" {
			removeNames[normalized] = struct{}{}
		}
	}
	var config sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND workflow_id = ? AND config_name = ?", userID, categoryID, workflowID, configName).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	rows, err := scriptResourceSeedRowsFromJSON(config.Rows)
	if err != nil {
		return nil
	}
	kept := make([]scriptResourceSeedRow, 0, len(rows))
	changed := false
	for _, row := range rows {
		name := strings.ToUpper(strings.TrimSpace(row.Name))
		if _, ok := removeNames[name]; ok {
			changed = true
			continue
		}
		kept = append(kept, row)
	}
	if !changed {
		return nil
	}
	data, err := json.Marshal(kept)
	if err != nil {
		return err
	}
	return db.Model(&config).Update("rows", string(data)).Error
}

func findResourceConfigByHost(db *gorm.DB, userID uint, categoryID uint, workflowID uint, rows []scriptResourceSeedRow) (sysModel.TbScriptResourceConfig, bool, error) {
	host := normalizeDatabaseResourceHost(resourceHostFromRows(rows))
	if host == "" {
		return sysModel.TbScriptResourceConfig{}, false, nil
	}
	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ? AND workflow_id = ?", userID, categoryID, workflowID).Order("id").Find(&configs).Error; err != nil {
		return sysModel.TbScriptResourceConfig{}, false, err
	}
	for _, config := range configs {
		configRows, err := scriptResourceSeedRowsFromJSON(config.Rows)
		if err != nil {
			continue
		}
		if normalizeDatabaseResourceHost(resourceHostFromRows(configRows)) == host {
			return config, true, nil
		}
	}
	return sysModel.TbScriptResourceConfig{}, false, nil
}

func resourceHostFromRows(rows []scriptResourceSeedRow) string {
	if value := databaseResourceValueFromRows(rows, "IP"); value != "" {
		return value
	}
	return databaseResourceValueFromRows(rows, "HOST")
}

func deployProjectNameSeedRow(db *gorm.DB, userID uint) scriptResourceSeedRow {
	row := scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "导出任务名称", Value: "postgres-db-export"}
	var category sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, defaultDeployResourceCategoryName).First(&category).Error
	if err != nil {
		return row
	}
	var configs []sysModel.TbScriptResourceConfig
	if err := db.Where("user_id = ? AND category_id = ?", userID, category.ID).Order("id").Find(&configs).Error; err != nil {
		return row
	}
	for _, config := range configs {
		if strings.TrimSpace(config.Rows) == "" {
			continue
		}
		var rows []scriptResourceSeedRow
		if err := json.Unmarshal([]byte(config.Rows), &rows); err != nil {
			continue
		}
		for _, existingRow := range rows {
			if strings.EqualFold(strings.TrimSpace(existingRow.Name), "PROJECT_NAME") {
				if strings.TrimSpace(existingRow.Placeholder) != "" {
					row.Placeholder = existingRow.Placeholder
				}
				if strings.TrimSpace(existingRow.Value) != "" {
					row.Value = existingRow.Value
				}
				return row
			}
		}
	}
	return row
}

func reverseDeployProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "反向导出任务名称", Value: "postgres-db-export-to-local"}
}

func reverseTableProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "反向单表导出任务名称", Value: "postgres-table-export-to-local"}
}

func mysqlProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "MySQL 导出任务名称", Value: "mysql-db-export"}
}

func mysqlTableProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "MySQL 单表导出任务名称", Value: "mysql-table-export"}
}

func mysqlReverseProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "MySQL 反向导出任务名称", Value: "mysql-db-export-to-local"}
}

func mysqlReverseTableProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "MySQL 反向单表导出任务名称", Value: "mysql-table-export-to-local"}
}

func clickHouseProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "ClickHouse 导出任务名称", Value: "clickhouse-db-export"}
}

func clickHouseTableProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "ClickHouse 单表导出任务名称", Value: "clickhouse-table-export"}
}

func clickHouseReverseProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "ClickHouse 反向导出任务名称", Value: "clickhouse-db-export-to-mac-mini"}
}

func clickHouseReverseTableProjectNameSeedRow() scriptResourceSeedRow {
	return scriptResourceSeedRow{Name: "PROJECT_NAME", Placeholder: "ClickHouse 反向单表导出任务名称", Value: "clickhouse-table-export-to-mac-mini"}
}

func removeDefaultDeployResourceConfigs(db *gorm.DB, userID uint) error {
	var category sysModel.TbScriptResourceCategory
	err := db.Where("user_id = ? AND category_name = ?", userID, defaultDeployResourceCategoryName).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := db.Where("user_id = ? AND category_id = ? AND config_name IN ?", userID, category.ID, []string{
		defaultDeployResourceConfigName,
		legacyDeployResourceConfigName,
	}).Delete(&sysModel.TbScriptResourceConfig{}).Error; err != nil {
		return err
	}
	var count int64
	if err := db.Model(&sysModel.TbScriptResourceConfig{}).
		Where("user_id = ? AND category_id = ?", userID, category.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Delete(&category).Error
}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func displayName(primary string, secondary string, fallback string) string {
	if primary != "" && secondary != "" {
		return primary + " / " + secondary
	}
	return firstText(primary, secondary, fallback)
}

type scriptSeedPlaceholder struct {
	Placeholder        string `json:"placeholder"`
	Name               string `json:"name"`
	ValueKind          string `json:"valueKind"`
	Value              string `json:"value"`
	ResourceCategoryId uint   `json:"resourceCategoryId,omitempty"`
	ResourceConfigId   uint   `json:"resourceConfigId,omitempty"`
	CustomValue        string `json:"customValue,omitempty"`
}

func defaultExportPlaceholders(databaseResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("SOURCE_DB", "源数据库配置", databaseResourceID),
		}, placeholders...)
	} else if connectionID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			connectionSeedPlaceholder("SOURCE_DB", "源数据库配置", connectionID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	placeholders = appendDynamicManualPlaceholder(placeholders, dynamicCategoryID, dynamicConfigID, "SOURCE_DB_DATABASE", "源数据库名称", "easy_deploy")
	return marshalSeedPlaceholders(placeholders)
}

func defaultTableExportPlaceholders(databaseResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("SOURCE_DB", "源数据库配置", databaseResourceID),
		}, placeholders...)
	} else if connectionID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			connectionSeedPlaceholder("SOURCE_DB", "源数据库配置", connectionID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultUploadPlaceholders(serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultTargetDownloadPlaceholders(serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultTargetExecPlaceholders(databaseResourceID uint, serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_DB", "目标数据库配置", databaseResourceID),
		}, placeholders...)
	} else if connectionID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			connectionSeedPlaceholder("TARGET_DB", "目标数据库配置", connectionID),
		}, placeholders...)
	}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	placeholders = appendDynamicManualPlaceholder(placeholders, dynamicCategoryID, dynamicConfigID, "TARGET_DB_DATABASE", "目标数据库名称", "easy_deploy")
	return marshalSeedPlaceholders(placeholders)
}

func defaultRemoteExportPlaceholders(databaseResourceID uint, serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("SOURCE_DB", "源数据库配置", databaseResourceID))
	} else if connectionID != 0 {
		placeholders = append(placeholders, connectionSeedPlaceholder("SOURCE_DB", "源数据库配置", connectionID))
	}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("SOURCE_SERVER", "源服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("SOURCE_SERVER", "源服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	placeholders = appendDynamicManualPlaceholder(placeholders, dynamicCategoryID, dynamicConfigID, "SOURCE_DB_DATABASE", "源数据库名称", "easy_deploy")
	return marshalSeedPlaceholders(placeholders)
}

func defaultRemoteTableExportPlaceholders(databaseResourceID uint, serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("SOURCE_DB", "源数据库配置", databaseResourceID))
	} else if connectionID != 0 {
		placeholders = append(placeholders, connectionSeedPlaceholder("SOURCE_DB", "源数据库配置", connectionID))
	}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("SOURCE_SERVER", "源服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("SOURCE_SERVER", "源服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultTableTargetExecPlaceholders(databaseResourceID uint, serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint, connectionID uint, serverID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_DB", "目标数据库配置", databaseResourceID),
		}, placeholders...)
	} else if connectionID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			connectionSeedPlaceholder("TARGET_DB", "目标数据库配置", connectionID),
		}, placeholders...)
	}
	if serverResourceID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			resourceSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverResourceID),
		}, placeholders...)
	} else if serverID != 0 {
		placeholders = append([]scriptSeedPlaceholder{
			serverSeedPlaceholder("TARGET_SERVER", "目标服务器配置", serverID),
		}, placeholders...)
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultClickHouseExportPlaceholders(databaseResourceID uint, sourceServerResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("SOURCE_DB", "源 ClickHouse 数据库配置（mac mini，只读导出）", databaseResourceID))
	}
	if sourceServerResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("SOURCE_SERVER", "源服务器配置（mac mini，Tailscale SSH）", sourceServerResourceID))
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func defaultClickHouseTargetExecPlaceholders(databaseResourceID uint, serverResourceID uint, dynamicCategoryID uint, dynamicConfigID uint, pathConstantsConfigID uint) string {
	placeholders := []scriptSeedPlaceholder{}
	if databaseResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("TARGET_DB", "目标 ClickHouse 数据库配置（本机）", databaseResourceID))
	}
	if serverResourceID != 0 {
		placeholders = append(placeholders, resourceSeedPlaceholder("TARGET_SERVER", "目标服务器配置（本机）", serverResourceID))
	}
	placeholders = appendExecutionParamsPlaceholder(placeholders, dynamicConfigID)
	placeholders = appendPathConstantsPlaceholder(placeholders, pathConstantsConfigID)
	return marshalSeedPlaceholders(placeholders)
}

func appendExecutionParamsPlaceholder(placeholders []scriptSeedPlaceholder, dynamicConfigID uint) []scriptSeedPlaceholder {
	if dynamicConfigID == 0 {
		return placeholders
	}
	return append(placeholders, resourceSeedPlaceholder("EXEC_PARAMS", "数据库导出执行参数", dynamicConfigID))
}

func appendPathConstantsPlaceholder(placeholders []scriptSeedPlaceholder, pathConstantsConfigID uint) []scriptSeedPlaceholder {
	if pathConstantsConfigID == 0 {
		return placeholders
	}
	return append(placeholders, resourceSeedPlaceholder("PATH_CONSTANTS", defaultPathConstantsCategoryName, pathConstantsConfigID))
}

func appendDynamicManualPlaceholder(placeholders []scriptSeedPlaceholder, dynamicCategoryID uint, dynamicConfigID uint, name string, description string, value string) []scriptSeedPlaceholder {
	if dynamicCategoryID == 0 || dynamicConfigID == 0 {
		return placeholders
	}
	return append(placeholders, scriptSeedPlaceholder{
		Placeholder:        description,
		Name:               name,
		ValueKind:          "manual",
		Value:              value,
		ResourceCategoryId: dynamicCategoryID,
		ResourceConfigId:   dynamicConfigID,
	})
}

func connectionSeedPlaceholder(name string, description string, connectionID uint) scriptSeedPlaceholder {
	return scriptSeedPlaceholder{Placeholder: description, Name: name, ValueKind: "connection", Value: uintToString(connectionID)}
}

func serverSeedPlaceholder(name string, description string, serverID uint) scriptSeedPlaceholder {
	return scriptSeedPlaceholder{Placeholder: description, Name: name, ValueKind: "server", Value: uintToString(serverID)}
}

func resourceSeedPlaceholder(name string, description string, resourceConfigID uint) scriptSeedPlaceholder {
	return scriptSeedPlaceholder{
		Placeholder:      description,
		Name:             name,
		ValueKind:        "resource",
		ResourceConfigId: resourceConfigID,
		Value:            uintToString(resourceConfigID),
		CustomValue:      resourceRoleForPlaceholderName(name),
	}
}

func resourceRoleForPlaceholderName(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "SOURCE_DB", "SOURCE_SERVER":
		return "source"
	case "TARGET_DB", "TARGET_SERVER":
		return "target"
	default:
		return ""
	}
}

func marshalSeedPlaceholders(placeholders []scriptSeedPlaceholder) string {
	data, err := json.Marshal(placeholders)
	if err != nil {
		return ""
	}
	return string(data)
}

func mergeSeedPlaceholders(existingJSON string, desiredJSON string) (string, bool) {
	if strings.TrimSpace(existingJSON) == "" || strings.TrimSpace(desiredJSON) == "" {
		return "", false
	}
	var existing []scriptSeedPlaceholder
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return "", false
	}
	var desired []scriptSeedPlaceholder
	if err := json.Unmarshal([]byte(desiredJSON), &desired); err != nil {
		return "", false
	}
	changed := false
	if pruned, ok := pruneDeprecatedDeployFlowPlaceholder(existing, desired); ok {
		existing = pruned
		changed = true
	}
	if upgraded, ok := upgradeDynamicManualPlaceholders(existing, desired); ok {
		existing = upgraded
		changed = true
	}
	if upgraded, ok := upgradeResourcePlaceholders(existing, desired); ok {
		existing = upgraded
		changed = true
	}
	if pruned, ok := pruneExtractedManualPlaceholders(existing, desired); ok {
		existing = pruned
		changed = true
	}
	known := make(map[string]struct{}, len(existing))
	for _, placeholder := range existing {
		if key := seedPlaceholderKey(placeholder); key != "" {
			known[key] = struct{}{}
		}
	}
	for _, placeholder := range desired {
		key := seedPlaceholderKey(placeholder)
		if key == "" {
			continue
		}
		if _, ok := known[key]; ok {
			continue
		}
		existing = append(existing, placeholder)
		known[key] = struct{}{}
		changed = true
	}
	if !changed {
		return "", false
	}
	data, err := json.Marshal(existing)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func seedPlaceholderKey(placeholder scriptSeedPlaceholder) string {
	if name := strings.ToUpper(strings.TrimSpace(placeholder.Name)); name != "" {
		return "name:" + name
	}
	if label := strings.TrimSpace(placeholder.Placeholder); label != "" {
		return "label:" + label
	}
	if strings.TrimSpace(placeholder.ValueKind) == "resource" && placeholder.ResourceConfigId != 0 {
		return fmt.Sprintf("resource:%d", placeholder.ResourceConfigId)
	}
	return ""
}

func upgradeDynamicManualPlaceholders(existing []scriptSeedPlaceholder, desired []scriptSeedPlaceholder) ([]scriptSeedPlaceholder, bool) {
	desiredDynamic := map[string]scriptSeedPlaceholder{}
	for _, placeholder := range desired {
		if strings.TrimSpace(placeholder.ValueKind) != "manual" || placeholder.ResourceCategoryId == 0 {
			continue
		}
		if name := strings.ToUpper(strings.TrimSpace(placeholder.Name)); name != "" {
			desiredDynamic[name] = placeholder
		}
	}
	if len(desiredDynamic) == 0 {
		return existing, false
	}
	changed := false
	for index, placeholder := range existing {
		if strings.TrimSpace(placeholder.ValueKind) != "manual" || placeholder.ResourceCategoryId != 0 {
			continue
		}
		name := strings.ToUpper(strings.TrimSpace(placeholder.Name))
		desiredPlaceholder, ok := desiredDynamic[name]
		if !ok {
			continue
		}
		existing[index].Placeholder = desiredPlaceholder.Placeholder
		existing[index].ResourceCategoryId = desiredPlaceholder.ResourceCategoryId
		existing[index].ResourceConfigId = desiredPlaceholder.ResourceConfigId
		if strings.TrimSpace(existing[index].Value) == "" {
			existing[index].Value = desiredPlaceholder.Value
		}
		changed = true
	}
	return existing, changed
}

func upgradeResourcePlaceholders(existing []scriptSeedPlaceholder, desired []scriptSeedPlaceholder) ([]scriptSeedPlaceholder, bool) {
	desiredResources := map[string]scriptSeedPlaceholder{}
	for _, placeholder := range desired {
		if strings.TrimSpace(placeholder.ValueKind) != "resource" {
			continue
		}
		if name := strings.ToUpper(strings.TrimSpace(placeholder.Name)); name != "" {
			desiredResources[name] = placeholder
		}
	}
	if len(desiredResources) == 0 {
		return existing, false
	}
	changed := false
	for index, placeholder := range existing {
		if strings.TrimSpace(placeholder.ValueKind) != "resource" {
			continue
		}
		name := strings.ToUpper(strings.TrimSpace(placeholder.Name))
		desiredPlaceholder, ok := desiredResources[name]
		if !ok {
			continue
		}
		if existing[index].ResourceConfigId != desiredPlaceholder.ResourceConfigId ||
			existing[index].ResourceCategoryId != desiredPlaceholder.ResourceCategoryId ||
			strings.TrimSpace(existing[index].Value) != strings.TrimSpace(desiredPlaceholder.Value) {
			existing[index].Placeholder = desiredPlaceholder.Placeholder
			existing[index].ResourceCategoryId = desiredPlaceholder.ResourceCategoryId
			existing[index].ResourceConfigId = desiredPlaceholder.ResourceConfigId
			existing[index].Value = desiredPlaceholder.Value
			changed = true
		}
		if strings.TrimSpace(existing[index].CustomValue) == "" && strings.TrimSpace(desiredPlaceholder.CustomValue) != "" {
			existing[index].CustomValue = desiredPlaceholder.CustomValue
			changed = true
		}
	}
	return existing, changed
}

func pruneDeprecatedDeployFlowPlaceholder(existing []scriptSeedPlaceholder, desired []scriptSeedPlaceholder) ([]scriptSeedPlaceholder, bool) {
	for _, placeholder := range desired {
		if strings.EqualFold(strings.TrimSpace(placeholder.Name), "DEPLOY_FLOW") {
			return existing, false
		}
	}
	kept := make([]scriptSeedPlaceholder, 0, len(existing))
	changed := false
	for _, placeholder := range existing {
		if strings.EqualFold(strings.TrimSpace(placeholder.Name), "DEPLOY_FLOW") &&
			strings.TrimSpace(placeholder.ValueKind) == "resource" {
			changed = true
			continue
		}
		kept = append(kept, placeholder)
	}
	return kept, changed
}

func pruneExtractedManualPlaceholders(existing []scriptSeedPlaceholder, desired []scriptSeedPlaceholder) ([]scriptSeedPlaceholder, bool) {
	desiredNames := map[string]struct{}{}
	for _, placeholder := range desired {
		if name := strings.ToUpper(strings.TrimSpace(placeholder.Name)); name != "" {
			desiredNames[name] = struct{}{}
		}
	}
	_, hasDeployConfig := desiredNames["DEPLOY_FLOW"]
	_, hasSourceDBConfig := desiredNames["SOURCE_DB"]
	_, hasTargetDBConfig := desiredNames["TARGET_DB"]
	_, hasExecParamsConfig := desiredNames["EXEC_PARAMS"]
	extractedDeployNames := map[string]struct{}{}
	for _, name := range []string{"PROJECT_NAME", "EXPORT_ROOT", "LOCAL_ENV", "TARGET_MANIFEST", "REMOTE_INBOX", "REMOTE_WORKDIR", "REMOTE_MANIFEST", "RESTORE_ENV"} {
		extractedDeployNames[name] = struct{}{}
	}
	extractedExecParamNames := map[string]struct{}{}
	for _, name := range []string{"PROJECT_NAME", "SOURCE_DB_KEY", "TARGET_DB_KEY", "TARGET_SERVER_KEY", "EXPORT_ROOT", "LOCAL_ENV", "TARGET_MANIFEST", "REMOTE_INBOX", "REMOTE_WORKDIR", "REMOTE_MANIFEST", "RESTORE_ENV", "TARGET_DB_RESET_BEFORE_IMPORT", "TARGET_DB_DROP_TABLES_BEFORE_IMPORT"} {
		extractedExecParamNames[name] = struct{}{}
	}

	kept := make([]scriptSeedPlaceholder, 0, len(existing))
	changed := false
	for _, placeholder := range existing {
		name := strings.ToUpper(strings.TrimSpace(placeholder.Name))
		if strings.TrimSpace(placeholder.ValueKind) == "manual" {
			if _, ok := extractedExecParamNames[name]; hasExecParamsConfig && ok {
				changed = true
				continue
			}
			_, isDeployName := extractedDeployNames[name]
			if hasDeployConfig && isDeployName && placeholder.ResourceCategoryId == 0 {
				changed = true
				continue
			}
			if hasSourceDBConfig && name == "SOURCE_DB_PASSWORD" {
				changed = true
				continue
			}
			if hasTargetDBConfig && name == "TARGET_DB_PASSWORD" {
				changed = true
				continue
			}
		}
		kept = append(kept, placeholder)
	}
	return kept, changed
}

func shouldRefreshDefaultStepScript(stepName string, script string) bool {
	if strings.TrimSpace(script) == "" {
		return true
	}
	switch stepName {
	case legacyDefaultExportStepName, defaultExportStepName:
		return strings.Contains(script, `:-`) || strings.Contains(script, `SOURCE_DB_KEY`) || !strings.Contains(script, `require_non_empty_env`)
	case reverseExportStepName:
		return !strings.Contains(script, `REMOTE_SCRIPT_LOCAL`) || !strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`)
	case defaultTableExportStepName:
		return !strings.Contains(script, `TABLE_NAME`) || !strings.Contains(script, `SOURCE_PG_DUMP_CMD`)
	case reverseTableExportStepName:
		return !strings.Contains(script, `REMOTE_SCRIPT_LOCAL`) ||
			!strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`) ||
			!strings.Contains(script, `SOURCE_TABLE_REF`)
	case defaultMySQLTableExportStepName, reverseMySQLTableExportStepName:
		return !strings.Contains(script, `TABLE_NAME`) || !strings.Contains(script, `SOURCE_MYSQLDUMP_CMD`)
	case "通过 Tailscale 上传导出文件":
		return strings.Contains(script, `:-`) || strings.Contains(script, `TARGET_SERVER_KEY`) || !strings.Contains(script, `TARGET_TAILSCALE_PORT`)
	case legacyReverseUploadStepName:
		return strings.Contains(script, `REMOTE_EXPORT_MANIFEST_LOCAL`) || !strings.Contains(script, `TARGET_TAILSCALE_PORT`)
	case "通过 Tailscale 上传单表导出文件":
		return !strings.Contains(script, `SOURCE_TABLE_NAME`) || !strings.Contains(script, `TARGET_TAILSCALE_PORT`)
	case "通过 Tailscale 上传 MySQL 单表导出文件", "通过 Tailscale 上传 MySQL 单表导出文件到本地":
		return !strings.Contains(script, `SOURCE_TABLE_NAME`) || !strings.Contains(script, `TARGET_TAILSCALE_PORT`)
	case "目标服务器整理并校验文件":
		return strings.Contains(script, `:-`) || !strings.Contains(script, `require_non_empty_env`)
	case "本地服务器整理并校验 PostgreSQL 文件":
		return strings.Contains(script, `本地导入清单`) || !strings.Contains(script, `REMOTE_MANIFEST`)
	case "目标服务器整理并校验单表文件":
		return !strings.Contains(script, `SOURCE_TABLE_NAME`) || !strings.Contains(script, `RESTORE_ENV`)
	case "目标服务器整理并校验 MySQL 单表文件", "本地服务器整理并校验 MySQL 单表文件":
		return !strings.Contains(script, `SOURCE_TABLE_NAME`) || !strings.Contains(script, `RESTORE_ENV`)
	case "目标服务器导入数据库", "本地服务器导入 PostgreSQL 数据库":
		return strings.Contains(script, `TARGET_DB_KEY`) ||
			!strings.Contains(script, `TARGET_DB_DROP_TABLES_BEFORE_IMPORT`) ||
			strings.Contains(script, `DROP SCHEMA IF EXISTS public CASCADE`)
	case "目标服务器导入 PostgreSQL 单表":
		return !strings.Contains(script, `TARGET_TABLE_DROP_BEFORE_IMPORT`) || !strings.Contains(script, `DROP TABLE IF EXISTS %I.%I`)
	case "目标服务器导入 MySQL 单表", "本地服务器导入 MySQL 单表":
		return !strings.Contains(script, `TARGET_TABLE_DROP_BEFORE_IMPORT`) || !strings.Contains(script, "DROP TABLE IF EXISTS `")
	case defaultClickHouseExportStepName:
		return !strings.Contains(script, `SOURCE_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`) || strings.Contains(script, `DROP TABLE`)
	case defaultClickHouseTableExportStepName:
		return !strings.Contains(script, `SOURCE_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`) || !strings.Contains(script, `TABLE_NAME`) || strings.Contains(script, `DROP TABLE`)
	case reverseClickHouseExportStepName:
		return !strings.Contains(script, `SOURCE_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`) || strings.Contains(script, `DROP TABLE`)
	case reverseClickHouseTableExportStepName:
		return !strings.Contains(script, `SOURCE_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `SOURCE_SERVER_TARGET_TAILSCALE_IP`) || !strings.Contains(script, `TABLE_NAME`) || strings.Contains(script, `DROP TABLE`)
	case "本机服务器导入 ClickHouse 数据库", "mac mini 服务器导入 ClickHouse 数据库":
		return !strings.Contains(script, `TARGET_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `TARGET_DB_DROP_TABLES_BEFORE_IMPORT`) || !strings.Contains(script, `python_stream_clickhouse_native`)
	case "本机服务器导入 ClickHouse 单表", "mac mini 服务器导入 ClickHouse 单表":
		return !strings.Contains(script, `TARGET_CLICKHOUSE_HTTP_CMD`) || !strings.Contains(script, `TARGET_TABLE_DROP_BEFORE_IMPORT`) || !strings.Contains(script, `python_stream_clickhouse_native`)
	default:
		return false
	}
}

func shouldRefreshDefaultStepType(stepName string, existingStepType string, desiredStepType string) bool {
	if strings.TrimSpace(desiredStepType) == "" || strings.TrimSpace(existingStepType) == strings.TrimSpace(desiredStepType) {
		return false
	}
	switch stepName {
	case reverseExportStepName, legacyReverseUploadStepName, "本地服务器整理并校验 PostgreSQL 文件", "本地服务器导入 PostgreSQL 数据库":
		return true
	default:
		return false
	}
}

func uintToString(value uint) string {
	return fmt.Sprintf("%d", value)
}

const defaultEasyDeployExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

run_configured_cmd() {
  local cmd_var="$1"
  shift
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER EXPORT_ROOT LOCAL_ENV
require_env SOURCE_DB_PASSWORD

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
DUMP_FILE="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}.dump"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"

mkdir -p "$EXPORT_ROOT"
mkdir -p "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

echo "开始导出 ${SOURCE_DB_TYPE} 数据库 ${SOURCE_DB_NAME}"
case "$SOURCE_DB_TYPE" in
  pgsql|postgres|postgresql)
    run_configured_cmd_with_env PGPASSWORD "$SOURCE_DB_PASSWORD" SOURCE_PG_DUMP_CMD \
      -h "$SOURCE_DB_HOST" \
      -p "$SOURCE_DB_PORT" \
      -U "$SOURCE_DB_USER" \
      -Fc \
      "$SOURCE_DB_NAME" > "$DUMP_FILE"
    ;;
  [mM][yY][sS][qQ][lL])
    run_configured_cmd_with_env MYSQL_PWD "$SOURCE_DB_PASSWORD" SOURCE_MYSQLDUMP_CMD \
      -h "$SOURCE_DB_HOST" \
      -P "$SOURCE_DB_PORT" \
      -u "$SOURCE_DB_USER" \
      "$SOURCE_DB_NAME" > "$DUMP_FILE"
    ;;
  sqlite)
    run_configured_cmd SOURCE_SQLITE3_CMD "$SOURCE_DB_NAME" ".backup '$DUMP_FILE'"
    ;;
  *)
    echo "不支持的 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

(cd "$(dirname "$DUMP_FILE")" && shasum -a 256 "$(basename "$DUMP_FILE")") > "$DUMP_SHA_FILE"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$(basename "$DUMP_FILE")
DUMP_SHA_BASENAME=$(basename "$DUMP_SHA_FILE")
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultPostgresRemoteDatabaseExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

write_env_assignment() {
  local name="$1"
  local value="$2"
  printf 'export %s=%q\n' "$name" "$value" >> "$REMOTE_ENV_LOCAL"
}

run_expect() {
  local delimiter
  local joined
  delimiter="$(printf '\034')"
  joined=""
  for arg in "$@"; do
    joined="${joined}${arg}${delimiter}"
  done
  EXPECT_COMMAND="$joined" "$LOCAL_EXPECT_CMD" <<'EXPECT_SCRIPT'
set timeout -1
set args {}
foreach part [split $env(EXPECT_COMMAND) "\034"] {
  if {$part ne ""} {
    lappend args $part
  }
}
spawn {*}$args
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(SOURCE_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCRIPT
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER EXPORT_ROOT LOCAL_ENV SOURCE_PG_DUMP_CMD LOCAL_EXPECT_CMD SOURCE_SERVER_TARGET_TAILSCALE_IP SOURCE_SERVER_TARGET_TAILSCALE_PORT SOURCE_SERVER_USER SOURCE_SERVER_PASSWORD
require_env SOURCE_DB_PASSWORD

case "$SOURCE_DB_TYPE" in
  pgsql|postgres|postgresql)
    ;;
  *)
    echo "远端导出仅支持 PostgreSQL，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_DB_CONNECT_HOST="${SOURCE_DB_REMOTE_HOST:-$SOURCE_DB_HOST}"
SOURCE_SSH_HOST="$SOURCE_SERVER_TARGET_TAILSCALE_IP"
SOURCE_SSH_PORT="$SOURCE_SERVER_TARGET_TAILSCALE_PORT"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
DUMP_BASENAME="${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}.dump"
DUMP_SHA_BASENAME="${DUMP_BASENAME}.sha256"
DUMP_FILE="${EXPORT_ROOT}/${DUMP_BASENAME}"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"
REMOTE_EXPORT_ROOT="${EXPORT_ROOT}/source"
REMOTE_DUMP_FILE="${REMOTE_EXPORT_ROOT}/${DUMP_BASENAME}"
REMOTE_DUMP_SHA_FILE="${REMOTE_DUMP_FILE}.sha256"
REMOTE_SCRIPT_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_pg_export.sh"
REMOTE_ENV_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_pg.env"
REMOTE_SCRIPT_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_pg_export.sh"
REMOTE_ENV_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_pg.env"

mkdir -p "$EXPORT_ROOT" "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

cat > "$REMOTE_SCRIPT_LOCAL" <<'REMOTE_SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

mkdir -p "$REMOTE_EXPORT_ROOT"
chmod 700 "$REMOTE_EXPORT_ROOT"

echo "开始在目标服务器本地导出 PostgreSQL 数据库 ${SOURCE_DB_NAME} (${SOURCE_DB_CONNECT_HOST}:${SOURCE_DB_PORT})"
run_configured_cmd_with_env PGPASSWORD "$SOURCE_DB_PASSWORD" SOURCE_PG_DUMP_CMD \
  -h "$SOURCE_DB_CONNECT_HOST" \
  -p "$SOURCE_DB_PORT" \
  -U "$SOURCE_DB_USER" \
  -Fc \
  "$SOURCE_DB_NAME" > "$REMOTE_DUMP_FILE"

(cd "$(dirname "$REMOTE_DUMP_FILE")" && shasum -a 256 "$(basename "$REMOTE_DUMP_FILE")") > "$REMOTE_DUMP_SHA_FILE"

echo "目标服务器 PostgreSQL 导出完成: $REMOTE_DUMP_FILE"
REMOTE_SCRIPT

: > "$REMOTE_ENV_LOCAL"
write_env_assignment SOURCE_PG_DUMP_CMD "$SOURCE_PG_DUMP_CMD"
write_env_assignment SOURCE_DB_CONNECT_HOST "$SOURCE_DB_CONNECT_HOST"
write_env_assignment SOURCE_DB_PORT "$SOURCE_DB_PORT"
write_env_assignment SOURCE_DB_NAME "$SOURCE_DB_NAME"
write_env_assignment SOURCE_DB_USER "$SOURCE_DB_USER"
write_env_assignment SOURCE_DB_PASSWORD "$SOURCE_DB_PASSWORD"
write_env_assignment REMOTE_EXPORT_ROOT "$REMOTE_EXPORT_ROOT"
write_env_assignment REMOTE_DUMP_FILE "$REMOTE_DUMP_FILE"
write_env_assignment REMOTE_DUMP_SHA_FILE "$REMOTE_DUMP_SHA_FILE"

chmod 600 "$REMOTE_ENV_LOCAL"

echo "通过 Tailscale SSH 上传 PostgreSQL 导出脚本到源服务器 ${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${SOURCE_SSH_PORT}"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "$REMOTE_SCRIPT_LOCAL" "$REMOTE_ENV_LOCAL" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:/tmp/"

echo "通过 Tailscale SSH 在源服务器导出 PostgreSQL"
run_expect ssh -p "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}" "chmod 700 '$REMOTE_SCRIPT_REMOTE' '$REMOTE_ENV_REMOTE' && source '$REMOTE_ENV_REMOTE' && bash '$REMOTE_SCRIPT_REMOTE'"

echo "通过 Tailscale 拉取源服务器 PostgreSQL 导出文件到本机"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_FILE}" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_SHA_FILE}" "$EXPORT_ROOT/"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultPostgresRemoteTableExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

write_env_assignment() {
  local name="$1"
  local value="$2"
  printf 'export %s=%q\n' "$name" "$value" >> "$REMOTE_ENV_LOCAL"
}

run_expect() {
  local delimiter
  local joined
  delimiter="$(printf '\034')"
  joined=""
  for arg in "$@"; do
    joined="${joined}${arg}${delimiter}"
  done
  EXPECT_COMMAND="$joined" "$LOCAL_EXPECT_CMD" <<'EXPECT_SCRIPT'
set timeout -1
set args {}
foreach part [split $env(EXPECT_COMMAND) "\034"] {
  if {$part ne ""} {
    lappend args $part
  }
}
spawn {*}$args
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(SOURCE_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCRIPT
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER TABLE_SCHEMA TABLE_NAME EXPORT_ROOT LOCAL_ENV SOURCE_PG_DUMP_CMD LOCAL_EXPECT_CMD SOURCE_SERVER_TARGET_TAILSCALE_IP SOURCE_SERVER_TARGET_TAILSCALE_PORT SOURCE_SERVER_USER SOURCE_SERVER_PASSWORD
require_env SOURCE_DB_PASSWORD

case "$SOURCE_DB_TYPE" in
  pgsql|postgres|postgresql)
    ;;
  *)
    echo "远端单表导出仅支持 PostgreSQL，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_TABLE_SCHEMA="$TABLE_SCHEMA"
SOURCE_TABLE_NAME="$TABLE_NAME"
SOURCE_TABLE_REF="${SOURCE_TABLE_SCHEMA}.${SOURCE_TABLE_NAME}"
SOURCE_DB_CONNECT_HOST="${SOURCE_DB_REMOTE_HOST:-$SOURCE_DB_HOST}"
SOURCE_SSH_HOST="$SOURCE_SERVER_TARGET_TAILSCALE_IP"
SOURCE_SSH_PORT="$SOURCE_SERVER_TARGET_TAILSCALE_PORT"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
SAFE_TABLE_NAME="$(printf "%s_%s" "$SOURCE_TABLE_SCHEMA" "$SOURCE_TABLE_NAME" | tr -c 'A-Za-z0-9_-' '_')"
DUMP_BASENAME="${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}.dump"
DUMP_SHA_BASENAME="${DUMP_BASENAME}.sha256"
DUMP_FILE="${EXPORT_ROOT}/${DUMP_BASENAME}"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"
REMOTE_EXPORT_ROOT="${EXPORT_ROOT}/source"
REMOTE_DUMP_FILE="${REMOTE_EXPORT_ROOT}/${DUMP_BASENAME}"
REMOTE_DUMP_SHA_FILE="${REMOTE_DUMP_FILE}.sha256"
REMOTE_SCRIPT_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_pg_table_export.sh"
REMOTE_ENV_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_pg_table.env"
REMOTE_SCRIPT_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_pg_table_export.sh"
REMOTE_ENV_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_pg_table.env"

mkdir -p "$EXPORT_ROOT" "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

cat > "$REMOTE_SCRIPT_LOCAL" <<'REMOTE_SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

mkdir -p "$REMOTE_EXPORT_ROOT"
chmod 700 "$REMOTE_EXPORT_ROOT"

echo "开始在目标服务器本地导出 PostgreSQL 单表 ${SOURCE_DB_NAME}.${SOURCE_TABLE_REF} (${SOURCE_DB_CONNECT_HOST}:${SOURCE_DB_PORT})"
run_configured_cmd_with_env PGPASSWORD "$SOURCE_DB_PASSWORD" SOURCE_PG_DUMP_CMD \
  -h "$SOURCE_DB_CONNECT_HOST" \
  -p "$SOURCE_DB_PORT" \
  -U "$SOURCE_DB_USER" \
  -Fc \
  --no-owner \
  -t "$SOURCE_TABLE_REF" \
  "$SOURCE_DB_NAME" > "$REMOTE_DUMP_FILE"

(cd "$(dirname "$REMOTE_DUMP_FILE")" && shasum -a 256 "$(basename "$REMOTE_DUMP_FILE")") > "$REMOTE_DUMP_SHA_FILE"

echo "目标服务器 PostgreSQL 单表导出完成: $REMOTE_DUMP_FILE"
REMOTE_SCRIPT

: > "$REMOTE_ENV_LOCAL"
write_env_assignment SOURCE_PG_DUMP_CMD "$SOURCE_PG_DUMP_CMD"
write_env_assignment SOURCE_DB_CONNECT_HOST "$SOURCE_DB_CONNECT_HOST"
write_env_assignment SOURCE_DB_PORT "$SOURCE_DB_PORT"
write_env_assignment SOURCE_DB_NAME "$SOURCE_DB_NAME"
write_env_assignment SOURCE_DB_USER "$SOURCE_DB_USER"
write_env_assignment SOURCE_DB_PASSWORD "$SOURCE_DB_PASSWORD"
write_env_assignment SOURCE_TABLE_REF "$SOURCE_TABLE_REF"
write_env_assignment REMOTE_EXPORT_ROOT "$REMOTE_EXPORT_ROOT"
write_env_assignment REMOTE_DUMP_FILE "$REMOTE_DUMP_FILE"
write_env_assignment REMOTE_DUMP_SHA_FILE "$REMOTE_DUMP_SHA_FILE"

chmod 600 "$REMOTE_ENV_LOCAL"

echo "通过 Tailscale SSH 上传 PostgreSQL 单表导出脚本到源服务器 ${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${SOURCE_SSH_PORT}"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "$REMOTE_SCRIPT_LOCAL" "$REMOTE_ENV_LOCAL" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:/tmp/"

echo "通过 Tailscale SSH 在源服务器导出 PostgreSQL 单表"
run_expect ssh -p "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}" "chmod 700 '$REMOTE_SCRIPT_REMOTE' '$REMOTE_ENV_REMOTE' && source '$REMOTE_ENV_REMOTE' && bash '$REMOTE_SCRIPT_REMOTE'"

echo "通过 Tailscale 拉取源服务器 PostgreSQL 单表导出文件到本机"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_FILE}" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_SHA_FILE}" "$EXPORT_ROOT/"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_SCHEMA=$SOURCE_TABLE_SCHEMA
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
SOURCE_TABLE_REF=$SOURCE_TABLE_REF
PROJECT_NAME=$PROJECT_NAME
EOF

echo "单表导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultEasyDeployUploadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env TARGET_SERVER_IP TARGET_SERVER_PORT TARGET_TAILSCALE_IP TARGET_TAILSCALE_PORT TARGET_SERVER_USER TARGET_SERVER_PASSWORD REMOTE_INBOX REMOTE_WORKDIR LOCAL_ENV TARGET_MANIFEST RESTORE_ENV LOCAL_EXPECT_CMD

if [ ! -f "$LOCAL_ENV" ]; then
  echo "未找到 $LOCAL_ENV，请先执行导出步骤" >&2
  exit 1
fi

is_local_target() {
  case "${TARGET_SERVER_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  case "${TARGET_TAILSCALE_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  return 1
}

source "$LOCAL_ENV"
mkdir -p "$(dirname "$TARGET_MANIFEST")"

cat > "$TARGET_MANIFEST" <<EOF
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
REMOTE_INBOX=$REMOTE_INBOX
REMOTE_WORKDIR=$REMOTE_WORKDIR
RESTORE_ENV=$RESTORE_ENV
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

if is_local_target; then
  echo "目标为本机，直接复制 PostgreSQL 导出文件到 ${REMOTE_INBOX}"
  mkdir -p "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  chmod 700 "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  cp "$DUMP_FILE" "$DUMP_SHA_FILE" "$TARGET_MANIFEST" "$REMOTE_INBOX/"
  echo "本机文件已就绪: ${REMOTE_INBOX}/${DUMP_BASENAME}"
  exit 0
fi

echo "准备通过 FRP SSH 初始化目录 ${TARGET_SERVER_USER}@${TARGET_SERVER_IP}:${TARGET_SERVER_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_SERVER_PORT="$TARGET_SERVER_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_SERVER_IP="$TARGET_SERVER_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
REMOTE_WORKDIR="$REMOTE_WORKDIR" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SSH'
set timeout -1
spawn ssh -p $env(TARGET_SERVER_PORT) -o StrictHostKeyChecking=accept-new "$env(TARGET_SERVER_USER)@$env(TARGET_SERVER_IP)" "mkdir -p '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)' && chmod 700 '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)'"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SSH

echo "准备通过 Tailscale 传输文件到 ${TARGET_SERVER_USER}@${TARGET_TAILSCALE_IP}:${TARGET_TAILSCALE_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_TAILSCALE_PORT="$TARGET_TAILSCALE_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_TAILSCALE_IP="$TARGET_TAILSCALE_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
DUMP_FILE="$DUMP_FILE" \
DUMP_SHA_FILE="$DUMP_SHA_FILE" \
TARGET_MANIFEST="$TARGET_MANIFEST" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SCP'
set timeout -1
spawn scp -P $env(TARGET_TAILSCALE_PORT) -o StrictHostKeyChecking=accept-new $env(DUMP_FILE) $env(DUMP_SHA_FILE) $env(TARGET_MANIFEST) "$env(TARGET_SERVER_USER)@$env(TARGET_TAILSCALE_IP):$env(REMOTE_INBOX)/"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCP

echo "上传完成"
echo "远端文件: ${REMOTE_INBOX}/${DUMP_BASENAME}"
`

const defaultEasyDeployTargetDownloadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env REMOTE_INBOX REMOTE_WORKDIR REMOTE_MANIFEST RESTORE_ENV

if [ ! -f "$REMOTE_MANIFEST" ]; then
  echo "未找到上传清单 $REMOTE_MANIFEST，请先执行本地上传步骤" >&2
  exit 1
fi

source "$REMOTE_MANIFEST"
require_non_empty_env DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME PROJECT_NAME
mkdir -p "$REMOTE_WORKDIR"
mkdir -p "$(dirname "$RESTORE_ENV")"
chmod 700 "$REMOTE_WORKDIR"

cp "${REMOTE_INBOX}/${DUMP_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_BASENAME}"
cp "${REMOTE_INBOX}/${DUMP_SHA_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}"

cd "$REMOTE_WORKDIR"
shasum -a 256 -c "$DUMP_SHA_BASENAME"

cat > "$RESTORE_ENV" <<EOF
RESTORE_DUMP=${REMOTE_WORKDIR}/${DUMP_BASENAME}
RESTORE_SHA=${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "目标服务器文件已就绪: ${REMOTE_WORKDIR}/${DUMP_BASENAME}"
`

const defaultEasyDeployTargetExecScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

run_configured_cmd() {
  local cmd_var="$1"
  shift
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

require_non_empty_env TARGET_DB_TYPE TARGET_DB_HOST TARGET_DB_PORT TARGET_DB_DATABASE TARGET_DB_USER TARGET_DB_DROP_TABLES_BEFORE_IMPORT REMOTE_WORKDIR RESTORE_ENV
require_env TARGET_DB_PASSWORD

case "$TARGET_DB_DROP_TABLES_BEFORE_IMPORT" in
  true|false)
    ;;
  *)
    echo "TARGET_DB_DROP_TABLES_BEFORE_IMPORT 只能配置为 true 或 false" >&2
    exit 1
    ;;
esac

TARGET_DB_NAME="$TARGET_DB_DATABASE"
TARGET_DB_CONNECT_HOST="${TARGET_DB_REMOTE_HOST:-$TARGET_DB_HOST}"

if [ ! -f "$RESTORE_ENV" ]; then
  echo "未找到 $RESTORE_ENV，请先执行目标下载步骤" >&2
  exit 1
fi

source "$RESTORE_ENV"
require_non_empty_env RESTORE_DUMP RESTORE_SHA SOURCE_DB_TYPE SOURCE_DB_NAME PROJECT_NAME

if [ "$TARGET_DB_DROP_TABLES_BEFORE_IMPORT" = "true" ]; then
  echo "开始删除目标数据库所有表 ${TARGET_DB_TYPE}/${TARGET_DB_NAME} (${TARGET_DB_CONNECT_HOST}:${TARGET_DB_PORT})"
  case "$TARGET_DB_TYPE" in
    pgsql|postgres|postgresql)
      run_configured_cmd_with_env PGPASSWORD "$TARGET_DB_PASSWORD" TARGET_PSQL_CMD \
        -h "$TARGET_DB_CONNECT_HOST" \
        -p "$TARGET_DB_PORT" \
        -U "$TARGET_DB_USER" \
        -d "$TARGET_DB_NAME" \
        -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE
  table_record record;
BEGIN
  FOR table_record IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
  LOOP
    EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE', table_record.schemaname, table_record.tablename);
  END LOOP;
END $$;
SQL
      ;;
    [mM][yY][sS][qQ][lL])
      run_configured_cmd_with_env MYSQL_PWD "$TARGET_DB_PASSWORD" TARGET_MYSQL_CMD \
        -h "$TARGET_DB_CONNECT_HOST" \
        -P "$TARGET_DB_PORT" \
        -u "$TARGET_DB_USER" \
        "$TARGET_DB_NAME" <<'SQL'
SET SESSION group_concat_max_len = 1000000;
SET FOREIGN_KEY_CHECKS = 0;
SELECT GROUP_CONCAT(CONCAT('` + "`" + `', table_name, '` + "`" + `')) INTO @tables
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_type = 'BASE TABLE';
SET @drop_stmt = IF(@tables IS NULL, 'SELECT 1', CONCAT('DROP TABLE IF EXISTS ', @tables));
PREPARE stmt FROM @drop_stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
SET FOREIGN_KEY_CHECKS = 1;
SQL
      ;;
    sqlite)
      rm -f "$TARGET_DB_NAME"
      ;;
    *)
      echo "不支持的 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
      exit 1
      ;;
  esac
  echo "目标数据库所有表已删除"
fi

echo "开始导入到目标数据库 ${TARGET_DB_TYPE}/${TARGET_DB_NAME} (${TARGET_DB_CONNECT_HOST}:${TARGET_DB_PORT})"
case "$TARGET_DB_TYPE" in
  pgsql|postgres|postgresql)
    run_configured_cmd_with_env PGPASSWORD "$TARGET_DB_PASSWORD" TARGET_PG_RESTORE_CMD \
      -h "$TARGET_DB_CONNECT_HOST" \
      -p "$TARGET_DB_PORT" \
      -U "$TARGET_DB_USER" \
      -d "$TARGET_DB_NAME" \
      --clean \
      --if-exists \
      --no-owner < "$RESTORE_DUMP"
    ;;
  [mM][yY][sS][qQ][lL])
    run_configured_cmd_with_env MYSQL_PWD "$TARGET_DB_PASSWORD" TARGET_MYSQL_CMD \
      -h "$TARGET_DB_CONNECT_HOST" \
      -P "$TARGET_DB_PORT" \
      -u "$TARGET_DB_USER" \
      "$TARGET_DB_NAME" < "$RESTORE_DUMP"
    ;;
  sqlite)
    run_configured_cmd TARGET_SQLITE3_CMD "$TARGET_DB_NAME" ".restore '$RESTORE_DUMP'"
    ;;
  *)
    echo "不支持的 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
    exit 1
    ;;
esac

echo "目标数据库导入完成"
`

const defaultPostgresTableExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER TABLE_SCHEMA TABLE_NAME EXPORT_ROOT LOCAL_ENV SOURCE_PG_DUMP_CMD
require_env SOURCE_DB_PASSWORD

case "$SOURCE_DB_TYPE" in
  pgsql|postgres|postgresql)
    ;;
  *)
    echo "单表导出仅支持 PostgreSQL，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_TABLE_SCHEMA="$TABLE_SCHEMA"
SOURCE_TABLE_NAME="$TABLE_NAME"
SOURCE_TABLE_REF="${SOURCE_TABLE_SCHEMA}.${SOURCE_TABLE_NAME}"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
SAFE_TABLE_NAME="$(printf "%s_%s" "$SOURCE_TABLE_SCHEMA" "$SOURCE_TABLE_NAME" | tr -c 'A-Za-z0-9_-' '_')"
DUMP_FILE="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}.dump"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"

mkdir -p "$EXPORT_ROOT"
mkdir -p "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

echo "开始导出 PostgreSQL 单表 ${SOURCE_DB_NAME}.${SOURCE_TABLE_REF}"
run_configured_cmd_with_env PGPASSWORD "$SOURCE_DB_PASSWORD" SOURCE_PG_DUMP_CMD \
  -h "$SOURCE_DB_HOST" \
  -p "$SOURCE_DB_PORT" \
  -U "$SOURCE_DB_USER" \
  -Fc \
  --no-owner \
  -t "$SOURCE_TABLE_REF" \
  "$SOURCE_DB_NAME" > "$DUMP_FILE"

(cd "$(dirname "$DUMP_FILE")" && shasum -a 256 "$(basename "$DUMP_FILE")") > "$DUMP_SHA_FILE"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$(basename "$DUMP_FILE")
DUMP_SHA_BASENAME=$(basename "$DUMP_SHA_FILE")
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_SCHEMA=$SOURCE_TABLE_SCHEMA
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
SOURCE_TABLE_REF=$SOURCE_TABLE_REF
PROJECT_NAME=$PROJECT_NAME
EOF

echo "单表导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultPostgresTableUploadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env TARGET_SERVER_IP TARGET_SERVER_PORT TARGET_TAILSCALE_IP TARGET_TAILSCALE_PORT TARGET_SERVER_USER TARGET_SERVER_PASSWORD REMOTE_INBOX REMOTE_WORKDIR LOCAL_ENV TARGET_MANIFEST RESTORE_ENV LOCAL_EXPECT_CMD

if [ ! -f "$LOCAL_ENV" ]; then
  echo "未找到 $LOCAL_ENV，请先执行单表导出步骤" >&2
  exit 1
fi

is_local_target() {
  case "${TARGET_SERVER_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  case "${TARGET_TAILSCALE_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  return 1
}

source "$LOCAL_ENV"
require_non_empty_env DUMP_FILE DUMP_SHA_FILE DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_SCHEMA SOURCE_TABLE_NAME SOURCE_TABLE_REF PROJECT_NAME
mkdir -p "$(dirname "$TARGET_MANIFEST")"

cat > "$TARGET_MANIFEST" <<EOF
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
REMOTE_INBOX=$REMOTE_INBOX
REMOTE_WORKDIR=$REMOTE_WORKDIR
RESTORE_ENV=$RESTORE_ENV
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_SCHEMA=$SOURCE_TABLE_SCHEMA
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
SOURCE_TABLE_REF=$SOURCE_TABLE_REF
PROJECT_NAME=$PROJECT_NAME
EOF

if is_local_target; then
  echo "目标为本机，直接复制单表导出文件到 ${REMOTE_INBOX}"
  mkdir -p "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  chmod 700 "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  cp "$DUMP_FILE" "$DUMP_SHA_FILE" "$TARGET_MANIFEST" "$REMOTE_INBOX/"
  echo "本机单表文件已就绪: ${REMOTE_INBOX}/${DUMP_BASENAME}"
  exit 0
fi

echo "准备通过 FRP SSH 初始化目录 ${TARGET_SERVER_USER}@${TARGET_SERVER_IP}:${TARGET_SERVER_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_SERVER_PORT="$TARGET_SERVER_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_SERVER_IP="$TARGET_SERVER_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
REMOTE_WORKDIR="$REMOTE_WORKDIR" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SSH'
set timeout -1
spawn ssh -p $env(TARGET_SERVER_PORT) -o StrictHostKeyChecking=accept-new "$env(TARGET_SERVER_USER)@$env(TARGET_SERVER_IP)" "mkdir -p '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)' && chmod 700 '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)'"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SSH

echo "准备通过 Tailscale 传输单表导出文件到 ${TARGET_SERVER_USER}@${TARGET_TAILSCALE_IP}:${TARGET_TAILSCALE_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_TAILSCALE_PORT="$TARGET_TAILSCALE_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_TAILSCALE_IP="$TARGET_TAILSCALE_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
DUMP_FILE="$DUMP_FILE" \
DUMP_SHA_FILE="$DUMP_SHA_FILE" \
TARGET_MANIFEST="$TARGET_MANIFEST" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SCP'
set timeout -1
spawn scp -P $env(TARGET_TAILSCALE_PORT) -o StrictHostKeyChecking=accept-new $env(DUMP_FILE) $env(DUMP_SHA_FILE) $env(TARGET_MANIFEST) "$env(TARGET_SERVER_USER)@$env(TARGET_TAILSCALE_IP):$env(REMOTE_INBOX)/"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCP

echo "上传完成"
echo "远端文件: ${REMOTE_INBOX}/${DUMP_BASENAME}"
`

const defaultPostgresTableTargetDownloadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env REMOTE_INBOX REMOTE_WORKDIR REMOTE_MANIFEST RESTORE_ENV

if [ ! -f "$REMOTE_MANIFEST" ]; then
  echo "未找到上传清单 $REMOTE_MANIFEST，请先执行本地上传步骤" >&2
  exit 1
fi

source "$REMOTE_MANIFEST"
require_non_empty_env DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_SCHEMA SOURCE_TABLE_NAME SOURCE_TABLE_REF PROJECT_NAME
mkdir -p "$REMOTE_WORKDIR"
mkdir -p "$(dirname "$RESTORE_ENV")"
chmod 700 "$REMOTE_WORKDIR"

cp "${REMOTE_INBOX}/${DUMP_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_BASENAME}"
cp "${REMOTE_INBOX}/${DUMP_SHA_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}"

cd "$REMOTE_WORKDIR"
shasum -a 256 -c "$DUMP_SHA_BASENAME"

cat > "$RESTORE_ENV" <<EOF
RESTORE_DUMP=${REMOTE_WORKDIR}/${DUMP_BASENAME}
RESTORE_SHA=${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_SCHEMA=$SOURCE_TABLE_SCHEMA
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
SOURCE_TABLE_REF=$SOURCE_TABLE_REF
PROJECT_NAME=$PROJECT_NAME
EOF

echo "目标服务器单表文件已就绪: ${REMOTE_WORKDIR}/${DUMP_BASENAME}"
`

const defaultPostgresTableTargetExecScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

require_non_empty_env TARGET_DB_TYPE TARGET_DB_HOST TARGET_DB_PORT TARGET_DB_DATABASE TARGET_DB_USER TARGET_TABLE_DROP_BEFORE_IMPORT RESTORE_ENV TARGET_PSQL_CMD TARGET_PG_RESTORE_CMD
require_env TARGET_DB_PASSWORD

case "$TARGET_DB_TYPE" in
  pgsql|postgres|postgresql)
    ;;
  *)
    echo "单表导入仅支持 PostgreSQL，当前 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
    exit 1
    ;;
esac

case "$TARGET_TABLE_DROP_BEFORE_IMPORT" in
  true|false)
    ;;
  *)
    echo "TARGET_TABLE_DROP_BEFORE_IMPORT 只能配置为 true 或 false" >&2
    exit 1
    ;;
esac

TARGET_DB_NAME="$TARGET_DB_DATABASE"
TARGET_DB_CONNECT_HOST="${TARGET_DB_REMOTE_HOST:-$TARGET_DB_HOST}"

if [ ! -f "$RESTORE_ENV" ]; then
  echo "未找到 $RESTORE_ENV，请先执行目标下载步骤" >&2
  exit 1
fi

source "$RESTORE_ENV"
require_non_empty_env RESTORE_DUMP SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_SCHEMA SOURCE_TABLE_NAME SOURCE_TABLE_REF PROJECT_NAME

case "$SOURCE_DB_TYPE" in
  pgsql|postgres|postgresql)
    ;;
  *)
    echo "单表导入仅支持 PostgreSQL 导出文件，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

echo "准备目标 PostgreSQL 表 ${TARGET_DB_NAME}.${SOURCE_TABLE_REF} (${TARGET_DB_CONNECT_HOST}:${TARGET_DB_PORT})"
run_configured_cmd_with_env PGPASSWORD "$TARGET_DB_PASSWORD" TARGET_PSQL_CMD \
  -h "$TARGET_DB_CONNECT_HOST" \
  -p "$TARGET_DB_PORT" \
  -U "$TARGET_DB_USER" \
  -d "$TARGET_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -v table_schema="$SOURCE_TABLE_SCHEMA" <<'SQL'
SELECT format('CREATE SCHEMA IF NOT EXISTS %I', :'table_schema') \gexec
SQL

if [ "$TARGET_TABLE_DROP_BEFORE_IMPORT" = "true" ]; then
  echo "开始删除目标表 ${TARGET_DB_NAME}.${SOURCE_TABLE_REF}"
  run_configured_cmd_with_env PGPASSWORD "$TARGET_DB_PASSWORD" TARGET_PSQL_CMD \
    -h "$TARGET_DB_CONNECT_HOST" \
    -p "$TARGET_DB_PORT" \
    -U "$TARGET_DB_USER" \
    -d "$TARGET_DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -v table_schema="$SOURCE_TABLE_SCHEMA" \
    -v table_name="$SOURCE_TABLE_NAME" <<'SQL'
SELECT format('DROP TABLE IF EXISTS %I.%I CASCADE', :'table_schema', :'table_name') \gexec
SQL
  echo "目标表已删除"
fi

echo "开始导入 PostgreSQL 单表 ${TARGET_DB_NAME}.${SOURCE_TABLE_REF}"
run_configured_cmd_with_env PGPASSWORD "$TARGET_DB_PASSWORD" TARGET_PG_RESTORE_CMD \
  -h "$TARGET_DB_CONNECT_HOST" \
  -p "$TARGET_DB_PORT" \
  -U "$TARGET_DB_USER" \
  -d "$TARGET_DB_NAME" \
  --no-owner < "$RESTORE_DUMP"

echo "PostgreSQL 单表导入完成"
`

const defaultMySQLTableExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER TABLE_NAME EXPORT_ROOT LOCAL_ENV SOURCE_MYSQLDUMP_CMD
require_env SOURCE_DB_PASSWORD

case "$SOURCE_DB_TYPE" in
  [mM][yY][sS][qQ][lL])
    ;;
  *)
    echo "单表导出仅支持 MySQL，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_TABLE_SCHEMA="default"
SOURCE_TABLE_NAME="$TABLE_NAME"
SOURCE_TABLE_REF="$SOURCE_TABLE_NAME"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
SAFE_TABLE_NAME="$(printf "%s" "$SOURCE_TABLE_NAME" | tr -c 'A-Za-z0-9_-' '_')"
DUMP_FILE="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}.sql"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"

mkdir -p "$EXPORT_ROOT"
mkdir -p "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

echo "开始导出 MySQL 单表 ${SOURCE_DB_NAME}.${SOURCE_TABLE_NAME}"
run_configured_cmd_with_env MYSQL_PWD "$SOURCE_DB_PASSWORD" SOURCE_MYSQLDUMP_CMD \
  -h "$SOURCE_DB_HOST" \
  -P "$SOURCE_DB_PORT" \
  -u "$SOURCE_DB_USER" \
  --single-transaction \
  --skip-lock-tables \
  "$SOURCE_DB_NAME" "$SOURCE_TABLE_NAME" > "$DUMP_FILE"

(cd "$(dirname "$DUMP_FILE")" && shasum -a 256 "$(basename "$DUMP_FILE")") > "$DUMP_SHA_FILE"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$(basename "$DUMP_FILE")
DUMP_SHA_BASENAME=$(basename "$DUMP_SHA_FILE")
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_SCHEMA=$SOURCE_TABLE_SCHEMA
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
SOURCE_TABLE_REF=$SOURCE_TABLE_REF
PROJECT_NAME=$PROJECT_NAME
EOF

echo "MySQL 单表导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultMySQLTableTargetExecScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd_with_env() {
  local env_name="$1"
  local env_value="$2"
  local cmd_var="$3"
  shift 3
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  for i in "${!cmd_parts[@]}"; do
    if [ "${cmd_parts[$i]##*/}" = "docker" ] && [ "${cmd_parts[$((i + 1))]:-}" = "exec" ]; then
      cmd_parts=("${cmd_parts[@]:0:$((i + 2))}" -e "${env_name}=${env_value}" "${cmd_parts[@]:$((i + 2))}")
      "${cmd_parts[@]}" "$@"
      return
    fi
  done
  env "$env_name=$env_value" "${cmd_parts[@]}" "$@"
}

require_non_empty_env TARGET_DB_TYPE TARGET_DB_HOST TARGET_DB_PORT TARGET_DB_DATABASE TARGET_DB_USER TARGET_TABLE_DROP_BEFORE_IMPORT RESTORE_ENV TARGET_MYSQL_CMD
require_env TARGET_DB_PASSWORD

case "$TARGET_DB_TYPE" in
  [mM][yY][sS][qQ][lL])
    ;;
  *)
    echo "单表导入仅支持 MySQL，当前 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
    exit 1
    ;;
esac

case "$TARGET_TABLE_DROP_BEFORE_IMPORT" in
  true|false)
    ;;
  *)
    echo "TARGET_TABLE_DROP_BEFORE_IMPORT 只能配置为 true 或 false" >&2
    exit 1
    ;;
esac

TARGET_DB_NAME="$TARGET_DB_DATABASE"
TARGET_DB_CONNECT_HOST="${TARGET_DB_REMOTE_HOST:-$TARGET_DB_HOST}"

if [ ! -f "$RESTORE_ENV" ]; then
  echo "未找到 $RESTORE_ENV，请先执行目标下载步骤" >&2
  exit 1
fi

source "$RESTORE_ENV"
require_non_empty_env RESTORE_DUMP SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_NAME PROJECT_NAME

case "$SOURCE_DB_TYPE" in
  [mM][yY][sS][qQ][lL])
    ;;
  *)
    echo "单表导入仅支持 MySQL 导出文件，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

if [ "$TARGET_TABLE_DROP_BEFORE_IMPORT" = "true" ]; then
  echo "开始删除目标 MySQL 表 ${TARGET_DB_NAME}.${SOURCE_TABLE_NAME}"
  run_configured_cmd_with_env MYSQL_PWD "$TARGET_DB_PASSWORD" TARGET_MYSQL_CMD \
    -h "$TARGET_DB_CONNECT_HOST" \
    -P "$TARGET_DB_PORT" \
    -u "$TARGET_DB_USER" \
    "$TARGET_DB_NAME" <<SQL
SET FOREIGN_KEY_CHECKS = 0;
SET @table_name = REPLACE('$SOURCE_TABLE_NAME', CHAR(96), CONCAT(CHAR(96), CHAR(96)));
SET @drop_stmt = CONCAT('DROP TABLE IF EXISTS ', CHAR(96), @table_name, CHAR(96));
PREPARE stmt FROM @drop_stmt;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
SET FOREIGN_KEY_CHECKS = 1;
SQL
  echo "目标表已删除"
fi

echo "开始导入 MySQL 单表 ${TARGET_DB_NAME}.${SOURCE_TABLE_NAME}"
run_configured_cmd_with_env MYSQL_PWD "$TARGET_DB_PASSWORD" TARGET_MYSQL_CMD \
  -h "$TARGET_DB_CONNECT_HOST" \
  -P "$TARGET_DB_PORT" \
  -u "$TARGET_DB_USER" \
  "$TARGET_DB_NAME" < "$RESTORE_DUMP"

echo "MySQL 单表导入完成"
`

const defaultClickHouseExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

write_env_assignment() {
  local name="$1"
  local value="$2"
  printf 'export %s=%q\n' "$name" "$value" >> "$REMOTE_ENV_LOCAL"
}

run_expect() {
  local delimiter
  local joined
  delimiter="$(printf '\034')"
  joined=""
  for arg in "$@"; do
    joined="${joined}${arg}${delimiter}"
  done
  EXPECT_COMMAND="$joined" "$LOCAL_EXPECT_CMD" <<'EXPECT_SCRIPT'
set timeout -1
set args {}
foreach part [split $env(EXPECT_COMMAND) "\034"] {
  if {$part ne ""} {
    lappend args $part
  }
}
spawn {*}$args
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(SOURCE_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCRIPT
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER EXPORT_ROOT LOCAL_ENV SOURCE_CLICKHOUSE_HTTP_CMD LOCAL_EXPECT_CMD SOURCE_SERVER_TARGET_TAILSCALE_IP SOURCE_SERVER_TARGET_TAILSCALE_PORT SOURCE_SERVER_USER SOURCE_SERVER_PASSWORD
require_env SOURCE_DB_PASSWORD

case "$(printf "%s" "$SOURCE_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 导出仅支持 ClickHouse，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_DB_CONNECT_HOST="${SOURCE_DB_REMOTE_HOST:-$SOURCE_DB_HOST}"
SOURCE_SSH_HOST="$SOURCE_SERVER_TARGET_TAILSCALE_IP"
SOURCE_SSH_PORT="$SOURCE_SERVER_TARGET_TAILSCALE_PORT"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
DUMP_BASENAME="${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}.tar.gz"
DUMP_SHA_BASENAME="${DUMP_BASENAME}.sha256"
DUMP_FILE="${EXPORT_ROOT}/${DUMP_BASENAME}"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"
REMOTE_EXPORT_ROOT="${EXPORT_ROOT}/source"
REMOTE_EXPORT_DIR="${REMOTE_EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}"
REMOTE_DUMP_FILE="${REMOTE_EXPORT_ROOT}/${DUMP_BASENAME}"
REMOTE_DUMP_SHA_FILE="${REMOTE_DUMP_FILE}.sha256"
REMOTE_SCRIPT_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_export.sh"
REMOTE_ENV_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote.env"
REMOTE_SCRIPT_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote_export.sh"
REMOTE_ENV_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${RUN_ID}_remote.env"

mkdir -p "$EXPORT_ROOT" "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

cat > "$REMOTE_SCRIPT_LOCAL" <<'REMOTE_SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

run_configured_cmd() {
  local cmd_var="$1"
  shift
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

clickhouse_url() {
  local host="$1"
  local port="$2"
  local database="${3:-}"
  local scheme="http"
  if [ "$port" = "8443" ]; then
    scheme="https"
  fi
  local base
  case "$host" in
    http://*|https://*) base="${host%/}" ;;
    *) base="${scheme}://${host}:${port}" ;;
  esac
  if [ -n "$database" ]; then
    printf "%s/?database=%s" "$base" "$database"
  else
    printf "%s/" "$base"
  fi
}

source_clickhouse_query() {
  local query="$1"
  run_configured_cmd SOURCE_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $SOURCE_DB_USER" \
    -H "X-ClickHouse-Key: $SOURCE_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$SOURCE_DB_CONNECT_HOST" "$SOURCE_DB_PORT" "$SOURCE_DB_NAME")"
}

ch_ident() {
  local value="$1"
  local bt
  bt="$(printf '\140')"
  value="${value//$bt/$bt$bt}"
  printf "%s%s%s" "$bt" "$value" "$bt"
}

safe_filename() {
  printf "%s" "$1" | tr -c 'A-Za-z0-9_-' '_'
}

SCHEMA_DIR="${REMOTE_EXPORT_DIR}/schemas"
DATA_DIR="${REMOTE_EXPORT_DIR}/data"
MANIFEST_FILE="${REMOTE_EXPORT_DIR}/manifest.tsv"
TABLE_LIST_FILE="${REMOTE_EXPORT_DIR}/tables.tsv"

rm -rf "$REMOTE_EXPORT_DIR"
mkdir -p "$SCHEMA_DIR" "$DATA_DIR" "$REMOTE_EXPORT_ROOT"
chmod 700 "$REMOTE_EXPORT_ROOT" "$REMOTE_EXPORT_DIR"
: > "$MANIFEST_FILE"

echo "开始只读导出 mac mini ClickHouse 数据库 ${SOURCE_DB_NAME}"
source_clickhouse_query "SELECT name FROM system.tables WHERE database = currentDatabase() AND is_temporary = 0 AND engine NOT LIKE '%View' ORDER BY name FORMAT TSVRaw" > "$TABLE_LIST_FILE"

while IFS= read -r table_name || [ -n "$table_name" ]; do
  [ -n "$table_name" ] || continue
  safe_table="$(safe_filename "$table_name")"
  schema_file="${SCHEMA_DIR}/${safe_table}.sql"
  data_file="${DATA_DIR}/${safe_table}.native"
  table_ref="$(ch_ident "$SOURCE_DB_NAME").$(ch_ident "$table_name")"

  echo "导出表结构: ${SOURCE_DB_NAME}.${table_name}"
  source_clickhouse_query "SHOW CREATE TABLE ${table_ref}" > "$schema_file"

  echo "导出表数据: ${SOURCE_DB_NAME}.${table_name}"
  source_clickhouse_query "SELECT * FROM ${table_ref} FORMAT Native" > "$data_file"

  printf "%s\t%s\t%s\n" "$table_name" "schemas/${safe_table}.sql" "data/${safe_table}.native" >> "$MANIFEST_FILE"
done < "$TABLE_LIST_FILE"

tar -C "$REMOTE_EXPORT_DIR" -czf "$REMOTE_DUMP_FILE" manifest.tsv tables.tsv schemas data
(cd "$(dirname "$REMOTE_DUMP_FILE")" && shasum -a 256 "$(basename "$REMOTE_DUMP_FILE")") > "$REMOTE_DUMP_SHA_FILE"

echo "ClickHouse 数据库导出完成: $REMOTE_DUMP_FILE"
REMOTE_SCRIPT

: > "$REMOTE_ENV_LOCAL"
write_env_assignment SOURCE_CLICKHOUSE_HTTP_CMD "$SOURCE_CLICKHOUSE_HTTP_CMD"
write_env_assignment SOURCE_DB_CONNECT_HOST "$SOURCE_DB_CONNECT_HOST"
write_env_assignment SOURCE_DB_PORT "$SOURCE_DB_PORT"
write_env_assignment SOURCE_DB_NAME "$SOURCE_DB_NAME"
write_env_assignment SOURCE_DB_USER "$SOURCE_DB_USER"
write_env_assignment SOURCE_DB_PASSWORD "$SOURCE_DB_PASSWORD"
write_env_assignment REMOTE_EXPORT_ROOT "$REMOTE_EXPORT_ROOT"
write_env_assignment REMOTE_EXPORT_DIR "$REMOTE_EXPORT_DIR"
write_env_assignment REMOTE_DUMP_FILE "$REMOTE_DUMP_FILE"
write_env_assignment REMOTE_DUMP_SHA_FILE "$REMOTE_DUMP_SHA_FILE"

chmod 600 "$REMOTE_ENV_LOCAL"

echo "通过 Tailscale SSH 上传 ClickHouse 导出脚本到源服务器 ${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${SOURCE_SSH_PORT}"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "$REMOTE_SCRIPT_LOCAL" "$REMOTE_ENV_LOCAL" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:/tmp/"

echo "通过 Tailscale SSH 在源服务器只读导出 ClickHouse"
run_expect ssh -p "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}" "chmod 700 '$REMOTE_SCRIPT_REMOTE' '$REMOTE_ENV_REMOTE' && source '$REMOTE_ENV_REMOTE' && bash '$REMOTE_SCRIPT_REMOTE'"

echo "通过 Tailscale 拉取源服务器 ClickHouse 导出文件到本机"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_FILE}" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_SHA_FILE}" "$EXPORT_ROOT/"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "ClickHouse 数据库导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultClickHouseUploadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env TARGET_SERVER_IP TARGET_SERVER_PORT TARGET_TAILSCALE_IP TARGET_TAILSCALE_PORT TARGET_SERVER_USER TARGET_SERVER_PASSWORD REMOTE_INBOX REMOTE_WORKDIR LOCAL_ENV TARGET_MANIFEST RESTORE_ENV LOCAL_EXPECT_CMD

if [ ! -f "$LOCAL_ENV" ]; then
  echo "未找到 $LOCAL_ENV，请先执行 ClickHouse 导出步骤" >&2
  exit 1
fi

is_local_target() {
  case "${TARGET_SERVER_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  case "${TARGET_TAILSCALE_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  return 1
}

source "$LOCAL_ENV"
require_non_empty_env DUMP_FILE DUMP_SHA_FILE DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME PROJECT_NAME
mkdir -p "$(dirname "$TARGET_MANIFEST")"

cat > "$TARGET_MANIFEST" <<EOF
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
REMOTE_INBOX=$REMOTE_INBOX
REMOTE_WORKDIR=$REMOTE_WORKDIR
RESTORE_ENV=$RESTORE_ENV
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

if is_local_target; then
  echo "目标为本机，直接复制 ClickHouse 导出文件到 ${REMOTE_INBOX}"
  mkdir -p "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  chmod 700 "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  cp "$DUMP_FILE" "$DUMP_SHA_FILE" "$TARGET_MANIFEST" "$REMOTE_INBOX/"
  echo "本机 ClickHouse 文件已就绪: ${REMOTE_INBOX}/${DUMP_BASENAME}"
  exit 0
fi

echo "准备初始化本机目标目录 ${TARGET_SERVER_USER}@${TARGET_SERVER_IP}:${TARGET_SERVER_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_SERVER_PORT="$TARGET_SERVER_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_SERVER_IP="$TARGET_SERVER_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
REMOTE_WORKDIR="$REMOTE_WORKDIR" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SSH'
set timeout -1
spawn ssh -p $env(TARGET_SERVER_PORT) -o StrictHostKeyChecking=accept-new "$env(TARGET_SERVER_USER)@$env(TARGET_SERVER_IP)" "mkdir -p '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)' && chmod 700 '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)'"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SSH

echo "准备通过 Tailscale 传输 ClickHouse 导出文件到本机 ${TARGET_SERVER_USER}@${TARGET_TAILSCALE_IP}:${TARGET_TAILSCALE_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_TAILSCALE_PORT="$TARGET_TAILSCALE_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_TAILSCALE_IP="$TARGET_TAILSCALE_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
DUMP_FILE="$DUMP_FILE" \
DUMP_SHA_FILE="$DUMP_SHA_FILE" \
TARGET_MANIFEST="$TARGET_MANIFEST" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SCP'
set timeout -1
spawn scp -P $env(TARGET_TAILSCALE_PORT) -o StrictHostKeyChecking=accept-new $env(DUMP_FILE) $env(DUMP_SHA_FILE) $env(TARGET_MANIFEST) "$env(TARGET_SERVER_USER)@$env(TARGET_TAILSCALE_IP):$env(REMOTE_INBOX)/"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCP

echo "上传完成"
echo "本机远端文件: ${REMOTE_INBOX}/${DUMP_BASENAME}"
`

const defaultClickHouseTargetDownloadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env REMOTE_INBOX REMOTE_WORKDIR REMOTE_MANIFEST RESTORE_ENV

if [ ! -f "$REMOTE_MANIFEST" ]; then
  echo "未找到上传清单 $REMOTE_MANIFEST，请先执行本地上传步骤" >&2
  exit 1
fi

source "$REMOTE_MANIFEST"
require_non_empty_env DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME PROJECT_NAME
mkdir -p "$REMOTE_WORKDIR"
mkdir -p "$(dirname "$RESTORE_ENV")"
chmod 700 "$REMOTE_WORKDIR"

cp "${REMOTE_INBOX}/${DUMP_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_BASENAME}"
cp "${REMOTE_INBOX}/${DUMP_SHA_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}"

cd "$REMOTE_WORKDIR"
shasum -a 256 -c "$DUMP_SHA_BASENAME"

cat > "$RESTORE_ENV" <<EOF
RESTORE_DUMP=${REMOTE_WORKDIR}/${DUMP_BASENAME}
RESTORE_SHA=${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "本机 ClickHouse 文件已就绪: ${REMOTE_WORKDIR}/${DUMP_BASENAME}"
`

const defaultClickHouseTargetExecScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd() {
  local cmd_var="$1"
  shift
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

clickhouse_url() {
  local host="$1"
  local port="$2"
  local database="${3:-}"
  local scheme="http"
  if [ "$port" = "8443" ]; then
    scheme="https"
  fi
  local base
  case "$host" in
    http://*|https://*) base="${host%/}" ;;
    *) base="${scheme}://${host}:${port}" ;;
  esac
  if [ -n "$database" ]; then
    printf "%s/?database=%s" "$base" "$database"
  else
    printf "%s/" "$base"
  fi
}

target_clickhouse_root_query() {
  local query="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT")"
}

target_clickhouse_query() {
  local query="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")"
}

target_clickhouse_file() {
  local file="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary @"$file" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")"
}

target_clickhouse_insert_native() {
  local table_name="$1"
  local data_file="$2"
  local query="INSERT INTO $(ch_ident "$TARGET_DB_NAME").$(ch_ident "$table_name") FORMAT Native"
  python_stream_clickhouse_native "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")" "$TARGET_DB_USER" "$TARGET_DB_PASSWORD" "$query" "$data_file"
}

python_stream_clickhouse_native() {
  local url="$1"
  local user="$2"
  local password="$3"
  local query="$4"
  local data_file="$5"
  "${TARGET_PYTHON_CMD:-python3}" - "$url" "$user" "$password" "$query" "$data_file" <<'PY'
import http.client
import os
import sys
import urllib.parse

url, user, password, query, data_file = sys.argv[1:6]
parsed = urllib.parse.urlparse(url)
params = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
params.extend([
    ("query", query),
    ("max_partitions_per_insert_block", "10000"),
])
path = parsed.path or "/"
if params:
    path += "?" + urllib.parse.urlencode(params)
conn_cls = http.client.HTTPSConnection if parsed.scheme == "https" else http.client.HTTPConnection
host = parsed.hostname or "127.0.0.1"
port = parsed.port or (443 if parsed.scheme == "https" else 80)
conn = conn_cls(host, port, timeout=3600)
headers = {
    "X-ClickHouse-User": user,
    "X-ClickHouse-Key": password,
    "Content-Length": str(os.path.getsize(data_file)),
}
conn.putrequest("POST", path)
for key, value in headers.items():
    conn.putheader(key, value)
conn.endheaders()
with open(data_file, "rb") as handle:
    while True:
        chunk = handle.read(1024 * 1024)
        if not chunk:
            break
        conn.send(chunk)
resp = conn.getresponse()
body = resp.read()
if resp.status >= 400:
    sys.stderr.write(body.decode("utf-8", "replace"))
    sys.exit(1)
PY
}

ch_ident() {
  local value="$1"
  local bt
  bt="$(printf '\140')"
  value="${value//$bt/$bt$bt}"
  printf "%s%s%s" "$bt" "$value" "$bt"
}

normalize_create_table_sql() {
  local source_file="$1"
  local target_file="$2"
  local decoded_file="${target_file}.decoded"
  local bt
  bt="$(printf '\140')"
  perl -0pe "s/\\\\n/\n/g; s/\\\\'/'/g; s/\\\\t/\t/g" "$source_file" > "$decoded_file"
  sed -E \
    -e "0,/^CREATE TABLE /s/^CREATE TABLE /CREATE TABLE IF NOT EXISTS /" \
    -e "0,/^CREATE TABLE IF NOT EXISTS IF NOT EXISTS /s//CREATE TABLE IF NOT EXISTS /" \
    -e "0,/^CREATE TABLE IF NOT EXISTS /s/^CREATE TABLE IF NOT EXISTS (${bt}[^${bt}]+${bt}|[A-Za-z0-9_]+)\./CREATE TABLE IF NOT EXISTS /" \
    "$decoded_file" > "$target_file"
}

require_non_empty_env TARGET_DB_TYPE TARGET_DB_HOST TARGET_DB_PORT TARGET_DB_DATABASE TARGET_DB_USER TARGET_DB_DROP_TABLES_BEFORE_IMPORT RESTORE_ENV TARGET_CLICKHOUSE_HTTP_CMD
require_env TARGET_DB_PASSWORD

case "$(printf "%s" "$TARGET_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 导入仅支持 ClickHouse，当前 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
    exit 1
    ;;
esac

case "$TARGET_DB_DROP_TABLES_BEFORE_IMPORT" in
  true|false)
    ;;
  *)
    echo "TARGET_DB_DROP_TABLES_BEFORE_IMPORT 只能配置为 true 或 false" >&2
    exit 1
    ;;
esac

TARGET_DB_NAME="$TARGET_DB_DATABASE"
TARGET_DB_CONNECT_HOST="${TARGET_DB_REMOTE_HOST:-$TARGET_DB_HOST}"

if [ ! -f "$RESTORE_ENV" ]; then
  echo "未找到 $RESTORE_ENV，请先执行目标下载步骤" >&2
  exit 1
fi

source "$RESTORE_ENV"
require_non_empty_env RESTORE_DUMP RESTORE_SHA SOURCE_DB_TYPE SOURCE_DB_NAME PROJECT_NAME

case "$(printf "%s" "$SOURCE_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 导入仅支持 ClickHouse 导出文件，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

RESTORE_DIR="${REMOTE_WORKDIR}/${PROJECT_NAME}_${SOURCE_DB_NAME}_clickhouse_restore"
rm -rf "$RESTORE_DIR"
mkdir -p "$RESTORE_DIR"
tar -xzf "$RESTORE_DUMP" -C "$RESTORE_DIR"
MANIFEST_FILE="${RESTORE_DIR}/manifest.tsv"
require_non_empty_env MANIFEST_FILE
if [ ! -f "$MANIFEST_FILE" ]; then
  echo "ClickHouse 导出包缺少 manifest.tsv" >&2
  exit 1
fi

echo "准备本机 ClickHouse 目标数据库 ${TARGET_DB_NAME} (${TARGET_DB_CONNECT_HOST}:${TARGET_DB_PORT})"
target_clickhouse_root_query "CREATE DATABASE IF NOT EXISTS $(ch_ident "$TARGET_DB_NAME")"

if [ "$TARGET_DB_DROP_TABLES_BEFORE_IMPORT" = "true" ]; then
  DROP_LIST="${RESTORE_DIR}/target_tables.tsv"
  target_clickhouse_query "SELECT name FROM system.tables WHERE database = currentDatabase() AND is_temporary = 0 ORDER BY name FORMAT TSVRaw" > "$DROP_LIST"
  while IFS= read -r target_table || [ -n "$target_table" ]; do
    [ -n "$target_table" ] || continue
    echo "删除本机目标表: ${TARGET_DB_NAME}.${target_table}"
    target_clickhouse_query "DROP TABLE IF EXISTS $(ch_ident "$TARGET_DB_NAME").$(ch_ident "$target_table") SYNC"
  done < "$DROP_LIST"
  echo "本机目标 ClickHouse 数据库表已删除"
fi

while IFS=$'\t' read -r table_name schema_path data_path || [ -n "$table_name" ]; do
  [ -n "$table_name" ] || continue
  source_schema="${RESTORE_DIR}/${schema_path}"
  target_schema="${RESTORE_DIR}/${schema_path}.target.sql"
  data_file="${RESTORE_DIR}/${data_path}"
  normalize_create_table_sql "$source_schema" "$target_schema"
  echo "创建本机目标表结构: ${TARGET_DB_NAME}.${table_name}"
  target_clickhouse_file "$target_schema"
  echo "导入本机目标表数据: ${TARGET_DB_NAME}.${table_name}"
  target_clickhouse_insert_native "$table_name" "$data_file"
done < "$MANIFEST_FILE"

echo "本机 ClickHouse 数据库导入完成"
`

const defaultClickHouseTableExportScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

safe_filename() {
  printf "%s" "$1" | tr -c 'A-Za-z0-9_-' '_'
}

write_env_assignment() {
  local name="$1"
  local value="$2"
  printf 'export %s=%q\n' "$name" "$value" >> "$REMOTE_ENV_LOCAL"
}

run_expect() {
  local delimiter
  local joined
  delimiter="$(printf '\034')"
  joined=""
  for arg in "$@"; do
    joined="${joined}${arg}${delimiter}"
  done
  EXPECT_COMMAND="$joined" "$LOCAL_EXPECT_CMD" <<'EXPECT_SCRIPT'
set timeout -1
set args {}
foreach part [split $env(EXPECT_COMMAND) "\034"] {
  if {$part ne ""} {
    lappend args $part
  }
}
spawn {*}$args
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(SOURCE_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCRIPT
}

require_non_empty_env PROJECT_NAME SOURCE_DB_TYPE SOURCE_DB_HOST SOURCE_DB_PORT SOURCE_DB_DATABASE SOURCE_DB_USER TABLE_NAME EXPORT_ROOT LOCAL_ENV SOURCE_CLICKHOUSE_HTTP_CMD LOCAL_EXPECT_CMD SOURCE_SERVER_TARGET_TAILSCALE_IP SOURCE_SERVER_TARGET_TAILSCALE_PORT SOURCE_SERVER_USER SOURCE_SERVER_PASSWORD
require_env SOURCE_DB_PASSWORD

case "$(printf "%s" "$SOURCE_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 单表导出仅支持 ClickHouse，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

SOURCE_DB_NAME="$SOURCE_DB_DATABASE"
SOURCE_DB_CONNECT_HOST="${SOURCE_DB_REMOTE_HOST:-$SOURCE_DB_HOST}"
SOURCE_TABLE_NAME="$TABLE_NAME"
SOURCE_SSH_HOST="$SOURCE_SERVER_TARGET_TAILSCALE_IP"
SOURCE_SSH_PORT="$SOURCE_SERVER_TARGET_TAILSCALE_PORT"
RUN_ID="$(date +%Y%m%d_%H%M%S)"
SAFE_TABLE_NAME="$(safe_filename "$SOURCE_TABLE_NAME")"
DUMP_BASENAME="${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}.tar.gz"
DUMP_SHA_BASENAME="${DUMP_BASENAME}.sha256"
DUMP_FILE="${EXPORT_ROOT}/${DUMP_BASENAME}"
DUMP_SHA_FILE="${DUMP_FILE}.sha256"
LATEST_ENV="$LOCAL_ENV"
REMOTE_EXPORT_ROOT="${EXPORT_ROOT}/source"
REMOTE_EXPORT_DIR="${REMOTE_EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}"
REMOTE_DUMP_FILE="${REMOTE_EXPORT_ROOT}/${DUMP_BASENAME}"
REMOTE_DUMP_SHA_FILE="${REMOTE_DUMP_FILE}.sha256"
REMOTE_SCRIPT_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_export.sh"
REMOTE_ENV_LOCAL="${EXPORT_ROOT}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote.env"
REMOTE_SCRIPT_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote_export.sh"
REMOTE_ENV_REMOTE="/tmp/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SAFE_TABLE_NAME}_${RUN_ID}_remote.env"

mkdir -p "$EXPORT_ROOT" "$(dirname "$LATEST_ENV")"
chmod 700 "$EXPORT_ROOT"

cat > "$REMOTE_SCRIPT_LOCAL" <<'REMOTE_SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

run_configured_cmd() {
  local cmd_var="$1"
  shift
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

clickhouse_url() {
  local host="$1"
  local port="$2"
  local database="${3:-}"
  local scheme="http"
  if [ "$port" = "8443" ]; then
    scheme="https"
  fi
  local base
  case "$host" in
    http://*|https://*) base="${host%/}" ;;
    *) base="${scheme}://${host}:${port}" ;;
  esac
  if [ -n "$database" ]; then
    printf "%s/?database=%s" "$base" "$database"
  else
    printf "%s/" "$base"
  fi
}

source_clickhouse_query() {
  local query="$1"
  run_configured_cmd SOURCE_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $SOURCE_DB_USER" \
    -H "X-ClickHouse-Key: $SOURCE_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$SOURCE_DB_CONNECT_HOST" "$SOURCE_DB_PORT" "$SOURCE_DB_NAME")"
}

ch_ident() {
  local value="$1"
  local bt
  bt="$(printf '\140')"
  value="${value//$bt/$bt$bt}"
  printf "%s%s%s" "$bt" "$value" "$bt"
}

SCHEMA_DIR="${REMOTE_EXPORT_DIR}/schemas"
DATA_DIR="${REMOTE_EXPORT_DIR}/data"
MANIFEST_FILE="${REMOTE_EXPORT_DIR}/manifest.tsv"
schema_file="${SCHEMA_DIR}/${SAFE_TABLE_NAME}.sql"
data_file="${DATA_DIR}/${SAFE_TABLE_NAME}.native"
table_ref="$(ch_ident "$SOURCE_DB_NAME").$(ch_ident "$SOURCE_TABLE_NAME")"

rm -rf "$REMOTE_EXPORT_DIR"
mkdir -p "$SCHEMA_DIR" "$DATA_DIR" "$REMOTE_EXPORT_ROOT"
chmod 700 "$REMOTE_EXPORT_ROOT" "$REMOTE_EXPORT_DIR"

echo "开始只读导出 mac mini ClickHouse 单表 ${SOURCE_DB_NAME}.${SOURCE_TABLE_NAME}"
source_clickhouse_query "SHOW CREATE TABLE ${table_ref}" > "$schema_file"
source_clickhouse_query "SELECT * FROM ${table_ref} FORMAT Native" > "$data_file"
printf "%s\t%s\t%s\n" "$SOURCE_TABLE_NAME" "schemas/${SAFE_TABLE_NAME}.sql" "data/${SAFE_TABLE_NAME}.native" > "$MANIFEST_FILE"

tar -C "$REMOTE_EXPORT_DIR" -czf "$REMOTE_DUMP_FILE" manifest.tsv schemas data
(cd "$(dirname "$REMOTE_DUMP_FILE")" && shasum -a 256 "$(basename "$REMOTE_DUMP_FILE")") > "$REMOTE_DUMP_SHA_FILE"
echo "ClickHouse 单表导出完成: $REMOTE_DUMP_FILE"
REMOTE_SCRIPT

: > "$REMOTE_ENV_LOCAL"
write_env_assignment SOURCE_CLICKHOUSE_HTTP_CMD "$SOURCE_CLICKHOUSE_HTTP_CMD"
write_env_assignment SOURCE_DB_CONNECT_HOST "$SOURCE_DB_CONNECT_HOST"
write_env_assignment SOURCE_DB_PORT "$SOURCE_DB_PORT"
write_env_assignment SOURCE_DB_NAME "$SOURCE_DB_NAME"
write_env_assignment SOURCE_DB_USER "$SOURCE_DB_USER"
write_env_assignment SOURCE_DB_PASSWORD "$SOURCE_DB_PASSWORD"
write_env_assignment SOURCE_TABLE_NAME "$SOURCE_TABLE_NAME"
write_env_assignment SAFE_TABLE_NAME "$SAFE_TABLE_NAME"
write_env_assignment REMOTE_EXPORT_ROOT "$REMOTE_EXPORT_ROOT"
write_env_assignment REMOTE_EXPORT_DIR "$REMOTE_EXPORT_DIR"
write_env_assignment REMOTE_DUMP_FILE "$REMOTE_DUMP_FILE"
write_env_assignment REMOTE_DUMP_SHA_FILE "$REMOTE_DUMP_SHA_FILE"

chmod 600 "$REMOTE_ENV_LOCAL"

echo "通过 Tailscale SSH 上传 ClickHouse 单表导出脚本到源服务器 ${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${SOURCE_SSH_PORT}"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "$REMOTE_SCRIPT_LOCAL" "$REMOTE_ENV_LOCAL" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:/tmp/"

echo "通过 Tailscale SSH 在源服务器只读导出 ClickHouse 单表"
run_expect ssh -p "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}" "chmod 700 '$REMOTE_SCRIPT_REMOTE' '$REMOTE_ENV_REMOTE' && source '$REMOTE_ENV_REMOTE' && bash '$REMOTE_SCRIPT_REMOTE'"

echo "通过 Tailscale 拉取源服务器 ClickHouse 单表导出文件到本机"
run_expect scp -P "$SOURCE_SSH_PORT" -o StrictHostKeyChecking=accept-new "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_FILE}" "${SOURCE_SERVER_USER}@${SOURCE_SSH_HOST}:${REMOTE_DUMP_SHA_FILE}" "$EXPORT_ROOT/"

cat > "$LATEST_ENV" <<EOF
DUMP_FILE=$DUMP_FILE
DUMP_SHA_FILE=$DUMP_SHA_FILE
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "ClickHouse 单表导出完成: $DUMP_FILE"
echo "校验文件: $DUMP_SHA_FILE"
echo "流程清单: $LATEST_ENV"
`

const defaultClickHouseTableUploadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env TARGET_SERVER_IP TARGET_SERVER_PORT TARGET_TAILSCALE_IP TARGET_TAILSCALE_PORT TARGET_SERVER_USER TARGET_SERVER_PASSWORD REMOTE_INBOX REMOTE_WORKDIR LOCAL_ENV TARGET_MANIFEST RESTORE_ENV LOCAL_EXPECT_CMD

if [ ! -f "$LOCAL_ENV" ]; then
  echo "未找到 $LOCAL_ENV，请先执行 ClickHouse 单表导出步骤" >&2
  exit 1
fi

is_local_target() {
  case "${TARGET_SERVER_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  case "${TARGET_TAILSCALE_IP}" in
    127.*|localhost|::1)
      return 0
      ;;
  esac
  return 1
}

source "$LOCAL_ENV"
require_non_empty_env DUMP_FILE DUMP_SHA_FILE DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_NAME PROJECT_NAME
mkdir -p "$(dirname "$TARGET_MANIFEST")"

cat > "$TARGET_MANIFEST" <<EOF
DUMP_BASENAME=$DUMP_BASENAME
DUMP_SHA_BASENAME=$DUMP_SHA_BASENAME
REMOTE_INBOX=$REMOTE_INBOX
REMOTE_WORKDIR=$REMOTE_WORKDIR
RESTORE_ENV=$RESTORE_ENV
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

if is_local_target; then
  echo "目标为本机，直接复制 ClickHouse 单表导出文件到 ${REMOTE_INBOX}"
  mkdir -p "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  chmod 700 "$REMOTE_INBOX" "$REMOTE_WORKDIR"
  cp "$DUMP_FILE" "$DUMP_SHA_FILE" "$TARGET_MANIFEST" "$REMOTE_INBOX/"
  echo "本机 ClickHouse 单表文件已就绪: ${REMOTE_INBOX}/${DUMP_BASENAME}"
  exit 0
fi

echo "准备初始化本机目标目录 ${TARGET_SERVER_USER}@${TARGET_SERVER_IP}:${TARGET_SERVER_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_SERVER_PORT="$TARGET_SERVER_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_SERVER_IP="$TARGET_SERVER_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
REMOTE_WORKDIR="$REMOTE_WORKDIR" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SSH'
set timeout -1
spawn ssh -p $env(TARGET_SERVER_PORT) -o StrictHostKeyChecking=accept-new "$env(TARGET_SERVER_USER)@$env(TARGET_SERVER_IP)" "mkdir -p '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)' && chmod 700 '$env(REMOTE_INBOX)' '$env(REMOTE_WORKDIR)'"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SSH

echo "准备通过 Tailscale 传输 ClickHouse 单表导出文件到本机 ${TARGET_SERVER_USER}@${TARGET_TAILSCALE_IP}:${TARGET_TAILSCALE_PORT}:${REMOTE_INBOX}"
TARGET_SERVER_PASSWORD="$TARGET_SERVER_PASSWORD" \
TARGET_TAILSCALE_PORT="$TARGET_TAILSCALE_PORT" \
TARGET_SERVER_USER="$TARGET_SERVER_USER" \
TARGET_TAILSCALE_IP="$TARGET_TAILSCALE_IP" \
REMOTE_INBOX="$REMOTE_INBOX" \
DUMP_FILE="$DUMP_FILE" \
DUMP_SHA_FILE="$DUMP_SHA_FILE" \
TARGET_MANIFEST="$TARGET_MANIFEST" \
"$LOCAL_EXPECT_CMD" <<'EXPECT_SCP'
set timeout -1
spawn scp -P $env(TARGET_TAILSCALE_PORT) -o StrictHostKeyChecking=accept-new $env(DUMP_FILE) $env(DUMP_SHA_FILE) $env(TARGET_MANIFEST) "$env(TARGET_SERVER_USER)@$env(TARGET_TAILSCALE_IP):$env(REMOTE_INBOX)/"
expect {
  -nocase -re "password:" {
    log_user 0
    send -- "$env(TARGET_SERVER_PASSWORD)\r"
    log_user 1
    exp_continue
  }
  eof
}
catch wait result
exit [lindex $result 3]
EXPECT_SCP

echo "上传完成"
echo "本机远端文件: ${REMOTE_INBOX}/${DUMP_BASENAME}"
`

const defaultClickHouseTableTargetDownloadScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env REMOTE_INBOX REMOTE_WORKDIR REMOTE_MANIFEST RESTORE_ENV

if [ ! -f "$REMOTE_MANIFEST" ]; then
  echo "未找到上传清单 $REMOTE_MANIFEST，请先执行本地上传步骤" >&2
  exit 1
fi

source "$REMOTE_MANIFEST"
require_non_empty_env DUMP_BASENAME DUMP_SHA_BASENAME SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_NAME PROJECT_NAME
mkdir -p "$REMOTE_WORKDIR"
mkdir -p "$(dirname "$RESTORE_ENV")"
chmod 700 "$REMOTE_WORKDIR"

cp "${REMOTE_INBOX}/${DUMP_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_BASENAME}"
cp "${REMOTE_INBOX}/${DUMP_SHA_BASENAME}" "${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}"

cd "$REMOTE_WORKDIR"
shasum -a 256 -c "$DUMP_SHA_BASENAME"

cat > "$RESTORE_ENV" <<EOF
RESTORE_DUMP=${REMOTE_WORKDIR}/${DUMP_BASENAME}
RESTORE_SHA=${REMOTE_WORKDIR}/${DUMP_SHA_BASENAME}
SOURCE_DB_TYPE=$SOURCE_DB_TYPE
SOURCE_DB_NAME=$SOURCE_DB_NAME
SOURCE_TABLE_NAME=$SOURCE_TABLE_NAME
PROJECT_NAME=$PROJECT_NAME
EOF

echo "本机 ClickHouse 单表文件已就绪: ${REMOTE_WORKDIR}/${DUMP_BASENAME}"
`

const defaultClickHouseTableTargetExecScript = `#!/usr/bin/env bash
set -euo pipefail

require_env() {
  for name in "$@"; do
    if [ -z "${!name+x}" ]; then
      echo "缺少必要资源配置: $name" >&2
      exit 1
    fi
  done
}

require_non_empty_env() {
  for name in "$@"; do
    require_env "$name"
    if [ -z "${!name}" ]; then
      echo "资源配置不能为空: $name" >&2
      exit 1
    fi
  done
}

run_configured_cmd() {
  local cmd_var="$1"
  shift
  require_non_empty_env "$cmd_var"
  read -r -a cmd_parts <<< "${!cmd_var}"
  "${cmd_parts[@]}" "$@"
}

clickhouse_url() {
  local host="$1"
  local port="$2"
  local database="${3:-}"
  local scheme="http"
  if [ "$port" = "8443" ]; then
    scheme="https"
  fi
  local base
  case "$host" in
    http://*|https://*) base="${host%/}" ;;
    *) base="${scheme}://${host}:${port}" ;;
  esac
  if [ -n "$database" ]; then
    printf "%s/?database=%s" "$base" "$database"
  else
    printf "%s/" "$base"
  fi
}

target_clickhouse_root_query() {
  local query="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT")"
}

target_clickhouse_query() {
  local query="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary "$query" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")"
}

target_clickhouse_file() {
  local file="$1"
  run_configured_cmd TARGET_CLICKHOUSE_HTTP_CMD \
    -sS --fail --show-error \
    -H "X-ClickHouse-User: $TARGET_DB_USER" \
    -H "X-ClickHouse-Key: $TARGET_DB_PASSWORD" \
    --data-binary @"$file" \
    "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")"
}

target_clickhouse_insert_native() {
  local table_name="$1"
  local data_file="$2"
  local query="INSERT INTO $(ch_ident "$TARGET_DB_NAME").$(ch_ident "$table_name") FORMAT Native"
  python_stream_clickhouse_native "$(clickhouse_url "$TARGET_DB_CONNECT_HOST" "$TARGET_DB_PORT" "$TARGET_DB_NAME")" "$TARGET_DB_USER" "$TARGET_DB_PASSWORD" "$query" "$data_file"
}

python_stream_clickhouse_native() {
  local url="$1"
  local user="$2"
  local password="$3"
  local query="$4"
  local data_file="$5"
  "${TARGET_PYTHON_CMD:-python3}" - "$url" "$user" "$password" "$query" "$data_file" <<'PY'
import http.client
import os
import sys
import urllib.parse

url, user, password, query, data_file = sys.argv[1:6]
parsed = urllib.parse.urlparse(url)
params = urllib.parse.parse_qsl(parsed.query, keep_blank_values=True)
params.extend([
    ("query", query),
    ("max_partitions_per_insert_block", "10000"),
])
path = parsed.path or "/"
if params:
    path += "?" + urllib.parse.urlencode(params)
conn_cls = http.client.HTTPSConnection if parsed.scheme == "https" else http.client.HTTPConnection
host = parsed.hostname or "127.0.0.1"
port = parsed.port or (443 if parsed.scheme == "https" else 80)
conn = conn_cls(host, port, timeout=3600)
headers = {
    "X-ClickHouse-User": user,
    "X-ClickHouse-Key": password,
    "Content-Length": str(os.path.getsize(data_file)),
}
conn.putrequest("POST", path)
for key, value in headers.items():
    conn.putheader(key, value)
conn.endheaders()
with open(data_file, "rb") as handle:
    while True:
        chunk = handle.read(1024 * 1024)
        if not chunk:
            break
        conn.send(chunk)
resp = conn.getresponse()
body = resp.read()
if resp.status >= 400:
    sys.stderr.write(body.decode("utf-8", "replace"))
    sys.exit(1)
PY
}

ch_ident() {
  local value="$1"
  local bt
  bt="$(printf '\140')"
  value="${value//$bt/$bt$bt}"
  printf "%s%s%s" "$bt" "$value" "$bt"
}

normalize_create_table_sql() {
  local source_file="$1"
  local target_file="$2"
  local decoded_file="${target_file}.decoded"
  local bt
  bt="$(printf '\140')"
  perl -0pe "s/\\\\n/\n/g; s/\\\\'/'/g; s/\\\\t/\t/g" "$source_file" > "$decoded_file"
  sed -E \
    -e "0,/^CREATE TABLE /s/^CREATE TABLE /CREATE TABLE IF NOT EXISTS /" \
    -e "0,/^CREATE TABLE IF NOT EXISTS IF NOT EXISTS /s//CREATE TABLE IF NOT EXISTS /" \
    -e "0,/^CREATE TABLE IF NOT EXISTS /s/^CREATE TABLE IF NOT EXISTS (${bt}[^${bt}]+${bt}|[A-Za-z0-9_]+)\./CREATE TABLE IF NOT EXISTS /" \
    "$decoded_file" > "$target_file"
}

require_non_empty_env TARGET_DB_TYPE TARGET_DB_HOST TARGET_DB_PORT TARGET_DB_DATABASE TARGET_DB_USER TARGET_TABLE_DROP_BEFORE_IMPORT RESTORE_ENV TARGET_CLICKHOUSE_HTTP_CMD
require_env TARGET_DB_PASSWORD

case "$(printf "%s" "$TARGET_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 单表导入仅支持 ClickHouse，当前 TARGET_DB_TYPE: $TARGET_DB_TYPE" >&2
    exit 1
    ;;
esac

case "$TARGET_TABLE_DROP_BEFORE_IMPORT" in
  true|false)
    ;;
  *)
    echo "TARGET_TABLE_DROP_BEFORE_IMPORT 只能配置为 true 或 false" >&2
    exit 1
    ;;
esac

TARGET_DB_NAME="$TARGET_DB_DATABASE"
TARGET_DB_CONNECT_HOST="${TARGET_DB_REMOTE_HOST:-$TARGET_DB_HOST}"

if [ ! -f "$RESTORE_ENV" ]; then
  echo "未找到 $RESTORE_ENV，请先执行目标下载步骤" >&2
  exit 1
fi

source "$RESTORE_ENV"
require_non_empty_env RESTORE_DUMP RESTORE_SHA SOURCE_DB_TYPE SOURCE_DB_NAME SOURCE_TABLE_NAME PROJECT_NAME

case "$(printf "%s" "$SOURCE_DB_TYPE" | tr '[:upper:]' '[:lower:]')" in
  clickhouse)
    ;;
  *)
    echo "ClickHouse 单表导入仅支持 ClickHouse 导出文件，当前 SOURCE_DB_TYPE: $SOURCE_DB_TYPE" >&2
    exit 1
    ;;
esac

RESTORE_DIR="${REMOTE_WORKDIR}/${PROJECT_NAME}_${SOURCE_DB_NAME}_${SOURCE_TABLE_NAME}_clickhouse_restore"
rm -rf "$RESTORE_DIR"
mkdir -p "$RESTORE_DIR"
tar -xzf "$RESTORE_DUMP" -C "$RESTORE_DIR"
MANIFEST_FILE="${RESTORE_DIR}/manifest.tsv"
if [ ! -f "$MANIFEST_FILE" ]; then
  echo "ClickHouse 单表导出包缺少 manifest.tsv" >&2
  exit 1
fi

echo "准备本机 ClickHouse 目标数据库 ${TARGET_DB_NAME} (${TARGET_DB_CONNECT_HOST}:${TARGET_DB_PORT})"
target_clickhouse_root_query "CREATE DATABASE IF NOT EXISTS $(ch_ident "$TARGET_DB_NAME")"

while IFS=$'\t' read -r table_name schema_path data_path || [ -n "$table_name" ]; do
  [ -n "$table_name" ] || continue
  source_schema="${RESTORE_DIR}/${schema_path}"
  target_schema="${RESTORE_DIR}/${schema_path}.target.sql"
  data_file="${RESTORE_DIR}/${data_path}"
  normalize_create_table_sql "$source_schema" "$target_schema"

  if [ "$TARGET_TABLE_DROP_BEFORE_IMPORT" = "true" ]; then
    echo "删除本机目标表: ${TARGET_DB_NAME}.${table_name}"
    target_clickhouse_query "DROP TABLE IF EXISTS $(ch_ident "$TARGET_DB_NAME").$(ch_ident "$table_name") SYNC"
  fi

  echo "创建本机目标表结构: ${TARGET_DB_NAME}.${table_name}"
  target_clickhouse_file "$target_schema"
  echo "导入本机目标表数据: ${TARGET_DB_NAME}.${table_name}"
  target_clickhouse_insert_native "$table_name" "$data_file"
done < "$MANIFEST_FILE"

echo "本机 ClickHouse 单表导入完成"
`
