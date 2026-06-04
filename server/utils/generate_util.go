package utils

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/xwb1989/sqlparser"
	"go.uber.org/zap"
)

type GenerateProjectInfo struct {
	ID             string `json:"id"`
	ModuleName     string `json:"moduleName"`
	ModuleComment  string `json:"moduleComment"`
	TableStructure string `json:"tableStructure"`
	DbType         string `json:"dbType"`
}

type ExtractedField struct {
	OriginalName     string // 原始字段 (下划线)
	JavaDataType     string // Java类型
	LengthInfo       string // 长度信息 (带括号)
	Comment          string // 注释
	JavaField        string // Java驼峰字段名 (首字母小写)
	PyType           string // Python类型
	CapitalizedField string // Java驼峰字段名 (首字母大写)
	VueDataType      string // Vue类型
	RawLengthInfo    string // 长度信息 (不带括号)
	PythonVoType     string // Python vo类型
}

func codeGenTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"contains":       strings.Contains,
		"hasSuffix":      strings.HasSuffix,
		"isCodeField":    isCodeField,
		"leafRuleName":   leafRuleName,
		"leafRulePrefix": leafRulePrefix,
	}
}

func isCodeField(field ExtractedField) bool {
	originalName := strings.ToLower(field.OriginalName)
	javaField := strings.ToLower(field.JavaField)
	comment := field.Comment
	return originalName == "code" ||
		strings.HasSuffix(originalName, "_code") ||
		strings.Contains(originalName, "_code_") ||
		strings.HasSuffix(javaField, "code") ||
		strings.Contains(comment, "编码")
}

func leafRuleName(field ExtractedField) string {
	return strings.ToUpper(field.OriginalName)
}

func leafRulePrefix(moduleName string, field ExtractedField) string {
	prefix := strings.TrimSuffix(strings.ToUpper(field.OriginalName), "_CODE")
	if prefix == "" || prefix == strings.ToUpper(field.OriginalName) {
		prefix = strings.ToUpper(moduleName)
	}
	return prefix
}

type RenderContext struct {
	ModuleName      string
	ModuleNameUpper string
	ModuleComment   string
	RawTableName    string // 原始数据库表名，例如 SCSL_EC_EXP_ORDER
	EntireTableName string // 例如 TbUser
	Table_name      string // tb_user 或者 user 如果带前缀
	TableName       string // 首字母大写 User
	TableNameLower  string // 首字母小写 user
	TABLENAME       string // 全大写 USER
	TABLE_NAME      string // 数据库部分名称的大写
	UrlTableName    string // 比如 sys/user
	VueTableName    string // 比如 sys-user
	TableComment    string
	FieldLines      []ExtractedField
	Date            string
	Placeholders    map[string]string
}

func ConvertDataType(sqlType string) string {
	t := strings.ToLower(sqlType)
	if strings.Contains(t, "bigint") {
		return "Long"
	}
	if strings.Contains(t, "int") || strings.Contains(t, "tinyint") {
		return "Integer"
	}
	if strings.Contains(t, "varchar") || strings.Contains(t, "text") {
		return "String"
	}
	if strings.Contains(t, "datetime") || strings.Contains(t, "date") {
		return "Date"
	}
	if strings.Contains(t, "decimal") {
		return "BigDecimal"
	}
	if strings.Contains(t, "jsonb") {
		return "JSONObject"
	}
	return "String"
}

func SqlToPyType(sqlType string) string {
	t := strings.ToLower(sqlType)
	if strings.Contains(t, "bigint") || strings.Contains(t, "int") || strings.Contains(t, "tinyint") {
		return "Integer"
	}
	if strings.Contains(t, "varchar") || strings.Contains(t, "text") {
		return "String"
	}
	if strings.Contains(t, "datetime") || strings.Contains(t, "date") {
		return "Date"
	}
	if strings.Contains(t, "decimal") {
		return "Numeric"
	}
	return "String"
}

func SqlToPyVoType(sqlType string) string {
	t := strings.ToLower(strings.Split(sqlType, "(")[0])
	switch t {
	case "bigint", "int", "integer", "tinyint", "smallint":
		return "int"
	case "varchar", "char", "text", "nvarchar":
		return "str"
	case "datetime", "timestamp", "date":
		return "date"
	case "decimal", "numeric":
		return "Decimal"
	case "float", "double", "real":
		return "float"
	case "boolean", "bit":
		return "bool"
	default:
		return "str"
	}
}

func ConvertVueType(sqlType string) string {
	t := strings.ToLower(sqlType)
	if strings.Contains(t, "bigint") || strings.Contains(t, "int") || strings.Contains(t, "tinyint") || strings.Contains(t, "decimal") {
		return "number"
	}
	return "string"
}

func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var result string
	for i, p := range parts {
		if i == 0 {
			result += p
		} else {
			if len(p) > 0 {
				result += strings.ToUpper(string(p[0])) + p[1:]
			}
		}
	}
	return result
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func SqlToJava(sql string, dbType string) (RenderContext, error) {
	if dbType == "" || dbType == "mysql" {
		return parseMySQLAst(sql)
	}
	return parseGenericDDL(sql, dbType)
}

func parseMySQLAst(sql string) (RenderContext, error) {
	var ctx RenderContext

	stmt, err := sqlparser.ParseStrictDDL(sql)
	if err != nil {
		return ctx, fmt.Errorf("解析 DDL 失败: %w", err)
	}

	ddl, ok := stmt.(*sqlparser.DDL)
	if !ok {
		return ctx, fmt.Errorf("SQL 不是 CREATE TABLE 结构")
	}

	entireTable := ddl.NewName.Name.String()
	entireTableName := Capitalize(SnakeToCamel(entireTable))

	firstUnderscore := strings.Index(entireTable, "_")
	tableNameRaw := ""
	TABLE_NAME := ""
	urlTableName := ""
	vueTableName := ""

	if firstUnderscore != -1 {
		tableNameRaw = entireTable[firstUnderscore+1:]
		urlTableName = strings.ReplaceAll(tableNameRaw, "_", "/")
		vueTableName = strings.ReplaceAll(tableNameRaw, "_", "-")
		TABLE_NAME = strings.ToUpper(tableNameRaw)
	}

	TableNameStr := Capitalize(SnakeToCamel(tableNameRaw))
	tableNameLower := ""
	if len(TableNameStr) > 0 {
		tableNameLower = strings.ToLower(string(TableNameStr[0])) + TableNameStr[1:]
	}
	TABLENAME_UPPER := strings.ToUpper(tableNameLower)

	tableComment := ""
	if ddl.TableSpec != nil && ddl.TableSpec.Options != "" {
		reComment := regexp.MustCompile(`(?i)COMMENT\s*=\s*'([^']+)'`)
		cMatch := reComment.FindStringSubmatch(ddl.TableSpec.Options)
		if len(cMatch) >= 2 {
			tableComment = cMatch[1]
		}
	}

	var fields []ExtractedField
	if ddl.TableSpec != nil {
		for _, col := range ddl.TableSpec.Columns {
			orig := col.Name.String()
			dbType := col.Type.Type

			rawLen := ""
			lenInfo := ""
			if col.Type.Length != nil {
				rawLen = string(col.Type.Length.Val)
				lenInfo = "(" + rawLen + ")"
			}

			comment := ""
			if col.Type.Comment != nil {
				comment = string(col.Type.Comment.Val)
			}

			javaType := ConvertDataType(dbType)
			pyType := SqlToPyType(dbType)
			voType := SqlToPyVoType(dbType)
			vueType := ConvertVueType(dbType)

			javaField := SnakeToCamel(orig)
			capField := Capitalize(javaField)

			fields = append(fields, ExtractedField{
				OriginalName:     orig,
				JavaDataType:     javaType,
				LengthInfo:       lenInfo,
				Comment:          comment,
				JavaField:        javaField,
				PyType:           pyType,
				CapitalizedField: capField,
				VueDataType:      vueType,
				RawLengthInfo:    rawLen,
				PythonVoType:     voType,
			})
		}
	}

	ctx = RenderContext{
		RawTableName:    entireTable,
		EntireTableName: entireTableName,
		Table_name:      tableNameRaw,
		TableName:       TableNameStr,
		TableNameLower:  tableNameLower,
		TABLENAME:       TABLENAME_UPPER,
		TABLE_NAME:      TABLE_NAME,
		UrlTableName:    urlTableName,
		VueTableName:    vueTableName,
		TableComment:    tableComment,
		FieldLines:      fields,
	}
	return ctx, nil
}

func parseGenericDDL(sql string, dbType string) (RenderContext, error) {
	var ctx RenderContext

	sql = strings.ReplaceAll(sql, "\r\n", "\n")

	reTable := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:[a-zA-Z0-9_\-"'\[\]]+\.)?["\[` + "`" + `]?([a-zA-Z0-9_]+)["\]` + "`" + `]?`)
	matches := reTable.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return ctx, fmt.Errorf("通用解析失败: 无法定位 CREATE TABLE 表名")
	}
	entireTable := matches[1]

	entireTableName := Capitalize(SnakeToCamel(entireTable))
	firstUnderscore := strings.Index(entireTable, "_")
	tableNameRaw := entireTable
	if firstUnderscore != -1 {
		tableNameRaw = entireTable[firstUnderscore+1:]
	}

	urlTableName := strings.ReplaceAll(tableNameRaw, "_", "/")
	vueTableName := strings.ReplaceAll(tableNameRaw, "_", "-")
	TABLE_NAME := strings.ToUpper(tableNameRaw)
	TableNameStr := Capitalize(SnakeToCamel(tableNameRaw))
	tableNameLower := ""
	if len(TableNameStr) > 0 {
		tableNameLower = strings.ToLower(string(TableNameStr[0])) + TableNameStr[1:]
	}
	TABLENAME_UPPER := strings.ToUpper(tableNameLower)

	tableComment := ""
	if dbType == "mssql" || dbType == "sqlserver" {
		// MS SQL extension property naive extraction
		reTabCom := regexp.MustCompile(`(?i)@value\s*=\s*N'([^']+)'\s*,[^@]*@level1type\s*=\s*N'TABLE'\s*,[^@]*@level1name\s*=\s*N'([^']+)'`)
		subs := reTabCom.FindAllStringSubmatch(sql, -1)
		for _, s := range subs {
			if len(s) >= 3 && !strings.Contains(s[0], "@level2type") {
				tableComment = s[1]
			}
		}
	} else {
		reTabCom := regexp.MustCompile(`(?i)COMMENT\s+ON\s+TABLE\s+[a-zA-Z0-9_".\[\]]+\s+IS\s+'([^']+)'`)
		cMatch := reTabCom.FindStringSubmatch(sql)
		if len(cMatch) >= 2 {
			tableComment = cMatch[1]
		}
	}

	startIdx := strings.Index(sql, "(")
	endIdx := strings.LastIndex(sql, ")")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return ctx, fmt.Errorf("通用解析失败: 找不到列定义括号区块")
	}

	colsBlock := sql[startIdx+1 : endIdx]

	lines := strings.Split(colsBlock, "\n")
	var fields []ExtractedField
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(strings.ToUpper(l), "PRIMARY KEY") || strings.HasPrefix(strings.ToUpper(l), "CONSTRAINT") {
			continue
		}

		reCol := regexp.MustCompile(`^["\[` + "`" + `]?([a-zA-Z0-9_]+)["\]` + "`" + `]?\s+([a-zA-Z0-9_]+)(\([0-9,]+\))?`)
		cMatch := reCol.FindStringSubmatch(l)
		if len(cMatch) < 3 {
			continue
		}
		orig := cMatch[1]
		dbTypeRaw := cMatch[2]
		lenInfo := ""
		if len(cMatch) > 3 {
			lenInfo = cMatch[3]
		}
		rawLen := strings.Trim(lenInfo, "()")

		comment := ""
		if dbType == "mssql" || dbType == "sqlserver" {
			reColCom := regexp.MustCompile(`(?i)@value\s*=\s*N'([^']+)'\s*,[^@]*@level2type\s*=\s*N'COLUMN'\s*,[^@]*@level2name\s*=\s*N'` + orig + `'`)
			ccMatch := reColCom.FindStringSubmatch(sql)
			if len(ccMatch) >= 2 {
				comment = ccMatch[1]
			}
		} else {
			reColCom := regexp.MustCompile(`(?i)COMMENT\s+ON\s+COLUMN\s+[a-zA-Z0-9_".\[\]]+\.` + orig + `\s+IS\s+'([^']+)'`)
			ccMatch := reColCom.FindStringSubmatch(sql)
			if len(ccMatch) >= 2 {
				comment = ccMatch[1]
			}
		}

		javaType := ConvertDataType(dbTypeRaw)
		pyType := SqlToPyType(dbTypeRaw)
		voType := SqlToPyVoType(dbTypeRaw)
		vueType := ConvertVueType(dbTypeRaw)

		javaField := SnakeToCamel(orig)
		capField := Capitalize(javaField)

		fields = append(fields, ExtractedField{
			OriginalName:     orig,
			JavaDataType:     javaType,
			LengthInfo:       lenInfo,
			Comment:          comment,
			JavaField:        javaField,
			PyType:           pyType,
			CapitalizedField: capField,
			VueDataType:      vueType,
			RawLengthInfo:    rawLen,
			PythonVoType:     voType,
		})
	}

	ctx = RenderContext{
		RawTableName:    entireTable,
		EntireTableName: entireTableName,
		Table_name:      tableNameRaw,
		TableName:       TableNameStr,
		TableNameLower:  tableNameLower,
		TABLENAME:       TABLENAME_UPPER,
		TABLE_NAME:      TABLE_NAME,
		UrlTableName:    urlTableName,
		VueTableName:    vueTableName,
		TableComment:    tableComment,
		FieldLines:      fields,
	}
	return ctx, nil
}

func CustomFormat(pathTpl string, ctx map[string]string) string {
	re := regexp.MustCompile(`{\[<([^>]+)>\]}`)
	return re.ReplaceAllStringFunc(pathTpl, func(m string) string {
		match := re.FindStringSubmatch(m)
		if len(match) > 1 {
			if val, ok := ctx[match[1]]; ok {
				return val
			}
		}
		return m
	})
}

func GenerateCode(genProject GenerateProjectInfo, paths []system.TbGenerateProjectPath, models []system.TbGenerateProjectPathModel, holders []system.TbGenerateProjectPlaceHolder, baseHolders []system.TbGeneratePlaceHolder, diskPath string) ([]string, error) {
	ctx, err := SqlToJava(genProject.TableStructure, genProject.DbType)
	if err != nil {
		global.GVA_LOG.Error("SQL解析失败", zap.Error(err))
		return nil, err
	}

	ctx.ModuleName = genProject.ModuleName
	ctx.ModuleNameUpper = Capitalize(genProject.ModuleName)
	ctx.ModuleComment = genProject.ModuleComment
	ctx.Date = time.Now().Format("2006-01-02 15:04:05")

	holderDict := make(map[string]string)
	for _, bh := range baseHolders {
		k := strings.ReplaceAll(bh.HolderKey, "{[<", "")
		k = strings.ReplaceAll(k, ">]}", "")
		holderDict[k] = bh.HolderValue
	}
	for _, ph := range holders {
		k := strings.ReplaceAll(ph.HolderKey, "{[<", "")
		k = strings.ReplaceAll(k, ">]}", "")
		holderDict[k] = ph.HolderValue
	}
	ctx.Placeholders = holderDict

	formatCtx := map[string]string{
		"moduleName":      ctx.ModuleName,
		"ModuleName":      ctx.ModuleNameUpper,
		"rawTableName":    ctx.RawTableName,
		"tableName":       ctx.TableNameLower,
		"table_name":      ctx.Table_name,
		"TableName":       ctx.TableName,
		"entireTableName": ctx.EntireTableName,
		"TABLENAME":       ctx.TABLENAME,
		"TABLE_NAME":      ctx.TABLE_NAME,
		"urlTableName":    ctx.UrlTableName,
		"vueTableName":    ctx.VueTableName,
	}

	generatedFiles := make([]string, 0)
	generatedFileSet := make(map[string]struct{})

	for _, p := range paths {
		var relModels []system.TbGenerateProjectPathModel
		for _, m := range models {
			if m.PathId == int(p.ID) {
				relModels = append(relModels, m)
			}
		}
		if len(relModels) == 0 {
			continue
		}

		writeFlag := p.Incremented == 0

		fileUrl := CustomFormat(p.FileUrl, formatCtx)
		fileName := CustomFormat(p.FileName, formatCtx)

		fullPath := filepath.Join(diskPath, fileUrl, fileName)

		for _, m := range relModels {
			// Template rendering!
			tmpl := template.New("model").Funcs(codeGenTemplateFuncs())
			tmpl.Delims("{[<", ">]}")

			// Map some missing variables that used to be passed flatly in jinja:
			t, err := tmpl.Parse(m.Content)
			if err != nil {
				global.GVA_LOG.Error("模板解析失败", zap.Error(err))
				continue
			}
			var buf bytes.Buffer

			// Since text/template expects . Field, we wrap in a map for easy indexing
			wrapper := map[string]interface{}{
				"moduleName":        ctx.ModuleName,
				"ModuleName":        ctx.ModuleNameUpper,
				"moduleComment":     ctx.ModuleComment,
				"rawTableName":      ctx.RawTableName,
				"entire_table_name": ctx.EntireTableName, // Fallback for old templates
				"entireTableName":   ctx.EntireTableName,
				"table_name":        ctx.Table_name,
				"TableName":         ctx.TableName,
				"tableName":         ctx.TableNameLower,
				"TABLENAME":         ctx.TABLENAME,
				"TABLE_NAME":        ctx.TABLE_NAME,
				"urlTableName":      ctx.UrlTableName,
				"vueTableName":      ctx.VueTableName,
				"tableComment":      ctx.TableComment,
				"field_lines":       ctx.FieldLines,
				"date":              ctx.Date,
			}
			for k, v := range holderDict {
				wrapper[k] = v
			}

			if err := t.Execute(&buf, wrapper); err != nil {
				global.GVA_LOG.Error("模板执行失败", zap.Error(err))
				continue
			}

			if err := generateAndWriteFile(fullPath, buf.String(), writeFlag); err != nil {
				global.GVA_LOG.Error("文件写入失败", zap.String("path", fullPath), zap.Error(err))
				return generatedFiles, err
			}
			if _, ok := generatedFileSet[fullPath]; !ok {
				generatedFiles = append(generatedFiles, fullPath)
				generatedFileSet[fullPath] = struct{}{}
			}
		}
	}
	return generatedFiles, nil
}

func generateAndWriteFile(path string, content string, writeFlag bool) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) || writeFlag {
		return os.WriteFile(path, []byte(content), os.ModePerm)
	} else if err != nil {
		return err
	} else {
		// 追加内容
		ext := filepath.Ext(path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if ext == ".java" {
			txt := string(data)
			lastBrace := strings.LastIndex(txt, "}")
			if lastBrace != -1 {
				newTxt := txt[:lastBrace] + content + txt[lastBrace:]
				return os.WriteFile(path, []byte(newTxt), os.ModePerm)
			} else {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				if _, err := f.WriteString(content); err != nil {
					f.Close()
					return err
				}
				return f.Close()
			}
		} else {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			if _, err := f.WriteString(content); err != nil {
				f.Close()
				return err
			}
			return f.Close()
		}
	}
}
