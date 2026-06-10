package system

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
)

type TbGenerateFieldSnippetService struct{}

type GenerateFieldColumn struct {
	ColumnName  string `json:"columnName"`
	DbType      string `json:"dbType"`
	Comment     string `json:"comment"`
	JavaField   string `json:"javaField"`
	SnakeField  string `json:"snakeField"`
	PascalField string `json:"pascalField"`
	UpperField  string `json:"upperField"`
	JavaType    string `json:"javaType"`
	TsType      string `json:"tsType"`
	PythonType  string `json:"pythonType"`
}

type GenerateFieldSnippetPreview struct {
	Columns  []GenerateFieldColumn `json:"columns"`
	Rendered map[string]string     `json:"rendered"`
}

var (
	fieldLinePattern    = regexp.MustCompile(`^\s*[` + "`" + `"]?([A-Za-z_][A-Za-z0-9_]*)[` + "`" + `"]?\s+([A-Za-z0-9_()]+)`)
	fieldCommentPattern = regexp.MustCompile(`(?i)comment\s+['"]([^'"]*)['"]`)
	fieldSkipPattern    = regexp.MustCompile(`(?i)^\s*(primary|unique|key|index|constraint|foreign|check)\b`)
)

func (s *TbGenerateFieldSnippetService) GetLatestGenerateFieldSnippet(businessType string) (system.TbGenerateFieldSnippet, error) {
	var record system.TbGenerateFieldSnippet
	err := global.GVA_DB.Where("business_type = ?", strings.TrimSpace(businessType)).Order("id DESC").First(&record).Error
	return record, err
}

func (s *TbGenerateFieldSnippetService) GetGenerateFieldSnippetHistory(businessType string) ([]system.TbGenerateFieldSnippet, error) {
	var records []system.TbGenerateFieldSnippet
	err := global.GVA_DB.Where("business_type = ?", strings.TrimSpace(businessType)).Order("id DESC").Limit(50).Find(&records).Error
	return records, err
}

func (s *TbGenerateFieldSnippetService) PreviewGenerateFieldSnippet(req systemReq.PreviewGenerateFieldSnippetReq) (GenerateFieldSnippetPreview, error) {
	return buildGenerateFieldSnippetPreview(req.SourceText, req.Snippets), nil
}

func (s *TbGenerateFieldSnippetService) SaveGenerateFieldSnippet(req systemReq.SaveGenerateFieldSnippetReq) (system.TbGenerateFieldSnippet, error) {
	preview := buildGenerateFieldSnippetPreview(req.SourceText, req.Snippets)
	snippetsJSON, _ := json.Marshal(req.Snippets)
	renderedJSON, _ := json.Marshal(preview.Rendered)
	record := system.TbGenerateFieldSnippet{
		BusinessType: strings.TrimSpace(req.BusinessType),
		Name:         strings.TrimSpace(req.Name),
		SourceText:   req.SourceText,
		Snippets:     string(snippetsJSON),
		Rendered:     string(renderedJSON),
		UserName:     strings.TrimSpace(req.UserName),
	}
	if record.Name == "" {
		record.Name = "字段片段"
	}
	if err := global.GVA_DB.Create(&record).Error; err != nil {
		return system.TbGenerateFieldSnippet{}, err
	}
	return record, nil
}

func renderLatestGenerateFieldSnippets(businessType string) map[string]string {
	var record system.TbGenerateFieldSnippet
	if err := global.GVA_DB.Where("business_type = ?", strings.TrimSpace(businessType)).Order("id DESC").First(&record).Error; err != nil {
		return map[string]string{}
	}
	var snippets []systemReq.GenerateFieldSnippetTemplate
	if err := json.Unmarshal([]byte(record.Snippets), &snippets); err != nil {
		return map[string]string{}
	}
	return buildGenerateFieldSnippetPreview(record.SourceText, snippets).Rendered
}

func buildGenerateFieldSnippetPreview(sourceText string, snippets []systemReq.GenerateFieldSnippetTemplate) GenerateFieldSnippetPreview {
	columns := parseGenerateFieldColumns(sourceText)
	rendered := make(map[string]string)
	for _, snippet := range snippets {
		key := strings.TrimSpace(snippet.Key)
		if key == "" {
			continue
		}
		parts := make([]string, 0, len(columns))
		for _, column := range columns {
			if snippet.ExcludeAudit && isGenerateAuditColumn(column.ColumnName) {
				continue
			}
			parts = append(parts, renderGenerateFieldTemplate(snippet.Template, column))
		}
		rendered[key] = strings.Join(parts, snippet.Separator)
	}
	return GenerateFieldSnippetPreview{Columns: columns, Rendered: rendered}
}

func parseGenerateFieldColumns(sourceText string) []GenerateFieldColumn {
	lines := strings.Split(sourceText, "\n")
	columns := make([]GenerateFieldColumn, 0)
	seen := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimRight(line, ","))
		if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") || fieldSkipPattern.MatchString(trimmed) {
			continue
		}
		match := fieldLinePattern.FindStringSubmatch(trimmed)
		if len(match) < 3 {
			continue
		}
		columnName := strings.TrimSpace(match[1])
		if columnName == "" || seen[columnName] {
			continue
		}
		seen[columnName] = true
		dbType := strings.ToLower(strings.TrimSpace(match[2]))
		comment := columnName
		if commentMatch := fieldCommentPattern.FindStringSubmatch(trimmed); len(commentMatch) > 1 {
			comment = strings.TrimSpace(commentMatch[1])
		}
		javaField := toCamelName(columnName)
		columns = append(columns, GenerateFieldColumn{
			ColumnName:  columnName,
			DbType:      dbType,
			Comment:     comment,
			JavaField:   javaField,
			SnakeField:  toSnakeCase(columnName),
			PascalField: upperFirst(javaField),
			UpperField:  strings.ToUpper(toSnakeCase(columnName)),
			JavaType:    mapGenerateJavaType(dbType),
			TsType:      mapGenerateTsType(dbType),
			PythonType:  mapGeneratePythonType(dbType),
		})
	}
	sort.SliceStable(columns, func(i, j int) bool { return i < j })
	return columns
}

func renderGenerateFieldTemplate(template string, column GenerateFieldColumn) string {
	values := map[string]string{
		"columnName":  column.ColumnName,
		"dbType":      column.DbType,
		"comment":     column.Comment,
		"javaField":   column.JavaField,
		"snakeField":  column.SnakeField,
		"pascalField": column.PascalField,
		"upperField":  column.UpperField,
		"javaType":    column.JavaType,
		"tsType":      column.TsType,
		"pythonType":  column.PythonType,
	}
	result := template
	for key, value := range values {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	return result
}

func isGenerateAuditColumn(columnName string) bool {
	key := strings.ToLower(toSnakeCase(columnName))
	switch key {
	case "id", "deleted", "tenancy", "company_id", "creator", "create_time", "modifier", "modify_time", "create_user", "update_user", "update_time":
		return true
	default:
		return false
	}
}

func mapGenerateJavaType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.Contains(dbType, "bigint"):
		return "Long"
	case strings.Contains(dbType, "int"):
		return "Integer"
	case strings.Contains(dbType, "decimal"), strings.Contains(dbType, "numeric"):
		return "BigDecimal"
	case strings.Contains(dbType, "double"), strings.Contains(dbType, "float"):
		return "Double"
	case strings.Contains(dbType, "date"), strings.Contains(dbType, "time"):
		return "String"
	case strings.Contains(dbType, "bool"):
		return "Boolean"
	default:
		return "String"
	}
}

func mapGenerateTsType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.Contains(dbType, "bigint"), strings.Contains(dbType, "int"), strings.Contains(dbType, "decimal"), strings.Contains(dbType, "numeric"), strings.Contains(dbType, "double"), strings.Contains(dbType, "float"):
		return "number"
	case strings.Contains(dbType, "bool"):
		return "boolean"
	default:
		return "string"
	}
}

func mapGeneratePythonType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.Contains(dbType, "bigint"), strings.Contains(dbType, "int"):
		return "int"
	case strings.Contains(dbType, "decimal"), strings.Contains(dbType, "numeric"), strings.Contains(dbType, "double"), strings.Contains(dbType, "float"):
		return "float"
	case strings.Contains(dbType, "bool"):
		return "bool"
	default:
		return "str"
	}
}
