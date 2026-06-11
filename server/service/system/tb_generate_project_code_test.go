package system

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderCodeGenerationTextReplacesSupportedPlaceholders(t *testing.T) {
	vars := buildCodeGenerationVars("btStation", "BtStation")
	got := renderCodeGenerationText("{{module}}/{{TableName}}/{{table_name}}/{[<moduleName>]}/{[<TableName>]}/{{<moduleName>}}", vars)
	want := "btStation/BtStation/bt_station/btStation/BtStation/btStation"
	if got != want {
		t.Fatalf("renderCodeGenerationText = %q, want %q", got, want)
	}
}

func TestBuildGeneratedFileTargetRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	_, _, err := buildGeneratedFileTarget(root, "../outside", "Bad.java")
	if err == nil {
		t.Fatal("buildGeneratedFileTarget error = nil, want traversal error")
	}
}

func TestBuildGeneratedFileTargetBuildsRelativePath(t *testing.T) {
	root := t.TempDir()
	relativePath, targetPath, err := buildGeneratedFileTarget(root, "src/main/java/{{module}}", "{{TableName}}.java")
	if err != nil {
		t.Fatalf("buildGeneratedFileTarget returned error: %v", err)
	}
	if relativePath != "src/main/java/{{module}}/{{TableName}}.java" {
		t.Fatalf("relativePath = %q", relativePath)
	}
	if !strings.HasPrefix(targetPath, filepath.Clean(root)+string(filepath.Separator)) {
		t.Fatalf("targetPath %q is not under root %q", targetPath, root)
	}
}

func TestRenderGeneratedFileTargetReplacesPathPlaceholders(t *testing.T) {
	root := t.TempDir()
	vars := buildCodeGenerationVars("btStation", "BtStation")

	relativePath, targetPath, err := renderGeneratedFileTarget(root, "src/main/java/{{module}}", "{{TableName}}.java", vars)
	if err != nil {
		t.Fatalf("renderGeneratedFileTarget returned error: %v", err)
	}
	if relativePath != "src/main/java/btStation/BtStation.java" {
		t.Fatalf("relativePath = %q", relativePath)
	}
	if !strings.HasPrefix(targetPath, filepath.Clean(root)+string(filepath.Separator)) {
		t.Fatalf("targetPath %q is not under root %q", targetPath, root)
	}
}

func TestRenderCodeGenerationTextReplacesTemplateContentPlaceholders(t *testing.T) {
	vars := buildCodeGenerationVars("btStation", "BtStation")
	got := renderCodeGenerationText("package {{module}};\npublic class {{TableName}}Item {}", vars)
	want := "package btStation;\npublic class BtStationItem {}"
	if got != want {
		t.Fatalf("renderCodeGenerationText = %q, want %q", got, want)
	}
}

func TestRenderCodeGenerationTextReplacesDynamicPlaceholders(t *testing.T) {
	vars := mergeCodeGenerationPlaceholderValues(buildCodeGenerationVars("btStation", "BtStation"), map[string]string{
		"{{tenant_code}}": "pzh",
		"${menuCode}":     "btWaybillList",
	})
	got := renderCodeGenerationText("{{tenant_code}}/${menuCode}/{{TableName}}", vars)
	want := "pzh/btWaybillList/BtStation"
	if got != want {
		t.Fatalf("renderCodeGenerationText = %q, want %q", got, want)
	}
}

func TestRenderCodeGenerationTextReplacesScopedPlaceholders(t *testing.T) {
	vars := mergeCodeGenerationPlaceholderValues(buildCodeGenerationVars("btStation", "BtStation"), map[string]string{
		"javaEntityFields": "private Long id;",
	})
	got := renderCodeGenerationText("{{manual:moduleName}}/{{field:TableName}}/${field:javaEntityFields}/{{snippet:table_name}}/{{manual:<moduleName>}}", vars)
	want := "btStation/BtStation/private Long id;/bt_station/btStation"
	if got != want {
		t.Fatalf("renderCodeGenerationText = %q, want %q", got, want)
	}
}

func TestBuildGenerateFieldSourcePlaceholdersDerivesTablePlaceholders(t *testing.T) {
	sql := `
CREATE TABLE cs_bt_train_operation_tracking (
    id bigint COMMENT '主键ID',
    train_no varchar(64) COMMENT '班列编号'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班列作业跟踪表';
`
	values := buildGenerateFieldSourcePlaceholders(sql)
	wants := map[string]string{
		"tableName":      "btTrainOperationTracking",
		"TableName":      "BtTrainOperationTracking",
		"table_name":     "bt_train_operation_tracking",
		"TABLE_NAME":     "BT_TRAIN_OPERATION_TRACKING",
		"kebabTableName": "bt-train-operation-tracking",
		"commentName":    "班列作业跟踪",
	}
	for key, want := range wants {
		if values[key] != want {
			t.Fatalf("values[%q] = %q, want %q", key, values[key], want)
		}
	}
	if _, ok := values["moduleName"]; ok {
		t.Fatalf("moduleName should not be derived from field snippets: %#v", values)
	}
}

func TestParseGenerateFieldColumnsSkipsCreateTableLine(t *testing.T) {
	sql := `
CREATE TABLE cs_bt_train_operation_tracking (
    id bigint COMMENT '主键ID',
    train_no varchar(64) COMMENT '班列编号'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班列作业跟踪表';
`
	columns := parseGenerateFieldColumns(sql)
	if len(columns) != 2 {
		t.Fatalf("len(columns) = %d, want 2: %#v", len(columns), columns)
	}
	if columns[0].ColumnName != "id" || columns[1].ColumnName != "train_no" {
		t.Fatalf("columns parsed incorrectly: %#v", columns)
	}
}

func TestApplyIncrementalTemplateContentInsertsJavaFragmentBeforeLastBrace(t *testing.T) {
	existing := "package demo;\n\npublic interface DemoSqlId {\n    String OLD = \"old\";\n}\n"
	fragment := "    /**\n     * 运单管理分页查询\n     */\n    String BT_WAYBILL_QUERY_GET_PAGE_LIST = \"btWaybill_query_getPageList\";"

	got, status, applied := applyIncrementalTemplateContent("/tmp/DemoSqlId.java", existing, fragment)
	if !applied {
		t.Fatal("applyIncrementalTemplateContent applied = false, want true")
	}
	if status != "incremented" {
		t.Fatalf("status = %q, want incremented", status)
	}
	if !strings.Contains(got, fragment+"\n}") {
		t.Fatalf("incremental content was not inserted before final brace:\n%s", got)
	}
	if strings.Contains(got, "}\n    /**") {
		t.Fatalf("incremental content was appended after final brace:\n%s", got)
	}
}

func TestApplyIncrementalTemplateContentSkipsDuplicateJavaConstant(t *testing.T) {
	existing := "package demo;\n\npublic interface DemoSqlId {\n    String BT_WAYBILL_QUERY_GET_PAGE_LIST = \"btWaybill_query_getPageList\";\n}\n"
	fragment := "    /**\n     * 运单管理分页查询\n     */\n    String BT_WAYBILL_QUERY_GET_PAGE_LIST = \"btWaybill_query_getPageList\";"

	got, status, applied := applyIncrementalTemplateContent("/tmp/DemoSqlId.java", existing, fragment)
	if !applied {
		t.Fatal("applyIncrementalTemplateContent applied = false, want true")
	}
	if status != "skipped" {
		t.Fatalf("status = %q, want skipped", status)
	}
	if got != existing {
		t.Fatalf("duplicate content changed existing file:\n%s", got)
	}
}

func TestBuildCodeGenerationTaskPromptContentIncludesTargets(t *testing.T) {
	drafts := []generateProjectCodeDraft{
		{
			File: GenerateProjectCodeFile{
				AbsolutePath: "/tmp/output/src/demo/DemoItem.java",
				RelativePath: "src/demo/DemoItem.java",
				Status:       "generated",
				Instruction:  "读取目标文件并生成最终代码。",
			},
			TemplateContent: "public class DemoItem {}",
			FilePrompt:      "请根据产品文档生成新增编辑入参类。",
		},
	}

	content := buildCodeGenerationTaskPromptContent("demo", "Demo", "测试项目", "/tmp/output", 1, drafts)
	for _, want := range []string{
		"/tmp/output/src/demo/DemoItem.java",
		"读取目标文件并生成最终代码。",
		"请根据产品文档生成新增编辑入参类。",
		"读取目标文件所在目录",
		"字段只保留前端或业务真正使用的字段",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("prompt content does not contain %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "public class DemoItem {}") {
		t.Fatalf("prompt content should not include template content:\n%s", content)
	}
}
