package initialize

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var postgresIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type postgresIDColumnMetadata struct {
	ColumnDefault sql.NullString `gorm:"column:column_default"`
	IsIdentity    string         `gorm:"column:is_identity"`
}

func repairLegacyPostgresAutoIncrementIDs(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return nil
	}

	return ensurePostgresAutoIncrementID(db, "tb_connection")
}

func ensurePostgresAutoIncrementID(db *gorm.DB, tableName string) error {
	var columns []postgresIDColumnMetadata
	if err := db.Raw(`
SELECT column_default, is_identity
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = ?
  AND column_name = 'id'
LIMIT 1`, tableName).Scan(&columns).Error; err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	column := columns[0]
	if strings.EqualFold(column.IsIdentity, "YES") {
		return nil
	}
	if column.ColumnDefault.Valid && strings.TrimSpace(column.ColumnDefault.String) != "" {
		return nil
	}

	repairSQL, err := postgresAutoIncrementIDRepairSQL(tableName)
	if err != nil {
		return err
	}
	return db.Exec(repairSQL).Error
}

func postgresAutoIncrementIDRepairSQL(tableName string) (string, error) {
	if !postgresIdentifierPattern.MatchString(tableName) {
		return "", fmt.Errorf("unsafe postgres table name %q", tableName)
	}

	sequenceName := tableName + "_id_seq"
	quotedTable := `"` + tableName + `"`
	return fmt.Sprintf(`
CREATE SEQUENCE IF NOT EXISTS %[1]s;
ALTER SEQUENCE %[1]s OWNED BY %[2]s."id";
SELECT setval('%[1]s', COALESCE((SELECT MAX("id") FROM %[2]s), 0) + 1, false);
ALTER TABLE %[2]s ALTER COLUMN "id" SET DEFAULT nextval('%[1]s');`,
		sequenceName,
		quotedTable,
	), nil
}
