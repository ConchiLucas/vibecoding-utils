package system

type TbGenerateRecord struct {
	ID             uint   `gorm:"primarykey" json:"ID"`
	ProjectId      int    `json:"projectId" gorm:"comment:生成项目id"`
	UserName       string `json:"userName" gorm:"type:varchar(255);comment:用户名称"`
	ModuleName     string `json:"moduleName" gorm:"type:varchar(255);comment:模块名称"`
	ModuleComment  string `json:"moduleComment" gorm:"type:varchar(255);comment:模块注释"`
	TableStructure string `json:"tableStructure" gorm:"type:text;comment:表结构"`
	DbType         string `json:"dbType" gorm:"type:varchar(64);default:mysql;comment:数据库类型"`
}

func (TbGenerateRecord) TableName() string {
	return "tb_generate_record"
}
