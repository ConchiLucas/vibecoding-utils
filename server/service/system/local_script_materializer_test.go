package system

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
)

func createMaterializationRequest(t *testing.T, dbProject modelSystem.TbProject, route modelSystem.TbProjectRoute, scriptPath string) localScriptMaterializationRequest {
	t.Helper()
	return localScriptMaterializationRequest{Project: dbProject, RouteID: route.ID, ScriptPath: scriptPath}
}

func TestLoadLocalScriptsForMaterializationPrevalidatesWholeBatch(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	first, firstRoute := createAggregateChildRoute(t, db, 1, "go", "valid", filepath.Join(root, "valid"), filepath.Join(root, "deploy", "valid"), 0, false)
	second, secondRoute := createAggregateChildRoute(t, db, 1, "go", "broken", filepath.Join(root, "broken"), filepath.Join(root, "deploy", "broken"), 0, false)
	valid := modelSystem.TbProjectScript{ProjectId: int(first.ID), RouteId: int(firstRoute.ID), ScriptType: 1, FileName: "docker-compose.yml", Content: "services:\n  api:\n    image: example/api:1\n"}
	broken := modelSystem.TbProjectScript{ProjectId: int(second.ID), RouteId: int(secondRoute.ID), ScriptType: 1, FileName: "docker-compose.yml", Content: "services:\n  api: [\n"}
	if err := db.Create(&valid).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&broken).Error; err != nil {
		t.Fatal(err)
	}

	_, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{
		createMaterializationRequest(t, first, firstRoute, firstRoute.LocalScriptPath),
		createMaterializationRequest(t, second, secondRoute, secondRoute.LocalScriptPath),
	})
	if err == nil || !strings.Contains(err.Error(), "broken") && !strings.Contains(err.Error(), "script=") {
		t.Fatalf("error = %v, want malformed final-script diagnostic", err)
	}
	if _, err := os.Stat(filepath.Join(firstRoute.LocalScriptPath, valid.FileName)); !os.IsNotExist(err) {
		t.Fatalf("preflight wrote first file: %v", err)
	}
	var reloaded modelSystem.TbProjectScript
	if err := db.First(&reloaded, valid.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Content != valid.Content {
		t.Fatal("preflight changed database content")
	}
}

func TestLoadLocalScriptsForMaterializationRejectsConflictingTargets(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	target := filepath.Join(root, "deploy", "shared")
	first, firstRoute := createAggregateChildRoute(t, db, 2, "go", "first", filepath.Join(root, "first"), target, 0, false)
	second, secondRoute := createAggregateChildRoute(t, db, 2, "java", "second", filepath.Join(root, "second"), target, 0, false)
	for _, script := range []modelSystem.TbProjectScript{
		{ProjectId: int(first.ID), RouteId: int(firstRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "first"},
		{ProjectId: int(second.ID), RouteId: int(secondRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "second"},
	} {
		if err := db.Create(&script).Error; err != nil {
			t.Fatal(err)
		}
	}

	_, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{
		createMaterializationRequest(t, first, firstRoute, target),
		createMaterializationRequest(t, second, secondRoute, target),
	})
	if err == nil || !strings.Contains(err.Error(), "目标文件冲突") {
		t.Fatalf("error = %v, want target conflict", err)
	}
}

func TestPublishPreparedLocalScriptsRollsBackReplacedAndNewFiles(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	firstPath := filepath.Join(root, "deploy", "first")
	secondPath := filepath.Join(root, "deploy", "second")
	first, firstRoute := createAggregateChildRoute(t, db, 3, "go", "first", filepath.Join(root, "first"), firstPath, 0, false)
	second, secondRoute := createAggregateChildRoute(t, db, 3, "java", "second", filepath.Join(root, "second"), secondPath, 0, false)
	for _, script := range []modelSystem.TbProjectScript{
		{ProjectId: int(first.ID), RouteId: int(firstRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "new-first"},
		{ProjectId: int(second.ID), RouteId: int(secondRoute.ID), ScriptType: 1, FileName: "start.sh", Content: "new-second"},
	} {
		if err := db.Create(&script).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(firstPath, 0o755); err != nil {
		t.Fatal(err)
	}
	firstTarget := filepath.Join(firstPath, "start.sh")
	if err := os.WriteFile(firstTarget, []byte("original-first"), 0o600); err != nil {
		t.Fatal(err)
	}

	prepared, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{
		createMaterializationRequest(t, first, firstRoute, firstPath),
		createMaterializationRequest(t, second, secondRoute, secondPath),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = publishPreparedLocalScripts(db, prepared, func(index int) error {
		if index == 1 {
			return errors.New("forced publish failure")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "forced publish failure") {
		t.Fatalf("error = %v, want forced failure", err)
	}
	content, err := os.ReadFile(firstTarget)
	if err != nil || string(content) != "original-first" {
		t.Fatalf("first file = %q, err=%v; want exact original", content, err)
	}
	if _, err := os.Stat(filepath.Join(secondPath, "start.sh")); !os.IsNotExist(err) {
		t.Fatalf("new second file survived rollback: %v", err)
	}
}

func TestPublishPreparedLocalScriptsRejectsStaleDatabaseBeforeFileReplacement(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	project, route := createAggregateChildRoute(t, db, 4, "go", "api", root, filepath.Join(root, "deploy"), 0, false)
	original := "services:\n  api:\n    image: example/api:1\n"
	script := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 1, FileName: "docker-compose.yml", Content: original}
	if err := db.Create(&script).Error; err != nil {
		t.Fatal(err)
	}
	prepared, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{createMaterializationRequest(t, project, route, route.LocalScriptPath)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&modelSystem.TbProjectScript{}).Where("id = ?", script.ID).Update("content", "newer").Error; err != nil {
		t.Fatal(err)
	}

	err = publishPreparedLocalScripts(db, prepared, nil)
	if err == nil || !strings.Contains(err.Error(), "并发修改") {
		t.Fatalf("error = %v, want optimistic conflict", err)
	}
	if _, err := os.Stat(filepath.Join(route.LocalScriptPath, script.FileName)); !os.IsNotExist(err) {
		t.Fatalf("stale publish wrote a file: %v", err)
	}
}

func TestPublishPreparedLocalScriptsNormalizesAndWritesMode(t *testing.T) {
	db := setupAggregateRouteTestDB(t)
	root := t.TempDir()
	project, route := createAggregateChildRoute(t, db, 5, "python", "word-agent", root, filepath.Join(root, "deploy"), 0, false)
	script := modelSystem.TbProjectScript{ProjectId: int(project.ID), RouteId: int(route.ID), ScriptType: 1, FileName: "docker-compose.yml", Content: "services:\n  api:\n    image: example/api:1\n"}
	if err := db.Create(&script).Error; err != nil {
		t.Fatal(err)
	}
	prepared, err := loadLocalScriptsForMaterialization(db, []localScriptMaterializationRequest{createMaterializationRequest(t, project, route, route.LocalScriptPath)})
	if err != nil {
		t.Fatal(err)
	}
	if err := publishPreparedLocalScripts(db, prepared, nil); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(route.LocalScriptPath, script.FileName)
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateComposeSharedNetwork(string(content)); err != nil {
		t.Fatalf("materialized Compose network invalid: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %o, want 644", info.Mode().Perm())
	}
	var reloaded modelSystem.TbProjectScript
	if err := db.First(&reloaded, script.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateComposeSharedNetwork(reloaded.Content); err != nil {
		t.Fatalf("stored Compose network invalid: %v", err)
	}
}
