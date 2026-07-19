package system

import (
	"fmt"
	"strings"
	"testing"

	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const correctSharedCompose = `services:
  api:
    image: example/api:1
    ports:
      - "8080:8080"
networks:
  default:
    name: vibedeploy-shared
    external: true
`

func TestNormalizeComposeSharedNetwork(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantChanged bool
		wantErr     bool
	}{
		{
			name:        "already correct external default",
			content:     correctSharedCompose,
			wantChanged: false,
		},
		{
			name: "implicit default",
			content: `services:
  api:
    image: example/api:1
    ports: ["8080:8080"]
`,
			wantChanged: true,
		},
		{
			name: "service project network",
			content: `services:
  api:
    image: example/api:1
    networks: [project]
networks:
  project: {}
`,
			wantChanged: true,
		},
		{
			name: "multiple user networks",
			content: `services:
  api:
    image: example/api:1
    networks:
      - backend
      - metrics
networks:
  backend: {}
  metrics: {}
`,
			wantChanged: true,
		},
		{
			name:    "malformed yaml",
			content: "services:\n  api: [\n",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			normalized, changed, err := normalizeComposeSharedNetwork(test.content)
			if test.wantErr {
				if err == nil {
					t.Fatal("normalizeComposeSharedNetwork() unexpectedly succeeded")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeComposeSharedNetwork() error = %v", err)
			}
			if changed != test.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, test.wantChanged)
			}
			if !changed && normalized != test.content {
				t.Fatal("already-correct Compose content was reformatted")
			}
			if err := validateComposeSharedNetwork(normalized); err != nil {
				t.Fatalf("normalized Compose validation failed: %v\n%s", err, normalized)
			}
			if !strings.Contains(normalized, "example/api:1") {
				t.Fatalf("normalization lost service fields:\n%s", normalized)
			}
			if strings.Contains(test.content, "8080:8080") && !strings.Contains(normalized, "8080:8080") {
				t.Fatalf("normalization lost published ports:\n%s", normalized)
			}
		})
	}
}

func setupComposeNetworkTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&modelSystem.TbProjectScript{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestReconcileStoredComposeScriptsUpdatesOnlyLocalCompose(t *testing.T) {
	db := setupComposeNetworkTestDB(t)
	implicit := "services:\n  api:\n    image: example/api:1\n"
	scripts := []modelSystem.TbProjectScript{
		{ProjectId: 1, RouteId: 1, ScriptType: 1, FileName: "docker-compose.yml", Content: implicit},
		{ProjectId: 2, RouteId: 2, ScriptType: 1, FileName: "compose.yaml", Content: correctSharedCompose},
		{ProjectId: 3, RouteId: 3, ScriptType: 2, FileName: "docker-compose.yml", Content: implicit},
		{ProjectId: 4, RouteId: 4, ScriptType: 1, FileName: "Dockerfile", Content: "FROM scratch\n"},
	}
	if err := db.Create(&scripts).Error; err != nil {
		t.Fatal(err)
	}

	changed, err := reconcileStoredComposeScripts(db)

	if err != nil {
		t.Fatalf("reconcileStoredComposeScripts() error = %v", err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	var reloaded []modelSystem.TbProjectScript
	if err := db.Order("id asc").Find(&reloaded).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateComposeSharedNetwork(reloaded[0].Content); err != nil {
		t.Fatalf("local Compose was not normalized: %v", err)
	}
	if reloaded[1].Content != correctSharedCompose {
		t.Fatal("correct Compose script was unexpectedly rewritten")
	}
	if reloaded[2].Content != implicit {
		t.Fatal("remote-only Compose script was unexpectedly rewritten")
	}
	if reloaded[3].Content != "FROM scratch\n" {
		t.Fatal("non-Compose script was unexpectedly rewritten")
	}
}

func TestReconcileStoredComposeScriptsRollsBackOnMalformedYAML(t *testing.T) {
	db := setupComposeNetworkTestDB(t)
	implicit := "services:\n  api:\n    image: example/api:1\n"
	scripts := []modelSystem.TbProjectScript{
		{ProjectId: 1, RouteId: 1, ScriptType: 1, FileName: "docker-compose.yml", Content: implicit},
		{ProjectId: 2, RouteId: 2, ScriptType: 1, FileName: "compose.yml", Content: "services:\n  api: [\n"},
	}
	if err := db.Create(&scripts).Error; err != nil {
		t.Fatal(err)
	}

	_, err := reconcileStoredComposeScripts(db)

	if err == nil || !strings.Contains(err.Error(), "project=2") {
		t.Fatalf("error = %v, want project/script diagnostic", err)
	}
	var first modelSystem.TbProjectScript
	if err := db.First(&first, scripts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if first.Content != implicit {
		t.Fatal("transaction did not roll back an earlier Compose update")
	}
}
