package request

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
)

// TbProjectScriptSearch 项目脚本分页查询请求
type TbProjectScriptSearch struct {
	request.PageInfo
	ProjectId int    `json:"projectId" form:"projectId"` // 项目id
	RouteId   int    `json:"routeId" form:"routeId"`     // 关联路由id
	FileName  string `json:"fileName" form:"fileName"`   // 文件名称
}

// ScriptUpdateContentReq 更新脚本内容请求
type ScriptUpdateContentReq struct {
	ID      uint   `json:"id" binding:"required"`      // 脚本记录ID
	Content string `json:"content" binding:"required"` // 新文件内容
}
