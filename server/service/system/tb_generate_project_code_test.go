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
