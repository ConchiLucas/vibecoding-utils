package system

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseAggregateChildScriptPathsPreservesOrderAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	aggregateDir := filepath.Join(root, "deploy", "compose", "full")
	content := `#!/bin/sh
# ignored
echo "prepare"
sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"
bash "$ROOT_DIR/deploy/backend/word_agent/local_full/start.sh" --force
sh $ROOT_DIR/deploy/backend/rob_english_word/local_full/start.sh
sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"
`

	got, err := parseAggregateChildScriptPaths(root, aggregateDir, content)
	if err != nil {
		t.Fatalf("parseAggregateChildScriptPaths() error = %v", err)
	}
	want := []string{
		filepath.Join(root, "deploy", "backend", "word_agent", "build_project"),
		filepath.Join(root, "deploy", "backend", "word_agent", "local_full"),
		filepath.Join(root, "deploy", "backend", "rob_english_word", "local_full"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func TestParseAggregateChildScriptPathsRejectsUnsafeOrUnsupportedReferences(t *testing.T) {
	root := t.TempDir()
	aggregateDir := filepath.Join(root, "deploy", "compose", "full")
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "unsupported command", content: `source "$ROOT_DIR/deploy/backend/api/local_full/start.sh"`, want: "第 1 行"},
		{name: "traversal", content: `sh "$ROOT_DIR/../outside/start.sh"`, want: "越过项目目录"},
		{name: "self reference", content: `sh "$ROOT_DIR/deploy/compose/full/start.sh"`, want: "引用自身"},
		{name: "empty", content: "#!/bin/sh\necho ready\n", want: "未引用任何"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseAggregateChildScriptPaths(root, aggregateDir, test.content)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want diagnostic containing %q", err, test.want)
			}
		})
	}
}

func TestParseAggregateChildScriptPathsRequiresAbsoluteRoot(t *testing.T) {
	_, err := parseAggregateChildScriptPaths("relative", "relative/deploy/full", `sh "$ROOT_DIR/deploy/api/start.sh"`)
	if err == nil || !strings.Contains(err.Error(), "绝对路径") {
		t.Fatalf("error = %v, want absolute-root diagnostic", err)
	}
}
