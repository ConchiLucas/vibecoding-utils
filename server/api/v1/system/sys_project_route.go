package system

import (
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/gin-gonic/gin"
)

type ProjectRouteApi struct{}

// SaveOrUpdateRoute
// @Tags      ProjectRoute
// @Summary   创建或更新部署路由配置
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{msg=string}  "成功"
// @Router    /projectRoute/saveOrUpdate [post]
func (a *ProjectRouteApi) SaveOrUpdateRoute(c *gin.Context) {
	var route system.TbProjectRoute
	err := c.ShouldBindJSON(&route)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = projectRouteService.SaveOrUpdateRoute(route)
	if err != nil {
		response.FailWithMessage("保存失败", c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

// DeleteRoute
// @Tags      ProjectRoute
// @Summary   删除路由配置
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{msg=string}  "成功"
// @Router    /projectRoute/deleteRoute [delete]
func (a *ProjectRouteApi) DeleteRoute(c *gin.Context) {
	var route system.TbProjectRoute
	err := c.ShouldBindJSON(&route)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = projectRouteService.DeleteRoute(route.ID)
	if err != nil {
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
