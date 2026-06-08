package system

import (
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/easy-deploy/server/model/common"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//@author: [piexlmax](https://github.com/piexlmax)
//@function: Register
//@description: 用户注册
//@param: u model.TbUser
//@return: userInter system.TbUser, err error

type UserService struct{}

var UserServiceApp = new(UserService)

func (userService *UserService) Register(u system.TbUser) (userInter system.TbUser, err error) {
	var user system.TbUser
	if !errors.Is(global.GVA_DB.Where("username = ?", u.Username).First(&user).Error, gorm.ErrRecordNotFound) { // 判断用户名是否注册
		return userInter, errors.New("用户名已注册")
	}
	// 否则 附加uuid 密码hash加密 注册
	u.Password = utils.BcryptHash(u.Password)
	u.UUID = uuid.New()
	err = global.GVA_DB.Create(&u).Error
	return u, err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@author: [SliverHorn](https://github.com/SliverHorn)
//@function: Login
//@description: 用户登录
//@param: u *model.TbUser
//@return: err error, userInter *model.TbUser

func (userService *UserService) Login(u *system.TbUser) (userInter *system.TbUser, err error) {
	if nil == global.GVA_DB {
		return nil, fmt.Errorf("db not init")
	}

	var user system.TbUser
	err = global.GVA_DB.Where("username = ?", u.Username).First(&user).Error
	if err == nil {
		if ok := utils.BcryptCheck(u.Password, user.Password); !ok {
			return nil, errors.New("密码错误")
		}
	}
	return &user, err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: ChangePassword
//@description: 修改用户密码
//@param: u *model.TbUser, newPassword string
//@return: err error

func (userService *UserService) ChangePassword(u *system.TbUser, newPassword string) (err error) {
	var user system.TbUser
	err = global.GVA_DB.Select("id, password").Where("id = ?", u.ID).First(&user).Error
	if err != nil {
		return err
	}
	if ok := utils.BcryptCheck(u.Password, user.Password); !ok {
		return errors.New("原密码错误")
	}
	pwd := utils.BcryptHash(newPassword)
	err = global.GVA_DB.Model(&user).Update("password", pwd).Error
	return err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: GetUserInfoList
//@description: 分页获取数据
//@param: info request.PageInfo
//@return: err error, list interface{}, total int64

func (userService *UserService) GetUserInfoList(info systemReq.GetUserList) (list interface{}, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&system.TbUser{})
	var userList []system.TbUser

	if info.NickName != "" {
		db = db.Where("nick_name LIKE ?", "%"+info.NickName+"%")
	}
	if info.Phone != "" {
		db = db.Where("phone LIKE ?", "%"+info.Phone+"%")
	}
	if info.Username != "" {
		db = db.Where("username LIKE ?", "%"+info.Username+"%")
	}
	if info.Email != "" {
		db = db.Where("email LIKE ?", "%"+info.Email+"%")
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	orderStr := "id desc"
	if info.OrderKey != "" {
		allowedOrders := map[string]bool{
			"id":        true,
			"username":  true,
			"nick_name": true,
			"phone":     true,
			"email":     true,
		}
		if allowedOrders[info.OrderKey] {
			orderStr = info.OrderKey
			if info.Desc {
				orderStr = info.OrderKey + " desc"
			}
		}
	}

	err = db.Limit(limit).Offset(offset).Order(orderStr).Find(&userList).Error
	return userList, total, err
}

func (userService *UserService) DeleteUser(id int) (err error) {
	return global.GVA_DB.Where("id = ?", id).Unscoped().Delete(&system.TbUser{}).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: SetUserInfo
//@description: 设置用户信息
//@param: reqUser model.TbUser
//@return: err error, user model.TbUser

func (userService *UserService) SetUserInfo(req system.TbUser) error {
	return global.GVA_DB.Model(&system.TbUser{}).
		Select("updated_at", "nick_name", "header_img", "phone", "email", "enable").
		Where("id=?", req.ID).
		Updates(map[string]interface{}{
			"updated_at": time.Now(),
			"nick_name":  req.NickName,
			"header_img": req.HeaderImg,
			"phone":      req.Phone,
			"email":      req.Email,
			"enable":     req.Enable,
		}).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: SetSelfInfo
//@description: 设置用户信息
//@param: reqUser model.TbUser
//@return: err error, user model.TbUser

func (userService *UserService) SetSelfInfo(req system.TbUser) error {
	return global.GVA_DB.Model(&system.TbUser{}).
		Where("id=?", req.ID).
		Updates(req).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: SetSelfSetting
//@description: 设置用户配置
//@param: req datatypes.JSON, uid uint
//@return: err error

func (userService *UserService) SetSelfSetting(req common.JSONMap, uid uint) error {
	var user system.TbUser
	if err := global.GVA_DB.Select("id", "origin_setting").Where("id = ?", uid).First(&user).Error; err != nil {
		return err
	}
	nextSetting := common.JSONMap{}
	for key, value := range user.OriginSetting {
		nextSetting[key] = value
	}
	for key, value := range req {
		nextSetting[key] = value
	}
	return global.GVA_DB.Model(&system.TbUser{}).Where("id = ?", uid).Update("origin_setting", nextSetting).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@author: [SliverHorn](https://github.com/SliverHorn)
//@function: GetUserInfo
//@description: 获取用户信息
//@param: uuid uuid.UUID
//@return: err error, user system.TbUser

func (userService *UserService) GetUserInfo(uuid uuid.UUID) (user system.TbUser, err error) {
	var reqUser system.TbUser
	err = global.GVA_DB.First(&reqUser, "uuid = ?", uuid).Error
	if err != nil {
		return reqUser, err
	}
	return reqUser, err
}

//@author: [SliverHorn](https://github.com/SliverHorn)
//@function: FindUserById
//@description: 通过id获取用户信息
//@param: id int
//@return: err error, user *model.TbUser

func (userService *UserService) FindUserById(id int) (user *system.TbUser, err error) {
	var u system.TbUser
	err = global.GVA_DB.Where("id = ?", id).First(&u).Error
	return &u, err
}

//@author: [SliverHorn](https://github.com/SliverHorn)
//@function: FindUserByUuid
//@description: 通过uuid获取用户信息
//@param: uuid string
//@return: err error, user *model.TbUser

func (userService *UserService) FindUserByUuid(uuid string) (user *system.TbUser, err error) {
	var u system.TbUser
	if err = global.GVA_DB.Where("uuid = ?", uuid).First(&u).Error; err != nil {
		return &u, errors.New("用户不存在")
	}
	return &u, nil
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: ResetPassword
//@description: 修改用户密码
//@param: ID uint
//@return: err error

func (userService *UserService) ResetPassword(ID uint, password string) (err error) {
	err = global.GVA_DB.Model(&system.TbUser{}).Where("id = ?", ID).Update("password", utils.BcryptHash(password)).Error
	return err
}
