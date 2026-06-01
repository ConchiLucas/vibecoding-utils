package system

import (
	"fmt"

	"github.com/flipped-aurora/easy-deploy/server/global"
	"github.com/flipped-aurora/easy-deploy/server/model/common/request"
	"github.com/flipped-aurora/easy-deploy/server/model/common/response"
	"github.com/flipped-aurora/easy-deploy/server/model/system"
	systemReq "github.com/flipped-aurora/easy-deploy/server/model/system/request"
	"github.com/flipped-aurora/easy-deploy/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TbConnectionApi struct{}

func (a *TbConnectionApi) CreateTbConnection(c *gin.Context) {
	var conn system.TbConnection
	err := c.ShouldBindJSON(&conn)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbConnectionService.CreateTbConnection(conn)
	if err != nil {
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
	} else {
		response.OkWithMessage("创建成功", c)
	}
}

func (a *TbConnectionApi) DeleteTbConnection(c *gin.Context) {
	var conn system.TbConnection
	err := c.ShouldBindJSON(&conn)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbConnectionService.DeleteTbConnection(conn)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
	} else {
		response.OkWithMessage("删除成功", c)
	}
}

func (a *TbConnectionApi) DeleteTbConnectionByIds(c *gin.Context) {
	var IDS request.IdsReq
	err := c.ShouldBindJSON(&IDS)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbConnectionService.DeleteTbConnectionByIds(IDS.Ids)
	if err != nil {
		global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败", c)
	} else {
		response.OkWithMessage("批量删除成功", c)
	}
}

func (a *TbConnectionApi) UpdateTbConnection(c *gin.Context) {
	var conn system.TbConnection
	err := c.ShouldBindJSON(&conn)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = tbConnectionService.UpdateTbConnection(&conn)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
	} else {
		response.OkWithMessage("更新成功", c)
	}
}

func (a *TbConnectionApi) GetTbConnection(c *gin.Context) {
	var conn system.TbConnection
	err := c.ShouldBindQuery(&conn)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(conn, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	conn, err = tbConnectionService.GetTbConnection(conn.ID)
	if err != nil {
		global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败", c)
	} else {
		response.OkWithDetailed(gin.H{"connection": conn}, "查询成功", c)
	}
}

func (a *TbConnectionApi) GetTbConnectionList(c *gin.Context) {
	var pageInfo systemReq.ConnectionSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := tbConnectionService.GetTbConnectionInfoList(pageInfo)
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

// TestConnection pings the target database and returns success/fail.
func (a *TbConnectionApi) TestConnection(c *gin.Context) {
	var conn system.TbConnection
	if err := c.ShouldBindQuery(&conn); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbConnectionService.TestConnection(conn.ID); err != nil {
		global.GVA_LOG.Error("连接测试失败!", zap.Error(err))
		response.FailWithMessage("连接失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("连接成功", c)
}

// TestConnectionPayload pings the target database using payload.
func (a *TbConnectionApi) TestConnectionPayload(c *gin.Context) {
	var conn system.TbConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbConnectionService.TestConnectionByPayload(conn); err != nil {
		global.GVA_LOG.Error("连接测试失败!", zap.Error(err))
		response.FailWithMessage("连接失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("连接成功", c)
}

// InitConnection imports all tables and columns from the target database.
func (a *TbConnectionApi) InitConnection(c *gin.Context) {
	var conn system.TbConnection
	if err := c.ShouldBindQuery(&conn); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userName := utils.GetUserName(c)
	if err := tbConnectionService.InitConnection(conn.ID, userName); err != nil {
		global.GVA_LOG.Error("导入表结构失败!", zap.Error(err))
		response.FailWithMessage("导入失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("导入成功", c)
}

// GetRemoteTables returns all table names from the target database in real-time.
func (a *TbConnectionApi) GetRemoteTables(c *gin.Context) {
	var conn system.TbConnection
	if err := c.ShouldBindQuery(&conn); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	tables, err := tbConnectionService.GetRemoteTables(conn.ID, conn.DatabaseName)
	if err != nil {
		global.GVA_LOG.Error("获取远程表列表失败!", zap.Error(err))
		response.FailWithMessage("获取表列表失败: "+err.Error(), c)
		return
	}
	response.OkWithData(tables, c)
}

// GetRemoteDatabases returns database names from configured data sources in real-time.
func (a *TbConnectionApi) GetRemoteDatabases(c *gin.Context) {
	var query systemReq.ConnectionSearch
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	databases, err := tbConnectionService.GetRemoteDatabases(query.ConnectionGroup, query.EnvName, query.ID)
	if err != nil {
		global.GVA_LOG.Error("获取远程数据库列表失败!", zap.Error(err))
		response.FailWithMessage("获取数据库列表失败: "+err.Error(), c)
		return
	}
	response.OkWithData(databases, c)
}

// GetRemoteTablePreview returns a single record from the table at a given offset,
// along with column descriptions and total count.
func (a *TbConnectionApi) GetRemoteTablePreview(c *gin.Context) {
	connIDStr := c.Query("ID")
	databaseName := c.Query("databaseName")
	tableName := c.Query("tableName")
	offsetStr := c.DefaultQuery("offset", "0")
	filterColumn := c.Query("filterColumn")
	filterValue := c.Query("filterValue")

	if connIDStr == "" || tableName == "" {
		response.FailWithMessage("缺少必要参数 ID 或 tableName", c)
		return
	}

	var connID uint
	if _, err := fmt.Sscanf(connIDStr, "%d", &connID); err != nil {
		response.FailWithMessage("ID 参数格式错误", c)
		return
	}

	var offset int
	if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
		offset = 0
	}

	preview, err := tbConnectionService.PreviewTableRecord(connID, databaseName, tableName, offset, filterColumn, filterValue)
	if err != nil {
		global.GVA_LOG.Error("预览表记录失败!", zap.Error(err))
		response.FailWithMessage("预览失败: "+err.Error(), c)
		return
	}
	response.OkWithData(preview, c)
}

// GetRemoteTableDDL returns a CREATE TABLE statement for the selected table.
func (a *TbConnectionApi) GetRemoteTableDDL(c *gin.Context) {
	connIDStr := c.Query("ID")
	databaseName := c.Query("databaseName")
	tableName := c.Query("tableName")

	if connIDStr == "" || tableName == "" {
		response.FailWithMessage("缺少必要参数 ID 或 tableName", c)
		return
	}

	var connID uint
	if _, err := fmt.Sscanf(connIDStr, "%d", &connID); err != nil {
		response.FailWithMessage("ID 参数格式错误", c)
		return
	}

	ddl, err := tbConnectionService.GetRemoteTableDDL(connID, databaseName, tableName)
	if err != nil {
		global.GVA_LOG.Error("获取建表 SQL 失败!", zap.Error(err))
		response.FailWithMessage("获取建表 SQL 失败: "+err.Error(), c)
		return
	}
	response.OkWithData(ddl, c)
}

// GetRemoteTableComments returns all table comments for a database/schema.
func (a *TbConnectionApi) GetRemoteTableComments(c *gin.Context) {
	connIDStr := c.Query("ID")
	databaseName := c.Query("databaseName")
	if connIDStr == "" {
		response.FailWithMessage("缺少必要参数 ID", c)
		return
	}

	var connID uint
	if _, err := fmt.Sscanf(connIDStr, "%d", &connID); err != nil {
		response.FailWithMessage("ID 参数格式错误", c)
		return
	}

	comments, err := tbConnectionService.GetRemoteTableComments(connID, databaseName)
	if err != nil {
		global.GVA_LOG.Error("获取远程表注释失败!", zap.Error(err))
		response.FailWithMessage("获取表注释失败: "+err.Error(), c)
		return
	}
	response.OkWithData(comments, c)
}

// QueryRemoteSQL runs a read-only query and returns a preview of the first rows.
func (a *TbConnectionApi) QueryRemoteSQL(c *gin.Context) {
	var req systemReq.RemoteSQLQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("缺少必要参数 ID", c)
		return
	}
	result, err := tbConnectionService.QueryRemoteSQL(req.ID, req.DatabaseName, req.SQL, req.Limit)
	if err != nil {
		global.GVA_LOG.Error("SQL 查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败: "+err.Error(), c)
		return
	}
	response.OkWithData(result, c)
}

// GetRemoteSQLHistory returns successful SQL query snapshots scoped by project
// and user, optionally filtered by environment, data source, and database.
func (a *TbConnectionApi) GetRemoteSQLHistory(c *gin.Context) {
	var req systemReq.RemoteSQLHistoryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, err := tbConnectionService.ListRemoteSQLHistory(req, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("获取 SQL 历史失败!", zap.Error(err))
		response.FailWithMessage("获取 SQL 历史失败: "+err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

// SaveRemoteSQLHistory stores a successful SQL query without storing result rows.
func (a *TbConnectionApi) SaveRemoteSQLHistory(c *gin.Context) {
	var req systemReq.RemoteSQLHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ConnectionID == 0 || req.SQL == "" {
		response.FailWithMessage("缺少必要参数 connectionId 或 sql", c)
		return
	}
	list, err := tbConnectionService.SaveRemoteSQLHistory(req, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("保存 SQL 历史失败!", zap.Error(err))
		response.FailWithMessage("保存 SQL 历史失败: "+err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

func (a *TbConnectionApi) DeleteRemoteSQLHistory(c *gin.Context) {
	var req systemReq.RemoteSQLHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 {
		response.FailWithMessage("缺少必要参数 id", c)
		return
	}
	list, err := tbConnectionService.DeleteRemoteSQLHistory(req, utils.GetUserName(c))
	if err != nil {
		global.GVA_LOG.Error("删除 SQL 历史失败!", zap.Error(err))
		response.FailWithMessage("删除 SQL 历史失败: "+err.Error(), c)
		return
	}
	response.OkWithData(list, c)
}

func (a *TbConnectionApi) ClearRemoteSQLHistory(c *gin.Context) {
	var req systemReq.RemoteSQLHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := tbConnectionService.ClearRemoteSQLHistory(req, utils.GetUserName(c)); err != nil {
		global.GVA_LOG.Error("清空 SQL 历史失败!", zap.Error(err))
		response.FailWithMessage("清空 SQL 历史失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("清空成功", c)
}

// UpdateRemoteTableRecord updates field values for the currently previewed row.
func (a *TbConnectionApi) UpdateRemoteTableRecord(c *gin.Context) {
	var req systemReq.UpdateRemoteTableRecordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if req.ID == 0 || req.TableName == "" {
		response.FailWithMessage("缺少必要参数 ID 或 tableName", c)
		return
	}
	if len(req.Changes) == 0 {
		response.FailWithMessage("没有需要修改的字段", c)
		return
	}

	changes := make(map[string]string, len(req.Changes))
	for _, change := range req.Changes {
		if change.Name == "" {
			response.FailWithMessage("字段名不能为空", c)
			return
		}
		changes[change.Name] = change.Value
	}

	preview, err := tbConnectionService.UpdateTableRecord(req.ID, req.DatabaseName, req.TableName, req.Offset, changes, req.FilterColumn, req.FilterValue)
	if err != nil {
		global.GVA_LOG.Error("修改表记录失败!", zap.Error(err))
		response.FailWithMessage("修改失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(preview, "修改成功", c)
}
