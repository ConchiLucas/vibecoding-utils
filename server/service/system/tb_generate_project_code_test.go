package system

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderCodeGenerationTextReplacesSupportedPlaceholders(t *testing.T) {
	vars := buildCodeGenerationVars("btStation", "BtStation")
	got := renderCodeGenerationText("{{module}}/{{TableName}}/{{table_name}}/{[<moduleName>]}", vars)
	want := "btStation/BtStation/bt_station/btStation"
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

	content := buildCodeGenerationTaskPromptContent("demo", "Demo", false, "测试项目", "/tmp/output", 1, drafts)
	for _, want := range []string{
		"/tmp/output/src/demo/DemoItem.java",
		"读取目标文件并生成最终代码。",
		"请根据产品文档生成新增编辑入参类。",
		"public class DemoItem {}",
		"字段只保留前端或业务真正使用的字段",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("prompt content does not contain %q:\n%s", want, content)
		}
	}
}
