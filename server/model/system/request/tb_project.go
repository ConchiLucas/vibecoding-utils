package request

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
)

// TbProjectSearch 项目分页查询请求
type TbProjectSearch struct {
	request.PageInfo
	ProjectName      string `json:"projectName" form:"projectName"`           // 项目名称
	ComputerLanguage string `json:"computerLanguage" form:"computerLanguage"` // 语言类型
	UserId           uint   `json:"userId" form:"userId"`                     // 用户ID
	ServerName       string `json:"serverName" form:"serverName"`             // 服务器名称
}
