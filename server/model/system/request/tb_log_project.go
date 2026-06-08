package request

import "github.com/flipped-aurora/easy-deploy/server/model/common/request"

// TbLogProjectSearch 日志项目分页查询请求
type TbLogProjectSearch struct {
	request.PageInfo
	ProjectName       string `json:"projectName" form:"projectName"`
	ProjectConfigId   uint   `json:"projectConfigId" form:"projectConfigId"`
	ProjectConfigName string `json:"projectConfigName" form:"projectConfigName"`
	ComputerLanguage  string `json:"computerLanguage" form:"computerLanguage"`
	GroupId           uint   `json:"groupId" form:"groupId"`
	UserId            uint   `json:"userId" form:"userId"`
}
