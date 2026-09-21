package initialize

import (
	"strings"
	"testing"
)

func TestDefaultPostgresScriptsDoNotRestoreSourceRoles(t *testing.T) {
	t.Parallel()

	scripts := map[string]string{
		"local database export":  defaultEasyDeployExportScript,
		"remote database export": defaultPostgresRemoteDatabaseExportScript,
		"local table export":     defaultPostgresTableExportScript,
		"remote table export":    defaultPostgresRemoteTableExportScript,
		"database restore":       defaultEasyDeployTargetExecScript,
		"table restore":          defaultPostgresTableTargetExecScript,
	}

	for name, script := range scripts {
		script := script
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, option := range []string{"--no-owner", "--no-acl"} {
				if !strings.Contains(script, option) {
					t.Errorf("PostgreSQL script is missing %s", option)
				}
			}
		})
	}
}

func TestPostgresScriptsWithoutNoACLAreRefreshed(t *testing.T) {
	t.Parallel()

	stepNames := []string{
		defaultExportStepName,
		reverseExportStepName,
		defaultTableExportStepName,
		reverseTableExportStepName,
		"目标服务器导入数据库",
		"本地服务器导入 PostgreSQL 数据库",
		"目标服务器导入 PostgreSQL 单表",
	}

	for _, stepName := range stepNames {
		stepName := stepName
		t.Run(stepName, func(t *testing.T) {
			t.Parallel()
			if !shouldRefreshDefaultStepScript(stepName, "current script without PostgreSQL ACL filtering") {
				t.Errorf("step %q should refresh when --no-acl is missing", stepName)
			}
		})
	}
}
