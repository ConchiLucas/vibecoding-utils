package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common"
	"github.com/google/uuid"
)

type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() uint
	GetUserInfo() any
}

var _ Login = new(TbUser)

type TbUser struct {
	global.GVA_MODEL
	UUID          uuid.UUID      `json:"uuid" gorm:"index;comment:用户UUID"`
	Username      string         `json:"userName" gorm:"index;comment:用户登录名"`
	Password      string         `json:"-"  gorm:"comment:用户登录密码"`
	NickName      string         `json:"nickName" gorm:"default:系统用户;comment:用户昵称"`
	HeaderImg     string         `json:"headerImg" gorm:"default:https://qmplusimg.henrongyi.top/gva_header.jpg;comment:用户头像"`
	Phone         string         `json:"phone"  gorm:"comment:用户手机号"`
	Email         string         `json:"email"  gorm:"comment:用户邮箱"`
	Enable        int            `json:"enable" gorm:"default:1;comment:用户是否被冻结 1正常 2冻结"`
	OriginSetting common.JSONMap `json:"originSetting" form:"originSetting" gorm:"type:text;default:null;column:origin_setting;comment:配置;"`
}

func (TbUser) TableName() string {
	return "tb_user"
}

func (s *TbUser) GetUsername() string {
	return s.Username
}

func (s *TbUser) GetNickname() string {
	return s.NickName
}

func (s *TbUser) GetUUID() uuid.UUID {
	return s.UUID
}

func (s *TbUser) GetUserId() uint {
	return s.ID
}

func (s *TbUser) GetUserInfo() any {
	return *s
}
