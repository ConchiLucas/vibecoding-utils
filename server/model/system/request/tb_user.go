package request

import (
	common "github.com/flipped-aurora/easy-deploy/server/model/common/request"
)

// Register User register structure
type Register struct {
	Username  string `json:"userName" example:"用户名"`
	Password  string `json:"passWord" example:"密码"`
	NickName  string `json:"nickName" example:"昵称"`
	HeaderImg string `json:"headerImg" example:"头像链接"`
	Enable    int    `json:"enable" swaggertype:"string" example:"int 是否启用"`
	Phone     string `json:"phone" example:"电话号码"`
	Email     string `json:"email" example:"电子邮箱"`
}

// Login User login structure
type Login struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
}

// ChangePasswordReq Modify password structure
type ChangePasswordReq struct {
	ID          uint   `json:"-"`           // 从 JWT 中提取 user id，避免越权
	Password    string `json:"password"`    // 密码
	NewPassword string `json:"newPassword"` // 新密码
}

type ResetPassword struct {
	ID       uint   `json:"ID" form:"ID"`
	Password string `json:"password" form:"password" gorm:"comment:用户登录密码"`
}

type ChangeUserInfo struct {
	ID        uint   `gorm:"primarykey"`
	NickName  string `json:"nickName" gorm:"default:系统用户;comment:用户昵称"`
	Phone     string `json:"phone"  gorm:"comment:用户手机号"`
	Email     string `json:"email"  gorm:"comment:用户邮箱"`
	HeaderImg string `json:"headerImg" gorm:"default:https://qmplusimg.henrongyi.top/gva_header.jpg;comment:用户头像"`
	Enable    int    `json:"enable" gorm:"comment:冻结用户"`
}

type GetUserList struct {
	common.PageInfo
	Username string `json:"username" form:"username"`
	NickName string `json:"nickName" form:"nickName"`
	Phone    string `json:"phone" form:"phone"`
	Email    string `json:"email" form:"email"`
	OrderKey string `json:"orderKey" form:"orderKey"`
	Desc     bool   `json:"desc" form:"desc"`
}
