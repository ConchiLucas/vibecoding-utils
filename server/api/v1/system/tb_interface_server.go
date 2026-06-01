package system

import (
	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbInterfaceServerApi struct{}

func (a *TbInterfaceServerApi) CreateTbInterfaceServer(c *gin.Context) {
	var server system.TbInterfaceServer
	err := c.ShouldBindJSON(&server)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceServerService.CreateTbInterfaceServer(server)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbInterfaceServerApi) DeleteTbInterfaceServer(c *gin.Context) {
	var server system.TbInterfaceServer
	err := c.ShouldBindJSON(&server)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceServerService.DeleteTbInterfaceServer(server)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbInterfaceServerApi) DeleteTbInterfaceServerByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceServerService.DeleteTbInterfaceServerByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbInterfaceServerApi) UpdateTbInterfaceServer(c *gin.Context) {
	var server system.TbInterfaceServer
	err := c.ShouldBindJSON(&server)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbInterfaceServerService.UpdateTbInterfaceServer(&server)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbInterfaceServerApi) GetTbInterfaceServer(c *gin.Context) {
	var server system.TbInterfaceServer
	err := c.ShouldBindQuery(&server)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(server, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	server, err = tbInterfaceServerService.GetTbInterfaceServer(server.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"server": server}, "查询成功", c)
	}
}

func (a *TbInterfaceServerApi) GetTbInterfaceServerList(c *gin.Context) {
	var pageInfo systemReq.ServerSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbInterfaceServerService.GetTbInterfaceServerInfoList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
	} else {
		response.OkWithDetailed(response.PageResult{
			List:     list,
			Total:    total,
			Page:     pageInfo.Page,
			PageSize: pageInfo.PageSize,
		}, "获取成功", c)
	}
}

func (a *TbInterfaceServerApi) ImportSwaggerInterfaces(c *gin.Context) {
	serverName := c.PostForm("serverName")
	projectName := c.PostForm("projectName")

	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("请上传文件", c)
		return
	}

	f, err := file.Open()
	if err != nil {
		response.FailWithMessage("文件打开失败", c)
		return
	}
	defer f.Close()

	fileContent := make([]byte, file.Size)
	_, err = f.Read(fileContent)
	if err != nil {
		response.FailWithMessage("文件读取失败", c)
		return
	}

	// Get username from JWT context
	userName := utils.GetUserName(c)

	err = tbInterfaceServerService.ImportSwaggerInterfaces(fileContent, projectName, serverName, userName)
	if err != nil {
		global.GVA_LOG.Error("导入失败!", zap.Error(err))
		response.FailWithMessage("导入失败: "+err.Error(), c)
	} else {
		response.OkWithMessage("导入成功", c)
	}
}

// RenameServer renames a server node and cascades the new name to all
// related tb_interface / tb_entity / tb_column records.
func (a *TbInterfaceServerApi) RenameServer(c *gin.Context) {
	var req struct {
		ID         uint   `json:"ID"`
		ServerName string `json:"serverName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 || req.ServerName == "" {
		response.FailWithMessage("ID 和新名称不能为空", c)
		return
	}
	if err := tbInterfaceServerService.RenameServer(req.ID, req.ServerName); err != nil {
		global.GVA_LOG.Error("重命名失败!", zap.Error(err))
		response.FailWithMessage("重命名失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("重命名成功", c)
}
