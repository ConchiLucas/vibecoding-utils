package initialize

import (
	"os"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Gorm() *gorm.DB {
	switch global.GVA_CONFIG.System.DbType {
	case "mysql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mysql.Dbname
		return GormMysql()
	case "pgsql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Pgsql.Dbname
		return GormPgSql()
	case "oracle":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Oracle.Dbname
		return GormOracle()
	case "mssql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mssql.Dbname
		return GormMssql()
	case "sqlite":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Sqlite.Dbname
		return GormSqlite()
	default:
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mysql.Dbname
		return GormMysql()
	}
}

func RegisterTables() {
	if global.GVA_CONFIG.System.DisableAutoMigrate {
		global.GVA_LOG.Info("auto-migrate is disabled, skipping table registration")
		return
	}

	db := global.GVA_DB
	err := db.AutoMigrate(
		system.TbUser{},
		system.TbServer{},
		system.TbProjectGroup{},
		system.TbProject{},
		system.TbLogProjectGroup{},
		system.TbLogProject{},
		system.TbLogProjectRoute{},
		system.TbDevelopmentPrepare{},
		system.TbProjectScript{},
		system.TbScriptCategory{},
		system.TbScriptWorkflow{},
		system.TbScriptStep{},
		system.TbScriptResourceCategory{},
		system.TbScriptResourceConfig{},
		system.TbScriptExecution{},
		system.TbProjectRoute{},
		system.TbDictData{},
		system.TbInterfaceServer{},
		system.TbInterfaceEnv{},
		system.TbInterface{},
		system.TbConnection{},
		system.TbTable{},
		system.TbTableColumn{},
		system.TbTableRelate{},
		system.TbEntity{},
		system.TbColumn{},
		system.TbClient{},
		system.TbInterfaceParams{},
		system.TbInterfaceLog{},
		system.TbAgileRequestLog{},
		system.TbTablePrefer{},
		system.TbSQLQueryHistory{},
		system.TbInterfaceServerUser{},
		system.TbInterfaceProject{},
		system.TbGenerateProject{},
		system.TbGenerateProjectInstance{},
		system.TbGenerateDbTemplateType{},
		system.TbGenerateDbTemplateScript{},
		system.TbGenerateProjectPathGroup{},
		system.TbGenerateProjectPath{},
		system.TbGenerateProjectPathModel{},
		system.TbGenerateFieldSnippet{},
		system.TbAIConfig{},
	)

	// User requests that we physically clean up the orphaned legacy columns that are no longer in Go models.
	if err == nil {
		err = dropRemovedAIAssistantTables(db)
		if err != nil {
			global.GVA_LOG.Error("drop removed AI assistant tables failed", zap.Error(err))
			os.Exit(0)
		}

		err = repairLegacyPostgresAutoIncrementIDs(db)
		if err != nil {
			global.GVA_LOG.Error("repair postgres auto-increment ids failed", zap.Error(err))
			os.Exit(0)
		}

		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS server_project_path;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS local_execute_command;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS server_execute_command;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS local_start_command;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS user_name;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS file_name;")
		db.Exec("ALTER TABLE tb_project DROP COLUMN IF EXISTS server_id;") // server 关联已移至路由层

		db.Exec("ALTER TABLE tb_server DROP COLUMN IF EXISTS user_name;")
		dropRemovedScriptManagerColumns(db)

		if err = seedDefaultScriptManagerData(db); err != nil {
			global.GVA_LOG.Error("seed default script manager data failed", zap.Error(err))
			os.Exit(0)
		}
	}

	if err != nil {
		global.GVA_LOG.Error("register table failed", zap.Error(err))
		os.Exit(0)
	}

	err = bizModel()

	if err != nil {
		global.GVA_LOG.Error("register biz_table failed", zap.Error(err))
		os.Exit(0)
	}
	global.GVA_LOG.Info("register table success")
}
