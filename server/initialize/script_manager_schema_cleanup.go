package initialize

import "gorm.io/gorm"

func dropRemovedScriptManagerColumns(db *gorm.DB) {
	if db == nil {
		return
	}
	db.Exec("ALTER TABLE tb_script_category DROP COLUMN IF EXISTS sort;")

	db.Exec("ALTER TABLE tb_script_workflow DROP COLUMN IF EXISTS color;")
	db.Exec("ALTER TABLE tb_script_workflow DROP COLUMN IF EXISTS sort;")

	db.Exec("ALTER TABLE tb_script_step DROP COLUMN IF EXISTS sort;")
	db.Exec("ALTER TABLE tb_script_step DROP COLUMN IF EXISTS server_id;")
	db.Exec("ALTER TABLE tb_script_step DROP COLUMN IF EXISTS working_dir;")
	db.Exec("ALTER TABLE tb_script_step DROP COLUMN IF EXISTS enabled;")
	db.Exec("ALTER TABLE tb_script_step DROP COLUMN IF EXISTS stop_on_error;")
}
