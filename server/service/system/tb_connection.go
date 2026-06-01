package system

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
)

type TbConnectionService struct{}

const (
	defaultRemoteSQLQueryLimit = 200
	maxRemoteSQLQueryLimit     = 200
)

type RemoteDatabaseVO struct {
	ConnectionID   uint   `json:"connectionId"`
	ConnectionName string `json:"connectionName"`
	ConnectionType string `json:"connectionType"`
	DatabaseName   string `json:"databaseName"`
	EnvName        string `json:"envName"`
}

// GetRemoteDatabases connects through the configured data sources and returns
// the database names visible in the requested project/environment in real time.
func (s *TbConnectionService) GetRemoteDatabases(connectionGroup, envName string, connID uint) ([]RemoteDatabaseVO, error) {
	var conns []system.TbConnection
	dbQuery := global.GVA_DB.Model(&system.TbConnection{})
	if connID != 0 {
		dbQuery = dbQuery.Where("id = ?", connID)
	}
	if connectionGroup != "" {
		dbQuery = dbQuery.Where("connection_group = ?", connectionGroup)
	}
	if envName != "" {
		dbQuery = dbQuery.Where("env_name = ?", envName)
	}
	if err := dbQuery.Order("id").Find(&conns).Error; err != nil {
		return nil, err
	}

	var databases []RemoteDatabaseVO
	for _, conn := range conns {
		names, err := s.getRemoteDatabaseNames(conn)
		if err != nil {
			return nil, fmt.Errorf("%s 获取数据库列表失败: %w", conn.ConnectionName, err)
		}
		for _, name := range names {
			databases = append(databases, RemoteDatabaseVO{
				ConnectionID:   conn.ID,
				ConnectionName: conn.ConnectionName,
				ConnectionType: conn.ConnectionType,
				DatabaseName:   name,
				EnvName:        conn.EnvName,
			})
		}
	}
	return databases, nil
}

func (s *TbConnectionService) getRemoteDatabaseNames(conn system.TbConnection) ([]string, error) {
	query, args, ok := buildRemoteDatabaseListQuery(conn.ConnectionType, conn.DatabaseName)
	if !ok {
		return []string{conn.DatabaseName}, nil
	}

	dsn, driverName := buildDSN(conn)
	if driverName == "" {
		return nil, fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}
	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		return nil, formatRemoteDBConnectionError(conn, err)
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询数据库列表失败: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(names) == 0 && conn.DatabaseName != "" {
		names = append(names, conn.DatabaseName)
	}
	return names, nil
}

// GetRemoteTables connects to the given database and returns all table names in real-time.
func (s *TbConnectionService) GetRemoteTables(connID uint, databaseName string) ([]string, error) {
	var conn system.TbConnection
	if err := global.GVA_DB.Where("id = ?", connID).First(&conn).Error; err != nil {
		return nil, fmt.Errorf("连接配置不存在: %w", err)
	}
	if strings.TrimSpace(databaseName) == "" {
		databaseName = conn.DatabaseName
	}

	// When the requested database differs from the configured one, switch the DSN.
	// PostgreSQL & SQLServer require separate connections per database.
	// MySQL's information_schema is global, but switching the DSN ensures correct
	// context and avoids potential permission issues.
	// Oracle does NOT have separate databases; databaseName is a schema/owner name,
	// so the DSN must always use the original configured service name.
	connForDSN := conn
	ct := strings.ToLower(conn.ConnectionType)
	if databaseName != conn.DatabaseName && ct != "oracle" {
		connForDSN.DatabaseName = databaseName
	}

	dsn, driverName := buildDSN(connForDSN)
	if driverName == "" {
		return nil, fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}

	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		return nil, formatRemoteDBConnectionError(connForDSN, err)
	}
	defer db.Close()

	query, args, ok := buildRemoteTableListQuery(conn.ConnectionType, databaseName)
	if !ok {
		return nil, fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func buildRemoteDatabaseListQuery(connectionType, configuredDatabase string) (string, []interface{}, bool) {
	ct := strings.ToLower(connectionType)
	switch {
	case ct == "mysql":
		return "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME NOT IN (?, ?, ?, ?) ORDER BY SCHEMA_NAME",
			[]interface{}{"information_schema", "mysql", "performance_schema", "sys"},
			true
	case ct == "postgresql" || ct == "pgsql":
		// Query all non-template databases from the server, just like Navicat does.
		// The connection is made to the configured database, but pg_database is a global catalog.
		return "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname",
			nil,
			true
	case ct == "sqlserver" || ct == "mssql":
		// Query all user databases from the server.
		return "SELECT name FROM sys.databases WHERE name NOT IN ('master', 'tempdb', 'model', 'msdb') ORDER BY name",
			nil,
			true
	case ct == "oracle":
		// Oracle doesn't have separate databases; use schemas/users as the namespace.
		return "SELECT username FROM all_users WHERE username NOT IN ('SYS','SYSTEM','OUTLN','DIP','ORACLE_OCM','DBSNMP','APPQOSSYS','DBSFWUSER','GGSYS','ANONYMOUS','CTXSYS','DVSYS','DVF','GSMADMIN_INTERNAL','MDSYS','OLAPSYS','XDB','WMSYS','OJVMSYS','LBACSYS','APEX_PUBLIC_USER','FLOWS_FILES') ORDER BY username",
			nil,
			true
	case ct == "clickhouse":
		return "SELECT name FROM system.databases WHERE name NOT IN ('INFORMATION_SCHEMA', 'information_schema', 'system') ORDER BY name",
			nil,
			true
	case ct == "sqlite":
		if strings.TrimSpace(configuredDatabase) == "" {
			return "", nil, false
		}
		return "", []interface{}{configuredDatabase}, false
	default:
		return "", nil, false
	}
}

func buildRemoteTableListQuery(connectionType, databaseName string) (string, []interface{}, bool) {
	ct := strings.ToLower(connectionType)
	switch {
	case ct == "mysql":
		return "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME", []interface{}{databaseName}, true
	case ct == "postgresql" || ct == "pgsql":
		// The DSN is already switched to the target database in GetRemoteTables,
		// so we just need to list tables in the 'public' schema.
		return "SELECT t.table_name FROM information_schema.tables t WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE' ORDER BY t.table_name",
			nil, true
	case ct == "sqlserver" || ct == "mssql":
		return "SELECT t.name FROM sys.tables t ORDER BY t.name", nil, true
	case ct == "oracle":
		// Use ALL_TABLES with OWNER filter to support cross-schema table listing.
		// databaseName here is the Oracle schema/owner name (from all_users query).
		owner := strings.ToUpper(strings.TrimSpace(databaseName))
		if owner == "" {
			// Fall back to current user's tables
			return "SELECT TABLE_NAME FROM USER_TABLES ORDER BY TABLE_NAME", nil, true
		}
		return "SELECT TABLE_NAME FROM ALL_TABLES WHERE OWNER = :1 ORDER BY TABLE_NAME", []interface{}{owner}, true
	case ct == "sqlite":
		return "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name", nil, true
	case ct == "clickhouse":
		return "SELECT name FROM system.tables WHERE database = ? AND is_temporary = 0 ORDER BY name", []interface{}{databaseName}, true
	default:
		return "", nil, false
	}
}

func (s *TbConnectionService) CreateTbConnection(conn system.TbConnection) (err error) {
	err = global.GVA_DB.Create(&conn).Error
	return err
}

func (s *TbConnectionService) DeleteTbConnection(conn system.TbConnection) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&conn).Error
	return err
}

func (s *TbConnectionService) DeleteTbConnectionByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbConnection{}, "id in ?", ids).Error
	return err
}

func (s *TbConnectionService) UpdateTbConnection(conn *system.TbConnection) (err error) {
	err = global.GVA_DB.Updates(conn).Error
	return err
}

func (s *TbConnectionService) GetTbConnection(id uint) (conn system.TbConnection, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&conn).Error
	return
}

func (s *TbConnectionService) GetTbConnectionInfoList(info systemReq.ConnectionSearch) (list []system.TbConnection, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbConnection{})
	if info.ConnectionName != "" {
		db = db.Where("connection_name LIKE ?", "%"+info.ConnectionName+"%")
	}
	if info.ConnectionType != "" {
		db = db.Where("connection_type = ?", info.ConnectionType)
	}
	if info.ConnectionGroup != "" {
		db = db.Where("connection_group = ?", info.ConnectionGroup)
	}
	if info.EnvName != "" {
		db = db.Where("env_name = ?", info.EnvName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

// ColumnPreview represents a single column's preview data
type ColumnPreview struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
	IsNull      bool   `json:"isNull"`
	PrimaryKey  bool   `json:"primaryKey"`
}

// TableRecordPreview represents one record from a table with column metadata
type TableRecordPreview struct {
	Columns []ColumnPreview `json:"columns"`
	Total   int64           `json:"total"`
	Offset  int             `json:"offset"`
}

type RemoteTableDDL struct {
	DatabaseName string `json:"databaseName"`
	TableName    string `json:"tableName"`
	SQL          string `json:"sql"`
}

type postgresDDLColumn struct {
	Name     string
	DataType string
	UdtName  string
	CharLen  sql.NullInt64
	NumPrec  sql.NullInt64
	NumScale sql.NullInt64
	Nullable string
	Default  sql.NullString
	Comment  sql.NullString
	Ordinal  int
}

type RemoteSQLQueryResult struct {
	Columns      []string        `json:"columns"`
	Rows         [][]interface{} `json:"rows"`
	Limit        int             `json:"limit"`
	Returned     int             `json:"returned"`
	Truncated    bool            `json:"truncated"`
	ElapsedMs    int64           `json:"elapsedMs"`
	DatabaseName string          `json:"databaseName"`
}

// QueryRemoteSQL runs a read-only SQL query against a configured data source.
// It intentionally returns only a small preview window so the UI behaves like a
// query panel without becoming an unrestricted execution console.
func (s *TbConnectionService) QueryRemoteSQL(connID uint, databaseName, rawSQL string, limit int) (*RemoteSQLQueryResult, error) {
	query, err := normalizeRemoteSQLQuery(rawSQL)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultRemoteSQLQueryLimit
	}
	if limit > maxRemoteSQLQueryLimit {
		limit = maxRemoteSQLQueryLimit
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if ct == "oracle" && containsSQLKeyword(maskSQLLiteralsAndComments(query), "limit") {
		return nil, fmt.Errorf("Oracle 不支持 LIMIT 语法；请去掉 LIMIT，系统会自动只返回前 %d 条，或改用 FETCH FIRST %d ROWS ONLY", limit, limit)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	started := time.Now()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取字段失败: %w", err)
	}

	result := &RemoteSQLQueryResult{
		Columns:      columns,
		Rows:         make([][]interface{}, 0, limit),
		Limit:        limit,
		DatabaseName: databaseName,
	}

	for rows.Next() {
		if len(result.Rows) >= limit {
			result.Truncated = true
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取记录失败: %w", err)
		}

		row := make([]interface{}, len(columns))
		for i, value := range values {
			row[i] = normalizeSQLQueryCellValue(value)
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取查询结果失败: %w", err)
	}

	result.Returned = len(result.Rows)
	result.ElapsedMs = time.Since(started).Milliseconds()
	return result, nil
}

// GetRemoteTableComments returns table comments keyed by table name for a database/schema.
func (s *TbConnectionService) GetRemoteTableComments(connID uint, databaseName string) (map[string]string, error) {
	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query, args, ok := buildTableCommentsQuery(ct, databaseName)
	if !ok {
		return map[string]string{}, nil
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询表注释失败: %w", err)
	}
	defer rows.Close()

	comments := make(map[string]string)
	for rows.Next() {
		var tableName string
		var comment sql.NullString
		if err := rows.Scan(&tableName, &comment); err != nil {
			continue
		}
		if comment.Valid && strings.TrimSpace(comment.String) != "" {
			comments[tableName] = comment.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

// GetRemoteTableDDL returns a readable CREATE TABLE statement for the remote table.
func (s *TbConnectionService) GetRemoteTableDDL(connID uint, databaseName, tableName string) (*RemoteTableDDL, error) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, fmt.Errorf("缺少表名")
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	sqlText, err := queryRemoteTableDDL(db, ct, databaseName, tableName)
	if err != nil {
		return nil, err
	}
	return &RemoteTableDDL{
		DatabaseName: databaseName,
		TableName:    tableName,
		SQL:          sqlText,
	}, nil
}

// PreviewTableRecord connects to the remote database and returns a single record
// at the given offset along with column descriptions and total row count.
func (s *TbConnectionService) PreviewTableRecord(connID uint, databaseName, tableName string, offset int, filterColumn, filterValue string) (*TableRecordPreview, error) {
	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Get column definitions (name + description)
	colDefs := utils.GetColumnDefinitions(db, databaseName, tableName, ct)
	primaryKey := getTablePrimaryKey(db, databaseName, tableName, ct)

	// Build qualified table name
	qualifiedTable := buildQualifiedTableName(ct, databaseName, tableName)
	whereSQL, whereArgs, err := buildSingleColumnFilter(ct, colDefs, filterColumn, filterValue)
	if err != nil {
		return nil, err
	}

	// Get total row count
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", qualifiedTable, whereSQL)
	if err := db.QueryRow(countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询记录数失败: %w", err)
	}

	if total == 0 {
		cols := make([]ColumnPreview, len(colDefs))
		for i, cd := range colDefs {
			cols[i] = ColumnPreview{
				Name:        cd.Name,
				Value:       "",
				Description: cd.Description,
				IsNull:      true,
				PrimaryKey:  strings.EqualFold(cd.Name, primaryKey),
			}
		}
		return &TableRecordPreview{Columns: cols, Total: 0, Offset: 0}, nil
	}

	// Clamp offset
	if offset < 0 {
		offset = 0
	}
	if int64(offset) >= total {
		offset = int(total) - 1
	}

	// Build paginated query for a single record
	var dataQuery string
	switch {
	case ct == "oracle":
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s OFFSET %d ROWS FETCH NEXT 1 ROWS ONLY", qualifiedTable, whereSQL, offset)
	case ct == "sqlserver" || ct == "mssql":
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT 1 ROWS ONLY", qualifiedTable, whereSQL, offset)
	default:
		// mysql, postgresql, sqlite
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s LIMIT 1 OFFSET %d", qualifiedTable, whereSQL, offset)
	}

	rows, err := db.Query(dataQuery, whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询记录失败: %w", err)
	}
	defer rows.Close()

	// Build name-to-description map from column definitions
	descMap := make(map[string]string)
	typeMap := make(map[string]string)
	for _, cd := range colDefs {
		descMap[strings.ToUpper(cd.Name)] = cd.Description
		typeMap[strings.ToUpper(cd.Name)] = cd.ColumnType
	}

	result := &TableRecordPreview{Total: total, Offset: offset}

	if rows.Next() {
		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取记录失败: %w", err)
		}

		for i, col := range columns {
			val := values[i]
			var strVal string
			isNull := val == nil
			if val != nil {
				switch v := val.(type) {
				case time.Time:
					strVal = formatPreviewTime(v, typeMap[strings.ToUpper(col)])
				case []byte:
					strVal = string(v)
				case string:
					strVal = v
				default:
					strVal = fmt.Sprintf("%v", v)
				}
			}
			desc := descMap[strings.ToUpper(col)]
			result.Columns = append(result.Columns, ColumnPreview{
				Name:        col,
				Value:       strVal,
				Description: desc,
				IsNull:      isNull,
				PrimaryKey:  strings.EqualFold(col, primaryKey),
			})
		}
	} else {
		// No data at this offset, return column defs with empty values
		for _, cd := range colDefs {
			result.Columns = append(result.Columns, ColumnPreview{
				Name:        cd.Name,
				Value:       "",
				Description: cd.Description,
				IsNull:      true,
				PrimaryKey:  strings.EqualFold(cd.Name, primaryKey),
			})
		}
	}

	return result, nil
}

// UpdateTableRecord updates field values for one previewed row and returns the refreshed preview.
func (s *TbConnectionService) UpdateTableRecord(connID uint, databaseName, tableName string, offset int, changes map[string]string, filterColumn, filterValue string) (*TableRecordPreview, error) {
	if len(changes) == 0 {
		return nil, fmt.Errorf("没有需要修改的字段")
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if ct == "clickhouse" {
		return nil, fmt.Errorf("暂不支持修改 ClickHouse 表数据")
	}

	colDefs := utils.GetColumnDefinitions(db, databaseName, tableName, ct)
	if len(colDefs) == 0 {
		return nil, fmt.Errorf("未读取到表字段信息")
	}

	allowedColumns := make(map[string]utils.ClientColumnVO, len(colDefs))
	for _, col := range colDefs {
		allowedColumns[strings.ToUpper(col.Name)] = col
	}

	normalizedChanges := make(map[string]interface{}, len(changes))
	for name, value := range changes {
		colDef, ok := allowedColumns[strings.ToUpper(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("字段 %s 不存在", name)
		}
		normalizedValue, err := normalizeTableUpdateValue(ct, colDef, value)
		if err != nil {
			return nil, err
		}
		normalizedChanges[strings.ToUpper(colDef.Name)] = normalizedValue
	}

	qualifiedTable := buildQualifiedTableName(ct, databaseName, tableName)
	whereSQL, whereArgs, err := buildSingleColumnFilter(ct, colDefs, filterColumn, filterValue)
	if err != nil {
		return nil, err
	}
	rowColumns, rowValues, err := queryTableRecordValues(db, ct, qualifiedTable, offset, whereSQL, whereArgs)
	if err != nil {
		return nil, err
	}
	if len(rowColumns) == 0 {
		return nil, fmt.Errorf("未找到要修改的记录")
	}

	rowIndex := make(map[string]int, len(rowColumns))
	for i, col := range rowColumns {
		rowIndex[strings.ToUpper(col)] = i
	}

	primaryKey := getTablePrimaryKey(db, databaseName, tableName, ct)
	whereColumns := rowColumns
	whereValues := rowValues
	if primaryKey != "" {
		if idx, ok := rowIndex[strings.ToUpper(primaryKey)]; ok && rowValues[idx] != nil {
			whereColumns = []string{rowColumns[idx]}
			whereValues = []interface{}{rowValues[idx]}
		}
	}

	builder := sqlArgBuilder{ct: ct}
	setParts := make([]string, 0, len(normalizedChanges))
	for _, col := range rowColumns {
		newValue, ok := normalizedChanges[strings.ToUpper(col)]
		if !ok {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = %s", quoteColumnIdentifier(ct, col), builder.Add(newValue)))
	}
	if len(setParts) == 0 {
		return nil, fmt.Errorf("没有可修改的字段")
	}

	whereParts := make([]string, 0, len(whereColumns))
	for i, col := range whereColumns {
		if whereValues[i] == nil {
			whereParts = append(whereParts, fmt.Sprintf("%s IS NULL", quoteColumnIdentifier(ct, col)))
			continue
		}
		whereParts = append(whereParts, fmt.Sprintf("%s = %s", quoteColumnIdentifier(ct, col), builder.Add(whereValues[i])))
	}
	if len(whereParts) == 0 {
		return nil, fmt.Errorf("无法定位要修改的记录")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", qualifiedTable, strings.Join(setParts, ", "), strings.Join(whereParts, " AND "))
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}

	result, err := tx.Exec(query, builder.args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("执行更新失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("无法确认更新结果: %w", err)
	}
	if rowsAffected != 1 {
		_ = tx.Rollback()
		return nil, fmt.Errorf("本次更新影响了 %d 行，已取消；请确认记录唯一性后重试", rowsAffected)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交更新失败: %w", err)
	}

	return s.PreviewTableRecord(connID, databaseName, tableName, offset, filterColumn, filterValue)
}

func (s *TbConnectionService) openRemoteDB(connID uint, databaseName string) (*sql.DB, string, string, error) {
	var conn system.TbConnection
	if err := global.GVA_DB.Where("id = ?", connID).First(&conn).Error; err != nil {
		return nil, "", "", fmt.Errorf("连接配置不存在: %w", err)
	}
	if strings.TrimSpace(databaseName) == "" {
		databaseName = conn.DatabaseName
	}

	connForDSN := conn
	ct := strings.ToLower(conn.ConnectionType)
	if databaseName != conn.DatabaseName && ct != "oracle" {
		connForDSN.DatabaseName = databaseName
	}

	dsn, driverName := buildDSN(connForDSN)
	if driverName == "" {
		return nil, "", "", fmt.Errorf("暂不支持的数据源类型: %s", conn.ConnectionType)
	}

	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		return nil, "", "", formatRemoteDBConnectionError(connForDSN, err)
	}
	return db, ct, databaseName, nil
}

func queryTableRecordValues(db *sql.DB, ct, qualifiedTable string, offset int, whereSQL string, whereArgs []interface{}) ([]string, []interface{}, error) {
	if offset < 0 {
		offset = 0
	}

	var dataQuery string
	switch {
	case ct == "oracle":
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s OFFSET %d ROWS FETCH NEXT 1 ROWS ONLY", qualifiedTable, whereSQL, offset)
	case ct == "sqlserver" || ct == "mssql":
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT 1 ROWS ONLY", qualifiedTable, whereSQL, offset)
	default:
		dataQuery = fmt.Sprintf("SELECT * FROM %s%s LIMIT 1 OFFSET %d", qualifiedTable, whereSQL, offset)
	}

	rows, err := db.Query(dataQuery, whereArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("查询记录失败: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, fmt.Errorf("未找到要修改的记录")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("读取字段失败: %w", err)
	}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, nil, fmt.Errorf("读取记录失败: %w", err)
	}
	return columns, values, nil
}

func formatPreviewTime(value time.Time, columnType string) string {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	if columnType == "DATE" || strings.Contains(columnType, "TIMESTAMP") || strings.Contains(columnType, "TIME") {
		if value.Nanosecond() == 0 {
			return value.Format("2006-01-02 15:04:05")
		}
		return value.Format("2006-01-02 15:04:05.999999999")
	}
	return value.Format("2006-01-02 15:04:05")
}

func normalizeTableUpdateValue(ct string, colDef utils.ClientColumnVO, value string) (interface{}, error) {
	columnType := strings.ToUpper(strings.TrimSpace(colDef.ColumnType))
	if ct != "oracle" || !isOracleTemporalColumn(columnType) {
		return value, nil
	}

	parsed, err := parseTableUpdateTime(value)
	if err != nil {
		return nil, fmt.Errorf("字段 %s 的时间格式不正确，请使用 YYYY-MM-DD HH:mm:ss: %w", colDef.Name, err)
	}
	return parsed, nil
}

func isOracleTemporalColumn(columnType string) bool {
	return columnType == "DATE" || strings.HasPrefix(columnType, "TIMESTAMP")
}

func parseTableUpdateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("时间不能为空")
	}

	layoutsWithZone := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 -07:00",
		"2006-01-02 15:04:05 -0700 -07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05 -0700",
	}
	for _, layout := range layoutsWithZone {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}

	layoutsInLocation := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layoutsInLocation {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析 %q", value)
}

func buildSingleColumnFilter(ct string, colDefs []utils.ClientColumnVO, filterColumn, filterValue string) (string, []interface{}, error) {
	filterColumn = strings.TrimSpace(filterColumn)
	if filterColumn == "" {
		return "", nil, nil
	}

	var actualColumn string
	for _, col := range colDefs {
		if strings.EqualFold(col.Name, filterColumn) {
			actualColumn = col.Name
			break
		}
	}
	if actualColumn == "" {
		return "", nil, fmt.Errorf("过滤字段 %s 不存在", filterColumn)
	}

	builder := sqlArgBuilder{ct: ct}
	return fmt.Sprintf(" WHERE %s = %s", quoteColumnIdentifier(ct, actualColumn), builder.Add(filterValue)), builder.args, nil
}

func buildTableCommentsQuery(ct, databaseName string) (string, []interface{}, bool) {
	switch {
	case ct == "mysql":
		return "SELECT TABLE_NAME, TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME",
			[]interface{}{databaseName}, true
	case ct == "postgresql" || ct == "pgsql":
		return `SELECT c.relname, obj_description(c.oid) FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r' AND n.nspname = 'public'
ORDER BY c.relname`, nil, true
	case ct == "sqlserver" || ct == "mssql":
		return `SELECT t.name, CAST(ep.value AS NVARCHAR(MAX)) FROM sys.tables t
LEFT JOIN sys.extended_properties ep ON ep.major_id = t.object_id AND ep.minor_id = 0 AND ep.name = 'MS_Description'
ORDER BY t.name`, nil, true
	case ct == "oracle":
		owner := strings.ToUpper(strings.TrimSpace(databaseName))
		if owner == "" {
			return "SELECT TABLE_NAME, COMMENTS FROM USER_TAB_COMMENTS ORDER BY TABLE_NAME", nil, true
		}
		return "SELECT TABLE_NAME, COMMENTS FROM ALL_TAB_COMMENTS WHERE OWNER = :1 ORDER BY TABLE_NAME",
			[]interface{}{owner}, true
	case ct == "clickhouse":
		return "SELECT name, comment FROM system.tables WHERE database = ? AND is_temporary = 0 ORDER BY name",
			[]interface{}{databaseName}, true
	default:
		return "", nil, false
	}
}

func queryRemoteTableDDL(db *sql.DB, ct, databaseName, tableName string) (string, error) {
	switch {
	case ct == "mysql":
		var table, createSQL string
		query := "SHOW CREATE TABLE " + buildQuotedTableName(ct, databaseName, tableName)
		if err := db.QueryRow(query).Scan(&table, &createSQL); err != nil {
			return "", fmt.Errorf("读取建表 SQL 失败: %w", err)
		}
		return createSQL, nil
	case ct == "clickhouse":
		var createSQL string
		query := "SHOW CREATE TABLE " + buildQuotedTableName(ct, databaseName, tableName)
		if err := db.QueryRow(query).Scan(&createSQL); err != nil {
			return "", fmt.Errorf("读取建表 SQL 失败: %w", err)
		}
		return createSQL, nil
	case ct == "sqlite":
		var createSQL sql.NullString
		if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", tableName).Scan(&createSQL); err != nil {
			return "", fmt.Errorf("读取建表 SQL 失败: %w", err)
		}
		if !createSQL.Valid || strings.TrimSpace(createSQL.String) == "" {
			return "", fmt.Errorf("未找到表 %s 的建表 SQL", tableName)
		}
		return createSQL.String, nil
	case ct == "postgresql" || ct == "pgsql":
		return buildPostgresTableDDL(db, tableName)
	case ct == "sqlserver" || ct == "mssql":
		return buildSQLServerTableDDL(db, databaseName, tableName)
	case ct == "oracle":
		return buildOracleTableDDL(db, databaseName, tableName)
	default:
		return "", fmt.Errorf("暂不支持查看 %s 的建表 SQL", ct)
	}
}

func buildPostgresTableDDL(db *sql.DB, tableName string) (string, error) {
	rows, err := db.Query(`SELECT c.column_name, c.data_type, c.udt_name, c.character_maximum_length,
       c.numeric_precision, c.numeric_scale, c.is_nullable, c.column_default,
       col_description(format('%I.%I', c.table_schema, c.table_name)::regclass::oid, c.ordinal_position),
       c.ordinal_position
FROM information_schema.columns c
WHERE c.table_schema = 'public' AND c.table_name = $1
ORDER BY c.ordinal_position`, tableName)
	if err != nil {
		return "", fmt.Errorf("读取字段信息失败: %w", err)
	}
	defer rows.Close()

	var columns []postgresDDLColumn
	for rows.Next() {
		var col postgresDDLColumn
		if err := rows.Scan(&col.Name, &col.DataType, &col.UdtName, &col.CharLen, &col.NumPrec, &col.NumScale, &col.Nullable, &col.Default, &col.Comment, &col.Ordinal); err != nil {
			return "", fmt.Errorf("读取字段信息失败: %w", err)
		}
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("未读取到表字段信息")
	}

	pks, err := queryPostgresPrimaryKeys(db, tableName)
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, len(columns)+1)
	for _, col := range columns {
		parts := []string{quoteDDLIdentifier("postgresql", col.Name), postgresColumnType(col)}
		if col.Default.Valid && strings.TrimSpace(col.Default.String) != "" {
			parts = append(parts, "DEFAULT "+strings.TrimSpace(col.Default.String))
		}
		if strings.EqualFold(col.Nullable, "NO") {
			parts = append(parts, "NOT NULL")
		}
		lines = append(lines, "  "+strings.Join(parts, " "))
	}
	if len(pks) > 0 {
		lines = append(lines, "  PRIMARY KEY ("+joinQuotedDDLIdentifiers("postgresql", pks)+")")
	}

	var builder strings.Builder
	builder.WriteString("CREATE TABLE ")
	builder.WriteString(quoteDDLIdentifier("postgresql", "public"))
	builder.WriteString(".")
	builder.WriteString(quoteDDLIdentifier("postgresql", tableName))
	builder.WriteString(" (\n")
	builder.WriteString(strings.Join(lines, ",\n"))
	builder.WriteString("\n);")
	for _, col := range columns {
		if col.Comment.Valid && strings.TrimSpace(col.Comment.String) != "" {
			builder.WriteString("\nCOMMENT ON COLUMN ")
			builder.WriteString(quoteDDLIdentifier("postgresql", "public"))
			builder.WriteString(".")
			builder.WriteString(quoteDDLIdentifier("postgresql", tableName))
			builder.WriteString(".")
			builder.WriteString(quoteDDLIdentifier("postgresql", col.Name))
			builder.WriteString(" IS ")
			builder.WriteString(sqlStringLiteral(col.Comment.String))
			builder.WriteString(";")
		}
	}
	return builder.String(), nil
}

func queryPostgresPrimaryKeys(db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.Query(`SELECT a.attname
FROM pg_index i
JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
JOIN pg_class c ON c.oid = i.indrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname = $1 AND i.indisprimary
ORDER BY array_position(i.indkey, a.attnum)`, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取主键信息失败: %w", err)
	}
	defer rows.Close()
	var pks []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		pks = append(pks, name)
	}
	return pks, rows.Err()
}

func buildSQLServerTableDDL(db *sql.DB, databaseName, tableName string) (string, error) {
	type column struct {
		Name      string
		TypeName  string
		MaxLength int
		Precision int
		Scale     int
		Nullable  bool
		Default   sql.NullString
	}
	objectName := tableName
	if databaseName != "" {
		objectName = fmt.Sprintf("[%s].[dbo].[%s]", databaseName, tableName)
	}
	rows, err := db.Query(`SELECT c.name, typ.name, c.max_length, c.precision, c.scale, c.is_nullable, dc.definition
FROM sys.columns c
JOIN sys.types typ ON c.user_type_id = typ.user_type_id
LEFT JOIN sys.default_constraints dc ON c.default_object_id = dc.object_id
WHERE c.object_id = OBJECT_ID(@p1)
ORDER BY c.column_id`, objectName)
	if err != nil {
		return "", fmt.Errorf("读取字段信息失败: %w", err)
	}
	defer rows.Close()

	var columns []column
	for rows.Next() {
		var col column
		if err := rows.Scan(&col.Name, &col.TypeName, &col.MaxLength, &col.Precision, &col.Scale, &col.Nullable, &col.Default); err != nil {
			return "", fmt.Errorf("读取字段信息失败: %w", err)
		}
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("未读取到表字段信息")
	}

	pks, err := querySQLServerPrimaryKeys(db, objectName)
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, len(columns)+1)
	for _, col := range columns {
		parts := []string{quoteDDLIdentifier("sqlserver", col.Name), sqlServerColumnType(col.TypeName, col.MaxLength, col.Precision, col.Scale)}
		if col.Default.Valid && strings.TrimSpace(col.Default.String) != "" {
			parts = append(parts, "DEFAULT "+strings.TrimSpace(col.Default.String))
		}
		if col.Nullable {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, "NOT NULL")
		}
		lines = append(lines, "  "+strings.Join(parts, " "))
	}
	if len(pks) > 0 {
		lines = append(lines, "  PRIMARY KEY ("+joinQuotedDDLIdentifiers("sqlserver", pks)+")")
	}
	return "CREATE TABLE " + buildQuotedTableName("sqlserver", databaseName, tableName) + " (\n" + strings.Join(lines, ",\n") + "\n);", nil
}

func querySQLServerPrimaryKeys(db *sql.DB, objectName string) ([]string, error) {
	rows, err := db.Query(`SELECT c.name
FROM sys.indexes i
JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
WHERE i.is_primary_key = 1 AND i.object_id = OBJECT_ID(@p1)
ORDER BY ic.key_ordinal`, objectName)
	if err != nil {
		return nil, fmt.Errorf("读取主键信息失败: %w", err)
	}
	defer rows.Close()
	var pks []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		pks = append(pks, name)
	}
	return pks, rows.Err()
}

func buildOracleTableDDL(db *sql.DB, databaseName, tableName string) (string, error) {
	type column struct {
		Name      string
		DataType  string
		DataLen   int
		Precision sql.NullInt64
		Scale     sql.NullInt64
		Nullable  string
		Default   sql.NullString
	}
	owner := strings.ToUpper(strings.TrimSpace(databaseName))
	table := strings.ToUpper(strings.TrimSpace(tableName))
	var rows *sql.Rows
	var err error
	if owner != "" {
		rows, err = db.Query(`SELECT column_name, data_type, data_length, data_precision, data_scale, nullable, data_default
FROM all_tab_columns WHERE owner = :1 AND table_name = :2 ORDER BY column_id`, owner, table)
	} else {
		rows, err = db.Query(`SELECT column_name, data_type, data_length, data_precision, data_scale, nullable, data_default
FROM user_tab_columns WHERE table_name = :1 ORDER BY column_id`, table)
	}
	if err != nil {
		return "", fmt.Errorf("读取字段信息失败: %w", err)
	}
	defer rows.Close()

	var columns []column
	for rows.Next() {
		var col column
		if err := rows.Scan(&col.Name, &col.DataType, &col.DataLen, &col.Precision, &col.Scale, &col.Nullable, &col.Default); err != nil {
			return "", fmt.Errorf("读取字段信息失败: %w", err)
		}
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("未读取到表字段信息")
	}

	pks, err := queryOraclePrimaryKeys(db, owner, table)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(columns)+1)
	for _, col := range columns {
		parts := []string{quoteDDLIdentifier("oracle", col.Name), oracleColumnType(col.DataType, col.DataLen, col.Precision, col.Scale)}
		if col.Default.Valid && strings.TrimSpace(col.Default.String) != "" {
			parts = append(parts, "DEFAULT "+strings.TrimSpace(col.Default.String))
		}
		if strings.EqualFold(col.Nullable, "N") {
			parts = append(parts, "NOT NULL")
		}
		lines = append(lines, "  "+strings.Join(parts, " "))
	}
	if len(pks) > 0 {
		lines = append(lines, "  PRIMARY KEY ("+joinQuotedDDLIdentifiers("oracle", pks)+")")
	}
	return "CREATE TABLE " + buildQuotedTableName("oracle", owner, table) + " (\n" + strings.Join(lines, ",\n") + "\n);", nil
}

func queryOraclePrimaryKeys(db *sql.DB, owner, tableName string) ([]string, error) {
	var rows *sql.Rows
	var err error
	if owner != "" {
		rows, err = db.Query(`SELECT cols.column_name
FROM all_constraints cons
JOIN all_cons_columns cols ON cols.owner = cons.owner AND cols.constraint_name = cons.constraint_name
WHERE cols.owner = :1 AND cols.table_name = :2 AND cons.constraint_type = 'P'
ORDER BY cols.position`, owner, tableName)
	} else {
		rows, err = db.Query(`SELECT cols.column_name
FROM user_constraints cons
JOIN user_cons_columns cols ON cols.constraint_name = cons.constraint_name
WHERE cols.table_name = :1 AND cons.constraint_type = 'P'
ORDER BY cols.position`, tableName)
	}
	if err != nil {
		return nil, fmt.Errorf("读取主键信息失败: %w", err)
	}
	defer rows.Close()
	var pks []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		pks = append(pks, name)
	}
	return pks, rows.Err()
}

func postgresColumnType(col postgresDDLColumn) string {
	dataType := strings.ToLower(col.DataType)
	switch dataType {
	case "character varying", "character":
		if col.CharLen.Valid {
			return dataType + "(" + strconv.FormatInt(col.CharLen.Int64, 10) + ")"
		}
	case "numeric", "decimal":
		if col.NumPrec.Valid && col.NumScale.Valid {
			return dataType + "(" + strconv.FormatInt(col.NumPrec.Int64, 10) + "," + strconv.FormatInt(col.NumScale.Int64, 10) + ")"
		}
		if col.NumPrec.Valid {
			return dataType + "(" + strconv.FormatInt(col.NumPrec.Int64, 10) + ")"
		}
	case "user-defined":
		if col.UdtName != "" {
			return col.UdtName
		}
	}
	if col.UdtName == "int4" {
		return "integer"
	}
	if col.UdtName == "int8" {
		return "bigint"
	}
	return dataType
}

func sqlServerColumnType(typeName string, maxLength, precision, scale int) string {
	lower := strings.ToLower(typeName)
	switch lower {
	case "varchar", "char", "varbinary", "binary":
		if maxLength == -1 {
			return lower + "(max)"
		}
		return lower + "(" + strconv.Itoa(maxLength) + ")"
	case "nvarchar", "nchar":
		if maxLength == -1 {
			return lower + "(max)"
		}
		return lower + "(" + strconv.Itoa(maxLength/2) + ")"
	case "decimal", "numeric":
		return lower + "(" + strconv.Itoa(precision) + "," + strconv.Itoa(scale) + ")"
	default:
		return lower
	}
}

func oracleColumnType(dataType string, dataLen int, precision, scale sql.NullInt64) string {
	upper := strings.ToUpper(strings.TrimSpace(dataType))
	switch upper {
	case "VARCHAR2", "NVARCHAR2", "CHAR", "NCHAR", "RAW":
		return upper + "(" + strconv.Itoa(dataLen) + ")"
	case "NUMBER":
		if precision.Valid && scale.Valid {
			return upper + "(" + strconv.FormatInt(precision.Int64, 10) + "," + strconv.FormatInt(scale.Int64, 10) + ")"
		}
		if precision.Valid {
			return upper + "(" + strconv.FormatInt(precision.Int64, 10) + ")"
		}
	}
	return upper
}

func buildQuotedTableName(ct, databaseName, tableName string) string {
	switch {
	case (ct == "mysql" || ct == "clickhouse") && strings.TrimSpace(databaseName) != "":
		return quoteDDLIdentifier(ct, databaseName) + "." + quoteDDLIdentifier(ct, tableName)
	case (ct == "sqlserver" || ct == "mssql") && strings.TrimSpace(databaseName) != "":
		return quoteDDLIdentifier(ct, databaseName) + "." + quoteDDLIdentifier(ct, "dbo") + "." + quoteDDLIdentifier(ct, tableName)
	case ct == "oracle" && strings.TrimSpace(databaseName) != "":
		return quoteDDLIdentifier(ct, databaseName) + "." + quoteDDLIdentifier(ct, tableName)
	case (ct == "postgresql" || ct == "pgsql"):
		return quoteDDLIdentifier(ct, "public") + "." + quoteDDLIdentifier(ct, tableName)
	default:
		return quoteDDLIdentifier(ct, tableName)
	}
}

func quoteDDLIdentifier(ct, name string) string {
	switch {
	case ct == "mysql" || ct == "clickhouse":
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	case ct == "sqlserver" || ct == "mssql":
		return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
	case ct == "postgresql" || ct == "pgsql" || ct == "sqlite" || ct == "oracle":
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	default:
		return name
	}
}

func joinQuotedDDLIdentifiers(ct string, names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quoteDDLIdentifier(ct, name))
	}
	return strings.Join(quoted, ", ")
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func normalizeRemoteSQLQuery(rawSQL string) (string, error) {
	query := trimSQLStatementEnd(strings.TrimSpace(rawSQL))
	if query == "" {
		return "", fmt.Errorf("请输入 SQL 查询语句")
	}
	queryForKeyword := strings.TrimSpace(stripLeadingSQLComments(query))
	if queryForKeyword == "" {
		return "", fmt.Errorf("请输入 SQL 查询语句")
	}
	if containsSQLStatementSeparator(queryForKeyword) {
		return "", fmt.Errorf("一次只允许查询一条 SQL")
	}

	firstKeyword := firstSQLKeyword(queryForKeyword)
	switch firstKeyword {
	case "select", "with", "show", "desc", "describe", "explain":
	default:
		return "", fmt.Errorf("只支持查询类 SQL，不支持执行写入或结构变更语句")
	}

	if containsForbiddenSQLKeyword(maskSQLLiteralsAndComments(queryForKeyword)) {
		return "", fmt.Errorf("只支持只读查询，不支持 insert/update/delete/drop 等执行语句")
	}
	return queryForKeyword, nil
}

func trimSQLStatementEnd(query string) string {
	query = strings.TrimSpace(query)
	for strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	return query
}

func stripLeadingSQLComments(query string) string {
	for {
		query = strings.TrimLeft(query, " \t\r\n")
		if strings.HasPrefix(query, "--") {
			newline := strings.IndexAny(query, "\r\n")
			if newline == -1 {
				return ""
			}
			query = query[newline+1:]
			continue
		}
		if strings.HasPrefix(query, "/*") {
			end := strings.Index(query[2:], "*/")
			if end == -1 {
				return query
			}
			query = query[end+4:]
			continue
		}
		return query
	}
}

func firstSQLKeyword(query string) string {
	query = strings.TrimSpace(query)
	var builder strings.Builder
	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			builder.WriteRune(r)
			continue
		}
		break
	}
	return strings.ToLower(builder.String())
}

func containsForbiddenSQLKeyword(query string) bool {
	for _, token := range tokenizeSQLWords(query) {
		switch token {
		case "insert", "update", "delete", "drop", "alter", "create", "truncate", "merge", "replace", "grant", "revoke", "call", "exec", "execute":
			return true
		}
	}
	return false
}

func containsSQLKeyword(query, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return false
	}
	for _, token := range tokenizeSQLWords(query) {
		if token == keyword {
			return true
		}
	}
	return false
}

func tokenizeSQLWords(query string) []string {
	words := make([]string, 0)
	var builder strings.Builder
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		words = append(words, strings.ToLower(builder.String()))
		builder.Reset()
	}

	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return words
}

func containsSQLStatementSeparator(query string) bool {
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		next := byte(0)
		if i+1 < len(query) {
			next = query[i+1]
		}

		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inSingleQuote {
			if ch == '\'' {
				if next == '\'' {
					i++
					continue
				}
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			if ch == '"' {
				if next == '"' {
					i++
					continue
				}
				inDoubleQuote = false
			}
			continue
		}
		if inBacktick {
			if ch == '`' {
				inBacktick = false
			}
			continue
		}

		switch {
		case ch == '-' && next == '-':
			inLineComment = true
			i++
		case ch == '/' && next == '*':
			inBlockComment = true
			i++
		case ch == '\'':
			inSingleQuote = true
		case ch == '"':
			inDoubleQuote = true
		case ch == '`':
			inBacktick = true
		case ch == ';':
			return true
		}
	}
	return false
}

func maskSQLLiteralsAndComments(query string) string {
	var builder strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		next := byte(0)
		if i+1 < len(query) {
			next = query[i+1]
		}

		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
				builder.WriteByte(ch)
			} else {
				builder.WriteByte(' ')
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				builder.WriteString("  ")
				i++
			} else {
				builder.WriteByte(' ')
			}
			continue
		}
		if inSingleQuote {
			if ch == '\'' {
				if next == '\'' {
					builder.WriteString("  ")
					i++
					continue
				}
				inSingleQuote = false
			}
			builder.WriteByte(' ')
			continue
		}
		if inDoubleQuote {
			if ch == '"' {
				if next == '"' {
					builder.WriteString("  ")
					i++
					continue
				}
				inDoubleQuote = false
			}
			builder.WriteByte(' ')
			continue
		}
		if inBacktick {
			if ch == '`' {
				inBacktick = false
			}
			builder.WriteByte(' ')
			continue
		}

		switch {
		case ch == '-' && next == '-':
			inLineComment = true
			builder.WriteString("  ")
			i++
		case ch == '/' && next == '*':
			inBlockComment = true
			builder.WriteString("  ")
			i++
		case ch == '\'':
			inSingleQuote = true
			builder.WriteByte(' ')
		case ch == '"':
			inDoubleQuote = true
			builder.WriteByte(' ')
		case ch == '`':
			inBacktick = true
			builder.WriteByte(' ')
		default:
			builder.WriteByte(ch)
		}
	}
	return builder.String()
}

func normalizeSQLQueryCellValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		return formatPreviewTime(v, "")
	case []byte:
		return string(v)
	case string, bool, int64, float64:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

type sqlArgBuilder struct {
	ct   string
	args []interface{}
}

func (b *sqlArgBuilder) Add(value interface{}) string {
	b.args = append(b.args, value)
	index := strconv.Itoa(len(b.args))
	switch {
	case b.ct == "postgresql" || b.ct == "pgsql":
		return "$" + index
	case b.ct == "oracle":
		return ":" + index
	case b.ct == "sqlserver" || b.ct == "mssql":
		return "@p" + index
	default:
		return "?"
	}
}

func quoteColumnIdentifier(ct, name string) string {
	switch {
	case ct == "mysql" || ct == "clickhouse":
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	case ct == "sqlserver" || ct == "mssql":
		return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
	case ct == "postgresql" || ct == "pgsql" || ct == "sqlite" || ct == "oracle":
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	default:
		return name
	}
}

func getTablePrimaryKey(db *sql.DB, dbName, tableName, ct string) string {
	var pk sql.NullString
	switch {
	case ct == "mysql":
		_ = db.QueryRow("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY' LIMIT 1", dbName, tableName).Scan(&pk)
	case ct == "postgresql" || ct == "pgsql":
		_ = db.QueryRow("SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) JOIN pg_class c ON c.oid = i.indrelid JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relname = $1 AND i.indisprimary", tableName).Scan(&pk)
	case ct == "sqlserver" || ct == "mssql":
		if dbName != "" {
			query := fmt.Sprintf("SELECT top 1 c.name FROM [%s].sys.indexes i JOIN [%s].sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id JOIN [%s].sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id WHERE i.is_primary_key = 1 AND i.object_id = OBJECT_ID(@p1)", dbName, dbName, dbName)
			_ = db.QueryRow(query, fmt.Sprintf("[%s].[dbo].[%s]", dbName, tableName)).Scan(&pk)
		} else {
			_ = db.QueryRow("SELECT top 1 c.name FROM sys.indexes i JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id WHERE i.is_primary_key = 1 AND i.object_id = OBJECT_ID(@p1)", tableName).Scan(&pk)
		}
	case ct == "oracle":
		if dbName != "" {
			_ = db.QueryRow("SELECT cols.column_name FROM all_constraints cons, all_cons_columns cols WHERE cols.owner = :1 AND cols.table_name = :2 AND cons.constraint_type = 'P' AND cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner FETCH FIRST 1 ROWS ONLY", strings.ToUpper(dbName), strings.ToUpper(tableName)).Scan(&pk)
		} else {
			_ = db.QueryRow("SELECT cols.column_name FROM all_constraints cons, all_cons_columns cols WHERE cols.table_name = :1 AND cons.constraint_type = 'P' AND cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner FETCH FIRST 1 ROWS ONLY", strings.ToUpper(tableName)).Scan(&pk)
		}
	case ct == "sqlite":
		rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
		if err != nil {
			return ""
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull sql.NullInt64
			var dflt sql.NullString
			var ispk int
			if rows.Scan(&cid, &name, &typ, &notnull, &dflt, &ispk) == nil && ispk == 1 {
				return name
			}
		}
	}

	if pk.Valid {
		return pk.String
	}
	return ""
}

// buildQualifiedTableName returns the fully qualified table name for cross-database/schema queries.
func buildQualifiedTableName(ct, databaseName, tableName string) string {
	switch {
	case ct == "mysql" && databaseName != "":
		return "`" + databaseName + "`.`" + tableName + "`"
	case ct == "oracle" && databaseName != "":
		return strings.ToUpper(databaseName) + "." + strings.ToUpper(tableName)
	case (ct == "sqlserver" || ct == "mssql") && databaseName != "":
		return fmt.Sprintf("[%s].[dbo].[%s]", databaseName, tableName)
	case ct == "postgresql" || ct == "pgsql":
		// PostgreSQL databases are isolated; the DSN already connects to the correct database.
		// Table names should be schema-qualified as "public.tableName", not "database.tableName".
		return "public." + tableName
	case ct == "clickhouse" && databaseName != "":
		return "`" + databaseName + "`.`" + tableName + "`"
	default:
		return tableName
	}
}
