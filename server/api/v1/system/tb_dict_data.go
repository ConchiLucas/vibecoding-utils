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

type TbDictDataApi struct{}

// CreateDictData 创建字典数据
// @Tags TbDictData
// @Summary 创建字典数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body system.TbDictData true "创建字典数据"
// @Success 200 {object} response.Response{msg=string} "创建成功"
// @Router /dict/createDictData [post]
func (a *TbDictDataApi) CreateDictData(c *gin.Context) {
	var dictData system.TbDictData
	err := c.ShouldBindJSON(&dictData)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbDictDataService.CreateDictData(dictData)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

// DeleteDictData 删除字典数据
// @Tags TbDictData
// @Summary 删除字典数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body system.TbDictData true "字典数据模型"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /dict/deleteDictData [delete]
func (a *TbDictDataApi) DeleteDictData(c *gin.Context) {
	var dictData system.TbDictData
	err := c.ShouldBindJSON(&dictData)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbDictDataService.DeleteDictData(dictData)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

// DeleteDictDataByIds 批量删除字典数据
// @Tags TbDictData
// @Summary 批量删除字典数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.IdsReq true "批量删除字典数据"
// @Success 200 {object} response.Response{msg=string} "批量删除成功"
// @Router /dict/deleteDictDataByIds [delete]
func (a *TbDictDataApi) DeleteDictDataByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbDictDataService.DeleteDictDataByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

// UpdateDictData 更新字典数据
// @Tags TbDictData
// @Summary 更新字典数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body system.TbDictData true "更新字典数据"
// @Success 200 {object} response.Response{msg=string} "更新成功"
// @Router /dict/updateDictData [put]
func (a *TbDictDataApi) UpdateDictData(c *gin.Context) {
	var dictData system.TbDictData
	err := c.ShouldBindJSON(&dictData)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbDictDataService.UpdateDictData(&dictData)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

// GetDictData 用id查询字典数据
// @Tags TbDictData
// @Summary 用id查询字典数据
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query system.TbDictData true "用id查询字典数据"
// @Success 200 {object} response.Response{data=object{dictData=system.TbDictData},msg=string} "查询成功"
// @Router /dict/getDictData [get]
func (a *TbDictDataApi) GetDictData(c *gin.Context) {
	var dictData system.TbDictData
	err := c.ShouldBindQuery(&dictData)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(dictData, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	dictData, err = tbDictDataService.GetDictData(dictData.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"dictData": dictData}, "查询成功", c)
	}
}

// GetDictDataList 分页获取字典数据列表
// @Tags TbDictData
// @Summary 分页获取字典数据列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query systemReq.DictDataSearch true "分页获取字典数据列表"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /dict/getDictDataList [get]
func (a *TbDictDataApi) GetDictDataList(c *gin.Context) {
	var pageInfo systemReq.DictDataSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbDictDataService.GetDictDataInfoList(pageInfo)
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
