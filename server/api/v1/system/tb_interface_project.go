package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbInterfaceProjectApi struct{}

func (a *TbInterfaceProjectApi) CreateTbInterfaceProject(c *gin.Context) {
	var project system.TbInterfaceProject
	err := c.ShouldBindJSON(&project)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	project.UserName = utils.GetUserName(c)
	err = tbInterfaceProjectService.CreateTbInterfaceProject(project)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败: "+err.Error(), c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbInterfaceProjectApi) DeleteTbInterfaceProject(c *gin.Context) {
	var project system.TbInterfaceProject
	err := c.ShouldBindJSON(&project)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceProjectService.DeleteTbInterfaceProject(project)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbInterfaceProjectApi) GetTbInterfaceProjectList(c *gin.Context) {
	list, err := tbInterfaceProjectService.GetTbInterfaceProjectList()
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithData(list, c)
	}
}

func (a *TbInterfaceProjectApi) UpdateTbInterfaceProject(c *gin.Context) {
	var project system.TbInterfaceProject
	err := c.ShouldBindJSON(&project)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if project.ID == 0 {
		response.FailWithMessage("ID不能为空", c)
		return
	}
	err = tbInterfaceProjectService.UpdateTbInterfaceProject(project)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败: "+err.Error(), c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}
