package request

import "github.com/flipped-aurora/easy-deploy/server/model/common/request"

type DevelopmentPrepareSearch struct {
	request.PageInfo
	ProjectConfigId   uint   `json:"projectConfigId" form:"projectConfigId"`
	ProjectConfigName string `json:"projectConfigName" form:"projectConfigName"`
	BusinessGroup     string `json:"businessGroup" form:"businessGroup"`
	ItemType          string `json:"itemType" form:"itemType"`
	UserId            uint   `json:"userId" form:"userId"`
}
