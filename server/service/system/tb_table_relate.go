package system

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	"go.uber.org/zap"
)

type TbTableRelateService struct{}

type ImportTableRelationsResult struct {
	ProjectConfigID uint                         `json:"projectConfigId"`
	Created         int                          `json:"created"`
	Skipped         int                          `json:"skipped"`
	Failed          []ImportTableRelationFailure `json:"failed"`
	Items           []system.TbTableRelate       `json:"items"`
}

type ImportTableRelationFailure struct {
	Index    int                           `json:"index"`
	Reason   string                        `json:"reason"`
	Relation systemReq.ImportTableRelation `json:"relation"`
}

func (s *TbTableRelateService) CreateTbTableRelate(tr system.TbTableRelate) (err error) {
	normalizeTbTableRelate(&tr)
	err = global.GVA_DB.Create(&tr).Error
	return err
}

func (s *TbTableRelateService) ImportTableRelations(req systemReq.ImportTableRelationsRequest, defaultUser string) (ImportTableRelationsResult, error) {
	result := ImportTableRelationsResult{
		ProjectConfigID: req.ProjectConfigID,
		Failed:          []ImportTableRelationFailure{},
		Items:           []system.TbTableRelate{},
	}
	if global.GVA_DB == nil {
		return result, errors.New("数据库未初始化")
	}
	if req.ProjectConfigID == 0 {
		return result, errors.New("projectConfigId 不能为空")
	}
	if len(req.Relations) == 0 {
		return result, errors.New("relations 不能为空")
	}

	userName := strings.TrimSpace(req.UserName)
	if userName == "" {
		userName = strings.TrimSpace(defaultUser)
	}
	if userName == "" {
		userName = "ai"
	}

	for idx, relation := range req.Relations {
		record, err := buildImportedTableRelate(req.ProjectConfigID, relation, userName)
		if err != nil {
			result.Failed = append(result.Failed, ImportTableRelationFailure{
				Index:    idx,
				Reason:   err.Error(),
				Relation: relation,
			})
			continue
		}

		exists, err := s.tableRelationExists(record)
		if err != nil {
			result.Failed = append(result.Failed, ImportTableRelationFailure{
				Index:    idx,
				Reason:   err.Error(),
				Relation: relation,
			})
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		if err := global.GVA_DB.Create(&record).Error; err != nil {
			result.Failed = append(result.Failed, ImportTableRelationFailure{
				Index:    idx,
				Reason:   err.Error(),
				Relation: relation,
			})
			continue
		}
		result.Created++
		result.Items = append(result.Items, record)
	}

	return result, nil
}

func (s *TbTableRelateService) DeleteTbTableRelate(tr system.TbTableRelate) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&tr).Error
	return err
}

func (s *TbTableRelateService) DeleteTbTableRelateByIds(ids []int) (err error) {
	err = global.GVA_DB.Unscoped().Delete(&[]system.TbTableRelate{}, "id in ?", ids).Error
	return err
}

func (s *TbTableRelateService) UpdateTbTableRelate(tr *system.TbTableRelate) (err error) {
	normalizeTbTableRelate(tr)
	err = global.GVA_DB.Updates(tr).Error
	return err
}

func (s *TbTableRelateService) GetTbTableRelate(id uint) (tr system.TbTableRelate, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&tr).Error
	return
}

func (s *TbTableRelateService) GetTbTableRelateInfoList(info systemReq.TableRelateSearch) (list []system.TbTableRelate, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Model(&system.TbTableRelate{})
	if info.ProjectConfigID != 0 {
		db = db.Where("project_config_id = ?", info.ProjectConfigID)
	}
	if info.TbName != "" {
		db = db.Where("table_name = ?", info.TbName)
	}
	if info.RelateTableName != "" {
		db = db.Where("relate_table_name = ?", info.RelateTableName)
	}

	err = db.Count(&total).Error
	if err != nil {
		return list, total, err
	}
	err = db.Limit(limit).Offset(offset).Find(&list).Error
	return list, total, err
}

func projectConfigGroupFromQuery(query systemReq.ClientQueryModel) string {
	if query.ProjectConfigID != 0 {
		return strconv.FormatUint(uint64(query.ProjectConfigID), 10)
	}
	return strings.TrimSpace(query.ConnectionGroup)
}

func projectConfigIDFromQuery(query systemReq.ClientQueryModel) uint {
	if query.ProjectConfigID != 0 {
		return query.ProjectConfigID
	}
	group := strings.TrimSpace(query.ConnectionGroup)
	if group == "" {
		return 0
	}
	id, err := strconv.ParseUint(group, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

func normalizeTbTableRelate(tr *system.TbTableRelate) {
	if tr == nil {
		return
	}
	tr.DatabaseName, tr.TbName = normalizeDbTableFields(tr.DatabaseName, tr.TbName)
	tr.RelateDatabaseName, tr.RelateTableName = normalizeDbTableFields(tr.RelateDatabaseName, tr.RelateTableName)
}

func buildImportedTableRelate(projectConfigID uint, relation systemReq.ImportTableRelation, userName string) (system.TbTableRelate, error) {
	record := system.TbTableRelate{
		ProjectConfigID:    projectConfigID,
		DatabaseName:       strings.TrimSpace(relation.Source.DatabaseName),
		TbName:             strings.TrimSpace(relation.Source.TableName),
		ColumnName:         strings.TrimSpace(relation.Source.ColumnName),
		ColumnType:         strings.TrimSpace(relation.Source.ColumnType),
		RelateDatabaseName: strings.TrimSpace(relation.Target.DatabaseName),
		RelateTableName:    strings.TrimSpace(relation.Target.TableName),
		RelateColumnName:   strings.TrimSpace(relation.Target.ColumnName),
		RelateColumnType:   strings.TrimSpace(relation.Target.ColumnType),
		UserName:           userName,
	}
	normalizeTbTableRelate(&record)

	switch {
	case record.DatabaseName == "":
		return system.TbTableRelate{}, errors.New("source.databaseName 不能为空")
	case record.TbName == "":
		return system.TbTableRelate{}, errors.New("source.tableName 不能为空")
	case record.ColumnName == "":
		return system.TbTableRelate{}, errors.New("source.columnName 不能为空")
	case record.RelateDatabaseName == "":
		return system.TbTableRelate{}, errors.New("target.databaseName 不能为空")
	case record.RelateTableName == "":
		return system.TbTableRelate{}, errors.New("target.tableName 不能为空")
	case record.RelateColumnName == "":
		return system.TbTableRelate{}, errors.New("target.columnName 不能为空")
	}

	return record, nil
}

func (s *TbTableRelateService) tableRelationExists(record system.TbTableRelate) (bool, error) {
	var total int64
	err := global.GVA_DB.Model(&system.TbTableRelate{}).
		Where("project_config_id = ?", record.ProjectConfigID).
		Where("database_name = ?", record.DatabaseName).
		Where("table_name = ?", record.TbName).
		Where("column_name = ?", record.ColumnName).
		Where("relate_database_name = ?", record.RelateDatabaseName).
		Where("relate_table_name = ?", record.RelateTableName).
		Where("relate_column_name = ?", record.RelateColumnName).
		Count(&total).Error
	return total > 0, err
}

func normalizeDbTableFields(dbName, tableName string) (string, string) {
	dbName = strings.TrimSpace(dbName)
	tableName = strings.TrimSpace(tableName)
	if !strings.Contains(tableName, ":") {
		return dbName, tableName
	}
	parts := strings.SplitN(tableName, ":", 2)
	if dbName == "" || strings.EqualFold(dbName, "defaultDb") {
		dbName = strings.TrimSpace(parts[0])
	}
	tableName = strings.TrimSpace(parts[1])
	return dbName, tableName
}

type relationLookupTarget struct {
	DatabaseName string
	TableName    string
	LookupColumn string
	ValueColumn  string
}

func resolveRelationLookupTarget(relate system.TbTableRelate, currentDb, currentTable string) (relationLookupTarget, bool) {
	normalizeTbTableRelate(&relate)
	if equalFoldAny(relate.DatabaseName, currentDb) && equalFoldAny(relate.TbName, currentTable) {
		return relationLookupTarget{
			DatabaseName: relate.RelateDatabaseName,
			TableName:    relate.RelateTableName,
			LookupColumn: relate.RelateColumnName,
			ValueColumn:  relate.ColumnName,
		}, true
	}
	if equalFoldAny(relate.RelateDatabaseName, currentDb) && equalFoldAny(relate.RelateTableName, currentTable) {
		return relationLookupTarget{
			DatabaseName: relate.DatabaseName,
			TableName:    relate.TbName,
			LookupColumn: relate.ColumnName,
			ValueColumn:  relate.RelateColumnName,
		}, true
	}
	return relationLookupTarget{}, false
}

func (s *TbTableRelateService) GetClientData(query systemReq.ClientQueryModel) ([]*utils.ClientDatabaseVO, error) {
	parts := strings.SplitN(query.DatabaseStr, ":", 2)
	if len(parts) < 2 {
		return nil, nil
	}
	dbName := strings.TrimSpace(parts[0])
	tableName := strings.TrimSpace(parts[1])

	// Load all connections for this environment and project config.
	var conns []system.TbConnection
	connDB := global.GVA_DB.Model(&system.TbConnection{}).Where("env_name = ?", query.Environment)
	if query.ConnectionID != 0 {
		connDB = connDB.Where("id = ?", query.ConnectionID)
	}
	if projectConfigGroup := projectConfigGroupFromQuery(query); projectConfigGroup != "" {
		connDB = connDB.Where("connection_group = ?", projectConfigGroup)
	}
	connDB.Find(&conns)

	// Helper: resolve connection for a target database name using the same multi-level matching
	// logic as GetRemoteTableColumns (exact match → fallback by type).
	resolveConn := func(targetDb string) (*system.TbConnection, string) {
		return resolveRemoteConnection(conns, targetDb)
	}

	// Helper: build sourceDb array from a connection, overriding DatabaseName for
	// databases that require a separate connection per database (PostgreSQL, SQLServer).
	connToArr := func(c *system.TbConnection, overrideDb string) []string {
		dbForConn := c.DatabaseName
		ct := strings.ToLower(strings.TrimSpace(c.ConnectionType))
		needsNewConn := ct == "postgresql" || ct == "pgsql" || ct == "sqlserver" || ct == "mssql"
		if needsNewConn && overrideDb != "" && overrideDb != c.DatabaseName {
			dbForConn = overrideDb
		}
		return []string{c.ConnectionUrl, c.DbLoginName, c.DbLoginPassword, strconv.Itoa(c.Port), dbForConn}
	}

	result := []*utils.ClientDatabaseVO{}

	// --- Step 1: Always query the source table first ---
	sourceConn, resolvedDb := resolveConn(dbName)
	if sourceConn == nil {
		fmt.Printf("[GetClientData] no connection found for db=%s env=%s\n", dbName, query.Environment)
		return result, nil
	}
	mainRelate := system.TbTableRelate{
		DatabaseName: resolvedDb,
		TbName:       tableName,
		ColumnName:   "id",
	}
	sourceVO := utils.QueryDatabaseInfo(connToArr(sourceConn, resolvedDb), mainRelate, query.Value, sourceConn.ConnectionType)
	if sourceVO == nil || len(sourceVO.ColumnList) == 0 {
		return result, nil
	}
	// Build a value map from source table columns (used to chain into related tables)
	firstQueryValues := make(map[string]string)
	for _, col := range sourceVO.ColumnList {
		firstQueryValues[col.Name] = col.Value
	}
	result = append(result, sourceVO)

	// --- Step 2: Query configured relations ---
	var relates []system.TbTableRelate
	relateDB := global.GVA_DB.Model(&system.TbTableRelate{}).
		Where(
			"(database_name = ? AND table_name = ?) OR (relate_database_name = ? AND relate_table_name = ?)",
			dbName, tableName, dbName, tableName,
		)
	if projectConfigID := projectConfigIDFromQuery(query); projectConfigID != 0 {
		relateDB = relateDB.Where("project_config_id = ?", projectConfigID)
	}
	relateDB.Find(&relates)

	// Track tables already included in the result to avoid duplicates
	seenTables := map[string]bool{
		strings.ToUpper(dbName) + ":" + strings.ToUpper(tableName): true, // source table
	}

	for _, relate := range relates {
		target, ok := resolveRelationLookupTarget(relate, dbName, tableName)
		if !ok || target.TableName == "" || target.LookupColumn == "" {
			continue
		}
		if target.DatabaseName == "" || strings.EqualFold(target.DatabaseName, "defaultDb") {
			target.DatabaseName = dbName
		}

		// Skip if it points back to the source table itself
		if equalFoldAny(target.DatabaseName, dbName) && equalFoldAny(target.TableName, tableName) {
			continue
		}

		// Deduplicate: skip if we've already queried this db:table combination
		tableKey := strings.ToUpper(target.DatabaseName) + ":" + strings.ToUpper(target.TableName)
		if seenTables[tableKey] {
			continue
		}
		seenTables[tableKey] = true

		relConn, resolvedRelDb := resolveConn(target.DatabaseName)
		if relConn == nil {
			fmt.Printf("[GetClientData] no connection for relate target db=%s\n", target.DatabaseName)
			continue
		}

		queryVal := query.Value
		if v, ok := firstQueryValues[target.ValueColumn]; ok && v != "" {
			queryVal = v
		}

		// Build a synthetic relate pointing at the target table
		targetRelate := system.TbTableRelate{
			DatabaseName: resolvedRelDb,
			TbName:       target.TableName,
			ColumnName:   target.LookupColumn,
		}

		vo := utils.QueryDatabaseInfo(connToArr(relConn, resolvedRelDb), targetRelate, queryVal, relConn.ConnectionType)
		if vo != nil && len(vo.ColumnList) > 0 {
			result = append(result, vo)
		}
	}

	// Insert Prefer (best-effort, ignore error)
	prefer := system.TbTablePrefer{
		ProjectConfigID: query.ProjectConfigID,
		ConnectionID:    query.ConnectionID,
		DatabaseName:    dbName,
		TbName:          tableName,
		ColumnValue:     query.Value,
		UserName:        query.UserName,
	}
	global.GVA_DB.Create(&prefer)

	return result, nil
}

func (s *TbTableRelateService) GetRemoteTableColumns(query systemReq.ClientQueryModel) ([]utils.ClientColumnVO, error) {
	parts := strings.SplitN(query.DatabaseStr, ":", 2)
	if len(parts) < 2 {
		return []utils.ClientColumnVO{}, nil
	}
	dbName := strings.TrimSpace(parts[0])
	tableName := strings.TrimSpace(parts[1])

	var conns []system.TbConnection
	connDB := global.GVA_DB.Model(&system.TbConnection{}).Where("env_name = ?", query.Environment)
	if query.ConnectionID != 0 {
		connDB = connDB.Where("id = ?", query.ConnectionID)
	}
	if projectConfigGroup := projectConfigGroupFromQuery(query); projectConfigGroup != "" {
		connDB = connDB.Where("connection_group = ?", projectConfigGroup)
	}
	connDB.Find(&conns)

	targetConn, metadataSchema := resolveRemoteConnection(conns, dbName)
	if targetConn == nil {
		global.GVA_LOG.Warn("remote table columns connection not found", zap.String("envName", query.Environment), zap.String("database", dbName), zap.String("table", tableName))
		return []utils.ClientColumnVO{}, nil
	}

	// For PostgreSQL: each database requires a separate connection.
	// When using a fallback connection, reconnect to the requested database
	// using the same credentials (same host/port/user/password, different dbname).
	// This mirrors how Navicat handles multi-database PostgreSQL servers.
	connForDSN := *targetConn
	ct := strings.ToLower(strings.TrimSpace(targetConn.ConnectionType))
	if (ct == "postgresql" || ct == "pgsql") && connForDSN.DatabaseName != metadataSchema && metadataSchema != "" {
		connForDSN.DatabaseName = metadataSchema
	}

	dsn, driverName := buildDSN(connForDSN)
	if driverName == "" {
		return []utils.ClientColumnVO{}, nil
	}

	db, err := openRemoteSQLDB(driverName, dsn)
	if err != nil {
		global.GVA_LOG.Error("Dynamic connect failed", zap.Error(formatRemoteDBConnectionError(connForDSN, err)))
		return []utils.ClientColumnVO{}, nil
	}
	defer db.Close()

	// For PostgreSQL cross-database: dbName is the actual database (already connected),
	// pass empty string as schema so GetColumnDefinitions uses the default schema search path.
	schemaForQuery := metadataSchema
	if ct == "postgresql" || ct == "pgsql" {
		// metadataSchema is now the database name; within the connection, use "public" as default schema
		// unless the user specified a schema explicitly (detected if original dbName != targetConn.DatabaseName)
		if connForDSN.DatabaseName == metadataSchema {
			// We connected to the right database; clear schemaForQuery so all schemas are searched
			schemaForQuery = ""
		}
	}

	cols := utils.GetColumnDefinitions(db, schemaForQuery, tableName, targetConn.ConnectionType)
	if len(cols) == 0 {
		global.GVA_LOG.Warn("remote table columns empty", zap.String("envName", query.Environment), zap.String("database", metadataSchema), zap.String("table", tableName), zap.String("connectionType", targetConn.ConnectionType))
	}
	return cols, nil
}

func resolveRemoteConnection(conns []system.TbConnection, requestedNamespace string) (*system.TbConnection, string) {
	requestedNamespace = strings.TrimSpace(requestedNamespace)
	// 1. Exact match on DatabaseName
	for i := range conns {
		c := &conns[i]
		if equalFoldAny(requestedNamespace, c.DatabaseName) {
			return c, requestedNamespace
		}
	}
	// 2. Match on DbLoginName (schema = login user, common in Oracle/PG)
	for i := range conns {
		c := &conns[i]
		if equalFoldAny(requestedNamespace, c.DbLoginName) {
			return c, requestedNamespace
		}
	}
	// 3. Match on ConnectionName
	for i := range conns {
		c := &conns[i]
		if equalFoldAny(requestedNamespace, c.ConnectionName) {
			return c, requestedNamespace
		}
	}
	// 4. MySQL cross-database fallback: MySQL allows `dbName`.`tableName` cross-database syntax.
	for i := range conns {
		c := &conns[i]
		if strings.EqualFold(strings.TrimSpace(c.ConnectionType), "mysql") {
			return c, requestedNamespace
		}
	}
	// 5. PostgreSQL cross-schema fallback: treat requestedNamespace as a schema name within the connected database.
	for i := range conns {
		c := &conns[i]
		ct := strings.ToLower(strings.TrimSpace(c.ConnectionType))
		if ct == "postgresql" || ct == "pgsql" {
			return c, requestedNamespace
		}
	}
	// 6. SQLServer cross-database fallback: SQLServer allows [dbName].[dbo].[table] cross-database syntax.
	for i := range conns {
		c := &conns[i]
		ct := strings.ToLower(strings.TrimSpace(c.ConnectionType))
		if ct == "sqlserver" || ct == "mssql" {
			return c, requestedNamespace
		}
	}
	// 7. Oracle cross-schema fallback: treat requestedNamespace as an Oracle owner/schema.
	for i := range conns {
		c := &conns[i]
		if strings.EqualFold(strings.TrimSpace(c.ConnectionType), "oracle") {
			return c, requestedNamespace
		}
	}
	return nil, requestedNamespace
}

func equalFoldAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

// GetTableComments returns a map of "db:table" -> comment for the given list of db:table pairs.
func (s *TbTableRelateService) GetTableComments(projectConfigID uint, environment string, connectionID uint, tables []string) map[string]string {
	result := make(map[string]string)
	if len(tables) == 0 {
		return result
	}

	// Load connections
	var conns []system.TbConnection
	connDB := global.GVA_DB.Model(&system.TbConnection{}).Where("env_name = ?", environment)
	if connectionID != 0 {
		connDB = connDB.Where("id = ?", connectionID)
	}
	if projectConfigID != 0 {
		connDB = connDB.Where("connection_group = ?", strconv.FormatUint(uint64(projectConfigID), 10))
	}
	connDB.Find(&conns)
	if len(conns) == 0 {
		return result
	}

	// Group tables by resolved connection to minimize DB connections
	type tableEntry struct {
		dbName    string
		tableName string
		key       string // original "db:table" key
	}
	type connGroup struct {
		conn       *system.TbConnection
		resolvedDb string
		entries    []tableEntry
	}
	groups := make(map[uint]*connGroup) // keyed by connection ID

	for _, t := range tables {
		parts := strings.SplitN(t, ":", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		dbName, tableName := parts[0], parts[1]
		conn, resolvedDb := resolveRemoteConnection(conns, dbName)
		if conn == nil {
			continue
		}
		g, ok := groups[conn.ID]
		if !ok {
			g = &connGroup{conn: conn, resolvedDb: resolvedDb}
			groups[conn.ID] = g
		}
		g.entries = append(g.entries, tableEntry{dbName: resolvedDb, tableName: tableName, key: t})
	}

	// Query each connection group
	for _, g := range groups {
		ct := strings.ToLower(strings.TrimSpace(g.conn.ConnectionType))

		// For Oracle, don't change DSN database name
		connForDSN := *g.conn
		if ct != "oracle" && g.resolvedDb != "" && g.resolvedDb != g.conn.DatabaseName {
			connForDSN.DatabaseName = g.resolvedDb
		}

		dsn, driverName := buildDSN(connForDSN)
		if driverName == "" {
			continue
		}

		db, err := openRemoteSQLDB(driverName, dsn)
		if err != nil {
			global.GVA_LOG.Error("remote table comments connect failed", zap.Error(formatRemoteDBConnectionError(connForDSN, err)))
			continue
		}

		for _, entry := range g.entries {
			comment := utils.GetTableComment(db, entry.dbName, entry.tableName, ct)
			if comment != "" {
				result[entry.key] = comment
			}
		}
		db.Close()
	}

	return result
}
