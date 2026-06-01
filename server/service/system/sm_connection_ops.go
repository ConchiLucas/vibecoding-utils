package system

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)

const (
	remoteDBPingTimeout     = utils.RemoteDBConnectionTimeout
	remoteDBPingMaxAttempts = 2
)

func (s *TbConnectionService) TestConnection(connID uint) error {
	var conn system.TbConnection
	if err := global.GVA_DB.Where("id = ?", connID).First(&conn).Error; err != nil {
		return fmt.Errorf("连接配置不存在: %w", err)
	}
	return s.TestConnectionByPayload(conn)
}

// TestConnectionByPayload tests if a database connection is reachable without saving.
func (s *TbConnectionService) TestConnectionByPayload(conn system.TbConnection) error {
	dsn, driverName := buildDSN(conn)
	if driverName == "" {
		return fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}
	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		return formatRemoteDBConnectionError(conn, err)
	}
	defer db.Close()
	return nil
}

// InitConnection imports all tables and columns from the target DB
func (s *TbConnectionService) InitConnection(connID uint, userName string) error {
	var conn system.TbConnection
	if err := global.GVA_DB.Where("id = ?", connID).First(&conn).Error; err != nil {
		return fmt.Errorf("连接配置不存在: %w", err)
	}

	global.GVA_DB.Where("connection_id = ?", connID).Unscoped().Delete(&system.TbTable{})
	global.GVA_DB.Where("connection_id = ?", connID).Unscoped().Delete(&system.TbTableColumn{})

	dsn, driverName := buildDSN(conn)
	if driverName == "" {
		return fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}
	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		return formatRemoteDBConnectionError(conn, err)
	}
	defer db.Close()

	ct := strings.ToLower(conn.ConnectionType)
	if ct == "mysql" {
		return importMySQLSchema(db, conn, userName)
	} else if ct == "postgresql" || ct == "pgsql" {
		return importPostgresSchema(db, conn, userName)
	} else if ct == "sqlserver" || ct == "mssql" {
		return importSqlserverSchema(db, conn, userName)
	} else if ct == "oracle" {
		return importOracleSchema(db, conn, userName)
	} else if ct == "sqlite" {
		return importSqliteSchema(db, conn, userName)
	} else if ct == "clickhouse" {
		return importClickHouseSchema(db, conn, userName)
	}
	return fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
}

func openRemoteSQLDB(driverName, dsn string) (*sql.DB, error) {
	var lastErr error
	for attempt := 1; attempt <= remoteDBPingMaxAttempts; attempt++ {
		db, err := sql.Open(driverName, dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		ctx, cancel := context.WithTimeout(context.Background(), remoteDBPingTimeout)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return db, nil
		}
		_ = db.Close()
		lastErr = err
		if !isRemoteConnectionDrop(err) || attempt == remoteDBPingMaxAttempts {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, lastErr
}

func formatRemoteDBConnectionError(conn system.TbConnection, err error) error {
	msg := "数据库不可达"
	if isRemoteConnectionDrop(err) {
		msg = "数据库连接中断或被监听器关闭"
	}
	return fmt.Errorf("%s [%s]: %w", msg, remoteConnectionTarget(conn), err)
}

func remoteConnectionTarget(conn system.TbConnection) string {
	host := strings.TrimSpace(conn.ConnectionUrl)
	if conn.Port > 0 {
		host = fmt.Sprintf("%s:%d", host, conn.Port)
	}
	if databaseName := strings.TrimSpace(conn.DatabaseName); databaseName != "" {
		host = host + "/" + databaseName
	}
	return host
}

func isRemoteConnectionDrop(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	for _, token := range []string{
		"broken pipe",
		"connection reset by peer",
		"eof",
		"network is down",
		"i/o timeout",
		"operation timed out",
		"no route to host",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func buildDSN(conn system.TbConnection) (dsn, driverName string) {
	ct := strings.ToLower(conn.ConnectionType)
	if ct == "mysql" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			conn.DbLoginName, conn.DbLoginPassword,
			conn.ConnectionUrl, conn.Port, conn.DatabaseName)
		driverName = "mysql"
	} else if ct == "postgresql" || ct == "pgsql" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
			conn.ConnectionUrl, conn.Port,
			conn.DbLoginName, conn.DbLoginPassword, conn.DatabaseName)
		driverName = "postgres"
	} else if ct == "sqlserver" || ct == "mssql" {
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(conn.DbLoginName, conn.DbLoginPassword),
			Host:   fmt.Sprintf("%s:%d", conn.ConnectionUrl, conn.Port),
		}
		q := u.Query()
		q.Set("database", conn.DatabaseName)
		u.RawQuery = q.Encode()
		dsn = u.String()
		driverName = "sqlserver"
	} else if ct == "oracle" {
		dsn = utils.BuildOracleDSN(conn.ConnectionUrl, conn.Port, conn.DatabaseName, conn.DbLoginName, conn.DbLoginPassword)
		driverName = "oracle"
	} else if ct == "sqlite" {
		dsn = conn.DatabaseName
		driverName = "sqlite"
	} else if ct == "clickhouse" {
		dsn = buildClickHouseDSN(conn.ConnectionUrl, conn.Port, conn.DatabaseName, conn.DbLoginName, conn.DbLoginPassword)
		driverName = "clickhouse"
	}
	return
}

func buildClickHouseDSN(host string, port int, databaseName, userName, password string) string {
	scheme := "clickhouse"
	if port == 8123 {
		scheme = "http"
	} else if port == 8443 {
		scheme = "https"
	}

	if parsed, err := url.Parse(host); err == nil && parsed.Scheme != "" {
		scheme = parsed.Scheme
		host = parsed.Host
		if parsed.Path != "" && parsed.Path != "/" && databaseName == "" {
			databaseName = strings.TrimPrefix(parsed.Path, "/")
		}
	}

	u := &url.URL{
		Scheme: scheme,
		User:   url.UserPassword(userName, password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   "/" + databaseName,
	}
	q := u.Query()
	q.Set("dial_timeout", "10s")
	u.RawQuery = q.Encode()
	return u.String()
}

// ---- MySQL ----
func importMySQLSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT TABLE_NAME, TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME", conn.DatabaseName)
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableComment string
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName, Description: tableComment, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importMySQLColumns(db, conn, table, userName)
	}
	return nil
}

func importMySQLColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("SELECT COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION", conn.DatabaseName, table.TbName)
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var colName, dataType, isNullable string
		var maxLen sql.NullInt64
		var defaultVal, colComment sql.NullString
		if err := colRows.Scan(&colName, &dataType, &maxLen, &isNullable, &defaultVal, &colComment); err != nil {
			continue
		}
		isEmpty := 1
		if strings.EqualFold(isNullable, "NO") {
			isEmpty = 0
		}
		colSize := ""
		if maxLen.Valid {
			colSize = strconv.FormatInt(maxLen.Int64, 10)
		}
		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: colName, ColumnType: dataType, ColumnSize: colSize, IsEmpty: isEmpty, DefaultValue: defaultVal.String, Description: colComment.String})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}

// ---- PostgreSQL ----
func importPostgresSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT t.table_name, COALESCE(d.description,'') FROM information_schema.tables t LEFT JOIN pg_stat_user_tables s ON s.relname = t.table_name LEFT JOIN pg_description d ON d.objoid = s.relid AND d.objsubid = 0 WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE' ORDER BY t.table_name")
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableComment string
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName, Description: tableComment, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importPostgresColumns(db, conn, table, userName)
	}
	return nil
}

func importPostgresColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("SELECT c.column_name, c.data_type, c.character_maximum_length::text, c.is_nullable, c.column_default, COALESCE(d.description,'') FROM information_schema.columns c LEFT JOIN pg_stat_user_tables s ON s.relname = c.table_name LEFT JOIN pg_description d ON d.objoid = s.relid AND d.objsubid = c.ordinal_position WHERE c.table_schema = 'public' AND c.table_name = $1 ORDER BY c.ordinal_position", table.TbName)
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var colName, dataType, isNullable string
		var maxLen, defaultVal, colComment sql.NullString
		if err := colRows.Scan(&colName, &dataType, &maxLen, &isNullable, &defaultVal, &colComment); err != nil {
			continue
		}
		isEmpty := 1
		if strings.EqualFold(isNullable, "NO") {
			isEmpty = 0
		}
		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: colName, ColumnType: dataType, ColumnSize: maxLen.String, IsEmpty: isEmpty, DefaultValue: defaultVal.String, Description: colComment.String})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}

// ---- SQLServer ----
func importSqlserverSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT t.name, CAST(p.value AS VARCHAR(MAX)) FROM sys.tables t LEFT JOIN sys.extended_properties p ON p.major_id = t.object_id AND p.minor_id = 0 AND p.name = 'MS_Description' ORDER BY t.name")
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName sql.NullString
		var tableComment sql.NullString
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName.String, Description: tableComment.String, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importSqlserverColumns(db, conn, table, userName)
	}
	return nil
}

func importSqlserverColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("SELECT c.name, t.name as type_name, c.max_length, c.is_nullable, object_definition(c.default_object_id), CAST(p.value AS VARCHAR(MAX)) FROM sys.columns c JOIN sys.types t ON c.user_type_id = t.user_type_id LEFT JOIN sys.extended_properties p ON p.major_id = c.object_id AND p.minor_id = c.column_id AND p.name = 'MS_Description' WHERE c.object_id = OBJECT_ID(@p1) ORDER BY c.column_id", table.TbName)
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var colName, dataType sql.NullString
		var isNullable sql.NullBool
		var maxLen sql.NullInt64
		var defaultVal, colComment sql.NullString
		if err := colRows.Scan(&colName, &dataType, &maxLen, &isNullable, &defaultVal, &colComment); err != nil {
			continue
		}
		isEmpty := 1
		if isNullable.Valid && !isNullable.Bool {
			isEmpty = 0
		}
		colSize := ""
		if maxLen.Valid {
			colSize = strconv.FormatInt(maxLen.Int64, 10)
		}
		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: colName.String, ColumnType: dataType.String, ColumnSize: colSize, IsEmpty: isEmpty, DefaultValue: defaultVal.String, Description: colComment.String})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}

// ---- Oracle ----
func importOracleSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT TABLE_NAME, COMMENTS FROM USER_TAB_COMMENTS WHERE TABLE_TYPE = 'TABLE' ORDER BY TABLE_NAME")
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableComment sql.NullString
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName.String, Description: tableComment.String, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importOracleColumns(db, conn, table, userName)
	}
	return nil
}

func importOracleColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("SELECT c.COLUMN_NAME, c.DATA_TYPE, c.DATA_LENGTH, c.NULLABLE, c.DATA_DEFAULT, cc.COMMENTS FROM USER_TAB_COLUMNS c LEFT JOIN USER_COL_COMMENTS cc ON c.TABLE_NAME = cc.TABLE_NAME AND c.COLUMN_NAME = cc.COLUMN_NAME WHERE c.TABLE_NAME = :1 ORDER BY c.COLUMN_ID", strings.ToUpper(table.TbName))
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var colName, dataType, isNullable sql.NullString
		var maxLen sql.NullInt64
		var defaultVal, colComment sql.NullString
		if err := colRows.Scan(&colName, &dataType, &maxLen, &isNullable, &defaultVal, &colComment); err != nil {
			continue
		}
		isEmpty := 1
		if isNullable.String == "N" {
			isEmpty = 0
		}
		colSize := ""
		if maxLen.Valid {
			colSize = strconv.FormatInt(maxLen.Int64, 10)
		}
		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: colName.String, ColumnType: dataType.String, ColumnSize: colSize, IsEmpty: isEmpty, DefaultValue: defaultVal.String, Description: colComment.String})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}

// ---- SQLite ----
func importSqliteSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT name, '' as comment FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableComment sql.NullString
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName.String, Description: tableComment.String, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importSqliteColumns(db, conn, table, userName)
	}
	return nil
}

func importSqliteColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("PRAGMA table_info(" + table.TbName + ")")
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var cid int
		var name, typ string
		var notnull sql.NullInt64
		var dflt sql.NullString
		var ispk int
		if err := colRows.Scan(&cid, &name, &typ, &notnull, &dflt, &ispk); err != nil {
			continue
		}
		isEmpty := 1
		if notnull.Valid && notnull.Int64 == 1 {
			isEmpty = 0
		}

		colSize := "-"
		colType := typ
		if idx := strings.Index(colType, "("); idx != -1 {
			if endIdx := strings.Index(colType, ")"); endIdx != -1 {
				colSize = colType[idx+1 : endIdx]
				colType = colType[:idx]
			}
		}

		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: name, ColumnType: colType, ColumnSize: colSize, IsEmpty: isEmpty, DefaultValue: dflt.String, Description: ""})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}

// ---- ClickHouse ----
func importClickHouseSchema(db *sql.DB, conn system.TbConnection, userName string) error {
	rows, err := db.Query("SELECT name, comment FROM system.tables WHERE database = ? AND is_temporary = 0 ORDER BY name", conn.DatabaseName)
	if err != nil {
		return fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableComment sql.NullString
		if err := rows.Scan(&tableName, &tableComment); err != nil {
			continue
		}
		table := system.TbTable{DatabaseName: conn.DatabaseName, TbName: tableName.String, Description: tableComment.String, ConnectionId: int(conn.ID), UserName: userName}
		if err := global.GVA_DB.Create(&table).Error; err != nil {
			continue
		}
		importClickHouseColumns(db, conn, table, userName)
	}
	return nil
}

func importClickHouseColumns(db *sql.DB, conn system.TbConnection, table system.TbTable, userName string) {
	colRows, err := db.Query("SELECT name, type, default_expression, comment FROM system.columns WHERE database = ? AND table = ? ORDER BY position", conn.DatabaseName, table.TbName)
	if err != nil {
		return
	}
	defer colRows.Close()

	var cols []system.TbTableColumn
	for colRows.Next() {
		var colName, dataType, defaultVal, colComment sql.NullString
		if err := colRows.Scan(&colName, &dataType, &defaultVal, &colComment); err != nil {
			continue
		}
		isEmpty := 0
		if strings.HasPrefix(dataType.String, "Nullable(") {
			isEmpty = 1
		}
		cols = append(cols, system.TbTableColumn{ConnectionId: int(conn.ID), TableId: strconv.Itoa(int(table.ID)), ColumnName: colName.String, ColumnType: dataType.String, ColumnSize: "", IsEmpty: isEmpty, DefaultValue: defaultVal.String, Description: colComment.String})
	}
	if len(cols) > 0 {
		global.GVA_DB.CreateInBatches(cols, 100)
	}
}
