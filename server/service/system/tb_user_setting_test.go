package system

import (
	"testing"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common"
	modelSystem "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserSettingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	oldDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() {
		global.GVA_DB = oldDB
	})

	if err := db.AutoMigrate(&modelSystem.TbUser{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestSetSelfSettingMergesExistingOriginSetting(t *testing.T) {
	db := setupUserSettingTestDB(t)
	user := modelSystem.TbUser{
		Username: "admin",
		OriginSetting: common.JSONMap{
			"theme": "dark",
		},
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	err := (&UserService{}).SetSelfSetting(common.JSONMap{
		"activeSelection": map[string]interface{}{
			"activeProject":      "我的项目",
			"activeProjectId":    float64(16),
			"activeConnectionId": float64(6),
		},
	}, user.ID)
	if err != nil {
		t.Fatalf("set self setting: %v", err)
	}

	var got modelSystem.TbUser
	if err := db.First(&got, user.ID).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if got.OriginSetting["theme"] != "dark" {
		t.Fatalf("theme = %#v, want dark; full setting: %#v", got.OriginSetting["theme"], got.OriginSetting)
	}
	if _, ok := got.OriginSetting["activeSelection"]; !ok {
		t.Fatalf("activeSelection missing from setting: %#v", got.OriginSetting)
	}
}
