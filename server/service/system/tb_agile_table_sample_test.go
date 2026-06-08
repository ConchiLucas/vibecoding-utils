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

func setupAgileTableSampleTestDB(t *testing.T) *gorm.DB {
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

	if err := db.AutoMigrate(&modelSystem.TbAgileTableSample{}, &modelSystem.TbAgileTableSampleHistory{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestAgileTableSampleListUsesConnectionScopeNotProject(t *testing.T) {
	db := setupAgileTableSampleTestDB(t)
	record := modelSystem.TbAgileTableSample{
		ProjectConfigID: 16,
		ConnectionID:    5,
		DatabaseName:    "analytics",
		DBTableName:     "orders",
		UserName:        "admin",
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create sample: %v", err)
	}

	got, err := (&TbAgileTableSampleService{}).List(systemReq.AgileTableSampleScope{
		ConnectionID: 5,
	}, "admin")
	if err != nil {
		t.Fatalf("list samples: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("sample count = %d, want 1", len(got))
	}
	if got[0].DBTableName != "orders" {
		t.Fatalf("table name = %q, want orders", got[0].DBTableName)
	}
}

func TestAgileTableSampleSaveReplacesByConnectionScopeNotProject(t *testing.T) {
	db := setupAgileTableSampleTestDB(t)
	oldRecord := modelSystem.TbAgileTableSample{
		ProjectConfigID: 16,
		ConnectionID:    5,
		DatabaseName:    "analytics",
		DBTableName:     "old_orders",
		UserName:        "admin",
	}
	if err := db.Create(&oldRecord).Error; err != nil {
		t.Fatalf("create old sample: %v", err)
	}

	got, err := (&TbAgileTableSampleService{}).Save(systemReq.AgileTableSampleSave{
		AgileTableSampleScope: systemReq.AgileTableSampleScope{
			ProjectConfigID: 17,
			ConnectionID:    5,
		},
		Tables: []systemReq.AgileTableSampleItem{
			{
				DatabaseName: "analytics",
				TableName:    "new_orders",
			},
		},
	}, "admin")
	if err != nil {
		t.Fatalf("save samples: %v", err)
	}
	if len(got) != 1 || got[0].DBTableName != "new_orders" {
		t.Fatalf("saved samples = %#v, want new_orders only", got)
	}

	var records []modelSystem.TbAgileTableSample
	if err := db.Where("connection_id = ? AND user_name = ?", 5, "admin").Find(&records).Error; err != nil {
		t.Fatalf("query samples: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("stored sample count = %d, want 1: %#v", len(records), records)
	}
	if records[0].DBTableName != "new_orders" {
		t.Fatalf("stored table = %q, want new_orders", records[0].DBTableName)
	}
}

func TestAgileTableSampleHistoryUsesConnectionScopeNotProject(t *testing.T) {
	db := setupAgileTableSampleTestDB(t)
	history := modelSystem.TbAgileTableSampleHistory{
		ProjectConfigID: 16,
		ConnectionID:    5,
		UserName:        "admin",
		BusinessName:    "订单分析",
		TableCount:      1,
		TableSnapshot:   `[{"databaseName":"analytics","tableName":"orders"}]`,
	}
	if err := db.Create(&history).Error; err != nil {
		t.Fatalf("create history: %v", err)
	}

	got, err := (&TbAgileTableSampleService{}).History(systemReq.AgileTableSampleScope{
		ConnectionID: 5,
	}, "admin")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("history count = %d, want 1", len(got))
	}
	if got[0].HistoryName != "订单分析" {
		t.Fatalf("history name = %q, want 订单分析", got[0].HistoryName)
	}
}
