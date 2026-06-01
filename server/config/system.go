package config

type System struct {
	DbType        string `mapstructure:"db-type" json:"db-type" yaml:"db-type"`    // 数据库类型:mysql(默认)|sqlite|sqlserver|postgresql
	RouterPrefix  string `mapstructure:"router-prefix" json:"router-prefix" yaml:"router-prefix"`
	Addr          int    `mapstructure:"addr" json:"addr" yaml:"addr"`                  // 端口值
	UseMultipoint bool   `mapstructure:"use-multipoint" json:"use-multipoint" yaml:"use-multipoint"`    // 多点登录拦截
	UseRedis      bool   `mapstructure:"use-redis" json:"use-redis" yaml:"use-redis"`                   // 使用redis
	UseMongo      bool   `mapstructure:"use-mongo" json:"use-mongo" yaml:"use-mongo"`                   // 使用mongo
	DisableAutoMigrate   bool   `mapstructure:"disable-auto-migrate" json:"disable-auto-migrate" yaml:"disable-auto-migrate"`          // 自动迁移数据库表结构，生产环境建议设为false，手动迁移
}
