package request

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
)

// TbServerSearch 服务器分页查询请求
type TbServerSearch struct {
	request.PageInfo
	ServerName string `json:"serverName" form:"serverName"` // 服务器名称
	ServerIp   string `json:"serverIp" form:"serverIp"`     // 服务器ip
	UserId     uint   `json:"userId" form:"userId"`         // 用户ID
}
