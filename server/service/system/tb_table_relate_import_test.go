package system

import (
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestImportTableRelationsCreatesRecordsAndSkipsDuplicates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:table_relate_import?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	oldDB := global.GVA_DB
	global.GVA_DB = db
	defer func() {
		global.GVA_DB = oldDB
	}()

	if err := db.AutoMigrate(&modelSystem.TbTableRelate{}); err != nil {
		t.Fatalf("migrate table relate: %v", err)
	}

	existing := modelSystem.TbTableRelate{
		ProjectConfigID:    12,
		DatabaseName:       "order_db",
		TbName:             "orders",
		ColumnName:         "user_id",
		ColumnType:         "bigint",
		RelateDatabaseName: "user_db",
		RelateTableName:    "users",
		RelateColumnName:   "id",
		RelateColumnType:   "bigint",
		UserName:           "seed",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed duplicate relation: %v", err)
	}

	req := systemReq.ImportTableRelationsRequest{
		ProjectConfigID: 12,
		Relations: []systemReq.ImportTableRelation{
			{
				Source: systemReq.ImportTableRelationEndpoint{
					DatabaseName: "order_db",
					TableName:    "orders",
					ColumnName:   "user_id",
					ColumnType:   "bigint",
				},
				Target: systemReq.ImportTableRelationEndpoint{
					DatabaseName: "user_db",
					TableName:    "users",
					ColumnName:   "id",
					ColumnType:   "bigint",
				},
			},
			{
				Source: systemReq.ImportTableRelationEndpoint{
					DatabaseName: "order_db",
					TableName:    "orders",
					ColumnName:   "address_id",
					ColumnType:   "bigint",
				},
				Target: systemReq.ImportTableRelationEndpoint{
					DatabaseName: "user_db",
					TableName:    "addresses",
					ColumnName:   "id",
					ColumnType:   "bigint",
				},
			},
		},
	}

	result, err := (&TbTableRelateService{}).ImportTableRelations(req, "ai")
	if err != nil {
		t.Fatalf("import relations: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("created = %d, want 1", result.Created)
	}
	if result.Skipped != 1 {
		t.Fatalf("skipped = %d, want 1", result.Skipped)
	}
	if len(result.Failed) != 0 {
		t.Fatalf("failed = %#v, want empty", result.Failed)
	}

	var rows []modelSystem.TbTableRelate
	if err := db.Order("id asc").Find(&rows).Error; err != nil {
		t.Fatalf("query rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}

	inserted := rows[1]
	if inserted.ProjectConfigID != 12 ||
		inserted.DatabaseName != "order_db" ||
		inserted.TbName != "orders" ||
		inserted.ColumnName != "address_id" ||
		inserted.ColumnType != "bigint" ||
		inserted.RelateDatabaseName != "user_db" ||
		inserted.RelateTableName != "addresses" ||
		inserted.RelateColumnName != "id" ||
		inserted.RelateColumnType != "bigint" ||
		inserted.UserName != "ai" {
		t.Fatalf("inserted row mismatch: %#v", inserted)
	}
}
