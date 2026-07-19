package system

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	sysModel "github.com/flipped-aurora/easy-deploy/server/model/system"
)

func TestSafeLocalScriptFilePathKeepsFilesInsideOutputRoot(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "frontend", "Dockerfile")

	got, err := safeLocalScriptFilePath(root, "frontend/Dockerfile")
	if err != nil {
		t.Fatalf("safeLocalScriptFilePath() error = %v", err)
	}
	if got != want {
		t.Fatalf("safeLocalScriptFilePath() = %q, want %q", got, want)
	}

	for _, fileName := range []string{"../outside.sh", "../../outside.sh", filepath.Join(string(filepath.Separator), "tmp", "outside.sh")} {
		if _, err := safeLocalScriptFilePath(root, fileName); err == nil {
			t.Fatalf("safeLocalScriptFilePath(%q) unexpectedly succeeded", fileName)
		}
	}
}

func TestBuildDockerLogCommandUsesRouteScriptDirectory(t *testing.T) {
	projectDir := t.TempDir()
	scriptDir := t.TempDir()
	composePath := filepath.Join(scriptDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	project := sysModel.TbProject{ProjectName: "example"}
	route := sysModel.TbProjectRoute{
		LocalScriptPath: scriptDir,
		BuildType:       "docker_compose_deploy",
	}
	command, args, workDir := buildDockerLogCommand(project, route, projectDir, "web")

	if command != "docker" {
		t.Fatalf("command = %q, want docker", command)
	}
	wantArgs := []string{"compose", "-f", composePath, "--project-directory", projectDir, "logs", "--tail", dockerLogTailLines, "-f", "web"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
	if workDir != projectDir {
		t.Fatalf("workDir = %q, want %q", workDir, projectDir)
	}
}

func TestBuildDockerLogCommandUsesDirectContainerLogsForProjectCard(t *testing.T) {
	projectDir := t.TempDir()
	scriptDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(scriptDir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	project := sysModel.TbProject{
		ProjectName:      "ai-file-navigation-web",
		ComputerLanguage: "react",
	}
	route := sysModel.TbProjectRoute{
		LocalScriptPath: scriptDir,
		BuildType:       "docker_compose_deploy",
	}
	command, args, workDir := buildDockerLogCommand(project, route, projectDir, "")

	if command != "docker" {
		t.Fatalf("command = %q, want docker", command)
	}
	wantArgs := []string{"logs", "--tail", dockerLogTailLines, "-f", "ai-file-navigation-web"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
	if workDir != projectDir {
		t.Fatalf("workDir = %q, want %q", workDir, projectDir)
	}
}
