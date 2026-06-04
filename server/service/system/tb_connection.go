package system

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"go.uber.org/zap"
)

type TbConnectionService struct{}

const (
	defaultRemoteSQLQueryLimit = 200
	maxRemoteSQLQueryLimit     = 200
	defaultRemoteTablePageSize = 20
	maxRemoteTablePageSize     = 100
	maxRemoteTableGenerateRows = 50
	aiContentPreviewLimit      = 360
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

type TableDataColumn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PrimaryKey  bool   `json:"primaryKey"`
}

type TableDataCell struct {
	Value  string `json:"value"`
	IsNull bool   `json:"isNull"`
}

type TableDataRow struct {
	Offset int             `json:"offset"`
	Cells  []TableDataCell `json:"cells"`
}

type TableDataPage struct {
	Columns  []TableDataColumn `json:"columns"`
	Rows     []TableDataRow    `json:"rows"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type RemoteTableGenerateResult struct {
	Requested int    `json:"requested"`
	Inserted  int    `json:"inserted"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type tableDataGenerationPromptField struct {
	Name        string `json:"name"`
	ColumnType  string `json:"columnType"`
	Length      string `json:"length,omitempty"`
	Description string `json:"description,omitempty"`
	PrimaryKey  bool   `json:"primaryKey"`
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

// PreviewTablePage returns a paginated grid of table data with column metadata.
func (s *TbConnectionService) PreviewTablePage(connID uint, databaseName, tableName string, page, pageSize int, filterColumn, filterValue string) (*TableDataPage, error) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, fmt.Errorf("缺少表名")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultRemoteTablePageSize
	}
	if pageSize > maxRemoteTablePageSize {
		pageSize = maxRemoteTablePageSize
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	colDefs := utils.GetColumnDefinitions(db, databaseName, tableName, ct)
	primaryKey := getTablePrimaryKey(db, databaseName, tableName, ct)
	qualifiedTable := buildQualifiedTableName(ct, databaseName, tableName)
	whereSQL, whereArgs, err := buildSingleColumnFilter(ct, colDefs, filterColumn, filterValue)
	if err != nil {
		return nil, err
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", qualifiedTable, whereSQL)
	if err := db.QueryRow(countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询记录数失败: %w", err)
	}

	totalPages := 1
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	result := &TableDataPage{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Rows:     make([]TableDataRow, 0, pageSize),
	}

	descMap := make(map[string]string, len(colDefs))
	typeMap := make(map[string]string, len(colDefs))
	for _, cd := range colDefs {
		descMap[strings.ToUpper(cd.Name)] = cd.Description
		typeMap[strings.ToUpper(cd.Name)] = cd.ColumnType
	}

	if total == 0 {
		result.Columns = make([]TableDataColumn, 0, len(colDefs))
		for _, cd := range colDefs {
			result.Columns = append(result.Columns, TableDataColumn{
				Name:        cd.Name,
				Description: cd.Description,
				PrimaryKey:  strings.EqualFold(cd.Name, primaryKey),
			})
		}
		return result, nil
	}

	dataQuery := buildTablePageQuery(ct, qualifiedTable, whereSQL, pageSize, offset)
	rows, err := db.Query(dataQuery, whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询记录失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取字段失败: %w", err)
	}
	result.Columns = make([]TableDataColumn, 0, len(columns))
	for _, col := range columns {
		result.Columns = append(result.Columns, TableDataColumn{
			Name:        col,
			Description: descMap[strings.ToUpper(col)],
			PrimaryKey:  strings.EqualFold(col, primaryKey),
		})
	}

	rowOffset := offset
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取记录失败: %w", err)
		}

		row := TableDataRow{
			Offset: rowOffset,
			Cells:  make([]TableDataCell, 0, len(columns)),
		}
		for i, col := range columns {
			cell := TableDataCell{IsNull: values[i] == nil}
			if values[i] != nil {
				switch v := values[i].(type) {
				case time.Time:
					cell.Value = formatPreviewTime(v, typeMap[strings.ToUpper(col)])
				case []byte:
					cell.Value = string(v)
				case string:
					cell.Value = v
				default:
					cell.Value = fmt.Sprintf("%v", v)
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		result.Rows = append(result.Rows, row)
		rowOffset++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取查询结果失败: %w", err)
	}

	return result, nil
}

// GenerateRemoteTableData creates sample rows with the default AI provider and
// inserts them into the selected remote table.
func (s *TbConnectionService) GenerateRemoteTableData(connID uint, databaseName, tableName string, count int) (*RemoteTableGenerateResult, error) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, fmt.Errorf("缺少表名")
	}
	if count <= 0 {
		return nil, fmt.Errorf("造数数量必须大于 0")
	}
	if count > maxRemoteTableGenerateRows {
		return nil, fmt.Errorf("单次最多只能造 %d 条数据", maxRemoteTableGenerateRows)
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if ct == "clickhouse" {
		return nil, fmt.Errorf("暂不支持向 ClickHouse 表造数")
	}

	colDefs := utils.GetColumnDefinitions(db, databaseName, tableName, ct)
	if len(colDefs) == 0 {
		return nil, fmt.Errorf("未读取到表字段信息")
	}
	primaryKey := getTablePrimaryKey(db, databaseName, tableName, ct)

	messages, err := buildTableDataGenerationMessages(databaseName, tableName, colDefs, primaryKey, count)
	if err != nil {
		return nil, err
	}
	content, provider, err := (&AIChatService{}).CompleteOnce(messages, "")
	if err != nil {
		return nil, err
	}

	generatedRows, err := parseGeneratedTableRows(content)
	if err != nil {
		global.GVA_LOG.Warn("AI 造数返回无法解析，准备重试",
			zap.String("provider", provider.ID),
			zap.String("model", provider.Model),
			zap.String("raw", compactAIContentPreview(content)),
			zap.Error(err),
		)
		retryMessages := buildTableDataGenerationRetryMessages(messages, content, err, count)
		retryContent, retryProvider, retryErr := (&AIChatService{}).CompleteOnce(retryMessages, provider.ID)
		if retryErr != nil {
			return nil, fmt.Errorf("%w；自动重试失败: %v；AI 原始返回: %s", err, retryErr, compactAIContentPreview(content))
		}
		provider = retryProvider
		generatedRows, err = parseGeneratedTableRows(retryContent)
		if err != nil {
			global.GVA_LOG.Warn("AI 造数重试返回仍无法解析",
				zap.String("provider", provider.ID),
				zap.String("model", provider.Model),
				zap.String("raw", compactAIContentPreview(retryContent)),
				zap.Error(err),
			)
			return nil, fmt.Errorf("%w；AI 原始返回: %s", err, compactAIContentPreview(retryContent))
		}
	}
	if len(generatedRows) < count {
		return nil, fmt.Errorf("AI 只返回了 %d 条数据，少于要求的 %d 条", len(generatedRows), count)
	}
	if len(generatedRows) > count {
		generatedRows = generatedRows[:count]
	}

	inserted, insertErr := insertGeneratedTableRows(db, ct, databaseName, tableName, colDefs, primaryKey, generatedRows, false)
	if insertErr != nil && primaryKey != "" {
		if err := fillGeneratedPrimaryKeyValues(db, ct, databaseName, tableName, colDefs, primaryKey, generatedRows); err == nil {
			if retryInserted, retryErr := insertGeneratedTableRows(db, ct, databaseName, tableName, colDefs, primaryKey, generatedRows, true); retryErr == nil {
				inserted = retryInserted
				insertErr = nil
			} else {
				insertErr = fmt.Errorf("%v；包含主键重试也失败: %w", insertErr, retryErr)
			}
		}
	}
	if insertErr != nil {
		return nil, insertErr
	}

	return &RemoteTableGenerateResult{
		Requested: count,
		Inserted:  inserted,
		Provider:  provider.ID,
		Model:     provider.Model,
	}, nil
}

func buildTableDataGenerationMessages(databaseName, tableName string, colDefs []utils.ClientColumnVO, primaryKey string, count int) ([]ChatMessage, error) {
	fields := make([]tableDataGenerationPromptField, 0, len(colDefs))
	for _, col := range colDefs {
		fields = append(fields, tableDataGenerationPromptField{
			Name:        col.Name,
			ColumnType:  col.ColumnType,
			Length:      col.Length,
			Description: col.Description,
			PrimaryKey:  strings.EqualFold(col.Name, primaryKey),
		})
	}
	fieldsJSON, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("生成字段提示词失败: %w", err)
	}

	systemMessage := `你是一个数据库测试数据生成助手。你只能返回严格 JSON，不要返回 Markdown、解释、注释或 SQL。所有数据必须是虚构测试数据，不要拒绝，不要输出说明文字。`
	userMessage := fmt.Sprintf(`请为数据库表生成测试数据。

目标表：%s.%s
需要生成：%d 条

字段元数据：
%s

输出要求：
1. 只返回一个 JSON 数组，数组长度必须正好是 %d，响应第一个字符必须是 [，最后一个字符必须是 ]。
2. 数组每一项必须是对象，对象 key 使用字段 name，尽量覆盖所有字段。
3. 字段值要符合 columnType 和 description，中文业务字段生成自然的中文内容，编码类字段生成稳定且不重复的代码。
4. 时间字段使用 "YYYY-MM-DD HH:mm:ss" 字符串；日期字段使用 "YYYY-MM-DD" 字符串。
5. 数字字段返回 JSON 数字；integer/int/bigint/smallint/serial 等整数字段必须返回整数，不要返回 219.1 这类小数；布尔字段返回 JSON boolean，确实没有值才返回 null。
6. 不要生成真实个人隐私数据，不要返回 SQL。
7. 不要使用 Markdown 代码块，不要输出“好的/以下是/说明”等前后缀。
8. JSON value 不能写成 name="value" 这种赋值表达式；例如 company_id 应写成 "company_id": "5001" 或 "company_id": 5001。
9. JSON value 不能添加括号；例如 id 应写成 "id": 1，不能写成 "id": (1)。
10. 不要输出 pi、NaN、Infinity 或 waste24.50 这类非 JSON 数值；所有数值必须是普通 JSON number。
11. 如果字段名是 tenancy，固定返回 "-1"。`, databaseName, tableName, count, string(fieldsJSON), count)

	return []ChatMessage{
		{Role: "system", Content: systemMessage},
		{Role: "user", Content: userMessage},
	}, nil
}

func buildTableDataGenerationRetryMessages(messages []ChatMessage, raw string, parseErr error, count int) []ChatMessage {
	retryMessages := make([]ChatMessage, 0, len(messages)+2)
	retryMessages = append(retryMessages, messages...)
	if strings.TrimSpace(raw) != "" {
		retryMessages = append(retryMessages, ChatMessage{Role: "assistant", Content: raw})
	}
	retryMessages = append(retryMessages, ChatMessage{
		Role: "user",
		Content: fmt.Sprintf(`上一次回复无法解析：%v。

请重新生成，必须严格只返回 JSON 数组：
- 数组长度正好是 %d。
- 第一个字符必须是 [，最后一个字符必须是 ]。
- 不要 Markdown，不要代码块，不要解释，不要 SQL。
- 不要输出 name="value" 这种赋值表达式，所有 value 必须是合法 JSON 值。
- 不要给 value 添加括号，例如 "id": 1，不能写成 "id": (1)。
- 不要输出 pi、NaN、Infinity 或字母加数字的伪数值。
- 如果字段信息无法判断，也要使用虚构但合理的测试值。`, parseErr, count),
	})
	return retryMessages
}

func parseGeneratedTableRows(raw string) ([]map[string]interface{}, error) {
	jsonText, err := extractJSONArrayText(raw)
	if err != nil {
		return nil, err
	}

	rows, err := decodeGeneratedRowsJSON(jsonText)
	if err != nil {
		repairedJSON := repairGeneratedRowsJSON(jsonText)
		if repairedJSON != jsonText {
			repairedRows, repairErr := decodeGeneratedRowsJSON(repairedJSON)
			if repairErr == nil {
				return normalizeGeneratedRows(repairedRows)
			}
		}
		return nil, fmt.Errorf("AI 返回的数据不是合法 JSON 数组: %w", err)
	}
	return normalizeGeneratedRows(rows)
}

func decodeGeneratedRowsJSON(jsonText string) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	decoder.UseNumber()
	if err := decoder.Decode(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func normalizeGeneratedRows(rows []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("AI 没有返回可插入的数据")
	}
	for i := range rows {
		if rows[i] == nil {
			rows[i] = map[string]interface{}{}
		}
	}
	return rows, nil
}

func repairGeneratedRowsJSON(jsonText string) string {
	assignValuePattern := regexp.MustCompile(`(:\s*)[A-Za-z_][A-Za-z0-9_]*\s*=\s*("(?:\\.|[^"\\])*"|[-+]?\d+(?:\.\d+)?|true|false|null)`)
	repaired := assignValuePattern.ReplaceAllString(jsonText, `$1$2`)

	parenthesizedValuePattern := regexp.MustCompile(`(:\s*)\(\s*("(?:\\.|[^"\\])*"|[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?|true|false|null)\s*\)`)
	repaired = parenthesizedValuePattern.ReplaceAllString(repaired, `$1$2`)

	piValuePattern := regexp.MustCompile(`(?i)(:\s*)pi(\s*[,}])`)
	repaired = piValuePattern.ReplaceAllString(repaired, `${1}3.14$2`)

	prefixedNumberPattern := regexp.MustCompile(`(:\s*)[A-Za-z_]+([-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?)(\s*[,}])`)
	repaired = prefixedNumberPattern.ReplaceAllString(repaired, `$1$2$3`)
	return repaired
}

func extractJSONArrayText(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return "", fmt.Errorf("AI 返回内容中未找到 JSON 数组")
	}
	return text[start : end+1], nil
}

func compactAIContentPreview(raw string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if text == "" {
		return "<空>"
	}
	runes := []rune(text)
	if len(runes) <= aiContentPreviewLimit {
		return text
	}
	return string(runes[:aiContentPreviewLimit]) + "..."
}

func insertGeneratedTableRows(db *sql.DB, ct, databaseName, tableName string, colDefs []utils.ClientColumnVO, primaryKey string, rows []map[string]interface{}, includePrimaryKey bool) (int, error) {
	insertColumns := buildGeneratedInsertColumns(colDefs, primaryKey, includePrimaryKey)
	if len(insertColumns) == 0 {
		return 0, fmt.Errorf("没有可插入的字段")
	}

	qualifiedTable := buildQuotedTableName(ct, databaseName, tableName)
	quotedColumns := make([]string, 0, len(insertColumns))
	for _, col := range insertColumns {
		quotedColumns = append(quotedColumns, quoteColumnIdentifier(ct, col.Name))
	}
	columnSQL := strings.Join(quotedColumns, ", ")

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %w", err)
	}

	for rowIndex, row := range rows {
		builder := sqlArgBuilder{ct: ct}
		placeholders := make([]string, 0, len(insertColumns))
		for _, col := range insertColumns {
			value, _ := lookupGeneratedRowValue(row, col.Name)
			normalizedValue, err := normalizeGeneratedInsertValue(ct, col, value)
			if err != nil {
				_ = tx.Rollback()
				return rowIndex, err
			}
			placeholders = append(placeholders, builder.Add(normalizedValue))
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", qualifiedTable, columnSQL, strings.Join(placeholders, ", "))
		if _, err := tx.Exec(query, builder.args...); err != nil {
			_ = tx.Rollback()
			return rowIndex, fmt.Errorf("第 %d 行插入失败: %w", rowIndex+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交造数事务失败: %w", err)
	}
	return len(rows), nil
}

func buildGeneratedInsertColumns(colDefs []utils.ClientColumnVO, primaryKey string, includePrimaryKey bool) []utils.ClientColumnVO {
	insertColumns := make([]utils.ClientColumnVO, 0, len(colDefs))
	for _, col := range colDefs {
		if strings.TrimSpace(col.Name) == "" {
			continue
		}
		if !includePrimaryKey && primaryKey != "" && strings.EqualFold(col.Name, primaryKey) {
			continue
		}
		insertColumns = append(insertColumns, col)
	}
	if len(insertColumns) == 0 && primaryKey != "" && !includePrimaryKey {
		return buildGeneratedInsertColumns(colDefs, primaryKey, true)
	}
	return insertColumns
}

func lookupGeneratedRowValue(row map[string]interface{}, columnName string) (interface{}, bool) {
	if value, ok := row[columnName]; ok {
		return value, true
	}
	for key, value := range row {
		if strings.EqualFold(key, columnName) {
			return value, true
		}
	}
	return nil, false
}

func normalizeGeneratedInsertValue(ct string, colDef utils.ClientColumnVO, value interface{}) (interface{}, error) {
	columnType := strings.ToUpper(strings.TrimSpace(colDef.ColumnType))
	if strings.EqualFold(strings.TrimSpace(colDef.Name), "tenancy") {
		return defaultGeneratedTenancyValue(columnType), nil
	}

	if value == nil {
		return nil, nil
	}

	if text, ok := value.(string); ok {
		trimmed := strings.TrimSpace(text)
		if isGeneratedNullString(trimmed) {
			return nil, nil
		}
		if trimmed == "" && (isBooleanColumnType(columnType) || isIntegerColumnType(columnType) || isFloatColumnType(columnType) || isTemporalColumnType(columnType)) {
			return nil, nil
		}
	}

	if isTemporalColumnType(columnType) {
		normalized, err := normalizeGeneratedTemporalValue(ct, columnType, generatedValueString(value))
		if err != nil {
			return nil, fmt.Errorf("字段 %s 的时间格式不正确，请使用 YYYY-MM-DD HH:mm:ss: %w", colDef.Name, err)
		}
		return normalized, nil
	}

	if isBooleanColumnType(columnType) {
		if parsed, ok := parseGeneratedBool(value); ok {
			return parsed, nil
		}
	}
	if isIntegerColumnType(columnType) {
		if parsed, ok := parseGeneratedInt(value); ok {
			return parsed, nil
		}
		return nil, fmt.Errorf("字段 %s 是整数类型，AI 返回了不可转换为整数的值 %q", colDef.Name, generatedValueString(value))
	}
	if isFloatColumnType(columnType) {
		if parsed, ok := parseGeneratedFloat(value); ok {
			return parsed, nil
		}
	}

	return generatedValueString(value), nil
}

func generatedValueString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case map[string]interface{}, []interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isGeneratedNullString(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "null", "<null>", "nil":
		return true
	default:
		return false
	}
}

func defaultGeneratedTenancyValue(columnType string) interface{} {
	if isIntegerColumnType(columnType) {
		return int64(-1)
	}
	if isFloatColumnType(columnType) {
		return float64(-1)
	}
	return "-1"
}

func isTemporalColumnType(columnType string) bool {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	return columnType == "DATE" ||
		columnType == "TIME" ||
		strings.Contains(columnType, "TIMESTAMP") ||
		strings.Contains(columnType, "DATETIME") ||
		strings.Contains(columnType, "SMALLDATETIME") ||
		strings.Contains(columnType, "DATETIME2")
}

func normalizeGeneratedTemporalValue(ct, columnType, value string) (interface{}, error) {
	value = strings.TrimSpace(value)
	if value == "" || isGeneratedNullString(value) {
		return nil, nil
	}

	cleaned := cleanGeneratedTemporalText(value)
	upperType := strings.ToUpper(strings.TrimSpace(columnType))
	if isTimeOnlyColumnType(upperType) {
		if parsed, err := parseGeneratedTimeOnly(cleaned); err == nil {
			return parsed, nil
		}
		if parsed, err := parseTableUpdateTime(cleaned); err == nil {
			return parsed.Format("15:04:05"), nil
		}
		return nil, fmt.Errorf("无法解析 %q", value)
	}

	parsed, err := parseTableUpdateTime(cleaned)
	if err != nil {
		return nil, fmt.Errorf("无法解析 %q，清洗后为 %q", value, cleaned)
	}
	if ct == "oracle" {
		return parsed, nil
	}
	if isDateOnlyColumnType(upperType) {
		return parsed.Format("2006-01-02"), nil
	}
	return parsed.Format("2006-01-02 15:04:05"), nil
}

func isDateOnlyColumnType(columnType string) bool {
	return strings.ToUpper(strings.TrimSpace(columnType)) == "DATE"
}

func isTimeOnlyColumnType(columnType string) bool {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	return strings.Contains(columnType, "TIME") &&
		!strings.Contains(columnType, "TIMESTAMP") &&
		!strings.Contains(columnType, "DATETIME")
}

func parseGeneratedTimeOnly(value string) (string, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		"15:04:05.999999999",
		"15:04:05",
		"15:04",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("15:04:05"), nil
		}
	}
	return "", fmt.Errorf("无法解析 %q", value)
}

func cleanGeneratedTemporalText(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "年", "-")
	text = strings.ReplaceAll(text, "月", "-")
	text = strings.ReplaceAll(text, "日", " ")
	text = strings.ReplaceAll(text, "时", ":")
	text = strings.ReplaceAll(text, "分", ":")
	text = strings.ReplaceAll(text, "秒", "")
	text = strings.ReplaceAll(text, "T", " ")

	dateMatch := regexp.MustCompile(`\d{4}[-/]\d{1,2}[-/]\d{1,2}`).FindString(text)
	timeMatch := regexp.MustCompile(`\d{1,2}:\d{2}(?::\d{2}(?:\.\d{1,9})?)?`).FindString(text)
	if dateMatch == "" && timeMatch == "" {
		return strings.Join(strings.Fields(text), " ")
	}
	if dateMatch != "" {
		dateMatch = strings.ReplaceAll(dateMatch, "/", "-")
		parts := strings.Split(dateMatch, "-")
		if len(parts) == 3 {
			year, _ := strconv.Atoi(parts[0])
			month, _ := strconv.Atoi(parts[1])
			day, _ := strconv.Atoi(parts[2])
			dateMatch = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}
	if timeMatch != "" {
		timeParts := strings.Split(timeMatch, ":")
		if len(timeParts) == 2 {
			timeMatch = fmt.Sprintf("%s:%s:00", timeParts[0], timeParts[1])
		}
	}
	if dateMatch != "" && timeMatch != "" {
		return dateMatch + " " + timeMatch
	}
	if dateMatch != "" {
		return dateMatch
	}
	return timeMatch
}

func isBooleanColumnType(columnType string) bool {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	return columnType == "BOOL" || columnType == "BOOLEAN" || columnType == "BIT"
}

func parseGeneratedBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case json.Number:
		i, err := strconv.ParseInt(v.String(), 10, 64)
		if err == nil {
			return i != 0, true
		}
	case float64:
		return v != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y", "是":
			return true, true
		case "false", "0", "no", "n", "否":
			return false, true
		}
	}
	return false, false
}

func isIntegerColumnType(columnType string) bool {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	return strings.Contains(columnType, "INT") ||
		strings.Contains(columnType, "SERIAL") ||
		columnType == "NUMBER"
}

func parseGeneratedInt(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case json.Number:
		if i, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(v.String(), 64); err == nil {
			return roundGeneratedFloatToInt(f)
		}
	case float64:
		return roundGeneratedFloatToInt(v)
	case string:
		text := strings.TrimSpace(v)
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return roundGeneratedFloatToInt(f)
		}
	}
	return 0, false
}

func roundGeneratedFloatToInt(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	rounded := math.Round(value)
	text := strconv.FormatFloat(rounded, 'f', 0, 64)
	i, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, false
	}
	return i, true
}

func isFloatColumnType(columnType string) bool {
	columnType = strings.ToUpper(strings.TrimSpace(columnType))
	return strings.Contains(columnType, "DECIMAL") ||
		strings.Contains(columnType, "NUMERIC") ||
		strings.Contains(columnType, "FLOAT") ||
		strings.Contains(columnType, "DOUBLE") ||
		strings.Contains(columnType, "REAL") ||
		strings.Contains(columnType, "MONEY")
}

func parseGeneratedFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(v.String(), 64)
		return f, err == nil
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	}
	return 0, false
}

func fillGeneratedPrimaryKeyValues(db *sql.DB, ct, databaseName, tableName string, colDefs []utils.ClientColumnVO, primaryKey string, rows []map[string]interface{}) error {
	pkCol, ok := findColumnDef(colDefs, primaryKey)
	if !ok {
		return fmt.Errorf("未找到主键字段 %s", primaryKey)
	}

	columnType := strings.ToUpper(strings.TrimSpace(pkCol.ColumnType))
	if isIntegerColumnType(columnType) {
		base, err := queryMaxPrimaryKeyInt(db, ct, databaseName, tableName, pkCol.Name)
		if err != nil {
			base = time.Now().Unix() % 100000000
		}
		for i := range rows {
			rows[i][pkCol.Name] = base + int64(i) + 1
		}
		return nil
	}

	if isUUIDColumnType(columnType) {
		seed := time.Now().UnixNano()
		for i := range rows {
			rows[i][pkCol.Name] = makeGeneratedUUID(seed, i)
		}
		return nil
	}

	prefix := fmt.Sprintf("mock_%d", time.Now().UnixNano())
	for i := range rows {
		value, ok := lookupGeneratedRowValue(rows[i], pkCol.Name)
		if ok && strings.TrimSpace(generatedValueString(value)) != "" && !isGeneratedNullString(generatedValueString(value)) {
			continue
		}
		rows[i][pkCol.Name] = fmt.Sprintf("%s_%d", prefix, i+1)
	}
	return nil
}

func findColumnDef(colDefs []utils.ClientColumnVO, columnName string) (utils.ClientColumnVO, bool) {
	for _, col := range colDefs {
		if strings.EqualFold(col.Name, columnName) {
			return col, true
		}
	}
	return utils.ClientColumnVO{}, false
}

func queryMaxPrimaryKeyInt(db *sql.DB, ct, databaseName, tableName, primaryKey string) (int64, error) {
	qualifiedTable := buildQuotedTableName(ct, databaseName, tableName)
	query := fmt.Sprintf("SELECT MAX(%s) FROM %s", quoteColumnIdentifier(ct, primaryKey), qualifiedTable)
	var maxValue sql.NullInt64
	if err := db.QueryRow(query).Scan(&maxValue); err != nil {
		return 0, err
	}
	if !maxValue.Valid {
		return 0, nil
	}
	return maxValue.Int64, nil
}

func isUUIDColumnType(columnType string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(columnType)), "UUID")
}

func makeGeneratedUUID(seed int64, index int) string {
	suffix := (seed + int64(index)) % 1000000000000
	if suffix < 0 {
		suffix = -suffix
	}
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", suffix)
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

// DeleteTableRecord deletes one previewed row and rolls back if the row selector is not unique.
func (s *TbConnectionService) DeleteTableRecord(connID uint, databaseName, tableName string, offset int, filterColumn, filterValue string) error {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return fmt.Errorf("缺少表名")
	}

	db, ct, databaseName, err := s.openRemoteDB(connID, databaseName)
	if err != nil {
		return err
	}
	defer db.Close()
	if ct == "clickhouse" {
		return fmt.Errorf("暂不支持删除 ClickHouse 表数据")
	}

	colDefs := utils.GetColumnDefinitions(db, databaseName, tableName, ct)
	if len(colDefs) == 0 {
		return fmt.Errorf("未读取到表字段信息")
	}

	qualifiedTable := buildQualifiedTableName(ct, databaseName, tableName)
	whereSQL, whereArgs, err := buildSingleColumnFilter(ct, colDefs, filterColumn, filterValue)
	if err != nil {
		return err
	}
	rowColumns, rowValues, err := queryTableRecordValues(db, ct, qualifiedTable, offset, whereSQL, whereArgs)
	if err != nil {
		return err
	}
	if len(rowColumns) == 0 {
		return fmt.Errorf("未找到要删除的记录")
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
	whereParts := make([]string, 0, len(whereColumns))
	for i, col := range whereColumns {
		if whereValues[i] == nil {
			whereParts = append(whereParts, fmt.Sprintf("%s IS NULL", quoteColumnIdentifier(ct, col)))
			continue
		}
		whereParts = append(whereParts, fmt.Sprintf("%s = %s", quoteColumnIdentifier(ct, col), builder.Add(whereValues[i])))
	}
	if len(whereParts) == 0 {
		return fmt.Errorf("无法定位要删除的记录")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", qualifiedTable, strings.Join(whereParts, " AND "))
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	result, err := tx.Exec(query, builder.args...)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("执行删除失败: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("无法确认删除结果: %w", err)
	}
	if rowsAffected != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("本次删除影响了 %d 行，已取消；请确认记录唯一性后重试", rowsAffected)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交删除失败: %w", err)
	}

	return nil
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

func buildTablePageQuery(ct, qualifiedTable, whereSQL string, limit, offset int) string {
	switch {
	case ct == "oracle":
		return fmt.Sprintf("SELECT * FROM %s%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", qualifiedTable, whereSQL, offset, limit)
	case ct == "sqlserver" || ct == "mssql":
		return fmt.Sprintf("SELECT * FROM %s%s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", qualifiedTable, whereSQL, offset, limit)
	default:
		return fmt.Sprintf("SELECT * FROM %s%s LIMIT %d OFFSET %d", qualifiedTable, whereSQL, limit, offset)
	}
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
