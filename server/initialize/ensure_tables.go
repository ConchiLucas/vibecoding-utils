package initialize

import (
	"context"
	sysModel "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/service/system"
	"gorm.io/gorm"
)

const initOrderEnsureTables = system.InitOrderExternal - 1

type ensureTables struct{}

// auto run
func init() {
	system.RegisterInit(initOrderEnsureTables, &ensureTables{})
}

func (e *ensureTables) InitializerName() string {
	return "ensure_tables_created"
}
func (e *ensureTables) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, seedDefaultScriptManagerData(db)
}

func (e *ensureTables) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if !db.Migrator().HasTable(&sysModel.TbScriptWorkflow{}) {
		return false
	}
	return defaultScriptWorkflowExists(db)
}

func (e *ensureTables) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	tables := []interface{}{
		sysModel.TbUser{},
		sysModel.TbServer{},
		sysModel.TbProject{},
		sysModel.TbProjectRoute{},
		sysModel.TbProjectScript{},
		sysModel.TbScriptCategory{},
		sysModel.TbScriptWorkflow{},
		sysModel.TbScriptStep{},
		sysModel.TbScriptResourceCategory{},
		sysModel.TbScriptResourceConfig{},
		sysModel.TbScriptExecution{},
		sysModel.TbAgileRequestLog{},
		sysModel.TbSQLQueryHistory{},
		sysModel.TbAIChatHistory{},
	}
	for _, t := range tables {
		_ = db.AutoMigrate(&t)
	}
	dropRemovedScriptManagerColumns(db)
	return ctx, nil
}

func (e *ensureTables) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	tables := []interface{}{
		sysModel.TbUser{},
		sysModel.TbServer{},
		sysModel.TbProject{},
		sysModel.TbProjectRoute{},
		sysModel.TbProjectScript{},
		sysModel.TbScriptCategory{},
		sysModel.TbScriptWorkflow{},
		sysModel.TbScriptStep{},
		sysModel.TbScriptResourceCategory{},
		sysModel.TbScriptResourceConfig{},
		sysModel.TbScriptExecution{},
		sysModel.TbAgileRequestLog{},
		sysModel.TbSQLQueryHistory{},
		sysModel.TbAIChatHistory{},
	}
	yes := true
	for _, t := range tables {
		yes = yes && db.Migrator().HasTable(t)
	}
	return yes
}
