package utils

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)

// ClientColumnVO represents column data
type ClientColumnVO struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Value         string `json:"value"`
	ColumnType    string `json:"columnType"`
	Length        string `json:"length"`
	NotNull       bool   `json:"notNull"`
	HasDefault    bool   `json:"hasDefault"`
	DefaultValue  string `json:"defaultValue,omitempty"`
	AutoIncrement bool   `json:"autoIncrement"`
}

// ClientDatabaseVO represents table metadata and column data
type ClientDatabaseVO struct {
	TableAnnotation string           `json:"tableAnnotation"`
	DatabaseName    string           `json:"databaseName"`
	TableName       string           `json:"tableName"`
	RecordCount     int64            `json:"recordCount"`
	ColumnList      []ClientColumnVO `json:"columnList"`
}

// sourceDb: [url, username, password, port, db_name]
func QueryDatabaseInfo(sourceDb []string, relate system.TbTableRelate, columnValue string, connType string) *ClientDatabaseVO {
	if len(sourceDb) < 5 {
		return &ClientDatabaseVO{}
	}

	host := sourceDb[0]
	user := sourceDb[1]
	password := sourceDb[2]
	port := sourceDb[3]
	dbName := sourceDb[4]

	tableName := relate.TbName
	mainColumn := relate.ColumnName
	metadataNamespace := queryNamespaceForRelate(dbName, relate)

	var db *sql.DB
	var err error

	ct := strings.ToLower(connType)
	if ct == "mysql" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbName)
		db, err = sql.Open("mysql", dsn)
	} else if ct == "postgresql" || ct == "pgsql" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai", host, user, password, dbName, port)
		db, err = sql.Open("postgres", dsn)
	} else if ct == "sqlserver" || ct == "mssql" {
		u := &url.URL{
			Scheme: "sqlserver",
			User:   url.UserPassword(user, password),
			Host:   fmt.Sprintf("%s:%s", host, port),
		}
		q := u.Query()
		q.Set("database", dbName)
		u.RawQuery = q.Encode()
		db, err = sql.Open("sqlserver", u.String())
	} else if ct == "oracle" {
		portInt, _ := strconv.Atoi(port)
		dsn := BuildOracleDSN(host, portInt, dbName, user, password)
		db, err = sql.Open("oracle", dsn)
	} else if ct == "sqlite" {
		db, err = sql.Open("sqlite", dbName)
	} else if ct == "clickhouse" {
		portInt, _ := strconv.Atoi(port)
		db, err = sql.Open("clickhouse", buildClickHouseDSN(host, portInt, dbName, user, password))
	} else {
		return &ClientDatabaseVO{}
	}

	if err != nil {
		fmt.Printf("Dynamic connect failed: %v\n", err)
		return &ClientDatabaseVO{}
	}
	defer db.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), RemoteDBConnectionTimeout)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		fmt.Printf("Dynamic ping failed: %v\n", err)
		return &ClientDatabaseVO{}
	}

	tableComment := GetTableComment(db, metadataNamespace, tableName, ct)
	primaryKey := getPrimaryKey(db, metadataNamespace, tableName, ct)

	columns := GetColumnDefinitions(db, metadataNamespace, tableName, ct)
	colValues := getRecordValues(db, metadataNamespace, tableName, mainColumn, primaryKey, columnValue, ct)
	recordCount := getRecordCount(db, metadataNamespace, tableName, mainColumn, primaryKey, columnValue, ct)

	resultList := []ClientColumnVO{}
	for _, col := range columns {
		valStr := ""
		if v, ok := colValues[col.Name]; ok {
			valStr = v
		}
		resultList = append(resultList, ClientColumnVO{
			Name:        col.Name,
			Description: col.Description,
			Value:       valStr,
		})
	}

	return &ClientDatabaseVO{
		TableAnnotation: tableComment,
		DatabaseName:    metadataNamespace,
		TableName:       tableName,
		RecordCount:     recordCount,
		ColumnList:      resultList,
	}
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

func queryNamespaceForRelate(connectionDatabase string, relate system.TbTableRelate) string {
	if namespace := strings.TrimSpace(relate.DatabaseName); namespace != "" && namespace != "defaultDb" {
		return namespace
	}
	return strings.TrimSpace(connectionDatabase)
}

func GetTableComment(db *sql.DB, dbName, tableName, ct string) string {
	var comment sql.NullString

	if ct == "mysql" {
		// MySQL: always use dbName to support cross-database queries
		db.QueryRow("SELECT TABLE_COMMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", dbName, tableName).Scan(&comment)
	} else if ct == "postgresql" || ct == "pgsql" {
		// PostgreSQL: the DSN is already connected to the correct database.
		// Always use the 'public' schema.
		db.QueryRow("SELECT obj_description(pg_class.oid) FROM pg_class JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace WHERE pg_namespace.nspname = 'public' AND pg_class.relname = $1", tableName).Scan(&comment)
	} else if ct == "sqlserver" || ct == "mssql" {
		// SQLServer: qualify with database name for cross-database support
		if dbName != "" {
			query := fmt.Sprintf("SELECT CAST(p.value AS VARCHAR(MAX)) FROM [%s].sys.tables t LEFT JOIN [%s].sys.extended_properties p ON p.major_id = t.object_id AND p.minor_id = 0 AND p.name = 'MS_Description' WHERE t.name = @p1", dbName, dbName)
			db.QueryRow(query, tableName).Scan(&comment)
		} else {
			db.QueryRow("SELECT CAST(p.value AS VARCHAR(MAX)) FROM sys.tables t LEFT JOIN sys.extended_properties p ON p.major_id = t.object_id AND p.minor_id = 0 AND p.name = 'MS_Description' WHERE t.name = @p1", tableName).Scan(&comment)
		}
	} else if ct == "oracle" {
		// Oracle: use dbName as owner/schema for cross-schema queries
		if dbName != "" {
			db.QueryRow("SELECT COMMENTS FROM ALL_TAB_COMMENTS WHERE OWNER = :1 AND TABLE_NAME = :2 FETCH FIRST 1 ROWS ONLY", strings.ToUpper(dbName), strings.ToUpper(tableName)).Scan(&comment)
		} else {
			db.QueryRow("SELECT COMMENTS FROM ALL_TAB_COMMENTS WHERE TABLE_NAME = :1 AND OWNER NOT IN ('SYS', 'SYSTEM') FETCH FIRST 1 ROWS ONLY", strings.ToUpper(tableName)).Scan(&comment)
		}
	} else if ct == "sqlite" {
		return ""
	} else if ct == "clickhouse" {
		db.QueryRow("SELECT comment FROM system.tables WHERE database = ? AND name = ? LIMIT 1", dbName, tableName).Scan(&comment)
	}

	if comment.Valid {
		return comment.String
	}
	return ""
}

func getPrimaryKey(db *sql.DB, dbName, tableName, ct string) string {
	var pk sql.NullString

	if ct == "mysql" {
		// MySQL: always use dbName to support cross-database queries
		db.QueryRow("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY' LIMIT 1", dbName, tableName).Scan(&pk)
	} else if ct == "postgresql" || ct == "pgsql" {
		// PostgreSQL: the DSN is already connected to the correct database.
		// Always use the 'public' schema.
		db.QueryRow("SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) JOIN pg_class c ON c.oid = i.indrelid JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relname = $1 AND i.indisprimary", tableName).Scan(&pk)
	} else if ct == "sqlserver" || ct == "mssql" {
		// SQLServer: qualify with database name for cross-database support
		if dbName != "" {
			query := fmt.Sprintf("SELECT top 1 c.name FROM [%s].sys.indexes i JOIN [%s].sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id JOIN [%s].sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id WHERE i.is_primary_key = 1 AND i.object_id = OBJECT_ID(@p1)", dbName, dbName, dbName)
			db.QueryRow(query, fmt.Sprintf("[%s].[dbo].[%s]", dbName, tableName)).Scan(&pk)
		} else {
			db.QueryRow("SELECT top 1 c.name FROM sys.indexes i JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id WHERE i.is_primary_key = 1 AND i.object_id = OBJECT_ID(@p1)", tableName).Scan(&pk)
		}
	} else if ct == "oracle" {
		// Oracle: use dbName as owner for cross-schema queries
		if dbName != "" {
			db.QueryRow("SELECT cols.column_name FROM all_constraints cons, all_cons_columns cols WHERE cols.owner = :1 AND cols.table_name = :2 AND cons.constraint_type = 'P' AND cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner FETCH FIRST 1 ROWS ONLY", strings.ToUpper(dbName), strings.ToUpper(tableName)).Scan(&pk)
		} else {
			db.QueryRow("SELECT cols.column_name FROM all_constraints cons, all_cons_columns cols WHERE cols.table_name = :1 AND cons.constraint_type = 'P' AND cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner FETCH FIRST 1 ROWS ONLY", strings.ToUpper(tableName)).Scan(&pk)
		}
	} else if ct == "sqlite" {
		rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cid int
				var name, typ string
				var notnull sql.NullInt64
				var dflt sql.NullString
				var ispk int
				if rows.Scan(&cid, &name, &typ, &notnull, &dflt, &ispk) == nil {
					if ispk == 1 {
						return name
					}
				}
			}
		}
	} else if ct == "clickhouse" {
		return ""
	}

	if pk.Valid {
		return pk.String
	}
	return ""
}

func GetColumnDefinitions(db *sql.DB, dbName, tableName, ct string) []ClientColumnVO {
	cols := []ClientColumnVO{}
	ct = strings.ToLower(ct)

	if ct == "mysql" {
		rows, err := db.Query("SHOW FULL COLUMNS FROM `" + dbName + "`.`" + tableName + "`")
		if err != nil {
			fmt.Printf("mysql GetColumnDefinitions err: %v\n", err)
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var field, typ, collation, null, key, defaultVal, extra, privileges, comment sql.NullString
			if err = rows.Scan(&field, &typ, &collation, &null, &key, &defaultVal, &extra, &privileges, &comment); err == nil {
				colType := typ.String
				length := "-"
				if idx := strings.Index(colType, "("); idx != -1 {
					if endIdx := strings.Index(colType, ")"); endIdx != -1 {
						length = colType[idx+1 : endIdx]
						colType = colType[:idx]
					}
				}
				defaultText := strings.TrimSpace(defaultVal.String)
				cols = append(cols, ClientColumnVO{
					Name:          field.String,
					Description:   comment.String,
					ColumnType:    colType,
					Length:        length,
					NotNull:       strings.EqualFold(null.String, "NO"),
					HasDefault:    defaultVal.Valid,
					DefaultValue:  defaultText,
					AutoIncrement: strings.Contains(strings.ToLower(extra.String), "auto_increment"),
				})
			}
		}
	} else if ct == "postgresql" || ct == "pgsql" {
		// PostgreSQL: the DSN is already connected to the correct database.
		// Always use the 'public' schema (standard default schema for user tables).
		query := `SELECT c.column_name, pgd.description, c.data_type, c.character_maximum_length, c.numeric_precision, c.numeric_scale,
       c.is_nullable, c.column_default, c.is_identity
FROM information_schema.columns c
LEFT JOIN pg_catalog.pg_statio_all_tables st ON c.table_schema = st.schemaname AND c.table_name = st.relname
LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
WHERE c.table_name = $1 AND c.table_schema = 'public'`
		rows, err := db.Query(query, tableName)
		if err != nil {
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var field, comment, dataType, nullable, defaultVal, isIdentity sql.NullString
			var maxLen, precision, scale sql.NullInt64
			if err = rows.Scan(&field, &comment, &dataType, &maxLen, &precision, &scale, &nullable, &defaultVal, &isIdentity); err == nil {
				length := "-"
				if maxLen.Valid {
					length = strconv.FormatInt(maxLen.Int64, 10)
				} else if precision.Valid && scale.Valid {
					length = fmt.Sprintf("%d,%d", precision.Int64, scale.Int64)
				} else if precision.Valid {
					length = strconv.FormatInt(precision.Int64, 10)
				}
				defaultText := strings.TrimSpace(defaultVal.String)
				cols = append(cols, ClientColumnVO{
					Name:          field.String,
					Description:   comment.String,
					ColumnType:    dataType.String,
					Length:        length,
					NotNull:       strings.EqualFold(nullable.String, "NO"),
					HasDefault:    defaultVal.Valid && defaultText != "",
					DefaultValue:  defaultText,
					AutoIncrement: strings.EqualFold(isIdentity.String, "YES") || strings.Contains(strings.ToLower(defaultText), "nextval("),
				})
			}
		}
	} else if ct == "sqlserver" || ct == "mssql" {
		// SQLServer: use fully qualified [dbName].[dbo].[tableName] for cross-database queries
		var tableRef string
		if dbName != "" {
			tableRef = fmt.Sprintf("[%s].[dbo].[%s]", dbName, tableName)
		} else {
			tableRef = tableName
		}
		query := fmt.Sprintf(`SELECT c.name, t.name as type_name, c.max_length, CAST(p.value AS VARCHAR(MAX)), c.is_nullable, dc.definition, c.is_identity FROM [%s].sys.columns c
JOIN [%s].sys.types t ON c.user_type_id = t.user_type_id
LEFT JOIN [%s].sys.extended_properties p ON p.major_id = c.object_id AND p.minor_id = c.column_id AND p.name = 'MS_Description'
LEFT JOIN [%s].sys.default_constraints dc ON c.default_object_id = dc.object_id
WHERE c.object_id = OBJECT_ID(@p1)`,
			dbName, dbName, dbName, dbName)
		rows, err := db.Query(query, tableRef)
		if err != nil {
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var field, typeName sql.NullString
			var maxLen sql.NullInt64
			var comment sql.NullString
			var nullable, identity bool
			var defaultVal sql.NullString
			if err = rows.Scan(&field, &typeName, &maxLen, &comment, &nullable, &defaultVal, &identity); err == nil {
				length := "-"
				if maxLen.Valid {
					length = strconv.FormatInt(maxLen.Int64, 10)
				}
				defaultText := strings.TrimSpace(defaultVal.String)
				cols = append(cols, ClientColumnVO{
					Name:          field.String,
					Description:   comment.String,
					ColumnType:    typeName.String,
					Length:        length,
					NotNull:       !nullable,
					HasDefault:    defaultVal.Valid && defaultText != "",
					DefaultValue:  defaultText,
					AutoIncrement: identity,
				})
			}
		}
	} else if ct == "oracle" {
		query, args := OracleColumnDefinitionsQuery(dbName, tableName)
		rows, err := db.Query(query, args...)
		if err != nil {
			fmt.Printf("oracle GetColumnDefinitions err: %v\n", err)
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var field, dataType, comment, nullable, defaultVal sql.NullString
			var maxLen sql.NullInt64
			if err = rows.Scan(&field, &dataType, &maxLen, &comment, &nullable, &defaultVal); err == nil {
				length := "-"
				if maxLen.Valid {
					length = strconv.FormatInt(maxLen.Int64, 10)
				}
				defaultText := strings.TrimSpace(defaultVal.String)
				cols = append(cols, ClientColumnVO{
					Name:         field.String,
					Description:  comment.String,
					ColumnType:   dataType.String,
					Length:       length,
					NotNull:      strings.EqualFold(nullable.String, "N"),
					HasDefault:   defaultVal.Valid && defaultText != "",
					DefaultValue: defaultText,
				})
			}
		}
	} else if ct == "sqlite" {
		rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
		if err != nil {
			fmt.Printf("sqlite GetColumnDefinitions err: %v\n", err)
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, typ string
			var notnull sql.NullInt64
			var dflt sql.NullString
			var ispk int
			if err = rows.Scan(&cid, &name, &typ, &notnull, &dflt, &ispk); err == nil {
				colType := typ
				length := "-"
				if idx := strings.Index(colType, "("); idx != -1 {
					if endIdx := strings.Index(colType, ")"); endIdx != -1 {
						length = colType[idx+1 : endIdx]
						colType = colType[:idx]
					}
				}
				defaultText := strings.TrimSpace(dflt.String)
				cols = append(cols, ClientColumnVO{
					Name:          name,
					Description:   "",
					ColumnType:    colType,
					Length:        length,
					NotNull:       notnull.Valid && notnull.Int64 == 1,
					HasDefault:    dflt.Valid && defaultText != "",
					DefaultValue:  defaultText,
					AutoIncrement: ispk > 0 && strings.Contains(strings.ToUpper(colType), "INT"),
				})
			}
		}
	} else if ct == "clickhouse" {
		rows, err := db.Query("SELECT name, type, comment FROM system.columns WHERE database = ? AND table = ? ORDER BY position", dbName, tableName)
		if err != nil {
			fmt.Printf("clickhouse GetColumnDefinitions err: %v\n", err)
			return cols
		}
		defer rows.Close()

		for rows.Next() {
			var field, dataType, comment sql.NullString
			if err = rows.Scan(&field, &dataType, &comment); err == nil {
				cols = append(cols, ClientColumnVO{Name: field.String, Description: comment.String, ColumnType: dataType.String, Length: "-"})
			}
		}
	}
	return cols
}

func OracleColumnDefinitionsQuery(owner, tableName string) (string, []any) {
	baseQuery := `SELECT c.COLUMN_NAME, c.DATA_TYPE, c.DATA_LENGTH, cc.COMMENTS, c.NULLABLE, c.DATA_DEFAULT FROM ALL_TAB_COLUMNS c 
LEFT JOIN ALL_COL_COMMENTS cc ON c.TABLE_NAME = cc.TABLE_NAME AND c.COLUMN_NAME = cc.COLUMN_NAME AND c.OWNER = cc.OWNER`
	owner = strings.ToUpper(strings.TrimSpace(owner))
	tableName = strings.ToUpper(strings.TrimSpace(tableName))
	if owner == "" {
		return baseQuery + `
WHERE c.TABLE_NAME = :1 AND c.OWNER NOT IN ('SYS', 'SYSTEM') ORDER BY c.COLUMN_ID`, []any{tableName}
	}
	return baseQuery + `
WHERE c.OWNER = :1 AND c.TABLE_NAME = :2 ORDER BY c.COLUMN_ID`, []any{owner, tableName}
}

func getRecordValues(db *sql.DB, dbName, tableName, mainColumn, primaryKey, columnValue, ct string) map[string]string {
	result := make(map[string]string)
	var rows *sql.Rows
	var err error

	isPg := ct == "postgresql" || ct == "pgsql"
	isOrcl := ct == "oracle"

	// Qualify table name with database/schema to support cross-database/schema queries.
	qualifiedTable := qualifyTableName(dbName, tableName, ct)

	if columnValue != "" && mainColumn != "" {
		isNumeric := false
		if _, errInt := strconv.ParseFloat(columnValue, 64); errInt == nil {
			isNumeric = true
		}

		if isPg {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 OR %s = $2 LIMIT 1", qualifiedTable, primaryKey, mainColumn)
				rows, err = db.Query(query, columnValue, columnValue)
			} else {
				query := fmt.Sprintf("SELECT * FROM %s WHERE %s::text = $1 LIMIT 1", qualifiedTable, mainColumn)
				rows, err = db.Query(query, columnValue)
			}
		} else if isOrcl {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT * FROM %s WHERE %s = :1 OR %s = :2 FETCH FIRST 1 ROWS ONLY", qualifiedTable, primaryKey, mainColumn)
				rows, err = db.Query(query, columnValue, columnValue)
			} else {
				query := fmt.Sprintf("SELECT * FROM %s WHERE TO_CHAR(%s) = :1 FETCH FIRST 1 ROWS ONLY", qualifiedTable, mainColumn)
				rows, err = db.Query(query, columnValue)
			}
		} else if ct == "sqlserver" || ct == "mssql" {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE %s = @p1 OR %s = @p2", qualifiedTable, primaryKey, mainColumn)
				rows, err = db.Query(query, columnValue, columnValue)
			} else {
				query := fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE CAST(%s AS VARCHAR(MAX)) = @p1", qualifiedTable, mainColumn)
				rows, err = db.Query(query, columnValue)
			}
		} else {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = ? OR `%s` = ? LIMIT 1", qualifiedTable, primaryKey, mainColumn)
				rows, err = db.Query(query, columnValue, columnValue)
			} else {
				query := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = ? LIMIT 1", qualifiedTable, mainColumn)
				rows, err = db.Query(query, columnValue)
			}
		}
	} else if columnValue != "" && primaryKey != "" {
		if isPg {
			query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 LIMIT 1", qualifiedTable, primaryKey)
			rows, err = db.Query(query, columnValue)
		} else if isOrcl {
			query := fmt.Sprintf("SELECT * FROM %s WHERE %s = :1 FETCH FIRST 1 ROWS ONLY", qualifiedTable, primaryKey)
			rows, err = db.Query(query, columnValue)
		} else if ct == "sqlserver" || ct == "mssql" {
			query := fmt.Sprintf("SELECT TOP 1 * FROM %s WHERE %s = @p1", qualifiedTable, primaryKey)
			rows, err = db.Query(query, columnValue)
		} else {
			query := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = ? LIMIT 1", qualifiedTable, primaryKey)
			rows, err = db.Query(query, columnValue)
		}
	} else {
		if isOrcl {
			query := fmt.Sprintf("SELECT * FROM %s FETCH FIRST 1 ROWS ONLY", qualifiedTable)
			rows, err = db.Query(query)
		} else if ct == "sqlserver" || ct == "mssql" {
			query := fmt.Sprintf("SELECT TOP 1 * FROM %s", qualifiedTable)
			rows, err = db.Query(query)
		} else {
			query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", qualifiedTable)
			rows, err = db.Query(query)
		}
	}

	if err != nil {
		fmt.Printf("Query record err: %v\n", err)
		return result
	}
	defer rows.Close()

	if rows.Next() {
		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)

		for i, col := range columns {
			val := values[i]
			var strVal string
			if val != nil {
				switch v := val.(type) {
				case []byte:
					strVal = string(v)
				case string:
					strVal = v
				default:
					strVal = fmt.Sprintf("%v", v)
				}
			}
			result[col] = strVal
		}
	}
	return result
}

func getRecordCount(db *sql.DB, dbName, tableName, mainColumn, primaryKey, columnValue, ct string) int64 {
	var total int64
	var err error

	isPg := ct == "postgresql" || ct == "pgsql"
	isOrcl := ct == "oracle"
	isSqlServer := ct == "sqlserver" || ct == "mssql"
	qualifiedTable := qualifyTableName(dbName, tableName, ct)

	if columnValue != "" && mainColumn != "" {
		isNumeric := false
		if _, errInt := strconv.ParseFloat(columnValue, 64); errInt == nil {
			isNumeric = true
		}

		if isPg {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1 OR %s = $2", qualifiedTable, primaryKey, mainColumn)
				err = db.QueryRow(query, columnValue, columnValue).Scan(&total)
			} else {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s::text = $1", qualifiedTable, mainColumn)
				err = db.QueryRow(query, columnValue).Scan(&total)
			}
		} else if isOrcl {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = :1 OR %s = :2", qualifiedTable, primaryKey, mainColumn)
				err = db.QueryRow(query, columnValue, columnValue).Scan(&total)
			} else {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE TO_CHAR(%s) = :1", qualifiedTable, mainColumn)
				err = db.QueryRow(query, columnValue).Scan(&total)
			}
		} else if isSqlServer {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = @p1 OR %s = @p2", qualifiedTable, primaryKey, mainColumn)
				err = db.QueryRow(query, columnValue, columnValue).Scan(&total)
			} else {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE CAST(%s AS VARCHAR(MAX)) = @p1", qualifiedTable, mainColumn)
				err = db.QueryRow(query, columnValue).Scan(&total)
			}
		} else {
			if isNumeric && primaryKey != "" {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE `%s` = ? OR `%s` = ?", qualifiedTable, primaryKey, mainColumn)
				err = db.QueryRow(query, columnValue, columnValue).Scan(&total)
			} else {
				query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE `%s` = ?", qualifiedTable, mainColumn)
				err = db.QueryRow(query, columnValue).Scan(&total)
			}
		}
	} else if columnValue != "" && primaryKey != "" {
		if isPg {
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", qualifiedTable, primaryKey)
			err = db.QueryRow(query, columnValue).Scan(&total)
		} else if isOrcl {
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = :1", qualifiedTable, primaryKey)
			err = db.QueryRow(query, columnValue).Scan(&total)
		} else if isSqlServer {
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = @p1", qualifiedTable, primaryKey)
			err = db.QueryRow(query, columnValue).Scan(&total)
		} else {
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE `%s` = ?", qualifiedTable, primaryKey)
			err = db.QueryRow(query, columnValue).Scan(&total)
		}
	} else {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qualifiedTable)
		err = db.QueryRow(query).Scan(&total)
	}

	if err != nil {
		fmt.Printf("Query count err: %v\n", err)
		return 0
	}
	return total
}

func qualifyTableName(dbName, tableName, ct string) string {
	ct = strings.ToLower(ct)
	if (ct == "mysql" || ct == "clickhouse") && dbName != "" {
		return "`" + dbName + "`.`" + tableName + "`"
	}
	if (ct == "sqlserver" || ct == "mssql") && dbName != "" {
		return fmt.Sprintf("[%s].[dbo].[%s]", dbName, tableName)
	}
	if ct == "oracle" && dbName != "" {
		return strings.ToUpper(dbName) + "." + strings.ToUpper(tableName)
	}
	if ct == "postgresql" || ct == "pgsql" {
		return "public." + tableName
	}
	return tableName
}
