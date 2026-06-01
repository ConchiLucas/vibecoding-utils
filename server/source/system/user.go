package system

import (
	"context"
	sysModel "github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/service/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const initOrderUser = system.InitOrderInternal + 1

type initUser struct{}

// auto run
func init() {
	system.RegisterInit(initOrderUser, &initUser{})
}

func (i *initUser) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(&sysModel.TbUser{})
}

func (i *initUser) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&sysModel.TbUser{})
}

func (i *initUser) InitializerName() string {
	return sysModel.TbUser{}.TableName()
}

func (i *initUser) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	ap := ctx.Value("adminPassword")
	apStr, ok := ap.(string)
	if !ok {
		apStr = "123456"
	}

	password := utils.BcryptHash(apStr)
	adminPassword := utils.BcryptHash(apStr)

	entities := []sysModel.TbUser{
		{
			UUID:      uuid.New(),
			Username:  "admin",
			Password:  adminPassword,
			NickName:  "管理员",
			HeaderImg: "https://qmplusimg.henrongyi.top/gva_header.jpg",
			Phone:     "17611111111",
			Email:     "333333333@qq.com",
		},
		{
			UUID:      uuid.New(),
			Username:  "user1",
			Password:  password,
			NickName:  "用户1",
			HeaderImg: "https://qmplusimg.henrongyi.top/1572075907logo.png",
			Phone:     "17611111111",
			Email:     "333333333@qq.com",
		},
	}
	if err = db.Create(&entities).Error; err != nil {
		return ctx, errors.Wrap(err, sysModel.TbUser{}.TableName()+"表数据初始化失败!")
	}
	next = context.WithValue(ctx, i.InitializerName(), entities)
	return next, nil
}

func (i *initUser) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	if errors.Is(db.Where("username = ?", "admin").First(&sysModel.TbUser{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
