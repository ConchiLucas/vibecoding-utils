package system

import (
	"strconv"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbGenerateRecordApi struct{}

func (a *TbGenerateRecordApi) CreateTbGenerateRecord(c *gin.Context) {
	var req system.TbGenerateRecord
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateRecordService.CreateTbGenerateRecord(&req)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbGenerateRecordApi) DeleteTbGenerateRecord(c *gin.Context) {
	var req system.TbGenerateRecord
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateRecordService.DeleteTbGenerateRecord(req)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbGenerateRecordApi) UpdateTbGenerateRecord(c *gin.Context) {
	var req system.TbGenerateRecord
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbGenerateRecordService.UpdateTbGenerateRecord(&req)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbGenerateRecordApi) GetTbGenerateRecord(c *gin.Context) {
	id := c.Query("id")
	res, err := tbGenerateRecordService.GetTbGenerateRecord(id)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateRecordApi) GetTbGenerateRecordList(c *gin.Context) {
	res, err := tbGenerateRecordService.GetTbGenerateRecordList()
	if err != nil {
		global.GVA_LOG.Error("查询列表失败!", zap.Error(err))
		response.FailWithMessage("查询列表失败", c)
	} else {
		response.OkWithData(res, c)
	}
}

func (a *TbGenerateRecordApi) GetGenerateRecordByUser(c *gin.Context) {
	userName := getGenerateRecordUserKey(c)
	projectId, _ := strconv.Atoi(c.Query("projectId"))
	res, err := tbGenerateRecordService.GetLatestGenerateRecord(userName, projectId)
	if err != nil {
		// No record found is not an error — just return empty
		response.OkWithData(nil, c)
	} else {
		response.OkWithData(res, c)
	}
}

func getGenerateRecordUserKey(c *gin.Context) string {
	if userId := c.GetHeader("x-user-id"); userId != "" {
		return userId
	}
	if userId := utils.GetUserID(c); userId > 0 {
		return strconv.FormatUint(uint64(userId), 10)
	}
	if auth := c.GetHeader("Authorization"); auth != "" {
		return auth
	}
	return "anonymous"
}
